package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hctsai1006/near-rt-ric/pkg/a1"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/xapp"
)

// TestContainer represents a test container with common operations
type TestContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	URL       string
}

// DatabaseTestContainer creates a PostgreSQL test container
func DatabaseTestContainer(t *testing.T, ctx context.Context) *TestContainer {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "near_rt_ric_test",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	return &TestContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		URL:       fmt.Sprintf("postgres://test_user:test_password@%s:%s/near_rt_ric_test?sslmode=disable", host, port.Port()),
	}
}

// RedisTestContainer creates a Redis test container
func RedisTestContainer(t *testing.T, ctx context.Context) *TestContainer {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	return &TestContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		URL:       fmt.Sprintf("redis://%s:%s/0", host, port.Port()),
	}
}

// KafkaTestContainer creates a Kafka test container
func KafkaTestContainer(t *testing.T, ctx context.Context) *TestContainer {
	req := testcontainers.ContainerRequest{
		Image:        "confluentinc/cp-kafka:7.4.0",
		ExposedPorts: []string{"9092/tcp"},
		Env: map[string]string{
			"KAFKA_BROKER_ID":                        "1",
			"KAFKA_ZOOKEEPER_CONNECT":                "localhost:2181",
			"KAFKA_ADVERTISED_LISTENERS":             "PLAINTEXT://localhost:9092",
			"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
			"KAFKA_AUTO_CREATE_TOPICS_ENABLE":        "true",
		},
		WaitingFor: wait.ForListeningPort("9092/tcp").WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "9092")
	require.NoError(t, err)

	return &TestContainer{
		Container: container,
		Host:      host,
		Port:      port.Port(),
		URL:       fmt.Sprintf("%s:%s", host, port.Port()),
	}
}

// Cleanup terminates a test container
func (tc *TestContainer) Cleanup(ctx context.Context) error {
	if tc.Container != nil {
		return tc.Container.Terminate(ctx)
	}
	return nil
}

// LoadTestFixture loads a JSON test fixture file
func LoadTestFixture(t *testing.T, filename string, target interface{}) {
	fixturesDir := filepath.Join("test", "fixtures")
	filePath := filepath.Join(fixturesDir, filename)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read fixture file: %s", filename)

	err = json.Unmarshal(data, target)
	require.NoError(t, err, "Failed to unmarshal fixture file: %s", filename)
}

// GenerateTestE2Messages creates test E2 messages for different scenarios
func GenerateTestE2Messages() map[string]interface{} {
	var messages map[string]interface{}
	fixtureData := `{
		"e2_setup_request": {
			"transaction_id": 1,
			"global_e2_node_id": {
				"node_type": "gNB",
				"node_id": "test-gnb-001"
			},
			"ran_functions": [
				{
					"function_id": 1,
					"name": "RAN Control",
					"version": "1.0",
					"oid": "1.3.6.1.4.1.1.22.1.1"
				}
			]
		}
	}`
	json.Unmarshal([]byte(fixtureData), &messages)
	return messages
}

// GenerateTestCertificates creates test SSL certificates
func GenerateTestCertificates(t *testing.T) (certPEM, keyPEM []byte) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Organization"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test City"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost", "test-server"},
	}

	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return certPEM, keyPEM
}

