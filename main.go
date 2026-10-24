package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Links2004/acme-challenge-outpost/admin"
	"github.com/Links2004/acme-challenge-outpost/config"
	"github.com/Links2004/acme-challenge-outpost/health"
	"github.com/Links2004/acme-challenge-outpost/httpx"
	"github.com/Links2004/acme-challenge-outpost/public"
	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/rs/zerolog"
)

// shutdownTimeout stays below the k8s default terminationGracePeriodSeconds (30s)
// and systemd default TimeoutStopSec (90s).
const shutdownTimeout = 20 * time.Second

func newServer(port int, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
}

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to yaml config file")
	flag.Parse()
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	allowed, err := httpx.ParseNets(cfg.Admin.AllowedIPs)
	if err != nil {
		log.Fatal().Err(err).Msg("admin allowedIPs")
	}
	if allowed == nil {
		allowed = []*net.IPNet{} // empty list denies everyone
	}

	st := store.New()
	probe := health.New()
	pubH, err := httpx.WithAccessLog("public", log, cfg.TrustedProxyIPs, nil, public.Handler(st, probe))
	if err != nil {
		log.Fatal().Err(err).Msg("trustedProxyIPs")
	}
	admH, err := httpx.WithAccessLog("admin", log, cfg.TrustedProxyIPs, allowed, admin.Handler(st, log))
	if err != nil {
		log.Fatal().Err(err).Msg("trustedProxyIPs")
	}
	pub := newServer(cfg.Public.Port, pubH)
	adm := newServer(cfg.Admin.Port, admH)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	log.Info().Int("public_port", cfg.Public.Port).Int("admin_port", cfg.Admin.Port).Msg("starting")
	if err := serve(ctx, log, shutdownTimeout, probe.SetNotReady, pub, adm); err != nil {
		log.Error().Err(err).Msg("server stopped")
		stop()
		os.Exit(1)
	}
	log.Info().Msg("shutdown complete")
}

// serve runs all servers until ctx is cancelled (SIGTERM from k8s/systemd or
// SIGINT) or one of them fails, then calls beforeShutdown (e.g. to fail readiness) and gracefully shuts all of them down,
// waiting at most timeout for in-flight requests.
func serve(ctx context.Context, log zerolog.Logger, timeout time.Duration, beforeShutdown func(), servers ...*http.Server) error {
	errCh := make(chan error, len(servers))
	for _, srv := range servers {
		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
				return
			}
			errCh <- nil
		}()
	}

	var result error
	select {
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received")
	case result = <-errCh:
	}

	if beforeShutdown != nil {
		beforeShutdown()
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, srv := range servers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.Shutdown(shutCtx); err != nil {
				mu.Lock()
				result = errors.Join(result, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return result
}
