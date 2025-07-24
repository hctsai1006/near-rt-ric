package e2

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCTPTransport_Listen(t *testing.T) {
	transport := &SCTPTransport{
		connections: make(map[string]*sctp.SCTPConn),
	}

	err := transport.Listen()
	require.NoError(t, err)
	assert.NotNil(t, transport.listener)
	transport.listener.Close()
}

func TestSCTPTransport_SendMessage_NoConnection(t *testing.T) {
	transport := &SCTPTransport{
		connections: make(map[string]*sctp.SCTPConn),
	}

	err := transport.SendMessage("non-existent-node", []byte("test"), 0)
	assert.Error(t, err)
	assert.EqualError(t, err, "no connection for node non-existent-node")
}
