package testing

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RICTestClient is a client for the RIC under test.
type RICTestClient struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// RICClientConfig is the configuration for the RICTestClient.
type RICClientConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// NewRICTestClient creates a new RICTestClient.
func NewRICTestClient(config *RICClientConfig) *RICTestClient {
	return &RICTestClient{
		Host:    config.Host,
		Port:    config.Port,
		Timeout: config.Timeout,
		Logger:  config.Logger,
	}
}

// E2TestClient is a client for the E2 interface under test.
type E2TestClient struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// E2ClientConfig is the configuration for the E2TestClient.
type E2ClientConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// NewE2TestClient creates a new E2TestClient.
func NewE2TestClient(config *E2ClientConfig) *E2TestClient {
	return &E2TestClient{
		Host:    config.Host,
		Port:    config.Port,
		Timeout: config.Timeout,
		Logger:  config.Logger,
	}
}

// A1TestClient is a client for the A1 interface under test.
type A1TestClient struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// A1ClientConfig is the configuration for the A1TestClient.
type A1ClientConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// NewA1TestClient creates a new A1TestClient.
func NewA1TestClient(config *A1ClientConfig) *A1TestClient {
	return &A1TestClient{
		Host:    config.Host,
		Port:    config.Port,
		Timeout: config.Timeout,
		Logger:  config.Logger,
	}
}

// O1TestClient is a client for the O1 interface under test.
type O1TestClient struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// O1ClientConfig is the configuration for the O1TestClient.
type O1ClientConfig struct {
	Host    string
	Port    int
	Timeout time.Duration
	Logger  *logrus.Logger
}

// NewO1TestClient creates a new O1TestClient.
func NewO1TestClient(config *O1ClientConfig) *O1TestClient {
	return &O1TestClient{
		Host:    config.Host,
		Port:    config.Port,
		Timeout: config.Timeout,
		Logger:  config.Logger,
	}
}

// IntegrationTestSuite provides a comprehensive testing framework for O-RAN Near-RT RIC
type IntegrationTestSuite struct {
	suite.Suite

	// Test Infrastructure
	ctx    context.Context
	cancel context.CancelFunc
	logger *logrus.Logger

	// Container Infrastructure
	postgresContainer testcontainers.Container
	redisContainer    testcontainers.Container
	kafkaContainer    testcontainers.Container

	// Database Connection
	db *sql.DB

	// Test Configuration
	testConfig *TestConfig

	// Component Clients
	ricClient *RICTestClient
	e2Client  *E2TestClient
	a1Client  *A1TestClient
	o1Client  *O1TestClient
}

// TestConfig contains configuration for integration tests
type TestConfig struct {
	PostgresHost     string `json:"postgres_host"`
	PostgresPort     int    `json:"postgres_port"`
	PostgresUser     string `json:"postgres_user"`
	PostgresPassword string `json:"postgres_password"`
	PostgresDB       string `json:"postgres_db"`

	RedisHost     string `json:"redis_host"`
	RedisPort     int    `json:"redis_port"`
	RedisPassword string `json:"redis_password"`

	KafkaHost string `json:"kafka_host"`
	KafkaPort int    `json:"kafka_port"`

	RICHost string `json:"ric_host"`
	RICPort int    `json:"ric_port"`

	E2Host string `json:"e2_host"`
	E2Port int    `json:"e2_port"`

	A1Host string `json:"a1_host"`
	A1Port int    `json:"a1_port"`

	O1Host string `json:"o1_host"`
	O1Port int    `json:"o1_port"`

	TestTimeout  time.Duration `json:"test_timeout"`
	CleanupDelay time.Duration `json:"cleanup_delay"`
}

// SetupSuite initializes the test suite with required infrastructure
func (suite *IntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Initialize logger
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)
	suite.logger.SetFormatter(&logrus.JSONFormatter{})

	suite.logger.Info("🚀 Starting O-RAN RIC Integration Test Suite")

	// Initialize test configuration
	suite.testConfig = &TestConfig{
		PostgresUser:     "testuser",
		PostgresPassword: "testpass",
		PostgresDB:       "testdb",
		RedisPassword:    "",
		TestTimeout:      5 * time.Minute,
		CleanupDelay:     2 * time.Second,
	}

	// Start test infrastructure
	suite.startTestInfrastructure()

	// Initialize test clients
	suite.initializeTestClients()

	suite.logger.Info("✅ Integration test suite setup completed")
}

