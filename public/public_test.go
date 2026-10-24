package public

import (
	"net/http/httptest"
	"testing"

	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	st := store.New()
	st.Set("abc", "abc.xyz")
	h := Handler(st)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", acmePrefix+"abc", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, "abc.xyz", w.Body.String())

	for _, p := range []string{acmePrefix + "nope", "/other", "/"} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		require.Equal(t, 404, w.Code, p)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", acmePrefix+"abc", nil))
	require.Equal(t, 405, w.Code)
}
