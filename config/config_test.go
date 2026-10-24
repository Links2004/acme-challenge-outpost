package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 2\n"))
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Public.Port)
	require.False(t, cfg.DisableHealthCheckLogging)
	cfg, err = ParseConfig([]byte("disableHealthCheckLogging: true\npublic:\n  port: 1\nadmin:\n  port: 2\n"))
	require.NoError(t, err)
	require.True(t, cfg.DisableHealthCheckLogging)
	_, err = ParseConfig([]byte("public:\n  port: 1\nadmin:\n  port: 1\n"))
	require.Error(t, err)
}
