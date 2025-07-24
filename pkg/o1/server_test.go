package o1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetconfServer(t *testing.T) {
	ds := NewNetconfDatastore()
	server, err := NewNetconfServer(ds)
	require.NoError(t, err)
	require.NotNil(t, server)
	assert.NotNil(t, server.datastore)
	assert.NotNil(t, server.sessions)
}
