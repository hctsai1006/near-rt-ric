package a1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/a1"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// A1IntegrationTestSuite contains A1 interface integration tests
type A1IntegrationTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	a1Interface       *a1.A1Interface
	postgresContainer testcontainers.Container
	redisContainer    testcontainers.Container
	dbURL             string
	redisURL          string
	baseURL           string
	httpClient        *http.Client
	authToken         string
}

// SetupSuite runs before all tests
func (suite *A1IntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup test containers
	suite.setupPostgres()
	suite.setupRedis()
	suite.setupA1Interface()

	// Setup HTTP client
	suite.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	suite.baseURL = "http://127.0.0.1:8080"
}

// TearDownSuite runs after all tests
func (suite *A1IntegrationTestSuite) TearDownSuite() {
	if suite.a1Interface != nil {
		suite.a1Interface.Stop(suite.ctx)
	}
	if suite.postgresContainer != nil {
		suite.postgresContainer.Terminate(suite.ctx)
	}
	if suite.redisContainer != nil {
		suite.redisContainer.Terminate(suite.ctx)
	}
	suite.cancel()
}

func (suite *A1IntegrationTestSuite) setupPostgres() {
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

func (suite *A1IntegrationTestSuite) setupRedis() {
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	container, err := testcontainers.GenericContainer(suite.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(suite.T(), err)

	suite.redisContainer = container

	host, err := container.Host(suite.ctx)
	require.NoError(suite.T(), err)

	port, err := container.MappedPort(suite.ctx, "6379")
	require.NoError(suite.T(), err)

	suite.redisURL = fmt.Sprintf("redis://%s:%s/0", host, port.Port())
}

func (suite *A1IntegrationTestSuite) setupA1Interface() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	metrics := monitoring.NewMetricsCollector("near_rt_ric", "a1")

	cfg := &config.A1Config{
		ListenAddress: "127.0.0.1",
		ListenPort:    8080,
		TLSEnabled:    false, // Disable TLS for testing
		TLSCertPath:   "",
		TLSKeyPath:    "",
		Auth: config.AuthConfig{
			Enabled:            false, // Disable auth for testing
			PrivateKeyPath:     "",
			PublicKeyPath:      "",
			TokenExpiry:        3600,
			Issuer:             "test-issuer",
			Audience:           "test-audience",
			StrictIPValidation: false,
		},
		Database: config.DatabaseConfig{
			URL:      suite.dbURL,
			PoolSize: 10,
		},
		LogLevel: "debug",
	}

	var err error
	suite.a1Interface, err = a1.NewA1Interface(cfg, logger, metrics)
	require.NoError(suite.T(), err)

	// Start A1 interface
	err = suite.a1Interface.Start(suite.ctx)
	require.NoError(suite.T(), err)

	// Wait for interface to be ready
	time.Sleep(3 * time.Second)

	// Get authentication token
	suite.authenticateTestUser()
}

func (suite *A1IntegrationTestSuite) authenticateTestUser() {
	// Create test user credentials
	loginReq := map[string]interface{}{
		"username": "test_admin",
		"password": "test_password",
	}

	body, _ := json.Marshal(loginReq)
	resp, err := suite.httpClient.Post(
		fmt.Sprintf("%s/api/v1/auth/login", suite.baseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var loginResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	require.NoError(suite.T(), err)

	suite.authToken = loginResp.Token
}

// Test A1 Policy Type management
func (suite *A1IntegrationTestSuite) TestPolicyTypeManagement() {
	// Create a policy type
	policyType := map[string]interface{}{
		"policy_type_id":      1,
		"name":                "Test Policy Type",
		"description":         "Policy type for testing",
		"policy_type_version": "1.0.0",
		"create_schema": map[string]interface{}{
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

	body, _ := json.Marshal(policyType)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policy-types/1", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Verify policy type was created
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policy-types/1", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var retrievedType map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&retrievedType)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), float64(1), retrievedType["policy_type_id"])
	assert.Equal(suite.T(), "Test Policy Type", retrievedType["name"])
}

// Test A1 Policy Instance management
func (suite *A1IntegrationTestSuite) TestPolicyInstanceManagement() {
	// First create a policy type (prerequisite)
	suite.TestPolicyTypeManagement()

	// Create a policy instance
	policyInstance := map[string]interface{}{
		"policy_id":      "policy-001",
		"policy_type_id": 1,
		"ric_id":         "ric-001",
		"policy_data": map[string]interface{}{
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

	body, _ := json.Marshal(policyInstance)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/policy-001", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Verify policy instance was created
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policies/policy-001", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var retrievedPolicy map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&retrievedPolicy)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "policy-001", retrievedPolicy["policy_id"])
	assert.Equal(suite.T(), float64(1), retrievedPolicy["policy_type_id"])
	assert.Equal(suite.T(), "ric-001", retrievedPolicy["ric_id"])
}

// Test A1 Policy Status updates
func (suite *A1IntegrationTestSuite) TestPolicyStatusUpdates() {
	// First create a policy instance (prerequisite)
	suite.TestPolicyInstanceManagement()

	// Update policy status
	statusUpdate := map[string]interface{}{
		"policy_id":        "policy-001",
		"status":           "ENFORCED",
		"reason":           "Policy successfully enforced",
		"has_been_deleted": false,
		"deleted":          false,
	}

	body, _ := json.Marshal(statusUpdate)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/policy-001/status", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify policy status was updated
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policies/policy-001/status", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "ENFORCED", status["status"])
	assert.Equal(suite.T(), "Policy successfully enforced", status["reason"])
}

// Test A1 Enrichment Information Service
func (suite *A1IntegrationTestSuite) TestEnrichmentInformationService() {
	// Create enrichment information type
	eiType := map[string]interface{}{
		"ei_type_id":          "location-info",
		"ei_type_name":        "Location Information",
		"ei_type_description": "UE location information for policy decisions",
		"ei_job_data_schema": map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"area_of_interest": map[string]interface{}{
					"type": "string",
				},
				"tracking_accuracy": map[string]interface{}{
					"type": "integer",
				},
			},
		},
	}

	body, _ := json.Marshal(eiType)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/enrichment/ei-types/location-info", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Create enrichment information job
	eiJob := map[string]interface{}{
		"ei_job_id":  "location-job-001",
		"ei_type_id": "location-info",
		"job_owner":  "policy-service",
		"job_definition": map[string]interface{}{
			"area_of_interest":  "cell-001",
			"tracking_accuracy": 100,
		},
		"job_result_uri":          "http://policy-service/api/v1/location-updates",
		"status_notification_uri": "http://policy-service/api/v1/job-status",
	}

	body, _ = json.Marshal(eiJob)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/enrichment/ei-jobs/location-job-001", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Verify EI job was created
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/enrichment/ei-jobs/location-job-001", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var retrievedJob map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&retrievedJob)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "location-job-001", retrievedJob["ei_job_id"])
	assert.Equal(suite.T(), "location-info", retrievedJob["ei_type_id"])
}

// Test A1 ML Model Management Service
func (suite *A1IntegrationTestSuite) TestMLModelManagement() {
	// Register ML model
	model := map[string]interface{}{
		"model_id":          "qoe-prediction-v1",
		"model_name":        "QoE Prediction Model",
		"model_version":     "1.0.0",
		"model_description": "Machine learning model for QoE prediction",
		"model_type":        "tensorflow",
		"input_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"throughput": map[string]interface{}{
					"type": "number",
				},
				"latency": map[string]interface{}{
					"type": "number",
				},
				"signal_strength": map[string]interface{}{
					"type": "number",
				},
			},
		},
		"output_schema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"qoe_score": map[string]interface{}{
					"type":    "number",
					"minimum": 0,
					"maximum": 5,
				},
				"confidence": map[string]interface{}{
					"type":    "number",
					"minimum": 0,
					"maximum": 1,
				},
			},
		},
	}

	body, _ := json.Marshal(model)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/models", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Get model list
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/models", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var models []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&models)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), models, 1)
	assert.Equal(suite.T(), "qoe-prediction-v1", models[0]["model_id"])
}

