package admin

import (
	"net/http/httptest"
	"testing"

	"github.com/Links2004/acme-challenge-outpost/store"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	st := store.New()
	h := Handler(st, zerolog.Nop())
	code := func(target string) int {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", target, nil))
		return w.Code
	}
	require.Equal(t, 200, code("/x?mode=create&token=abc&keyauthorization=abc.xyz"))
	v, ok := st.Get("abc")
	require.True(t, ok)
	require.Equal(t, "abc.xyz", v)
	require.Equal(t, 200, code("/?mode=clean&token=abc"))
	_, ok = st.Get("abc")
	require.False(t, ok)
	require.Equal(t, 400, code("/?mode=create&token=abc"))
	require.Equal(t, 400, code("/?mode=bad&token=abc"))
	require.Equal(t, 400, code("/?mode=clean"))
	require.Equal(t, 200, code("/metrics"))
}
