package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

const acmePrefix = "/.well-known/acme-challenge/"

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// publicHandler serves only /.well-known/acme-challenge/{token}.
func publicHandler(store *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(acmePrefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimPrefix(r.URL.Path, acmePrefix)
		keyAuth, ok := store.Get(token)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, keyAuth)
	})
	return mux
}

// adminHandler handles ?mode=clean|create. The path is ignored, except
// /metrics which exposes prometheus metrics.
func adminHandler(store *Store, log zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		token := q.Get("token")
		if !tokenRe.MatchString(token) {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		switch q.Get("mode") {
		case "create":
			keyAuth := q.Get("keyauthorization")
			if keyAuth == "" {
				http.Error(w, "missing keyauthorization", http.StatusBadRequest)
				return
			}
			store.Set(token, keyAuth)
			log.Info().Str("token", token).Msg("challenge created")
			fmt.Fprintln(w, "created")
		case "clean":
			store.Delete(token)
			log.Info().Str("token", token).Msg("challenge cleaned")
			fmt.Fprintln(w, "cleaned")
		default:
			http.Error(w, "invalid mode", http.StatusBadRequest)
		}
	})
	return mux
}
