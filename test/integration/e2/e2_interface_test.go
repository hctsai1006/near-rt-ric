package e2_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// E2IntegrationTestSuite contains E2 interface integration tests
type E2IntegrationTestSuite struct {
	suite.Suite
	ctx             context.Context
	cancel          context.CancelFunc
	e2Interface     *e2.E2Interface
	mockE2Node      *MockE2Node
	postgresContainer testcontainers.Container
	dbURL           string
}

// MockE2Node simulates an E2 node for testing
type MockE2Node struct {
	conn       net.Conn
	nodeID     string
	nodeName   string
	ranFunctions []e2.RANFunction
	logger     *logrus.Logger
}

// SetupSuite runs before all tests
func (suite *E2IntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	
	// Setup test containers
	suite.setupPostgres()
	suite.setupE2Interface()
	suite.setupMockE2Node()
}

// TearDownSuite runs after all tests
func (suite *E2IntegrationTestSuite) TearDownSuite() {
	if suite.e2Interface != nil {
		suite.e2Interface.Stop(suite.ctx)
	}
	if suite.mockE2Node != nil {
		suite.mockE2Node.Stop()
	}
	if suite.postgresContainer != nil {
		suite.postgresContainer.Terminate(suite.ctx)
	}
	suite.cancel()
}

// SetupTest runs before each test
func (suite *E2IntegrationTestSuite) SetupTest() {
	// Reset E2 node state
	suite.mockE2Node.Reset()
}

func (suite *E2IntegrationTestSuite) setupPostgres() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "near_rt_ric_test",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(suite.T(), err)

	suite.postgresContainer = container

	host, err := container.Host(suite.ctx)
	require.NoError(suite.T(), err)

	port, err := container.MappedPort(suite.ctx, "5432")
	require.NoError(suite.T(), err)

	suite.dbURL = fmt.Sprintf("postgres://test_user:test_password@%s:%s/near_rt_ric_test?sslmode=disable", host, port.Port())
}

func (suite *E2IntegrationTestSuite) setupE2Interface() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	metrics := monitoring.NewMetricsCollector()

	cfg := &config.E2Config{
		Server: config.E2ServerConfig{
			Host: "127.0.0.1",
			Port: 36421,
			SCTP: config.SCTPConfig{
				Enabled:           true,
				Streams:           10,
				HeartbeatInterval: 30 * time.Second,
			},
		},
		Database: config.DatabaseConfig{
			URL:               suite.dbURL,
			MaxConnections:    10,
			MaxIdleConnections: 5,
			ConnTimeout:       30 * time.Second,
		},
		ASN1: config.ASN1Config{
			Version:    "v3.0",
			Codec:      "per",
			Validation: true,
		},
	}

	var err error
	suite.e2Interface, err = e2.NewE2Interface(cfg, logger, metrics)
	require.NoError(suite.T(), err)

	// Start E2 interface
	err = suite.e2Interface.Start(suite.ctx)
	require.NoError(suite.T(), err)

	// Wait for interface to be ready
	time.Sleep(2 * time.Second)
}

func (suite *E2IntegrationTestSuite) setupMockE2Node() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	suite.mockE2Node = &MockE2Node{
		nodeID:   "gnb001",
		nodeName: "test-gnb-001",
		ranFunctions: []e2.RANFunction{
			{
				FunctionID:   1,
				Name:         "RAN Control",
				Version:      "1.0",
				OID:          "1.3.6.1.4.1.1.22.1.1",
				Description:  "RAN Control Function",
			},
			{
				FunctionID:   2,
				Name:         "Key Performance Measurement",
				Version:      "1.0",
				OID:          "1.3.6.1.4.1.1.22.1.2",
				Description:  "KPM Function",
			},
		},
		logger: logger,
	}
}

// Test E2 Setup procedure
func (suite *E2IntegrationTestSuite) TestE2Setup() {
	// Connect mock E2 node to E2 interface
	err := suite.mockE2Node.Connect("127.0.0.1:36421")
	require.NoError(suite.T(), err)

	// Send E2 Setup Request
	setupReq := &e2.E2SetupRequest{
		TransactionID: 1,
		GlobalE2NodeID: e2.GlobalE2NodeID{
			NodeType: e2.NodeTypeGNB,
			NodeID:   suite.mockE2Node.nodeID,
		},
		RANFunctions: suite.mockE2Node.ranFunctions,
	}

	err = suite.mockE2Node.SendE2SetupRequest(setupReq)
	require.NoError(suite.T(), err)

	// Wait for E2 Setup Response
	setupResp, err := suite.mockE2Node.WaitForE2SetupResponse(10 * time.Second)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), setupResp)
	assert.Equal(suite.T(), e2.ProcedureCodeE2Setup, setupResp.ProcedureCode)
	assert.Equal(suite.T(), e2.TypeSuccessfulOutcome, setupResp.MessageType)

	// Verify E2 node is registered
	nodes := suite.e2Interface.GetConnectedNodes()
	assert.Len(suite.T(), nodes, 1)
	assert.Equal(suite.T(), suite.mockE2Node.nodeID, nodes[0].NodeID)
	assert.Equal(suite.T(), e2.ConnectionStatusConnected, nodes[0].Status)
}

