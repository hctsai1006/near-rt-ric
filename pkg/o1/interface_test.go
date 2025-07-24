package o1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
	"errors"
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

type MockYANGRepository struct {
	mock.Mock
}

func (m *MockYANGRepository) LoadModule(moduleName string, moduleData []byte) error {
	args := m.Called(moduleName, moduleData)
	return args.Error(0)
}

func (m *MockYANGRepository) GetModule(moduleName string) (*YANGModule, error) {
	args := m.Called(moduleName)
	return args.Get(0).(*YANGModule), args.Error(1)
}

func (m *MockYANGRepository) GetAllModules() ([]*YANGModule, error) {
	args := m.Called()
	return args.Get(0).([]*YANGModule), args.Error(1)
}

func (m *MockYANGRepository) ValidateConfig(config []byte) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockYANGRepository) GetCapabilities() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockYANGRepository) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) GetConfig(path string) (interface{}, error) {
	args := m.Called(path)
	return args.Get(0), args.Error(1)
}

func (m *MockConfigManager) SetConfig(path string, value interface{}) error {
	args := m.Called(path, value)
	return args.Error(0)
}

func (m *MockConfigManager) DeleteConfig(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockConfigManager) ValidateConfig(config []byte) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigManager) GetRunningConfig() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockConfigManager) GetCandidateConfig() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockConfigManager) CommitConfig() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) RollbackConfig() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockFaultManager struct {
	mock.Mock
}

func (m *MockFaultManager) ReportFault(fault *O1Fault) error {
	args := m.Called(fault)
	return args.Error(0)
}

func (m *MockFaultManager) GetActiveFaults() ([]*O1Fault, error) {
	args := m.Called()
	return args.Get(0).([]*O1Fault), args.Error(1)
}

func (m *MockFaultManager) ClearFault(faultID string) error {
	args := m.Called(faultID)
	return args.Error(0)
}

func (m *MockFaultManager) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockPerformanceManager struct {
	mock.Mock
}

func (m *MockPerformanceManager) CollectMetrics() (*O1PerformanceData, error) {
	args := m.Called()
	return args.Get(0).(*O1PerformanceData), args.Error(1)
}

func (m *MockPerformanceManager) GetPerformanceData(startTime, endTime time.Time) ([]*O1PerformanceData, error) {
	args := m.Called(startTime, endTime)
	return args.Get(0).([]*O1PerformanceData), args.Error(1)
}

func (m *MockPerformanceManager) StartCollection() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPerformanceManager) StopCollection() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPerformanceManager) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

type MockO1Metrics struct {
	mock.Mock
}

func (m *MockO1Metrics) Register() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockO1Metrics) Unregister() {
	m.Called()
}

// Test configuration helper
func createTestO1Config() *config.O1Config {
	return &config.O1Config{
		Enabled:     true,
		Port:        830,
		BindAddress: "127.0.0.1",
		SSH: config.SSHConfig{
			HostKeyPath:        "/tmp/test_host_key",
			AuthorizedKeysPath: "/tmp/test_authorized_keys",
			Timeout:           30 * time.Second,
			MaxConnections:    10,
		},
		NETCONF: config.NETCONFConfig{
			Capabilities: []string{
				"urn:ietf:params:netconf:base:1.0",
				"urn:ietf:params:netconf:capability:startup:1.0",
			},
			SessionTimeout: 600 * time.Second,
			MaxSessions:    10,
		},
		YANG: config.YANGConfig{
			ModulesPath:    "/etc/near-rt-ric/yang",
			ValidateConfig: true,
		},
		FileManagement: config.FileManagementConfig{
			Enabled:     true,
			UploadPath:  "/var/lib/near-rt-ric/uploads",
			MaxFileSize: 100 * 1024 * 1024, // 100MB
		},
		SoftwareManagement: config.SoftwareManagementConfig{
			Enabled:      true,
			PackagePath:  "/var/lib/near-rt-ric/packages",
			InstallPath:  "/opt/near-rt-ric",
			BackupPath:   "/var/lib/near-rt-ric/backups",
			MaxPackages:  10,
		},
	}
}

// TestNewO1Interface tests the creation of a new O1Interface
func TestNewO1Interface(t *testing.T) {
	cfg := createTestO1Config()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test basic configuration
	assert.NotNil(t, cfg)
	assert.Equal(t, 830, cfg.Port)
	assert.Equal(t, "127.0.0.1", cfg.BindAddress)
	assert.True(t, cfg.Enabled)
}

// TestO1InterfaceLifecycle tests the start and stop functionality
func TestO1InterfaceLifecycle(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	ctx := context.Background()

	// Test starting the interface
	err := o1Interface.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, o1Interface.running)

	// Test starting already running interface
	err = o1Interface.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stopping the interface
	err = o1Interface.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, o1Interface.running)

	// Test stopping already stopped interface
	err = o1Interface.Stop(ctx)
	assert.NoError(t, err)
}

