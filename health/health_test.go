package health

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProbe(t *testing.T) {
	p := New()
	w := httptest.NewRecorder()
	p.Liveness(w, httptest.NewRequest("GET", "/healthz", nil))
	require.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	p.Readiness(w, httptest.NewRequest("GET", "/readyz", nil))
	require.Equal(t, 200, w.Code)

	p.SetNotReady()
	w = httptest.NewRecorder()
	p.Readiness(w, httptest.NewRequest("GET", "/readyz", nil))
	require.Equal(t, 503, w.Code)
	w = httptest.NewRecorder()
	p.Liveness(w, httptest.NewRequest("GET", "/healthz", nil))
	require.Equal(t, 200, w.Code)
}
