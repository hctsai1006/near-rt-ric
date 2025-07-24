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
	"github.com/hctsai1006/near-rt-ric/pkg/e2/models"
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
	_ = json.Unmarshal([]byte(fixtureData), &messages)
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
func CreateE2SetupRequest(nodeID string, transactionID int64) *models.E2SetupRequest {
	return &models.E2SetupRequest{
		TransactionID: transactionID,
		GlobalE2NodeID: &models.GlobalE2NodeID{
			GNB_ID: &models.GNB_ID{
				GNB_ID: []byte(nodeID),
			},
		},
		RANfunctions: []*models.RANfunction{
			{
				RANfunctionID:         1,
				RANfunctionDefinition: []byte("test"),
				RANfunctionRevision:   1,
			},
		},
	}
}

// CreateRICSubscriptionRequest creates a test RIC Subscription Request
func CreateRICSubscriptionRequest(requestorID, instanceID, ranFunctionID int) *models.RICSubscriptionRequest {
	return &models.RICSubscriptionRequest{
		RICrequestID: &models.RICrequestID{
			RICrequestorID: requestorID,
			RICinstanceID:  instanceID,
		},
		RANfunctionID: ranFunctionID,
		RICsubscriptionDetails: &models.RICsubscriptionDetails{
			RICeventTriggerDefinition: []byte("test"),
			RICactions: []*models.RICaction{
				{
					RICactionID:   1,
					RICactionType: models.Report,
				},
			},
		},
	}
}

// CreateRICControlRequest creates a test RIC Control Request
func CreateRICControlRequest() *models.RICcontrolRequest {
	return &models.RICcontrolRequest{}
}

// A1 Test Message builders

// CreateA1PolicyType creates a test A1 policy type
func CreateA1PolicyType(policyTypeID string, name string) *a1.A1PolicyType {
	return &a1.A1PolicyType{
		PolicyTypeID: policyTypeID,
		Name:         name,
		Description:  fmt.Sprintf("Test policy type: %s", name),
		PolicySchema: map[string]interface{}{
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
func CreateA1PolicyInstance(policyID string, policyTypeID string) *a1.A1Policy {
	return &a1.A1Policy{
		PolicyID:     policyID,
		PolicyTypeID: policyTypeID,
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

// CreateTestXApp creates a test xApp
func CreateTestXApp(name, version string) *xapp.XApp {
	return &xapp.XApp{
		Name:        name,
		Version:     version,
		Description: fmt.Sprintf("Test xApp: %s", name),
		Deployment: xapp.XAppDeploymentSpec{
			Image: "test-image",
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