// Test Authentication and Authorization
func (suite *A1IntegrationTestSuite) TestAuthenticationAuthorization() {
	// Test unauthenticated request
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policy-types", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	// Test with invalid token
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policy-types", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer invalid_token")

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	// Test with valid token
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/policy-types", suite.baseURL), nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// Test API input validation
func (suite *A1IntegrationTestSuite) TestInputValidation() {
	// Test invalid policy type creation
	invalidPolicyType := map[string]interface{}{
		// Missing required fields
		"name": "Invalid Policy Type",
	}

	body, _ := json.Marshal(invalidPolicyType)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policy-types/999", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)

	// Test invalid JSON
	req, _ = http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policy-types/998", suite.baseURL), bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

// Test performance under load
func (suite *A1IntegrationTestSuite) TestA1Performance() {
	// First create a policy type for testing
	suite.TestPolicyTypeManagement()

	numRequests := 50
	results := make(chan error, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			policyInstance := map[string]interface{}{
				"policy_id":      fmt.Sprintf("perf-policy-%03d", requestID),
				"policy_type_id": 1,
				"ric_id":         fmt.Sprintf("ric-%03d", requestID%10),
				"policy_data": map[string]interface{}{
					"scope": map[string]interface{}{
						"ue_id": fmt.Sprintf("ue-%020d", requestID),
					},
					"statement": map[string]interface{}{
						"qos": map[string]interface{}{
							"gbr": 1000 + requestID,
						},
					},
				},
			}

			body, _ := json.Marshal(policyInstance)
			req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/policies/perf-policy-%03d", suite.baseURL, requestID), bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+suite.authToken)
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.httpClient.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			if err != nil || resp.StatusCode != http.StatusCreated {
				results <- fmt.Errorf("request %d failed: %v, status: %d", requestID, err, resp.StatusCode)
			} else {
				results <- nil
			}
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
	assert.Less(suite.T(), errors, numRequests/10)   // Less than 10% error rate
	assert.Less(suite.T(), duration, 60*time.Second) // Complete within 60 seconds

	suite.T().Logf("Performance test: %d requests in %v, %d errors", numRequests, duration, errors)
}

// Test health endpoint
func (suite *A1IntegrationTestSuite) TestHealthEndpoint() {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/health", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "healthy", health["status"])
	assert.Contains(suite.T(), health, "timestamp")
	assert.Contains(suite.T(), health, "version")
}

// Test runner
func TestA1IntegrationSuite(t *testing.T) {
	suite.Run(t, new(A1IntegrationTestSuite))
}
