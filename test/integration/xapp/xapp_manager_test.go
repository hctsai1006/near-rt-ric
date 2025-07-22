package xapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/xapp"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// XAppIntegrationTestSuite contains xApp Manager integration tests
type XAppIntegrationTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	xappManager       *xapp.Manager
	postgresContainer testcontainers.Container
	redisContainer    testcontainers.Container
	dbURL             string
	redisURL          string
	baseURL           string
	httpClient        *http.Client
	k8sClient         *fake.Clientset
}

// SetupSuite runs before all tests
func (suite *XAppIntegrationTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup test containers
	suite.setupPostgres()
	suite.setupRedis()
	suite.setupK8sClient()
	suite.setupXAppManager()

	// Setup HTTP client
	suite.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	suite.baseURL = "http://127.0.0.1:8088"
}

// TearDownSuite runs after all tests
func (suite *XAppIntegrationTestSuite) TearDownSuite() {
	if suite.xappManager != nil {
		suite.xappManager.Stop(suite.ctx)
	}
	if suite.postgresContainer != nil {
		suite.postgresContainer.Terminate(suite.ctx)
	}
	if suite.redisContainer != nil {
		suite.redisContainer.Terminate(suite.ctx)
	}
	suite.cancel()
}

func (suite *XAppIntegrationTestSuite) setupPostgres() {
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

func (suite *XAppIntegrationTestSuite) setupRedis() {
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

func (suite *XAppIntegrationTestSuite) setupK8sClient() {
	// Use fake Kubernetes client for testing
	suite.k8sClient = fake.NewSimpleClientset()

	// Create test namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "xapp-test",
		},
	}
	_, err := suite.k8sClient.CoreV1().Namespaces().Create(suite.ctx, namespace, metav1.CreateOptions{})
	require.NoError(suite.T(), err)
}

func (suite *XAppIntegrationTestSuite) setupXAppManager() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	metrics := monitoring.NewMetricsCollector()

	cfg := &config.XAppConfig{
		Manager: config.XAppManagerConfig{
			Host: "127.0.0.1",
			Port: 8088,
		},
		Database: config.DatabaseConfig{
			URL:                suite.dbURL,
			MaxConnections:     10,
			MaxIdleConnections: 5,
			ConnTimeout:        30 * time.Second,
		},
		Cache: config.CacheConfig{
			Type: "redis",
			Redis: config.RedisConfig{
				URL:         suite.redisURL,
				MaxRetries:  3,
				DialTimeout: 5 * time.Second,
			},
		},
		Kubernetes: config.KubernetesConfig{
			Namespace:      "xapp-test",
			InCluster:      false,
			ConfigPath:     "", // Will use fake client
			ServiceAccount: "xapp-manager",
		},
		Registry: config.XAppRegistryConfig{
			Type: "helm",
			Helm: config.HelmRegistryConfig{
				URL:      "oci://registry-1.docker.io",
				Username: "testuser",
				Password: "testpass",
			},
		},
		Health: config.HealthConfig{
			CheckInterval:    30 * time.Second,
			FailureThreshold: 3,
			TimeoutPerCheck:  10 * time.Second,
		},
	}

	var err error
	suite.xappManager, err = xapp.NewManager(cfg, logger, metrics, suite.k8sClient)
	require.NoError(suite.T(), err)

	// Start xApp Manager
	err = suite.xappManager.Start(suite.ctx)
	require.NoError(suite.T(), err)

	// Wait for manager to be ready
	time.Sleep(3 * time.Second)
}

