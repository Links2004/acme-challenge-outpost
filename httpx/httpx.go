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
	"github.com/rs/zerolog/hlog"
)

var httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "acme_outpost_http_requests_total",
	Help: "Total HTTP requests by server and status code.",
}, []string{"server", "code"})

// clientAddr resolves the client address from X-Forwarded-For. The list is
// walked from the right, skipping trusted proxies; the first untrusted entry
// is the client. Private addresses are valid clients.
func clientAddr(xffHeader string, trusted []*net.IPNet) net.IP {
	parts := strings.Split(xffHeader, ",")
	var ip net.IP
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
	return ip
}

var blockedRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "acme_outpost_blocked_requests_total",
	Help: "Requests rejected because the client IP is not allowed.",
}, []string{"server"})

// forwarded applies X-Forwarded-For, X-Forwarded-Proto and X-Forwarded-Host
// when the direct peer is a trusted proxy.
func forwarded(trusted []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if InNets(net.ParseIP(host), trusted) {
			if v := r.Header.Get("X-Forwarded-For"); v != "" {
				if ip := clientAddr(v, trusted); ip != nil {
					_, port, _ := net.SplitHostPort(r.RemoteAddr)
					r.RemoteAddr = net.JoinHostPort(ip.String(), port)
				}
			}
			if v := r.Header.Get("X-Forwarded-Proto"); v == "http" || v == "https" {
				r.URL.Scheme = v
			}
			if v := r.Header.Get("X-Forwarded-Host"); v != "" {
				r.Host = strings.TrimSpace(strings.Split(v, ",")[0])
			}
		}
		next.ServeHTTP(w, r)
	})
}

// allowOnly rejects clients whose (already resolved) address is not in allowed.
func allowOnly(server string, allowed []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if !InNets(net.ParseIP(host), allowed) {
			blockedRequests.WithLabelValues(server).Inc()
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithAccessLog wraps a handler with X-Forwarded-* handling (trustedProxies,
// as CIDRs or IPs), optional IP allow-listing (allowed == nil disables it)
// and hlog based access logging.
func WithAccessLog(server string, log zerolog.Logger, trustedProxies []string, allowed []*net.IPNet, next http.Handler) (http.Handler, error) {
	trusted, err := ParseNets(trustedProxies)
	if err != nil {
		return nil, err
	}
	h := next
	if allowed != nil {
		h = allowOnly(server, allowed, h)
	}
	h = hlog.AccessHandler(func(r *http.Request, status, size int, d time.Duration) {
		httpRequests.WithLabelValues(server, strconv.Itoa(status)).Inc()
		scheme := r.URL.Scheme
		if scheme == "" {
			scheme = "http"
			if r.TLS != nil {
				scheme = "https"
			}
		}
		hlog.FromRequest(r).Info().
			Str("server", server).
			Str("proto", scheme).
			Str("host", r.Host).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Int("bytes", size).
			Dur("duration", d).
			Msg("access")
	})(h)
	h = hlog.UserAgentHandler("user_agent")(h)
	h = hlog.RemoteAddrHandler("client_ip")(h)
	h = hlog.NewHandler(log)(h)
	// outermost: sees the direct peer address and rewrites RemoteAddr
	h = forwarded(trusted, h)
	return h, nil
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
				s = ip4.String() + "/32"
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