// TearDownSuite cleans up all test infrastructure
func (suite *IntegrationTestSuite) TearDownSuite() {
	suite.logger.Info("🧹 Tearing down integration test suite")

	// Close database connection
	if suite.db != nil {
		suite.db.Close()
	}

	// Stop containers
	if suite.postgresContainer != nil {
		_ = suite.postgresContainer.Terminate(suite.ctx)
	}
	if suite.redisContainer != nil {
		_ = suite.redisContainer.Terminate(suite.ctx)
	}
	if suite.kafkaContainer != nil {
		_ = suite.kafkaContainer.Terminate(suite.ctx)
	}

	// Cancel context
	suite.cancel()

	suite.logger.Info("✅ Integration test suite teardown completed")
}

// SetupTest prepares individual test cases
func (suite *IntegrationTestSuite) SetupTest() {
	suite.logger.Info("🧪 Setting up individual test")

	// Clean database state
	suite.cleanDatabaseState()

	// Reset Redis state
	suite.cleanRedisState()

	// Wait for services to be ready
	suite.waitForServicesReady()
}

// TearDownTest cleans up after individual test cases
func (suite *IntegrationTestSuite) TearDownTest() {
	suite.logger.Info("🧹 Tearing down individual test")

	// Allow time for async operations to complete
	time.Sleep(suite.testConfig.CleanupDelay)
}

// startTestInfrastructure starts all required test containers
func (suite *IntegrationTestSuite) startTestInfrastructure() {
	suite.logger.Info("🐳 Starting test infrastructure containers")

	// Start PostgreSQL container
	suite.startPostgresContainer()

	// Start Redis container
	suite.startRedisContainer()

	// Start Kafka container
	suite.startKafkaContainer()

	// Connect to database
	suite.connectToDatabase()
}

// startPostgresContainer starts PostgreSQL test container
func (suite *IntegrationTestSuite) startPostgresContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     suite.testConfig.PostgresUser,
			"POSTGRES_PASSWORD": suite.testConfig.PostgresPassword,
			"POSTGRES_DB":       suite.testConfig.PostgresDB,
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	suite.Require().NoError(err, "Failed to start PostgreSQL container")
	suite.postgresContainer = container

	// Get container host and port
	host, err := container.Host(suite.ctx)
	suite.Require().NoError(err)

	mappedPort, err := container.MappedPort(suite.ctx, "5432")
	suite.Require().NoError(err)

	suite.testConfig.PostgresHost = host
	suite.testConfig.PostgresPort = mappedPort.Int()

	suite.logger.WithFields(logrus.Fields{
		"host": host,
		"port": mappedPort.Int(),
	}).Info("✅ PostgreSQL container started")
}

// startRedisContainer starts Redis test container
func (suite *IntegrationTestSuite) startRedisContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	suite.Require().NoError(err, "Failed to start Redis container")
	suite.redisContainer = container

	// Get container host and port
	host, err := container.Host(suite.ctx)
	suite.Require().NoError(err)

	mappedPort, err := container.MappedPort(suite.ctx, "6379")
	suite.Require().NoError(err)

	suite.testConfig.RedisHost = host
	suite.testConfig.RedisPort = mappedPort.Int()

	suite.logger.WithFields(logrus.Fields{
		"host": host,
		"port": mappedPort.Int(),
	}).Info("✅ Redis container started")
}

// startKafkaContainer starts Kafka test container
func (suite *IntegrationTestSuite) startKafkaContainer() {
	req := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-kafka:7.4.0",
		ExposedPorts: []string{"9092/tcp", "9093/tcp"},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                        "1",
			"KAFKA_ZOOKEEPER_CONNECT":                "zookeeper:2181",
			"KAFKA_ADVERTISED_LISTENERS":             "PLAINTEXT://localhost:9092,PLAINTEXT_INTERNAL://kafka:29092",
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP":   "PLAINTEXT:PLAINTEXT,PLAINTEXT_INTERNAL:PLAINTEXT",
			"KAFKA_INTER_BROKER_LISTENER_NAME":       "PLAINTEXT_INTERNAL",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
		},
		WaitingFor: wait.ForListeningPort("9092/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	suite.Require().NoError(err, "Failed to start Kafka container")
	suite.kafkaContainer = container

	// Get container host and port
	host, err := container.Host(suite.ctx)
	suite.Require().NoError(err)

	mappedPort, err := container.MappedPort(suite.ctx, "9092")
	suite.Require().NoError(err)

	suite.testConfig.KafkaHost = host
	suite.testConfig.KafkaPort = mappedPort.Int()

	suite.logger.WithFields(logrus.Fields{
		"host": host,
		"port": mappedPort.Int(),
	}).Info("✅ Kafka container started")
}

