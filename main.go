package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

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

	cfg, err := LoadConfig(*cfgPath)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	trusted, err := ParseNets(cfg.TrustedProxyIPs)
	if err != nil {
		log.Fatal().Err(err).Msg("trustedProxyIPs")
	}
	allowed, err := ParseNets(cfg.Admin.AllowedIPs)
	if err != nil {
		log.Fatal().Err(err).Msg("admin allowedIPs")
	}
	if allowed == nil {
		allowed = []*net.IPNet{} // empty list denies everyone
	}

	store := NewStore()
	pub := newServer(cfg.Public.Port, withAccessLog("public", log, trusted, nil, publicHandler(store)))
	adm := newServer(cfg.Admin.Port, withAccessLog("admin", log, trusted, allowed, adminHandler(store, log)))

	errCh := make(chan error, 2)
	go func() { log.Info().Int("port", cfg.Public.Port).Msg("public listening"); errCh <- pub.ListenAndServe() }()
	go func() { log.Info().Int("port", cfg.Admin.Port).Msg("admin listening"); errCh <- adm.ListenAndServe() }()
	log.Fatal().Err(<-errCh).Msg("server stopped")
}