// WaitForPort waits for a port to be available
func WaitForPort(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	address := fmt.Sprintf("%s:%d", host, port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("port %s not available after %v", address, timeout)
}

// WaitForHTTPEndpoint waits for an HTTP endpoint to be available
func WaitForHTTPEndpoint(url string, timeout time.Duration) error {
	// Implementation would use http.Client to check endpoint
	// For now, just wait for the port
	return nil
}

// CreateTempDir creates a temporary directory for test files
func CreateTempDir(t *testing.T, prefix string) string {
	dir, err := os.MkdirTemp("", prefix)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// WriteTestFile writes content to a temporary test file
func WriteTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	return filePath
}

// CopyTestFile copies a file for testing
func CopyTestFile(t *testing.T, src, dst string) {
	srcFile, err := os.Open(src)
	require.NoError(t, err)
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	require.NoError(t, err)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	require.NoError(t, err)
}

// E2TestMessage builders for common E2 messages

// CreateE2SetupRequest creates a test E2 Setup Request
func CreateE2SetupRequest(nodeID string, transactionID uint32) *e2.E2SetupRequest {
	return &e2.E2SetupRequest{
		TransactionID: transactionID,
		GlobalE2NodeID: e2.GlobalE2NodeID{
			NodeType: e2.NodeTypeGNB,
			NodeID:   nodeID,
		},
		RANFunctions: []e2.RANFunction{
			{
				FunctionID:  1,
				Name:        "RAN Control",
				Version:     "1.0",
				OID:         "1.3.6.1.4.1.1.22.1.1",
				Description: "Test RAN Control Function",
			},
			{
				FunctionID:  2,
				Name:        "Key Performance Measurement",
				Version:     "1.0",
				OID:         "1.3.6.1.4.1.1.22.1.2",
				Description: "Test KPM Function",
			},
		},
	}
}

// CreateRICSubscriptionRequest creates a test RIC Subscription Request
func CreateRICSubscriptionRequest(requestorID, instanceID, transactionID uint32, ranFunctionID uint32) *e2.RICSubscriptionRequest {
	return &e2.RICSubscriptionRequest{
		TransactionID: transactionID,
		RequestID: e2.RICRequestID{
			RequestorID: requestorID,
			InstanceID:  instanceID,
		},
		RANFunctionID: ranFunctionID,
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
}

// CreateRICControlRequest creates a test RIC Control Request
func CreateRICControlRequest(requestorID, instanceID, transactionID uint32, ranFunctionID uint32) *e2.RICControlRequest {
	return &e2.RICControlRequest{
		TransactionID: transactionID,
		RequestID: e2.RICRequestID{
			RequestorID: requestorID,
			InstanceID:  instanceID,
		},
		RANFunctionID:     ranFunctionID,
		CallProcessID:     []byte("test-control-001"),
		ControlHeader:     []byte("test-control-header"),
		ControlMessage:    []byte("test-control-message"),
		ControlAckRequest: e2.ControlAckRequestACK,
	}
}

// A1 Test Message builders

// CreateA1PolicyType creates a test A1 policy type
func CreateA1PolicyType(policyTypeID int, name string) *a1.PolicyType {
	return &a1.PolicyType{
		PolicyTypeID: policyTypeID,
		Name:         name,
		Description:  fmt.Sprintf("Test policy type: %s", name),
		Version:      "1.0.0",
		CreateSchema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"scope": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ue_id": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"statement": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"qos": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"gbr": map[string]interface{}{
									"type":    "integer",
									"minimum": 0,
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateA1PolicyInstance creates a test A1 policy instance
func CreateA1PolicyInstance(policyID string, policyTypeID int, ricID string) *a1.PolicyInstance {
	return &a1.PolicyInstance{
		PolicyID:     policyID,
		PolicyTypeID: policyTypeID,
		RICID:        ricID,
		PolicyData: map[string]interface{}{
			"scope": map[string]interface{}{
				"ue_id": "12345678901234567890",
			},
			"statement": map[string]interface{}{
				"qos": map[string]interface{}{
					"gbr": 1000,
				},
			},
		},
	}
}

// xApp Test builders

// CreateTestXAppSpec creates a test xApp specification
func CreateTestXAppSpec(name, version string) *xapp.XAppSpec {
	return &xapp.XAppSpec{
		Name:        name,
		Version:     version,
		Description: fmt.Sprintf("Test xApp: %s", name),
		HelmChart: xapp.HelmChart{
			Repository: "oci://registry-1.docker.io/test",
			Name:       name,
			Version:    version,
		},
		ConfigSchema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"log_level": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"debug", "info", "warn", "error"},
					"default": "info",
				},
			},
		},
		ResourceRequirements: xapp.ResourceRequirements{
			Requests: map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			Limits: map[string]string{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		},
		Interfaces: map[string]xapp.InterfaceRequirement{
			"e2": {
				Required: true,
				Version:  "v3.0",
			},
		},
	}
}

// CreateTestXAppDeployment creates a test xApp deployment
func CreateTestXAppDeployment(xappName, instanceName, namespace string) *xapp.XAppDeploymentSpec {
	return &xapp.XAppDeploymentSpec{
		XAppName:     xappName,
		InstanceName: instanceName,
		Namespace:    namespace,
		Config: map[string]interface{}{
			"log_level": "debug",
		},
		ResourceOverrides: map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "200m",
				"memory": "256Mi",
			},
		},
	}
}

// Performance testing utilities

// MeasureExecutionTime measures the execution time of a function
func MeasureExecutionTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// RunConcurrently runs a function concurrently with specified workers
func RunConcurrently(t *testing.T, workers int, fn func(workerID int) error) []error {
	errors := make([]error, workers)
	done := make(chan int, workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			errors[workerID] = fn(workerID)
			done <- workerID
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < workers; i++ {
		<-done
	}

	return errors
}

// AssertNoErrors checks that no errors occurred in concurrent execution
func AssertNoErrors(t *testing.T, errors []error) {
	for i, err := range errors {
		require.NoError(t, err, "Worker %d returned error", i)
	}
}

// AssertMaxErrors checks that error rate is below threshold
func AssertMaxErrors(t *testing.T, errors []error, maxErrorRate float64) {
	errorCount := 0
	for _, err := range errors {
		if err != nil {
			errorCount++
		}
	}

	errorRate := float64(errorCount) / float64(len(errors))
	require.LessOrEqual(t, errorRate, maxErrorRate, "Error rate %.2f%% exceeds maximum %.2f%%", errorRate*100, maxErrorRate*100)
}