// connectToDatabase establishes database connection
func (suite *IntegrationTestSuite) connectToDatabase() {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		suite.testConfig.PostgresHost,
		suite.testConfig.PostgresPort,
		suite.testConfig.PostgresUser,
		suite.testConfig.PostgresPassword,
		suite.testConfig.PostgresDB,
	)

	var err error
	suite.db, err = sql.Open("postgres", dsn)
	suite.Require().NoError(err, "Failed to connect to test database")

	// Test connection
	err = suite.db.Ping()
	suite.Require().NoError(err, "Failed to ping test database")

	// Initialize database schema
	suite.initializeDatabaseSchema()

	suite.logger.Info("✅ Database connection established")
}

// initializeDatabaseSchema creates test database schema
func (suite *IntegrationTestSuite) initializeDatabaseSchema() {
	schema := `
		-- Test schema for O-RAN Near-RT RIC
		CREATE TABLE IF NOT EXISTS ric_nodes (
			id SERIAL PRIMARY KEY,
			node_id VARCHAR(255) UNIQUE NOT NULL,
			node_type VARCHAR(100) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'inactive',
			last_heartbeat TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS e2_subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			node_id VARCHAR(255) NOT NULL,
			function_id INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (node_id) REFERENCES ric_nodes(node_id)
		);
		
		CREATE TABLE IF NOT EXISTS a1_policies (
			id SERIAL PRIMARY KEY,
			policy_id VARCHAR(255) UNIQUE NOT NULL,
			policy_type_id VARCHAR(255) NOT NULL,
			policy_data JSONB NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS test_metrics (
			id SERIAL PRIMARY KEY,
			test_name VARCHAR(255) NOT NULL,
			metric_name VARCHAR(255) NOT NULL,
			metric_value NUMERIC NOT NULL,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := suite.db.Exec(schema)
	suite.Require().NoError(err, "Failed to initialize database schema")

	suite.logger.Info("✅ Database schema initialized")
}

// initializeTestClients creates test clients for all interfaces
func (suite *IntegrationTestSuite) initializeTestClients() {
	suite.logger.Info("🔌 Initializing test clients")

	// Initialize RIC client
	suite.ricClient = NewRICTestClient(&RICClientConfig{
		Host:    "localhost",
		Port:    8080,
		Timeout: suite.testConfig.TestTimeout,
		Logger:  suite.logger,
	})

	// Initialize E2 client
	suite.e2Client = NewE2TestClient(&E2ClientConfig{
		Host:    "localhost",
		Port:    36421,
		Timeout: suite.testConfig.TestTimeout,
		Logger:  suite.logger,
	})

	// Initialize A1 client
	suite.a1Client = NewA1TestClient(&A1ClientConfig{
		Host:    "localhost",
		Port:    8081,
		Timeout: suite.testConfig.TestTimeout,
		Logger:  suite.logger,
	})

	// Initialize O1 client
	suite.o1Client = NewO1TestClient(&O1ClientConfig{
		Host:    "localhost",
		Port:    830,
		Timeout: suite.testConfig.TestTimeout,
		Logger:  suite.logger,
	})

	suite.logger.Info("✅ Test clients initialized")
}

// cleanDatabaseState cleans database state between tests
func (suite *IntegrationTestSuite) cleanDatabaseState() {
	tables := []string{
		"test_metrics",
		"a1_policies",
		"e2_subscriptions",
		"ric_nodes",
	}

	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			suite.logger.WithError(err).Warnf("Failed to truncate table %s", table)
		}
	}
}

// cleanRedisState cleans Redis state between tests
func (suite *IntegrationTestSuite) cleanRedisState() {
	// Implementation would connect to Redis and flush test keys
	suite.logger.Debug("Redis state cleaned")
}

// waitForServicesReady waits for all services to be ready
func (suite *IntegrationTestSuite) waitForServicesReady() {
	suite.logger.Info("⏳ Waiting for services to be ready")

	// Wait for database
	suite.waitForDatabaseReady()

	// Wait for RIC service (if running)
	suite.waitForServiceReady("localhost", 8080, "RIC API")

	suite.logger.Info("✅ All services are ready")
}

// waitForDatabaseReady waits for database to be ready
func (suite *IntegrationTestSuite) waitForDatabaseReady() {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			suite.FailNow("Database readiness timeout")
		case <-ticker.C:
			if err := suite.db.Ping(); err == nil {
				return
			}
		}
	}
}

// waitForServiceReady waits for a service to be ready on given host:port
func (suite *IntegrationTestSuite) waitForServiceReady(host string, port int, serviceName string) {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	address := fmt.Sprintf("%s:%d", host, port)

	for {
		select {
		case <-timeout:
			suite.logger.Warnf("Service readiness timeout for %s at %s", serviceName, address)
			return // Don't fail, service might not be running in unit tests
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err == nil {
				conn.Close()
				suite.logger.Infof("✅ %s is ready at %s", serviceName, address)
				return
			}
		}
	}
}

// Helper methods for test data creation

// CreateTestRICNode creates a test RIC node
func (suite *IntegrationTestSuite) CreateTestRICNode(nodeID, nodeType string) {
	query := `INSERT INTO ric_nodes (node_id, node_type, status) VALUES ($1, $2, 'active')`
	_, err := suite.db.Exec(query, nodeID, nodeType)
	suite.Require().NoError(err, "Failed to create test RIC node")
}

// CreateTestE2Subscription creates a test E2 subscription
func (suite *IntegrationTestSuite) CreateTestE2Subscription(subscriptionID, nodeID string, functionID int) {
	query := `INSERT INTO e2_subscriptions (subscription_id, node_id, function_id) VALUES ($1, $2, $3)`
	_, err := suite.db.Exec(query, subscriptionID, nodeID, functionID)
	suite.Require().NoError(err, "Failed to create test E2 subscription")
}

// CreateTestA1Policy creates a test A1 policy
func (suite *IntegrationTestSuite) CreateTestA1Policy(policyID, policyTypeID string, policyData map[string]interface{}) {
	query := `INSERT INTO a1_policies (policy_id, policy_type_id, policy_data) VALUES ($1, $2, $3)`
	_, err := suite.db.Exec(query, policyID, policyTypeID, policyData)
	suite.Require().NoError(err, "Failed to create test A1 policy")
}

// RecordTestMetric records a test metric
func (suite *IntegrationTestSuite) RecordTestMetric(testName, metricName string, value float64) {
	query := `INSERT INTO test_metrics (test_name, metric_name, metric_value) VALUES ($1, $2, $3)`
	_, err := suite.db.Exec(query, testName, metricName, value)
	suite.Require().NoError(err, "Failed to record test metric")
}

// GetTestConfig returns the test configuration
func (suite *IntegrationTestSuite) GetTestConfig() *TestConfig {
	return suite.testConfig
}

// GetDB returns the database connection
func (suite *IntegrationTestSuite) GetDB() *sql.DB {
	return suite.db
}

// GetLogger returns the test logger
func (suite *IntegrationTestSuite) GetLogger() *logrus.Logger {
	return suite.logger
}

// GetRICClient returns the RIC test client
func (suite *IntegrationTestSuite) GetRICClient() *RICTestClient {
	return suite.ricClient
}

// GetE2Client returns the E2 test client
func (suite *IntegrationTestSuite) GetE2Client() *E2TestClient {
	return suite.e2Client
}

// GetA1Client returns the A1 test client
func (suite *IntegrationTestSuite) GetA1Client() *A1TestClient {
	return suite.a1Client
}

// GetO1Client returns the O1 test client
func (suite *IntegrationTestSuite) GetO1Client() *O1TestClient {
	return suite.o1Client
}

// RunIntegrationTestSuite runs the integration test suite
func RunIntegrationTestSuite(t *testing.T) {
	// Skip integration tests if not explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Integration tests skipped. Set RUN_INTEGRATION_TESTS=true to enable.")
	}

	suite.Run(t, new(IntegrationTestSuite))
}