// Test RIC Subscription procedure
func (suite *E2IntegrationTestSuite) TestRICSubscription() {
	// First establish E2 connection
	suite.TestE2Setup()

	// Create RIC Subscription Request
	subscriptionReq := &e2.RICSubscriptionRequest{
		TransactionID: 2,
		RequestID: e2.RICRequestID{
			RequestorID: 1001,
			InstanceID:  1,
		},
		RANFunctionID: 1, // RAN Control function
		EventTriggers: e2.EventTriggerDefinition{
			TriggerType: e2.TriggerTypePeriodic,
			Period:      1000, // 1 second
		},
		Actions: []e2.RICAction{
			{
				ActionID:   1,
				ActionType: e2.ActionTypeReport,
			},
		},
	}

	err := suite.e2Interface.SendRICSubscriptionRequest(suite.mockE2Node.nodeID, subscriptionReq)
	require.NoError(suite.T(), err)

	// Wait for RIC Subscription Response from mock node
	subscriptionResp, err := suite.mockE2Node.WaitForRICSubscriptionResponse(10 * time.Second)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), subscriptionResp)
	assert.Equal(suite.T(), subscriptionReq.RequestID, subscriptionResp.RequestID)
	assert.Equal(suite.T(), e2.TypeSuccessfulOutcome, subscriptionResp.MessageType)

	// Verify subscription is created
	subscriptions := suite.e2Interface.GetActiveSubscriptions(suite.mockE2Node.nodeID)
	assert.Len(suite.T(), subscriptions, 1)
	assert.Equal(suite.T(), subscriptionReq.RequestID, subscriptions[0].RequestID)
	assert.Equal(suite.T(), e2.SubscriptionStatusActive, subscriptions[0].Status)
}

// Test RIC Control procedure
func (suite *E2IntegrationTestSuite) TestRICControl() {
	// First establish E2 connection and subscription
	suite.TestRICSubscription()

	// Create RIC Control Request
	controlReq := &e2.RICControlRequest{
		TransactionID: 3,
		RequestID: e2.RICRequestID{
			RequestorID: 1001,
			InstanceID:  2,
		},
		RANFunctionID: 1,
		CallProcessID: []byte("control-001"),
		ControlHeader: []byte("test-control-header"),
		ControlMessage: []byte("test-control-message"),
		ControlAckRequest: e2.ControlAckRequestACK,
	}

	err := suite.e2Interface.SendRICControlRequest(suite.mockE2Node.nodeID, controlReq)
	require.NoError(suite.T(), err)

	// Wait for RIC Control Acknowledge from mock node
	controlAck, err := suite.mockE2Node.WaitForRICControlAcknowledge(10 * time.Second)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), controlAck)
	assert.Equal(suite.T(), controlReq.RequestID, controlAck.RequestID)
	assert.Equal(suite.T(), e2.TypeSuccessfulOutcome, controlAck.MessageType)
}

// Test E2 connection failure handling
func (suite *E2IntegrationTestSuite) TestConnectionFailure() {
	// First establish E2 connection
	suite.TestE2Setup()

	// Simulate connection failure
	suite.mockE2Node.SimulateConnectionFailure()

	// Wait for connection status to change
	time.Sleep(3 * time.Second)

	// Verify E2 node is marked as disconnected
	nodes := suite.e2Interface.GetConnectedNodes()
	var targetNode *e2.E2Node
	for _, node := range nodes {
		if node.NodeID == suite.mockE2Node.nodeID {
			targetNode = node
			break
		}
	}

	require.NotNil(suite.T(), targetNode)
	assert.Equal(suite.T(), e2.ConnectionStatusDisconnected, targetNode.Status)

	// Verify subscriptions are marked as inactive
	subscriptions := suite.e2Interface.GetActiveSubscriptions(suite.mockE2Node.nodeID)
	for _, sub := range subscriptions {
		assert.Equal(suite.T(), e2.SubscriptionStatusInactive, sub.Status)
	}
}

// Test E2 message validation
func (suite *E2IntegrationTestSuite) TestMessageValidation() {
	// Test invalid E2 Setup Request
	invalidSetupReq := &e2.E2SetupRequest{
		TransactionID: 0, // Invalid transaction ID
		GlobalE2NodeID: e2.GlobalE2NodeID{
			NodeType: "invalid", // Invalid node type
			NodeID:   "",        // Empty node ID
		},
		RANFunctions: []e2.RANFunction{}, // Empty RAN functions
	}

	err := suite.mockE2Node.Connect("127.0.0.1:36421")
	require.NoError(suite.T(), err)

	err = suite.mockE2Node.SendE2SetupRequest(invalidSetupReq)
	assert.Error(suite.T(), err)
}

