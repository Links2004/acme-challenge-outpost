package public

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Links2004/acme-challenge-outpost/health"
	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var tokenNotFound = promauto.NewCounter(prometheus.CounterOpts{
	Name: "acme_outpost_public_token_not_found_total",
	Help: "Public challenge requests for unknown tokens.",
})

const acmePrefix = "/.well-known/acme-challenge/"

// Handler serves /.well-known/acme-challenge/{token} plus the /healthz and
// /readyz probes. The probes live on the public port because the admin port is
// IP restricted and the kubelet would be blocked.
func Handler(st *store.Store, probe *health.Probe) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", probe.Liveness)
	mux.HandleFunc("/readyz", probe.Readiness)
	mux.HandleFunc(acmePrefix, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		token := strings.TrimPrefix(r.URL.Path, acmePrefix)
		keyAuth, ok := st.Get(token)
		if !ok {
			tokenNotFound.Inc()
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, keyAuth)
	})
	return mux
}
