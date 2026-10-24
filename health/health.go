// Package health provides Kubernetes liveness and readiness probe handlers.
package health

import (
	"net/http"
	"sync/atomic"
)

// Probe tracks readiness. It starts ready; call SetNotReady when shutting down
// so Kubernetes stops routing traffic before the listeners close.
type Probe struct {
	notReady atomic.Bool
}

func New() *Probe { return &Probe{} }

func (p *Probe) SetNotReady() { p.notReady.Store(true) }

func (p *Probe) Ready() bool { return !p.notReady.Load() }

// Liveness always answers 200 while the process can serve requests.
func (p *Probe) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok\n"))
}

// Readiness answers 200 until SetNotReady was called, then 503.
func (p *Probe) Readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	if !p.Ready() {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("shutting down\n"))
		return
	}
	w.Write([]byte("ok\n"))
}
