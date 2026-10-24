package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T, buf *bytes.Buffer) (http.Handler, http.Handler, *Store) {
	cfg, err := ParseConfig([]byte(`
trustedProxyIPs: [10.0.0.0/24, 10.2.2.2]
admin:
  port: 9000
  allowedIPs: [192.168.0.0/24, 192.168.1.1]
`))
	require.NoError(t, err)
	trusted, err := ParseNets(cfg.TrustedProxyIPs)
	require.NoError(t, err)
	allowed, err := ParseNets(cfg.Admin.AllowedIPs)
	require.NoError(t, err)
	log := zerolog.New(buf)
	s := NewStore()
	return withAccessLog("public", log, trusted, nil, publicHandler(s)),
		withAccessLog("admin", log, trusted, allowed, adminHandler(s, log)), s
}

func do(h http.Handler, target, remote string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", target, nil)
	r.RemoteAddr = remote
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 2\n"))
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Public.Port)
	_, err = ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 1\n"))
	require.Error(t, err)
	_, err = ParseNets([]string{"bogus"})
	require.Error(t, err)
}

func TestAdminAndPublicFlow(t *testing.T) {
	var buf bytes.Buffer
	pub, adm, _ := setup(t, &buf)

	require.Equal(t, 404, do(pub, "/.well-known/acme-challenge/abc", "1.2.3.4:1", nil).Code)
	require.Equal(t, 200, do(adm, "/x?mode=create&token=abc&keyauthorization=abc.xyz", "192.168.1.1:5", nil).Code)
	w := do(pub, "/.well-known/acme-challenge/abc", "1.2.3.4:1", nil)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "abc.xyz", w.Body.String())
	require.Equal(t, 404, do(pub, "/other", "1.2.3.4:1", nil).Code)
	require.Equal(t, 200, do(adm, "/?mode=clean&token=abc", "192.168.0.7:5", nil).Code)
	require.Equal(t, 404, do(pub, "/.well-known/acme-challenge/abc", "1.2.3.4:1", nil).Code)
}

func TestAdminValidation(t *testing.T) {
	var buf bytes.Buffer
	_, adm, _ := setup(t, &buf)
	require.Equal(t, 400, do(adm, "/?mode=create&token=abc", "192.168.1.1:5", nil).Code)
	require.Equal(t, 400, do(adm, "/?mode=bad&token=abc", "192.168.1.1:5", nil).Code)
	require.Equal(t, 400, do(adm, "/?mode=clean", "192.168.1.1:5", nil).Code)
}

func TestAdminAllowedIPs(t *testing.T) {
	var buf bytes.Buffer
	_, adm, _ := setup(t, &buf)
	q := "/?mode=clean&token=abc"
	require.Equal(t, 403, do(adm, q, "8.8.8.8:1", nil).Code)
	// untrusted peer cannot spoof via XFF
	require.Equal(t, 403, do(adm, q, "8.8.8.8:1", map[string]string{"X-Forwarded-For": "192.168.1.1"}).Code)
	// trusted proxy forwarding an allowed client
	require.Equal(t, 200, do(adm, q, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1"}).Code)
	// trusted proxy forwarding a disallowed client
	require.Equal(t, 403, do(adm, q, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "8.8.8.8"}).Code)
	// spoofed leftmost entry behind trusted proxy is ignored
	require.Equal(t, 403, do(adm, q, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1, 8.8.8.8"}).Code)
}

func TestAccessLogForwardedHeaders(t *testing.T) {
	var buf bytes.Buffer
	pub, _, _ := setup(t, &buf)
	do(pub, "/.well-known/acme-challenge/zzz", "10.2.2.2:9", map[string]string{
		"X-Forwarded-For":   "5.6.7.8",
		"X-Forwarded-Proto": "https",
		"X-Forwarded-Host":  "example.com",
	})
	out := buf.String()
	require.Contains(t, out, `"client_ip":"5.6.7.8"`)
	require.Contains(t, out, `"proto":"https"`)
	require.Contains(t, out, `"host":"example.com"`)
	require.Contains(t, out, `"status":404`)
}

func TestStore(t *testing.T) {
	s := NewStore()
	s.Set("a", "b")
	v, ok := s.Get("a")
	require.True(t, ok)
	require.Equal(t, "b", v)
	s.Delete("a")
	_, ok = s.Get("a")
	require.False(t, ok)
}
