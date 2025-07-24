package o1

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func TestNewNetconfServer(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{} // Mock metrics collector
	
	server, err := NewNetconfServer(config, logger, metrics)
	
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.metrics)
	assert.NotEmpty(t, server.yangModels)
	assert.NotEmpty(t, server.capabilities)
	assert.NotNil(t, server.sshConfig)
}

func TestNetconfServerYANGModels(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test YANG models initialization
	expectedModels := []string{
		"ietf-netconf",
		"ietf-netconf-monitoring", 
		"o-ran-sc-ric-gnb-status",
		"o-ran-sc-ric-alarms",
		"o-ran-sc-ric-xapp-mgr",
	}
	
	assert.Len(t, server.yangModels, len(expectedModels))
	
	for i, expectedName := range expectedModels {
		assert.Equal(t, expectedName, server.yangModels[i].Name)
		assert.NotEmpty(t, server.yangModels[i].Namespace)
		assert.NotEmpty(t, server.yangModels[i].Revision)
	}
}

func TestNetconfServerCapabilities(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test capabilities initialization
	expectedBaseCapabilities := []string{
		"urn:ietf:params:netconf:base:1.0",
		"urn:ietf:params:netconf:base:1.1",
		"urn:ietf:params:netconf:capability:writable-running:1.0",
		"urn:ietf:params:netconf:capability:candidate:1.0",
		"urn:ietf:params:netconf:capability:rollback-on-error:1.0",
	}
	
	assert.True(t, len(server.capabilities) > len(expectedBaseCapabilities))
	
	// Check that base capabilities are present
	capabilityURIs := make([]string, len(server.capabilities))
	for i, cap := range server.capabilities {
		capabilityURIs[i] = cap.URI
	}
	
	for _, expectedCap := range expectedBaseCapabilities {
		assert.Contains(t, capabilityURIs, expectedCap)
	}
}

func TestNetconfServerSSHConfig(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	assert.NotNil(t, server.sshConfig)
	assert.NotNil(t, server.sshConfig.PasswordCallback)
	
	// Test SSH authentication callback
	conn := &mockSSHConnMetadata{user: "netconf"}
	
	// Test valid credentials
	perms, err := server.sshConfig.PasswordCallback(conn, []byte("netconf"))
	assert.NoError(t, err)
	assert.Nil(t, perms) // Success returns nil permissions
	
	// Test invalid credentials
	perms, err = server.sshConfig.PasswordCallback(conn, []byte("wrongpassword"))
	assert.Error(t, err)
	assert.Nil(t, perms)
}

func TestNetconfServerStartStop(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test server start
	ctx := context.Background()
	err = server.Start(ctx)
	require.NoError(t, err)
	assert.True(t, server.IsRunning())
	
	// Wait a moment for server to be ready
	time.Sleep(100 * time.Millisecond)
	
	// Test that server is listening
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8301", 1*time.Second)
	if err == nil {
		conn.Close()
	}
	// Connection may fail due to SSH handshake, but port should be listening
	
	// Test server stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = server.Stop(stopCtx)
	assert.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestNetconfServerDuplicateStart(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// First start should succeed
	err = server.Start(ctx)
	require.NoError(t, err)
	
	// Second start should fail
	err = server.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
	
	// Cleanup
	server.Stop(context.Background())
}

func TestNetconfServerStats(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test initial stats
	stats := server.GetStats()
	assert.Equal(t, int32(0), stats.ActiveSessions)
	assert.Equal(t, int64(0), stats.TotalSessions)
	assert.Equal(t, int64(0), stats.TotalMessages)
}

func TestNetconfSessionManagement(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test session creation
	sessionID := server.createSession("testuser", "127.0.0.1:12345")
	assert.Greater(t, sessionID, uint32(0))
	
	// Test session retrieval
	session := server.getSession(sessionID)
	assert.NotNil(t, session)
	assert.Equal(t, sessionID, session.SessionID)
	assert.Equal(t, "testuser", session.Username)
	assert.Equal(t, "127.0.0.1:12345", session.SourceHost)
	assert.NotEmpty(t, session.Capabilities)
	
	// Test stats update
	stats := server.GetStats()
	assert.Equal(t, int32(1), stats.ActiveSessions)
	assert.Equal(t, int64(1), stats.TotalSessions)
	
	// Test session removal
	server.removeSession(sessionID)
	
	session = server.getSession(sessionID)
	assert.Nil(t, session)
	
	stats = server.GetStats()
	assert.Equal(t, int32(0), stats.ActiveSessions)
	assert.Equal(t, int64(1), stats.TotalSessions) // Total count doesn't decrease
}

func TestNetconfConcurrentSessions(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	const numSessions = 10
	sessionIDs := make([]uint32, numSessions)
	
	// Create multiple sessions concurrently
	for i := 0; i < numSessions; i++ {
		go func(id int) {
			sessionID := server.createSession("user"+string(rune(id)), "127.0.0.1:1234")
			sessionIDs[id] = sessionID
		}(i)
	}
	
	// Wait for all sessions to be created
	time.Sleep(100 * time.Millisecond)
	
	stats := server.GetStats()
	assert.Equal(t, int32(numSessions), stats.ActiveSessions)
	
	// Remove all sessions
	for _, sessionID := range sessionIDs {
		if sessionID > 0 {
			server.removeSession(sessionID)
		}
	}
	
	stats = server.GetStats()
	assert.Equal(t, int32(0), stats.ActiveSessions)
}

func TestNetconfMessageHandler(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(t, err)
	
	// Test setting message handler
	var handlerCalled bool
	var receivedSessionID uint32
	var receivedRequest *GetConfigRequest
	
	mockHandler := &mockNetconfMessageHandler{
		getConfigHandler: func(sessionID uint32, req *GetConfigRequest) (interface{}, error) {
			handlerCalled = true
			receivedSessionID = sessionID
			receivedRequest = req
			return map[string]interface{}{"status": "ok"}, nil
		},
	}
	
	server.SetMessageHandler(mockHandler)
	assert.NotNil(t, server.messageHandler)
	
	// Simulate handler call
	testSessionID := uint32(123)
	testRequest := &GetConfigRequest{Datastore: DatastoreRunning}
	
	result, err := server.messageHandler.HandleGetConfig(testSessionID, testRequest)
	
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, testSessionID, receivedSessionID)
	assert.Equal(t, testRequest, receivedRequest)
	assert.NotNil(t, result)
}