// TestO1InterfaceHealthCheck tests the health check functionality
func TestO1InterfaceHealthCheck(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Health check when not running
	err := o1Interface.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Start interface and test health check
	ctx := context.Background()
	err = o1Interface.Start(ctx)
	require.NoError(t, err)

	// Mock successful health checks
	o1Interface.netconfServer.(*MockNETCONFServer).On("HealthCheck").Return(nil)
	o1Interface.yangRepo.(*MockYANGRepository).On("HealthCheck").Return(nil)
	o1Interface.configMgr.(*MockConfigManager).On("HealthCheck").Return(nil)
	o1Interface.faultMgr.(*MockFaultManager).On("HealthCheck").Return(nil)
	o1Interface.perfMgr.(*MockPerformanceManager).On("HealthCheck").Return(nil)

	err = o1Interface.HealthCheck()
	assert.NoError(t, err)

	// Clean up
	err = o1Interface.Stop(ctx)
	require.NoError(t, err)
}

// TestO1InterfaceGetStatus tests the status reporting functionality
func TestO1InterfaceGetStatus(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Mock return values
	o1Interface.netconfServer.(*MockNETCONFServer).On("GetActiveConnections").Return(3)
	o1Interface.netconfServer.(*MockNETCONFServer).On("GetCapabilities").Return([]string{"base:1.0", "startup:1.0"})
	o1Interface.faultMgr.(*MockFaultManager).On("GetActiveFaults").Return([]*O1Fault{}, nil)

	status := o1Interface.GetStatus()

	assert.NotNil(t, status)
	assert.Equal(t, "127.0.0.1", status.BindAddress)
	assert.Equal(t, 830, status.Port)
	assert.Equal(t, 3, status.ActiveConnections)
	assert.Equal(t, 10, status.MaxConnections)
	assert.Len(t, status.Capabilities, 2)
}

// TestNETCONFSessionHandling tests NETCONF session management
func TestNETCONFSessionHandling(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	ctx := context.Background()
	err := o1Interface.Start(ctx)
	require.NoError(t, err)

	// Test session creation
	sessionID := "test-session-123"
	session := &NETCONFSession{
		SessionID:   sessionID,
		UserName:    "test-user",
		ClientAddr:  "192.168.1.100",
		StartTime:   time.Now(),
		LastActivity: time.Now(),
		Capabilities: []string{"base:1.0"},
	}

	o1Interface.addSession(session)

	// Test getting session
	retrievedSession := o1Interface.getSession(sessionID)
	assert.NotNil(t, retrievedSession)
	assert.Equal(t, sessionID, retrievedSession.SessionID)
	assert.Equal(t, "test-user", retrievedSession.UserName)

	// Test removing session
	o1Interface.removeSession(sessionID)
	removedSession := o1Interface.getSession(sessionID)
	assert.Nil(t, removedSession)

	// Clean up
	err = o1Interface.Stop(ctx)
	require.NoError(t, err)
}

// TestConfigurationManagement tests configuration operations
func TestConfigurationManagement(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test getting configuration
	testConfig := map[string]interface{}{
		"system": map[string]interface{}{
			"hostname": "test-ric",
			"location": "test-lab",
		},
	}

	o1Interface.configMgr.(*MockConfigManager).On("GetConfig", "/system").Return(testConfig, nil)

	config, err := o1Interface.GetConfig("/system")
	assert.NoError(t, err)
	assert.Equal(t, testConfig, config)

	// Test setting configuration
	newConfig := map[string]interface{}{
		"hostname": "new-ric",
	}

	o1Interface.configMgr.(*MockConfigManager).On("SetConfig", "/system/hostname", "new-ric").Return(nil)

	err = o1Interface.SetConfig("/system/hostname", "new-ric")
	assert.NoError(t, err)

	// Test committing configuration
	o1Interface.configMgr.(*MockConfigManager).On("CommitConfig").Return(nil)

	err = o1Interface.CommitConfig()
	assert.NoError(t, err)
}

// TestYANGModelManagement tests YANG model operations
func TestYANGModelManagement(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test loading YANG module
	moduleData := []byte(`
		module test-module {
			namespace "http://example.com/test";
			prefix "test";
			
			leaf hostname {
				type string;
				description "System hostname";
			}
		}
	`)

	o1Interface.yangRepo.(*MockYANGRepository).On("LoadModule", "test-module", moduleData).Return(nil)

	err := o1Interface.LoadYANGModule("test-module", moduleData)
	assert.NoError(t, err)

	// Test getting YANG module
	testModule := &YANGModule{
		Name:        "test-module",
		Namespace:   "http://example.com/test",
		Prefix:      "test",
		Version:     "1.0",
		Description: "Test YANG module",
		LoadTime:    time.Now(),
	}

	o1Interface.yangRepo.(*MockYANGRepository).On("GetModule", "test-module").Return(testModule, nil)

	module, err := o1Interface.GetYANGModule("test-module")
	assert.NoError(t, err)
	assert.Equal(t, "test-module", module.Name)
	assert.Equal(t, "http://example.com/test", module.Namespace)
}