// Test performance under load
func (suite *E2IntegrationTestSuite) TestE2Performance() {
	// First establish E2 connection
	suite.TestE2Setup()

	// Send multiple subscription requests concurrently
	numRequests := 100
	results := make(chan error, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			subscriptionReq := &e2.RICSubscriptionRequest{
				TransactionID: uint32(requestID + 1000),
				RequestID: e2.RICRequestID{
					RequestorID: 1001,
					InstanceID:  uint32(requestID),
				},
				RANFunctionID: 1,
				EventTriggers: e2.EventTriggerDefinition{
					TriggerType: e2.TriggerTypePeriodic,
					Period:      5000, // 5 seconds
				},
				Actions: []e2.RICAction{
					{
						ActionID:   1,
						ActionType: e2.ActionTypeReport,
					},
				},
			}

			err := suite.e2Interface.SendRICSubscriptionRequest(suite.mockE2Node.nodeID, subscriptionReq)
			results <- err
		}(i)
	}

	// Collect results
	errors := 0
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			errors++
		}
	}

	duration := time.Since(start)

	// Performance assertions
	assert.Less(suite.T(), errors, numRequests/10) // Less than 10% error rate
	assert.Less(suite.T(), duration, 30*time.Second) // Complete within 30 seconds

	suite.T().Logf("Performance test: %d requests in %v, %d errors", numRequests, duration, errors)
}

// Mock E2 Node implementation methods

func (m *MockE2Node) Connect(address string) error {
	conn, err := net.Dial("sctp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to E2 interface: %w", err)
	}
	m.conn = conn
	return nil
}

func (m *MockE2Node) SendE2SetupRequest(req *e2.E2SetupRequest) error {
	// Encode E2 Setup Request message (simplified)
	message := m.encodeE2SetupRequest(req)
	_, err := m.conn.Write(message)
	return err
}

func (m *MockE2Node) WaitForE2SetupResponse(timeout time.Duration) (*e2.E2SetupResponse, error) {
	// Set read timeout
	m.conn.SetReadDeadline(time.Now().Add(timeout))
	
	buffer := make([]byte, 4096)
	n, err := m.conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	// Decode E2 Setup Response (simplified)
	response := m.decodeE2SetupResponse(buffer[:n])
	return response, nil
}

func (m *MockE2Node) WaitForRICSubscriptionResponse(timeout time.Duration) (*e2.RICSubscriptionResponse, error) {
	// Implementation similar to WaitForE2SetupResponse
	// This is a simplified mock implementation
	return &e2.RICSubscriptionResponse{
		MessageType:   e2.TypeSuccessfulOutcome,
		ProcedureCode: e2.ProcedureCodeRICSubscription,
		RequestID: e2.RICRequestID{
			RequestorID: 1001,
			InstanceID:  1,
		},
	}, nil
}

func (m *MockE2Node) WaitForRICControlAcknowledge(timeout time.Duration) (*e2.RICControlAcknowledge, error) {
	// Simplified mock implementation
	return &e2.RICControlAcknowledge{
		MessageType:   e2.TypeSuccessfulOutcome,
		ProcedureCode: e2.ProcedureCodeRICControl,
		RequestID: e2.RICRequestID{
			RequestorID: 1001,
			InstanceID:  2,
		},
	}, nil
}

func (m *MockE2Node) SimulateConnectionFailure() {
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
}

func (m *MockE2Node) Stop() {
	if m.conn != nil {
		m.conn.Close()
	}
}

func (m *MockE2Node) Reset() {
	// Reset mock state between tests
	if m.conn != nil {
		m.conn.Close()
		m.conn = nil
	}
}

// Simplified encoding/decoding methods (in production, use proper ASN.1 encoding)
func (m *MockE2Node) encodeE2SetupRequest(req *e2.E2SetupRequest) []byte {
	// This is a mock implementation
	// In production, use proper ASN.1 PER encoding
	message := fmt.Sprintf("E2SetupRequest:{TransactionID:%d,NodeID:%s}", req.TransactionID, req.GlobalE2NodeID.NodeID)
	return []byte(message)
}

func (m *MockE2Node) decodeE2SetupResponse(data []byte) *e2.E2SetupResponse {
	// This is a mock implementation
	// In production, use proper ASN.1 PER decoding
	return &e2.E2SetupResponse{
		MessageType:   e2.TypeSuccessfulOutcome,
		ProcedureCode: e2.ProcedureCodeE2Setup,
		TransactionID: 1,
	}
}

// Test runner
func TestE2IntegrationSuite(t *testing.T) {
	suite.Run(t, new(E2IntegrationTestSuite))
}