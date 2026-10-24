package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 2\n"))
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Public.Port)
	_, err = ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 1\n"))
	require.Error(t, err)
}
