package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

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
	trusted, _ := ParseNets([]string{"10.0.0.0/24"})
	allowed, _ := ParseNets([]string{"192.168.0.0/24", "192.168.1.1"})
	h := WithAccessLog("admin", zerolog.Nop(), trusted, allowed, ok)
	require.Equal(t, 403, do(h, "8.8.8.8:1", nil).Code)
	require.Equal(t, 403, do(h, "8.8.8.8:1", map[string]string{"X-Forwarded-For": "192.168.1.1"}).Code)
	require.Equal(t, 204, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1"}).Code)
	require.Equal(t, 403, do(h, "10.0.0.5:1", map[string]string{"X-Forwarded-For": "192.168.1.1, 8.8.8.8"}).Code)
	require.Equal(t, 204, do(h, "192.168.0.9:1", nil).Code)
}

func TestAccessLogForwardedHeaders(t *testing.T) {
	var buf bytes.Buffer
	trusted, _ := ParseNets([]string{"10.2.2.2"})
	h := WithAccessLog("public", zerolog.New(&buf), trusted, nil, ok)
	do(h, "10.2.2.2:9", map[string]string{
		"X-Forwarded-For": "5.6.7.8", "X-Forwarded-Proto": "https", "X-Forwarded-Host": "example.com"})
	out := buf.String()
	require.Contains(t, out, `"client_ip":"5.6.7.8"`)
	require.Contains(t, out, `"proto":"https"`)
	require.Contains(t, out, `"host":"example.com"`)
	require.Contains(t, out, `"status":204`)
}
