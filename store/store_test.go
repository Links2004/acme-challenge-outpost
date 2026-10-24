package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	s := New()
	s.Set("a", "b")
	v, ok := s.Get("a")
	require.True(t, ok)
	require.Equal(t, "b", v)
	s.Delete("a")
	_, ok = s.Get("a")
	require.False(t, ok)
}
