// Package httpx holds HTTP helpers shared by the public and admin servers.
package httpx

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

var httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "acme_outpost_http_requests_total",
	Help: "Total HTTP requests by server and status code.",
}, []string{"server", "code"})

// remoteIP returns the IP of the direct peer.
func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

// clientInfo resolves client IP, scheme and host, honoring X-Forwarded-*
// headers only when the direct peer is a trusted proxy.
func clientInfo(r *http.Request, trusted []*net.IPNet) (ip net.IP, proto, host string) {
	ip = remoteIP(r)
	proto = "http"
	if r.TLS != nil {
		proto = "https"
	}
	host = r.Host
	if !InNets(ip, trusted) {
		return
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// walk from the right, skipping trusted proxies
		for i := len(parts) - 1; i >= 0; i-- {
			p := net.ParseIP(strings.TrimSpace(parts[i]))
			if p == nil {
				break
			}
			ip = p
			if !InNets(p, trusted) {
				break
			}
		}
	}
	if v := r.Header.Get("X-Forwarded-Proto"); v == "http" || v == "https" {
		proto = v
	}
	if v := r.Header.Get("X-Forwarded-Host"); v != "" {
		host = strings.TrimSpace(strings.Split(v, ",")[0])
	}
	return
}

type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// WithAccessLog wraps a handler with X-Forwarded-* handling, optional IP
// allow-listing (allowed == nil disables it) and access logging.
func WithAccessLog(server string, log zerolog.Logger, trusted, allowed []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ip, proto, host := clientInfo(r, trusted)
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		if allowed != nil && !InNets(ip, allowed) {
			http.Error(sw, "forbidden", http.StatusForbidden)
		} else {
			next.ServeHTTP(sw, r)
		}
		httpRequests.WithLabelValues(server, strconv.Itoa(sw.status)).Inc()
		log.Info().
			Str("server", server).
			Str("client_ip", ipString(ip)).
			Str("remote_addr", r.RemoteAddr).
			Str("proto", proto).
			Str("host", host).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", sw.status).
			Int("bytes", sw.bytes).
			Dur("duration", time.Since(start)).
			Str("user_agent", r.UserAgent()).
			Msg("access")
	})
}

func ipString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

// ParseNets parses a list of CIDRs or single IPs.
func ParseNets(list []string) ([]*net.IPNet, error) {
	var nets []*net.IPNet
	for _, s := range list {
		s = strings.TrimSpace(s)
		if !strings.Contains(s, "/") {
			ip := net.ParseIP(s)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP %q", s)
			}
			if ip4 := ip.To4(); ip4 != nil {
				s += "/32"
			} else {
				s += "/128"
			}
		}
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", s, err)
		}
		nets = append(nets, n)
	}
	return nets, nil
}

func InNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
