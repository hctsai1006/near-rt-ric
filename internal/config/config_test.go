package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadA1Config(t *testing.T) {
	cfg, err := LoadA1Config()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "0.0.0.0", cfg.ListenAddress)
}
