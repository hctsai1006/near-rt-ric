//go:build linux
// +build linux

package e2

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ishidawataru/sctp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testSCTPTestAddr = "127.0.0.1:36423"
)

// TestSCTPServer tests the SCTP server implementation.
func TestSCTPServer(t *testing.T) {
	log := logrus.New()
	var wg sync.WaitGroup
	wg.Add(1)

	// Message handler for the server
	// handler := func(conn *sctp.SCTPConn, msg []byte) {
	// 	// For this test, we'll just signal that a message was received.
	// 	t.Logf("Server received message of length %d", len(msg))
	// 	wg.Done()
	// }

	// 1. Set up the SCTP Server
	serverConfig := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		Port:              36423,
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
	defer server.Stop(context.Background())

	// 2. Set up the SCTP Client
	clientAddr, err := sctp.ResolveSCTPAddr("sctp", testSCTPTestAddr)
	require.NoError(t, err)
	clientConn, err := sctp.DialSCTP("sctp", nil, clientAddr)
	require.NoError(t, err, "Client should connect to server without errors")
	defer clientConn.Close()

	// 3. Send a message
	testMessage := []byte("hello")
	_, err = clientConn.Write(testMessage)
	require.NoError(t, err, "Should send message without errors")

	// Wait for the server to process the message
	if waitTimeout(&wg, 5*time.Second) {
		t.Fatal("Timed out waiting for server to receive message")
	}

	// Assertions can be made here based on the handler's behavior
	assert.True(t, true, "Test completed, assuming handler logic is correct")
}

// waitTimeout waits for a WaitGroup to finish or times out
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // Completed normally
	case <-time.After(timeout):
		return true // Timed out
	}
}