// Test xApp registration
func (suite *XAppIntegrationTestSuite) TestXAppRegistration() {
	xappSpec := map[string]interface{}{
		"name":        "test-xapp",
		"version":     "1.0.0",
		"description": "Test xApp for integration testing",
		"helm_chart": map[string]interface{}{
			"repository": "oci://registry-1.docker.io/testuser",
			"name":       "test-xapp",
			"version":    "1.0.0",
		},
		"config_schema": map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"log_level": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"debug", "info", "warn", "error"},
					"default": "info",
				},
				"batch_size": map[string]interface{}{
					"type":    "integer",
					"minimum": 1,
					"maximum": 1000,
					"default": 100,
				},
			},
		},
		"resource_requirements": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			"limits": map[string]interface{}{
				"cpu":    "500m",
				"memory": "512Mi",
			},
		},
		"interfaces": map[string]interface{}{
			"e2": map[string]interface{}{
				"required": true,
				"version":  "v3.0",
			},
			"a1": map[string]interface{}{
				"required": false,
				"version":  "v2.1",
			},
		},
	}

	body, _ := json.Marshal(xappSpec)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var registeredXApp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&registeredXApp)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "test-xapp", registeredXApp["name"])
	assert.Equal(suite.T(), "1.0.0", registeredXApp["version"])
}

// Test xApp deployment
func (suite *XAppIntegrationTestSuite) TestXAppDeployment() {
	// First register the xApp
	suite.TestXAppRegistration()

	deploymentSpec := map[string]interface{}{
		"xapp_name":    "test-xapp",
		"instance_name": "test-xapp-instance-001",
		"namespace":    "xapp-test",
		"config": map[string]interface{}{
			"log_level":  "debug",
			"batch_size": 50,
		},
		"resource_overrides": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "200m",
				"memory": "256Mi",
			},
		},
	}

	body, _ := json.Marshal(deploymentSpec)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	var deployment map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&deployment)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "test-xapp-instance-001", deployment["instance_name"])
	assert.Equal(suite.T(), "pending", deployment["status"])

	// Wait for deployment to process
	time.Sleep(5 * time.Second)

	// Verify Kubernetes resources were created
	deployments, err := suite.k8sClient.AppsV1().Deployments("xapp-test").List(suite.ctx, metav1.ListOptions{})
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), deployments.Items, 1)
	assert.Equal(suite.T(), "test-xapp-instance-001", deployments.Items[0].Name)
}

// Test xApp instance status monitoring
func (suite *XAppIntegrationTestSuite) TestXAppStatusMonitoring() {
	// First deploy an xApp instance
	suite.TestXAppDeployment()

	// Create a mock deployment in Kubernetes to simulate running xApp
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-xapp-instance-001",
			Namespace: "xapp-test",
			Labels: map[string]string{
				"app.kubernetes.io/name":     "test-xapp",
				"app.kubernetes.io/instance": "test-xapp-instance-001",
				"app.kubernetes.io/part-of":  "near-rt-ric",
			},
		},
		Spec: v1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     "test-xapp",
					"app.kubernetes.io/instance": "test-xapp-instance-001",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name":     "test-xapp",
						"app.kubernetes.io/instance": "test-xapp-instance-001",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-xapp",
							Image: "test-xapp:1.0.0",
						},
					},
				},
			},
		},
		Status: v1.DeploymentStatus{
			Replicas:          1,
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			Conditions: []v1.DeploymentCondition{
				{
					Type:   v1.DeploymentProgressing,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   v1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	_, err := suite.k8sClient.AppsV1().Deployments("xapp-test").Create(suite.ctx, deployment, metav1.CreateOptions{})
	require.NoError(suite.T(), err)

	// Wait for status monitoring
	time.Sleep(3 * time.Second)

	// Get instance status
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-xapp-instance-001/status", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "running", status["status"])
	assert.Equal(suite.T(), float64(1), status["ready_replicas"])
}

// Test xApp configuration updates
func (suite *XAppIntegrationTestSuite) TestXAppConfigurationUpdate() {
	// First deploy an xApp instance
	suite.TestXAppDeployment()

	configUpdate := map[string]interface{}{
		"log_level":  "error",
		"batch_size": 200,
		"new_setting": "test_value",
	}

	body, _ := json.Marshal(configUpdate)
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-xapp-instance-001/config", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify configuration was updated
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-xapp-instance-001/config", suite.baseURL), nil)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var config map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&config)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "error", config["log_level"])
	assert.Equal(suite.T(), float64(200), config["batch_size"])
	assert.Equal(suite.T(), "test_value", config["new_setting"])
}

// Test xApp health monitoring
func (suite *XAppIntegrationTestSuite) TestXAppHealthMonitoring() {
	// First deploy an xApp instance
	suite.TestXAppDeployment()

	// Get health status
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-xapp-instance-001/health", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(suite.T(), err)

	assert.Contains(suite.T(), health, "status")
	assert.Contains(suite.T(), health, "last_check")
	assert.Contains(suite.T(), health, "checks")
}

// Test xApp dependency resolution
func (suite *XAppIntegrationTestSuite) TestXAppDependencyResolution() {
	// Register a dependency xApp first
	dependencySpec := map[string]interface{}{
		"name":        "dependency-xapp",
		"version":     "1.0.0",
		"description": "Dependency xApp for testing",
		"provides": []string{
			"data-processing-service",
			"analytics-service",
		},
	}

	body, _ := json.Marshal(dependencySpec)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	resp.Body.Close()

	// Register main xApp with dependencies
	mainXAppSpec := map[string]interface{}{
		"name":        "main-xapp",
		"version":     "1.0.0",
		"description": "Main xApp with dependencies",
		"dependencies": []string{
			"data-processing-service",
		},
	}

	body, _ = json.Marshal(mainXAppSpec)
	req, _ = http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusCreated, resp.StatusCode)

	// Test dependency validation
	req, _ = http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps/main-xapp/validate-dependencies", suite.baseURL), nil)

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var validation map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&validation)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), validation["valid"].(bool))
	assert.Len(suite.T(), validation["resolved_dependencies"].([]interface{}), 1)
}

