package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Links2004/acme-challenge-outpost/admin"
	"github.com/Links2004/acme-challenge-outpost/config"
	"github.com/Links2004/acme-challenge-outpost/httpx"
	"github.com/Links2004/acme-challenge-outpost/public"
	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/rs/zerolog"
)

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
	trusted, err := httpx.ParseNets(cfg.TrustedProxyIPs)
	if err != nil {
		log.Fatal().Err(err).Msg("trustedProxyIPs")
	}
	allowed, err := httpx.ParseNets(cfg.Admin.AllowedIPs)
	if err != nil {
		log.Fatal().Err(err).Msg("admin allowedIPs")
	}
	if allowed == nil {
		allowed = []*net.IPNet{} // empty list denies everyone
	}

	st := store.New()
	pub := newServer(cfg.Public.Port, httpx.WithAccessLog("public", log, trusted, nil, public.Handler(st)))
	adm := newServer(cfg.Admin.Port, httpx.WithAccessLog("admin", log, trusted, allowed, admin.Handler(st, log)))

	errCh := make(chan error, 2)
	go func() { log.Info().Int("port", cfg.Public.Port).Msg("public listening"); errCh <- pub.ListenAndServe() }()
	go func() { log.Info().Int("port", cfg.Admin.Port).Msg("admin listening"); errCh <- adm.ListenAndServe() }()
	log.Fatal().Err(<-errCh).Msg("server stopped")
}
