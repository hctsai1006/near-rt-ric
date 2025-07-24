//go:build linux
// +build linux

package e2

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTransportAddr = "127.0.0.1:36424"
)

// TestSCTPServer_Listen tests the Listen method of the SCTPServer.
func TestSCTPServer_Listen(t *testing.T) {
	log := logrus.New()
	// handler := func(conn *sctp.SCTPConn, msg []byte) {}
	serverConfig := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		Port:              36424,
		MaxConnections:    10,
		ConnectionTimeout: 5 * time.Second,
		HeartbeatInterval: 1 * time.Second,
		BufferSize:        1500,
		Streams:           3,
	}
	server, err := NewSCTPServer(serverConfig, log)

	require.NoError(t, err, "Server should start without errors")
	err = server.Start(context.Background())
	require.NoError(t, err, "Server should start listening without errors")
	assert.NotNil(t, server.listener, "Listener should not be nil")
	server.Stop(context.Background())
}

// TestSCTPServer_SendMessage_NoConnection tests sending a message to a non-existent connection.
func TestSCTPServer_SendMessage_NoConnection(t *testing.T) {
	log := logrus.New()
	serverConfig := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		Port:              36425,
		MaxConnections:    10,
		ConnectionTimeout: 5 * time.Second,
		HeartbeatInterval: 1 * time.Second,
		BufferSize:        1500,
		Streams:           3,
	}
	server, err := NewSCTPServer(serverConfig, log)
	require.NoError(t, err, "Server should be created without errors")

	err = server.SendToNode("non-existent-node", []byte("test"))
	assert.Error(t, err, "Should return an error")
	assert.EqualError(t, err, "no connection found for node non-existent-node", "Error message should indicate no active connection")
}