// Test xApp undeployment
func (suite *XAppIntegrationTestSuite) TestXAppUndeployment() {
	// First deploy an xApp instance
	suite.TestXAppDeployment()

	// Undeploy the instance
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-xapp-instance-001", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Wait for undeployment to process
	time.Sleep(3 * time.Second)

	// Verify Kubernetes resources were deleted
	deployments, err := suite.k8sClient.AppsV1().Deployments("xapp-test").List(suite.ctx, metav1.ListOptions{})
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), deployments.Items, 0)
}

// Test bulk operations
func (suite *XAppIntegrationTestSuite) TestBulkOperations() {
	// Register multiple xApps
	for i := 1; i <= 5; i++ {
		xappSpec := map[string]interface{}{
			"name":        fmt.Sprintf("bulk-xapp-%d", i),
			"version":     "1.0.0",
			"description": fmt.Sprintf("Bulk test xApp %d", i),
		}

		body, _ := json.Marshal(xappSpec)
		req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps", suite.baseURL), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := suite.httpClient.Do(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()
	}

	// Get all xApps
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/xapps", suite.baseURL), nil)

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var xapps []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&xapps)
	require.NoError(suite.T(), err)

	// Should include the bulk xApps plus previously registered ones
	assert.GreaterOrEqual(suite.T(), len(xapps), 5)
}

// Test error handling
func (suite *XAppIntegrationTestSuite) TestErrorHandling() {
	// Test deploying non-existent xApp
	deploymentSpec := map[string]interface{}{
		"xapp_name":     "non-existent-xapp",
		"instance_name": "test-instance",
	}

	body, _ := json.Marshal(deploymentSpec)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/xapps/non-existent-xapp/instances", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusNotFound, resp.StatusCode)

	// Test invalid configuration
	configUpdate := map[string]interface{}{
		"log_level": "invalid_level", // Invalid enum value
	}

	body, _ = json.Marshal(configUpdate)
	req, _ = http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/xapps/test-xapp/instances/test-instance/config", suite.baseURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = suite.httpClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode)
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}

// Test runner
func TestXAppIntegrationSuite(t *testing.T) {
	suite.Run(t, new(XAppIntegrationTestSuite))
}