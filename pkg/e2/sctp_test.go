package e2

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSCTPManager(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38000,
		ConnectTimeout:    5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)

	assert.NotNil(t, manager)
	assert.Equal(t, "127.0.0.1", manager.listenAddress)
	assert.Equal(t, 38000, manager.listenPort)
	assert.Equal(t, 5*time.Second, manager.connectTimeout)
	assert.NotNil(t, manager.connections)
	assert.NotNil(t, manager.logger)
	assert.NotNil(t, manager.ctx)
}

func TestSCTPManagerConnectionTracking(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38001,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)

	// Test initial state
	status := manager.GetConnectionStatus()
	assert.Empty(t, status)

	stats := manager.GetConnectionStatistics()
	assert.Equal(t, 0, stats["total_connections"])
	assert.Equal(t, 0, stats["active_connections"])
}

func TestSCTPManagerMessageHandler(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38002,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)

	// Test setting message handler
	var receivedNodeID string
	var receivedData []byte
	var handlerCalled bool

	handler := func(nodeID string, data []byte) error {
		receivedNodeID = nodeID
		receivedData = data
		handlerCalled = true
		return nil
	}

	manager.SetMessageHandler(handler)
	assert.NotNil(t, manager.messageHandler)

	// Test handler execution (simulate)
	testNodeID := "test-node-1"
	testData := []byte("test-message")
	
	err := manager.messageHandler(testNodeID, testData)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, testNodeID, receivedNodeID)
	assert.Equal(t, testData, receivedData)
}

func TestSCTPConnectionTimeout(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38003,
		ConnectTimeout:    100 * time.Millisecond, // Very short timeout
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)

	// Try to connect to non-existent address
	start := time.Now()
	err := manager.Connect("test-node", "192.0.2.1", 38000) // RFC 5737 test address
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
	assert.Less(t, duration, 500*time.Millisecond) // Should timeout quickly
}

func TestSCTPManagerClose(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38004,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)

	// Test close
	err := manager.Close()
	assert.NoError(t, err)

	// Verify context is cancelled
	select {
	case <-manager.ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should be cancelled after close")
	}
}

func TestSCTPConnectionManagement(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38005,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)
	defer manager.Close()

	// Test disconnect non-existent connection
	err := manager.DisconnectNode("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no connection")

	// Test send to non-existent connection
	err = manager.SendToNode("non-existent", []byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestSCTPBroadcastToAllNodes(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38006,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)
	defer manager.Close()

	testData := []byte("broadcast-message")

	// Test broadcast with no connections
	errors := manager.BroadcastToAllNodes(testData)
	assert.Empty(t, errors)
}

// Mock TCP connection for testing SCTP behavior
func TestSCTPWithMockConnection(t *testing.T) {
	// This test simulates SCTP behavior using TCP for testing
	// In a real environment, SCTP would be used directly

	// Create a TCP server to simulate SCTP endpoint
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	
	// Handle one connection
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo back received data
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			return
		}

		conn.Write(buffer[:n])
	}()

	// Test TCP connection (simulating SCTP)
	conn, err := net.Dial("tcp", addr.String())
	require.NoError(t, err)
	defer conn.Close()

	testMessage := []byte("test-e2ap-message")
	
	// Send message
	_, err = conn.Write(testMessage)
	assert.NoError(t, err)

	// Read response
	response := make([]byte, len(testMessage))
	_, err = conn.Read(response)
	assert.NoError(t, err)
	assert.Equal(t, testMessage, response)
}

// Benchmark SCTP manager operations
func BenchmarkSCTPManagerCreation(b *testing.B) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38100,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		manager := NewSCTPManager(config)
		manager.Close()
	}
}

func BenchmarkConnectionStatusCheck(b *testing.B) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38101,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)
	defer manager.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = manager.GetConnectionStatus()
	}
}

