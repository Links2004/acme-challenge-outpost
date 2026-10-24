package admin

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

var adminOps = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "acme_outpost_admin_operations_total",
	Help: "Admin operations by mode (create, clean) and result (ok, invalid).",
}, []string{"mode", "result"})

var tokenRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Handler handles ?mode=clean|create. The path is ignored, except
// /metrics which exposes prometheus metrics.
func Handler(st *store.Store, log zerolog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		token := q.Get("token")
		mode := q.Get("mode")
		if mode != "create" && mode != "clean" {
			adminOps.WithLabelValues("other", "invalid").Inc()
			http.Error(w, "invalid mode", http.StatusBadRequest)
			return
		}
		if !tokenRe.MatchString(token) {
			adminOps.WithLabelValues(mode, "invalid").Inc()
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		if mode == "create" {
			keyAuth := q.Get("keyauthorization")
			if keyAuth == "" {
				adminOps.WithLabelValues(mode, "invalid").Inc()
				http.Error(w, "missing keyauthorization", http.StatusBadRequest)
				return
			}
			st.Set(token, keyAuth)
			adminOps.WithLabelValues(mode, "ok").Inc()
			log.Info().Str("token", token).Msg("challenge created")
			fmt.Fprintln(w, "created")
			return
		}
		st.Delete(token)
		adminOps.WithLabelValues(mode, "ok").Inc()
		log.Info().Str("token", token).Msg("challenge cleaned")
		fmt.Fprintln(w, "cleaned")
	})
	return mux
}