// TestFaultManagement tests fault management functionality
func TestFaultManagement(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test reporting fault
	fault := &O1Fault{
		FaultID:     "test-fault-001",
		Severity:    FaultSeverityMajor,
		Source:      "e2-interface",
		Description: "E2 connection timeout",
		Timestamp:   time.Now(),
		Status:      FaultStatusActive,
	}

	o1Interface.faultMgr.(*MockFaultManager).On("ReportFault", fault).Return(nil)

	err := o1Interface.ReportFault(fault)
	assert.NoError(t, err)

	// Test getting active faults
	activeFaults := []*O1Fault{fault}
	o1Interface.faultMgr.(*MockFaultManager).On("GetActiveFaults").Return(activeFaults, nil)

	faults, err := o1Interface.GetActiveFaults()
	assert.NoError(t, err)
	assert.Len(t, faults, 1)
	assert.Equal(t, "test-fault-001", faults[0].FaultID)

	// Test clearing fault
	o1Interface.faultMgr.(*MockFaultManager).On("ClearFault", "test-fault-001").Return(nil)

	err = o1Interface.ClearFault("test-fault-001")
	assert.NoError(t, err)
}

// TestPerformanceManagement tests performance management functionality
func TestPerformanceManagement(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test collecting metrics
	perfData := &O1PerformanceData{
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 60.2,
			"disk_usage":   45.0,
			"network_throughput": map[string]interface{}{
				"rx_bytes": 1024000,
				"tx_bytes": 512000,
			},
		},
		Source: "system-monitor",
	}

	o1Interface.perfMgr.(*MockPerformanceManager).On("CollectMetrics").Return(perfData, nil)

	metrics, err := o1Interface.CollectPerformanceMetrics()
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, 75.5, metrics.Metrics["cpu_usage"])

	// Test getting historical performance data
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()
	historicalData := []*O1PerformanceData{perfData}

	o1Interface.perfMgr.(*MockPerformanceManager).On("GetPerformanceData", startTime, endTime).Return(historicalData, nil)

	data, err := o1Interface.GetPerformanceData(startTime, endTime)
	assert.NoError(t, err)
	assert.Len(t, data, 1)
}

// TestFileManagement tests file management operations
func TestFileManagement(t *testing.T) {
	o1Interface := createTestO1Interface(t)

	// Test file upload
	fileData := []byte("test file content")
	fileName := "test-config.xml"

	err := o1Interface.UploadFile(fileName, fileData)
	assert.NoError(t, err)

	// Test file download
	downloadedData, err := o1Interface.DownloadFile(fileName)
	assert.NoError(t, err)
	assert.Equal(t, fileData, downloadedData)

	// Test file deletion
	err = o1Interface.DeleteFile(fileName)
	assert.NoError(t, err)
}

// TestSSHKeyGeneration tests SSH key generation for NETCONF
func TestSSHKeyGeneration(t *testing.T) {
	// Generate test SSH key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to SSH format
	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyBytes := ssh.MarshalAuthorizedKey(sshPublicKey)
	assert.NotEmpty(t, publicKeyBytes)

	// Convert private key to PEM format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	assert.NotEmpty(t, privateKeyPEM)
}

// BenchmarkO1RequestProcessing benchmarks O1 request processing
func BenchmarkO1RequestProcessing(b *testing.B) {
	o1Interface := createTestO1Interface(nil)

	// Setup mocks for benchmarking
	testConfig := map[string]interface{}{"test": "value"}
	o1Interface.configMgr.(*MockConfigManager).On("GetConfig", "/test").Return(testConfig, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = o1Interface.GetConfig("/test")
	}
}

// Helper function to create a test O1Interface with mocks
func createTestO1Interface(t *testing.T) *O1Interface {
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
	mockYANGRepo := &MockYANGRepository{}
	mockConfigMgr := &MockConfigManager{}
	mockFaultMgr := &MockFaultManager{}
	mockPerfMgr := &MockPerformanceManager{}
	mockMetrics := &MockO1Metrics{}

	// Set up mock expectations for basic operations
	mockNETCONFServer.On("Start", mock.Anything).Return(nil)
	mockNETCONFServer.On("Stop", mock.Anything).Return(nil)
	mockMetrics.On("Register").Return(nil)
	mockMetrics.On("Unregister").Return()

	// Create interface
	o1Interface := &O1Interface{
		config:       cfg,
		netconfServer: mockNETCONFServer,
		yangRepo:     mockYANGRepo,
		configMgr:    mockConfigMgr,
		faultMgr:     mockFaultMgr,
		perfMgr:      mockPerfMgr,
		logger:       logger.WithField("component", "o1-interface"),
		metrics:      mockMetrics,
		sessions:     make(map[string]*NETCONFSession),
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
		startTime:    time.Now(),
	}

	return o1Interface
}