// Test concurrent operations
func TestSCTPConcurrentOperations(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38007,
		ConnectTimeout:    100 * time.Millisecond,
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	manager := NewSCTPManager(config)
	defer manager.Close()

	// Test concurrent status checks
	const goroutines = 10
	const iterations = 100
	
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			
			for i := 0; i < iterations; i++ {
				// Mix different operations
				switch i % 3 {
				case 0:
					_ = manager.GetConnectionStatus()
				case 1:
					_ = manager.GetConnectionStatistics()
				case 2:
					errors := manager.BroadcastToAllNodes([]byte("test"))
					assert.Empty(t, errors)
				}
			}
		}(g)
	}

	wg.Wait()
}

// Test SCTP address parsing and validation
func TestSCTPAddressHandling(t *testing.T) {
	tests := []struct {
		name    string
		address string
		port    int
		valid   bool
	}{
		{"Valid IPv4", "127.0.0.1", 38000, true},
		{"Valid IPv4 any", "0.0.0.0", 38000, true},
		{"Valid port", "127.0.0.1", 65535, true},
		{"Invalid port high", "127.0.0.1", 65536, false},
		{"Invalid port zero", "127.0.0.1", 0, false},
		{"Invalid IP", "invalid-ip", 38000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SCTPConfig{
				ListenAddress:     tt.address,
				ListenPort:        tt.port,
				ConnectTimeout:    100 * time.Millisecond,
				ReadTimeout:       1 * time.Second,
				WriteTimeout:      1 * time.Second,
				KeepaliveInterval: 30 * time.Second,
			}

			manager := NewSCTPManager(config)
			defer manager.Close()

			// For invalid addresses, connection should fail quickly
			err := manager.Connect("test-node", tt.address, tt.port)
			
			if tt.valid {
				// Even valid addresses might fail to connect (no server)
				// but error shouldn't be due to address parsing
				if err != nil {
					assert.NotContains(t, err.Error(), "invalid")
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Test E2 interface latency requirements
func TestE2LatencyCompliance(t *testing.T) {
	config := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38008,
		ConnectTimeout:    1 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		KeepaliveInterval: 30 * time.Second,
	}

	// Test manager creation latency
	start := time.Now()
	manager := NewSCTPManager(config)
	creationDuration := time.Since(start)
	
	assert.Less(t, creationDuration, 10*time.Millisecond, 
		"SCTP manager creation should be < 10ms, got %v", creationDuration)

	// Test status check latency
	start = time.Now()
	_ = manager.GetConnectionStatus()
	statusDuration := time.Since(start)
	
	assert.Less(t, statusDuration, 1*time.Millisecond, 
		"Status check should be < 1ms, got %v", statusDuration)

	// Test broadcast latency (with no connections)
	start = time.Now()
	_ = manager.BroadcastToAllNodes([]byte("test"))
	broadcastDuration := time.Since(start)
	
	assert.Less(t, broadcastDuration, 1*time.Millisecond, 
		"Broadcast to no connections should be < 1ms, got %v", broadcastDuration)

	manager.Close()
	
	t.Logf("SCTP Manager Performance - Creation: %v, Status: %v, Broadcast: %v", 
		creationDuration, statusDuration, broadcastDuration)
}

// Test SCTP configuration validation
func TestSCTPConfigValidation(t *testing.T) {
	validConfig := &SCTPConfig{
		ListenAddress:     "127.0.0.1",
		ListenPort:        38000,
		ConnectTimeout:    5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		KeepaliveInterval: 60 * time.Second,
	}

	manager := NewSCTPManager(validConfig)
	defer manager.Close()

	assert.Equal(t, validConfig.ListenAddress, manager.listenAddress)
	assert.Equal(t, validConfig.ListenPort, manager.listenPort)
	assert.Equal(t, validConfig.ConnectTimeout, manager.connectTimeout)
	assert.Equal(t, validConfig.ReadTimeout, manager.readTimeout)
	assert.Equal(t, validConfig.WriteTimeout, manager.writeTimeout)
	assert.Equal(t, validConfig.KeepaliveInterval, manager.keepaliveInterval)
}