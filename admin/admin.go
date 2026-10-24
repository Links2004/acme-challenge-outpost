package admin

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Handler handles ?mode=clean|create. The path is ignored, except
// /metrics which exposes prometheus metrics.
func Handler(st *store.Store, log zerolog.Logger) http.Handler {
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
			st.Set(token, keyAuth)
			log.Info().Str("token", token).Msg("challenge created")
			fmt.Fprintln(w, "created")
		case "clean":
			st.Delete(token)
			log.Info().Str("token", token).Msg("challenge cleaned")
			fmt.Fprintln(w, "cleaned")
		default:
			http.Error(w, "invalid mode", http.StatusBadRequest)
		}
	})
	return mux
}