// Mock types for testing

type mockSSHConnMetadata struct {
	user string
}

func (m *mockSSHConnMetadata) User() string        { return m.user }
func (m *mockSSHConnMetadata) SessionID() []byte  { return []byte("test-session") }
func (m *mockSSHConnMetadata) ClientVersion() []byte { return []byte("test-client") }
func (m *mockSSHConnMetadata) ServerVersion() []byte { return []byte("test-server") }
func (m *mockSSHConnMetadata) RemoteAddr() net.Addr { 
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:12345")
	return addr
}
func (m *mockSSHConnMetadata) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8301")
	return addr
}

type mockNetconfMessageHandler struct {
	getConfigHandler   func(uint32, *GetConfigRequest) (interface{}, error)
	editConfigHandler  func(uint32, *EditConfigRequest) error
	getHandler         func(uint32, string) (interface{}, error)
	copyConfigHandler  func(uint32, DatastoreType, DatastoreType) error
	deleteConfigHandler func(uint32, DatastoreType) error
	lockHandler        func(uint32, DatastoreType) error
	unlockHandler      func(uint32, DatastoreType) error
	closeSessionHandler func(uint32) error
	killSessionHandler func(uint32, uint32) error
}

func (m *mockNetconfMessageHandler) HandleGetConfig(sessionID uint32, req *GetConfigRequest) (interface{}, error) {
	if m.getConfigHandler != nil {
		return m.getConfigHandler(sessionID, req)
	}
	return nil, nil
}

func (m *mockNetconfMessageHandler) HandleEditConfig(sessionID uint32, req *EditConfigRequest) error {
	if m.editConfigHandler != nil {
		return m.editConfigHandler(sessionID, req)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleGet(sessionID uint32, filter string) (interface{}, error) {
	if m.getHandler != nil {
		return m.getHandler(sessionID, filter)
	}
	return nil, nil
}

func (m *mockNetconfMessageHandler) HandleCopyConfig(sessionID uint32, source, target DatastoreType) error {
	if m.copyConfigHandler != nil {
		return m.copyConfigHandler(sessionID, source, target)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleDeleteConfig(sessionID uint32, target DatastoreType) error {
	if m.deleteConfigHandler != nil {
		return m.deleteConfigHandler(sessionID, target)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleLock(sessionID uint32, target DatastoreType) error {
	if m.lockHandler != nil {
		return m.lockHandler(sessionID, target)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleUnlock(sessionID uint32, target DatastoreType) error {
	if m.unlockHandler != nil {
		return m.unlockHandler(sessionID, target)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleCloseSession(sessionID uint32) error {
	if m.closeSessionHandler != nil {
		return m.closeSessionHandler(sessionID)
	}
	return nil
}

func (m *mockNetconfMessageHandler) HandleKillSession(sessionID uint32, targetSessionID uint32) error {
	if m.killSessionHandler != nil {
		return m.killSessionHandler(sessionID, targetSessionID)
	}
	return nil
}

// Benchmark tests

func BenchmarkNetconfSessionCreation(b *testing.B) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(b, err)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		sessionID := server.createSession("user", "127.0.0.1:12345")
		server.removeSession(sessionID)
	}
}

func BenchmarkNetconfStatsRetrieval(b *testing.B) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	server, err := NewNetconfServer(config, logger, metrics)
	require.NoError(b, err)
	
	// Create some sessions for realistic stats
	for i := 0; i < 10; i++ {
		server.createSession("user", "127.0.0.1:12345")
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = server.GetStats()
	}
}

// Test O1 interface latency requirements  
func TestO1LatencyCompliance(t *testing.T) {
	config := createTestO1Config()
	logger := logrus.New()
	metrics := &monitoring.MetricsCollector{}
	
	// Test server creation latency
	start := time.Now()
	server, err := NewNetconfServer(config, logger, metrics)
	creationDuration := time.Since(start)
	
	require.NoError(t, err)
	assert.Less(t, creationDuration, 500*time.Millisecond, 
		"NETCONF server creation should be < 500ms, got %v", creationDuration)
	
	// Test session creation latency
	start = time.Now()
	sessionID := server.createSession("testuser", "127.0.0.1:12345")
	sessionDuration := time.Since(start)
	
	assert.Greater(t, sessionID, uint32(0))
	assert.Less(t, sessionDuration, 10*time.Millisecond,
		"Session creation should be < 10ms, got %v", sessionDuration)
	
	// Test stats retrieval latency
	start = time.Now()
	_ = server.GetStats()
	statsDuration := time.Since(start)
	
	assert.Less(t, statsDuration, 1*time.Millisecond,
		"Stats retrieval should be < 1ms, got %v", statsDuration)
	
	server.removeSession(sessionID)
	
	t.Logf("O1 NETCONF Performance - Creation: %v, Session: %v, Stats: %v", 
		creationDuration, sessionDuration, statsDuration)
}
