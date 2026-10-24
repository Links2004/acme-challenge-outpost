package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })

func do(h http.Handler, remote string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", "/p", nil)
	r.RemoteAddr = remote
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestParseNets(t *testing.T) {
	n, err := ParseNets([]string{"10.0.0.0/24", "10.2.2.2", "::1"})
	require.NoError(t, err)
	require.Len(t, n, 3)
	_, err = ParseNets([]string{"bogus"})
	require.Error(t, err)
}

func TestAllowedIPs(t *testing.T) {
	allowed, _ := ParseNets([]string{"192.168.0.0/24", "192.168.1.1", "5.6.7.8"})
	h, err := WithAccessLog("admin", zerolog.Nop(), []string{"10.0.0.0/24"}, allowed, ok)
	require.NoError(t, err)
	before := testutil.ToFloat64(blockedRequests.WithLabelValues("admin"))
	require.Equal(t, 403, do(h, "8.8.8.8:1", nil).Code)
	require.Equal(t, 1.0, testutil.ToFloat64(blockedRequests.WithLabelValues("admin"))-before)
	// untrusted peer cannot spoof via XFF
	require.Equal(t, 403, do(h, "8.8.8.8:1", map[string]string{"X-Forwarded-For": "5.6.7.8"}).Code)
	// trusted proxy forwarding an allowed (public) client
	require.Equal(t, 204, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "5.6.7.8"}).Code)
	// private client address, via chain of trusted proxies
	require.Equal(t, 204, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.9"}).Code)
	// spoofed leftmost entry is ignored
	require.Equal(t, 403, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1, 8.8.8.8"}).Code)
	// trusted proxy forwarding a disallowed client
	require.Equal(t, 403, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "8.8.8.8"}).Code)
	require.Equal(t, 204, do(h, "192.168.0.9:1", nil).Code)
}

func TestInvalidTrustedProxy(t *testing.T) {
	_, err := WithAccessLog("x", zerolog.Nop(), []string{"bogus"}, nil, ok)
	require.Error(t, err)
}

func TestAccessLogForwardedHeaders(t *testing.T) {
	var buf bytes.Buffer
	h, err := WithAccessLog("public", zerolog.New(&buf), []string{"10.2.2.2"}, nil, ok)
	require.NoError(t, err)
	do(h, "10.2.2.2:9", map[string]string{
		"X-Forwarded-For": "5.6.7.8", "X-Forwarded-Proto": "https", "X-Forwarded-Host": "example.com"})
	out := buf.String()
	require.Contains(t, out, `"client_ip":"5.6.7.8:9"`)
	require.Contains(t, out, `"proto":"https"`)
	require.Contains(t, out, `"host":"example.com"`)
	require.Contains(t, out, `"status":204`)

	buf.Reset()
	do(h, "9.9.9.9:9", map[string]string{"X-Forwarded-For": "5.6.7.8", "X-Forwarded-Host": "evil.com"})
	require.Contains(t, buf.String(), `"client_ip":"9.9.9.9:9"`)
	require.NotContains(t, buf.String(), "evil.com")
}
