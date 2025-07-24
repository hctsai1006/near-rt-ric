package o1

import (
	"context"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type MockNETCONFServer struct {
	mock.Mock
}

func (m *MockNETCONFServer) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockNETCONFServer) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockNETCONFServer) GetCapabilities() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockNETCONFServer) HandleRPC(rpc string, data []byte) ([]byte, error) {
	args := m.Called(rpc, data)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockNETCONFServer) GetActiveConnections() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockNETCONFServer) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockFCAPSManager struct {
	mock.Mock
}

func (m *MockFCAPSManager) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockFCAPSManager) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockFCAPSManager) GetAlarms(filter map[string]interface{}) []*Alarm {
	args := m.Called(filter)
	return args.Get(0).([]*Alarm)
}

// Test configuration helper
func createTestO1Config() *config.O1Config {
	return &config.O1Config{
		Enabled:     true,
		Port:        830,
		BindAddress: "127.0.0.1",
		LogLevel:    "info",
		SSH: config.SSHConfig{
			HostKeyPath:        "/tmp/test_host_key",
			AuthorizedKeysPath: "/tmp/test_authorized_keys",
			Timeout:            30,
			MaxConnections:     10,
		},
		NETCONF: config.NETCONFConfig{
			ListenAddress:  "127.0.0.1",
			Port:           830,
			TLSPort:        6513,
			SessionTimeout: 600,
			MaxSessions:    10,
		},
		YANG: config.YANGConfig{
			ModulesPath:    "/etc/near-rt-ric/yang",
			ValidateConfig: true,
		},
		FileManagement: config.FileManagementConfig{
			Enabled:     true,
			UploadPath:  "/var/lib/near-rt-ric/uploads",
			MaxFileSize: 100,
		},
		SoftwareManagement: config.SoftwareManagementConfig{
			Enabled:     true,
			PackagePath: "/var/lib/near-rt-ric/packages",
			InstallPath: "/opt/near-rt-ric",
			BackupPath:  "/var/lib/near-rt-ric/backups",
			MaxPackages: 10,
		},
	}
}

// TestNewO1Interface tests the creation of a new O1Interface
func TestNewO1Interface(t *testing.T) {
	cfg := createTestO1Config()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	o1, err := NewO1Interface(cfg, logger, nil)
	require.NoError(t, err)
	assert.NotNil(t, o1)
}

// TestO1InterfaceLifecycle tests the start and stop functionality
func TestO1InterfaceLifecycle(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test starting the interface
	err := o1Interface.Start()
	assert.NoError(t, err)
	assert.True(t, o1Interface.running)

	// Test starting already running interface
	err = o1Interface.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stopping the interface
	err = o1Interface.Stop()
	assert.NoError(t, err)
	assert.False(t, o1Interface.running)

	// Test stopping already stopped interface
	err = o1Interface.Stop()
	assert.NoError(t, err)
}

// Helper function to create a test O1Interface with mocks
func createTestO1Interface(t *testing.T) *o1Interface {
	cfg := createTestO1Config()

	var logger *logrus.Logger
	if t != nil {
		logger = logrus.New()
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger = logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in benchmarks
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create mocks
	mockNETCONFServer := &MockNETCONFServer{}
	mockFCAPSManager := &MockFCAPSManager{}

	// Set up mock expectations for basic operations
	mockNETCONFServer.On("Start", mock.Anything).Return(nil)
	mockNETCONFServer.On("Stop", mock.Anything).Return(nil)
	mockFCAPSManager.On("Start", mock.Anything).Return(nil)
	mockFCAPSManager.On("Stop", mock.Anything).Return(nil)

	// Create interface
	o1Interface := &o1Interface{
		config:        cfg,
		logger:        logger.WithField("component", "o1-interface"),
		metrics:       nil,
		netconfServer: mockNETCONFServer,
		fcapsManager:  mockFCAPSManager,
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		startTime:     time.Now(),
	}

	return o1Interface
}
