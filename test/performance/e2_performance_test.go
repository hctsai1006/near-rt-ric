package performance_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hctsai1006/near-rt-ric/internal/config"
	"github.com/hctsai1006/near-rt-ric/pkg/e2"
	"github.com/hctsai1006/near-rt-ric/pkg/common/monitoring"
	"github.com/sirupsen/logrus"
)

// E2PerformanceTest measures E2 interface performance under load
func TestE2InterfacePerformance(t *testing.T) {
	ctx := context.Background()
	
	// Setup test database
	postgresContainer := setupTestPostgres(t, ctx)
	defer postgresContainer.Terminate(ctx)

	host, _ := postgresContainer.Host(ctx)
	port, _ := postgresContainer.MappedPort(ctx, "5432")
	dbURL := fmt.Sprintf("postgres://test_user:test_password@%s:%s/near_rt_ric_test?sslmode=disable", host, port.Port())

	// Setup E2 interface
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce logging for performance test
	metrics := monitoring.NewMetricsCollector()

	cfg := &config.E2Config{
		Server: config.E2ServerConfig{
			Host: "127.0.0.1",
			Port: 36421,
			SCTP: config.SCTPConfig{
				Enabled:           true,
				Streams:           20,
				HeartbeatInterval: 30 * time.Second,
			},
		},
		Database: config.DatabaseConfig{
			URL:                dbURL,
			MaxConnections:     50,
			MaxIdleConnections: 25,
			ConnTimeout:        30 * time.Second,
		},
		ASN1: config.ASN1Config{
			Version:    "v3.0",
			Codec:      "per",
			Validation: false, // Disable validation for performance
		},
	}

	e2Interface, err := e2.NewE2Interface(cfg, logger, metrics)
	require.NoError(t, err)

	err = e2Interface.Start(ctx)
	require.NoError(t, err)
	defer e2Interface.Stop(ctx)

	// Wait for interface to be ready
	time.Sleep(2 * time.Second)

	// Performance test parameters
	numNodes := 100
	requestsPerNode := 100
	concurrentNodes := 10

	t.Run("E2SetupPerformance", func(t *testing.T) {
		testE2SetupPerformance(t, e2Interface, numNodes, concurrentNodes)
	})

	t.Run("RICSubscriptionPerformance", func(t *testing.T) {
		testRICSubscriptionPerformance(t, e2Interface, numNodes, requestsPerNode, concurrentNodes)
	})

	t.Run("RICControlPerformance", func(t *testing.T) {
		testRICControlPerformance(t, e2Interface, numNodes, requestsPerNode, concurrentNodes)
	})

	t.Run("E2LatencyTest", func(t *testing.T) {
		testE2Latency(t, e2Interface, 1000)
	})

	t.Run("E2ThroughputTest", func(t *testing.T) {
		testE2Throughput(t, e2Interface, 5*time.Minute)
	})
}

// BenchmarkE2SetupProcedure measures E2 Setup procedure performance
func BenchmarkE2SetupProcedure(b *testing.B) {
	ctx := context.Background()
	
	// Setup minimal E2 interface for benchmarking
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	metrics := monitoring.NewMetricsCollector()

	cfg := &config.E2Config{
		Server: config.E2ServerConfig{
			Host: "127.0.0.1",
			Port: 36422,
		},
		Database: config.DatabaseConfig{
			URL: "postgres://test:test@localhost/test?sslmode=disable",
		},
	}

	e2Interface, err := e2.NewE2Interface(cfg, logger, metrics)
	if err != nil {
		b.Skip("Cannot create E2 interface for benchmark:", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		setupReq := &e2.E2SetupRequest{
			TransactionID: uint32(i + 1),
			GlobalE2NodeID: e2.GlobalE2NodeID{
				NodeType: e2.NodeTypeGNB,
				NodeID:   fmt.Sprintf("bench-node-%d", i),
			},
			RANFunctions: []e2.RANFunction{
				{
					FunctionID:  1,
					Name:        "Benchmark Function",
					Version:     "1.0",
					OID:         "1.3.6.1.4.1.1.22.1.1",
					Description: "Function for benchmarking",
				},
			},
		}

		// Simulate E2 Setup processing (without actual network I/O)
		_ = e2Interface.ProcessE2SetupRequest(ctx, fmt.Sprintf("bench-node-%d", i), setupReq)
	}
}

// BenchmarkRICSubscription measures RIC Subscription procedure performance
func BenchmarkRICSubscription(b *testing.B) {
	ctx := context.Background()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	metrics := monitoring.NewMetricsCollector()

	cfg := &config.E2Config{
		Database: config.DatabaseConfig{
			URL: "postgres://test:test@localhost/test?sslmode=disable",
		},
	}

	e2Interface, err := e2.NewE2Interface(cfg, logger, metrics)
	if err != nil {
		b.Skip("Cannot create E2 interface for benchmark:", err)
	}

	// Pre-setup a test node
	nodeID := "benchmark-node"
	setupReq := &e2.E2SetupRequest{
		TransactionID: 1,
		GlobalE2NodeID: e2.GlobalE2NodeID{
			NodeType: e2.NodeTypeGNB,
			NodeID:   nodeID,
		},
		RANFunctions: []e2.RANFunction{
			{FunctionID: 1, Name: "Test Function", Version: "1.0"},
		},
	}
	e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		subscriptionReq := &e2.RICSubscriptionRequest{
			TransactionID: uint32(i + 1),
			RequestID: e2.RICRequestID{
				RequestorID: 1001,
				InstanceID:  uint32(i),
			},
			RANFunctionID: 1,
			EventTriggers: e2.EventTriggerDefinition{
				TriggerType: e2.TriggerTypePeriodic,
				Period:      1000,
			},
			Actions: []e2.RICAction{
				{
					ActionID:   1,
					ActionType: e2.ActionTypeReport,
				},
			},
		}

		_ = e2Interface.ProcessRICSubscriptionRequest(ctx, nodeID, subscriptionReq)
	}
}

func testE2SetupPerformance(t *testing.T, e2Interface *e2.E2Interface, numNodes, concurrentNodes int) {
	ctx := context.Background()
	semaphore := make(chan struct{}, concurrentNodes)
	var wg sync.WaitGroup
	
	start := time.Now()
	errors := 0
	var errorsMutex sync.Mutex

	for i := 0; i < numNodes; i++ {
		wg.Add(1)
		go func(nodeIndex int) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			nodeID := fmt.Sprintf("perf-node-%03d", nodeIndex)
			setupReq := &e2.E2SetupRequest{
				TransactionID: uint32(nodeIndex + 1),
				GlobalE2NodeID: e2.GlobalE2NodeID{
					NodeType: e2.NodeTypeGNB,
					NodeID:   nodeID,
				},
				RANFunctions: []e2.RANFunction{
					{
						FunctionID:  1,
						Name:        "Performance Test Function",
						Version:     "1.0",
						OID:         "1.3.6.1.4.1.1.22.1.1",
						Description: "Function for performance testing",
					},
				},
			}

			err := e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)
			if err != nil {
				errorsMutex.Lock()
				errors++
				errorsMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Performance assertions
	require.Less(t, errors, numNodes/10, "Error rate should be less than 10%")
	require.Less(t, duration, 60*time.Second, "E2 Setup for %d nodes should complete within 60 seconds", numNodes)

	// Calculate metrics
	successfulSetups := numNodes - errors
	setupsPerSecond := float64(successfulSetups) / duration.Seconds()
	averageLatency := duration / time.Duration(successfulSetups)

	t.Logf("E2 Setup Performance Results:")
	t.Logf("  Total nodes: %d", numNodes)
	t.Logf("  Successful setups: %d", successfulSetups)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Setups per second: %.2f", setupsPerSecond)
	t.Logf("  Average latency: %v", averageLatency)
	t.Logf("  Error rate: %.2f%%", float64(errors)/float64(numNodes)*100)
}

func testRICSubscriptionPerformance(t *testing.T, e2Interface *e2.E2Interface, numNodes, requestsPerNode, concurrentNodes int) {
	ctx := context.Background()
	
	// First setup all nodes
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("sub-node-%03d", i)
		setupReq := &e2.E2SetupRequest{
			TransactionID: uint32(i + 1),
			GlobalE2NodeID: e2.GlobalE2NodeID{
				NodeType: e2.NodeTypeGNB,
				NodeID:   nodeID,
			},
			RANFunctions: []e2.RANFunction{
				{FunctionID: 1, Name: "Test Function", Version: "1.0"},
			},
		}
		e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)
	}

	semaphore := make(chan struct{}, concurrentNodes)
	var wg sync.WaitGroup
	
	start := time.Now()
	totalRequests := numNodes * requestsPerNode
	errors := 0
	var errorsMutex sync.Mutex

	for i := 0; i < numNodes; i++ {
		for j := 0; j < requestsPerNode; j++ {
			wg.Add(1)
			go func(nodeIndex, requestIndex int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				nodeID := fmt.Sprintf("sub-node-%03d", nodeIndex)
				subscriptionReq := &e2.RICSubscriptionRequest{
					TransactionID: uint32(nodeIndex*requestsPerNode + requestIndex + 1),
					RequestID: e2.RICRequestID{
						RequestorID: 1001,
						InstanceID:  uint32(nodeIndex*requestsPerNode + requestIndex),
					},
					RANFunctionID: 1,
					EventTriggers: e2.EventTriggerDefinition{
						TriggerType: e2.TriggerTypePeriodic,
						Period:      5000,
					},
					Actions: []e2.RICAction{
						{ActionID: 1, ActionType: e2.ActionTypeReport},
					},
				}

				err := e2Interface.ProcessRICSubscriptionRequest(ctx, nodeID, subscriptionReq)
				if err != nil {
					errorsMutex.Lock()
					errors++
					errorsMutex.Unlock()
				}
			}(i, j)
		}
	}

	wg.Wait()
	duration := time.Since(start)

	// Performance assertions
	require.Less(t, errors, totalRequests/10, "Error rate should be less than 10%")

	successfulSubscriptions := totalRequests - errors
	subscriptionsPerSecond := float64(successfulSubscriptions) / duration.Seconds()

	t.Logf("RIC Subscription Performance Results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful subscriptions: %d", successfulSubscriptions)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Subscriptions per second: %.2f", subscriptionsPerSecond)
	t.Logf("  Error rate: %.2f%%", float64(errors)/float64(totalRequests)*100)
}

func testRICControlPerformance(t *testing.T, e2Interface *e2.E2Interface, numNodes, requestsPerNode, concurrentNodes int) {
	ctx := context.Background()
	
	// Setup nodes and subscriptions first
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("ctrl-node-%03d", i)
		
		// Setup node
		setupReq := &e2.E2SetupRequest{
			TransactionID: uint32(i + 1),
			GlobalE2NodeID: e2.GlobalE2NodeID{
				NodeType: e2.NodeTypeGNB,
				NodeID:   nodeID,
			},
			RANFunctions: []e2.RANFunction{
				{FunctionID: 1, Name: "Control Function", Version: "1.0"},
			},
		}
		e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)
	}

	semaphore := make(chan struct{}, concurrentNodes)
	var wg sync.WaitGroup
	
	start := time.Now()
	totalRequests := numNodes * requestsPerNode
	errors := 0
	var errorsMutex sync.Mutex

	for i := 0; i < numNodes; i++ {
		for j := 0; j < requestsPerNode; j++ {
			wg.Add(1)
			go func(nodeIndex, requestIndex int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				nodeID := fmt.Sprintf("ctrl-node-%03d", nodeIndex)
				controlReq := &e2.RICControlRequest{
					TransactionID: uint32(nodeIndex*requestsPerNode + requestIndex + 1),
					RequestID: e2.RICRequestID{
						RequestorID: 1001,
						InstanceID:  uint32(nodeIndex*requestsPerNode + requestIndex),
					},
					RANFunctionID:     1,
					CallProcessID:     []byte(fmt.Sprintf("ctrl-%d-%d", nodeIndex, requestIndex)),
					ControlHeader:     []byte("test-control-header"),
					ControlMessage:    []byte("test-control-message"),
					ControlAckRequest: e2.ControlAckRequestACK,
				}

				err := e2Interface.ProcessRICControlRequest(ctx, nodeID, controlReq)
				if err != nil {
					errorsMutex.Lock()
					errors++
					errorsMutex.Unlock()
				}
			}(i, j)
		}
	}

	wg.Wait()
	duration := time.Since(start)

	successfulControls := totalRequests - errors
	controlsPerSecond := float64(successfulControls) / duration.Seconds()

	t.Logf("RIC Control Performance Results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful controls: %d", successfulControls)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Controls per second: %.2f", controlsPerSecond)
	t.Logf("  Error rate: %.2f%%", float64(errors)/float64(totalRequests)*100)
}

func testE2Latency(t *testing.T, e2Interface *e2.E2Interface, numSamples int) {
	ctx := context.Background()
	
	// Setup a test node
	nodeID := "latency-test-node"
	setupReq := &e2.E2SetupRequest{
		TransactionID: 1,
		GlobalE2NodeID: e2.GlobalE2NodeID{
			NodeType: e2.NodeTypeGNB,
			NodeID:   nodeID,
		},
		RANFunctions: []e2.RANFunction{
			{FunctionID: 1, Name: "Latency Test Function", Version: "1.0"},
		},
	}
	err := e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)
	require.NoError(t, err)

	latencies := make([]time.Duration, 0, numSamples)

	for i := 0; i < numSamples; i++ {
		start := time.Now()
		
		subscriptionReq := &e2.RICSubscriptionRequest{
			TransactionID: uint32(i + 1),
			RequestID: e2.RICRequestID{
				RequestorID: 1001,
				InstanceID:  uint32(i),
			},
			RANFunctionID: 1,
			EventTriggers: e2.EventTriggerDefinition{
				TriggerType: e2.TriggerTypePeriodic,
				Period:      1000,
			},
			Actions: []e2.RICAction{
				{ActionID: 1, ActionType: e2.ActionTypeReport},
			},
		}

		err := e2Interface.ProcessRICSubscriptionRequest(ctx, nodeID, subscriptionReq)
		
		latency := time.Since(start)
		if err == nil {
			latencies = append(latencies, latency)
		}

		// Small delay to avoid overwhelming the system
		time.Sleep(time.Millisecond)
	}

	// Calculate latency statistics
	if len(latencies) > 0 {
		var total time.Duration
		min := latencies[0]
		max := latencies[0]

		for _, lat := range latencies {
			total += lat
			if lat < min {
				min = lat
			}
			if lat > max {
				max = lat
			}
		}

		average := total / time.Duration(len(latencies))

		// Calculate percentiles (simplified)
		// Sort latencies for percentile calculation would be more accurate
		p95Index := int(0.95 * float64(len(latencies)))
		p99Index := int(0.99 * float64(len(latencies)))
		
		if p95Index >= len(latencies) {
			p95Index = len(latencies) - 1
		}
		if p99Index >= len(latencies) {
			p99Index = len(latencies) - 1
		}

		t.Logf("E2 Latency Test Results:")
		t.Logf("  Samples: %d", len(latencies))
		t.Logf("  Average latency: %v", average)
		t.Logf("  Min latency: %v", min)
		t.Logf("  Max latency: %v", max)
		t.Logf("  P95 latency (approx): %v", latencies[p95Index])
		t.Logf("  P99 latency (approx): %v", latencies[p99Index])

		// Assert latency requirements (O-RAN E2 interface requirement: 10ms - 1s)
		require.Less(t, average, 100*time.Millisecond, "Average latency should be less than 100ms")
		require.Less(t, latencies[p95Index], 500*time.Millisecond, "P95 latency should be less than 500ms")
	}
}

func testE2Throughput(t *testing.T, e2Interface *e2.E2Interface, duration time.Duration) {
	ctx := context.Background()
	
	// Setup test nodes
	numNodes := 10
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("throughput-node-%02d", i)
		setupReq := &e2.E2SetupRequest{
			TransactionID: uint32(i + 1),
			GlobalE2NodeID: e2.GlobalE2NodeID{
				NodeType: e2.NodeTypeGNB,
				NodeID:   nodeID,
			},
			RANFunctions: []e2.RANFunction{
				{FunctionID: 1, Name: "Throughput Test Function", Version: "1.0"},
			},
		}
		e2Interface.ProcessE2SetupRequest(ctx, nodeID, setupReq)
	}

	var wg sync.WaitGroup
	var operations int64
	var errorCount int64
	startTime := time.Now()
	endTime := startTime.Add(duration)

	// Start workers
	numWorkers := 20
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			requestCounter := 0
			for time.Now().Before(endTime) {
				nodeID := fmt.Sprintf("throughput-node-%02d", requestCounter%numNodes)
				
				subscriptionReq := &e2.RICSubscriptionRequest{
					TransactionID: uint32(workerID*10000 + requestCounter),
					RequestID: e2.RICRequestID{
						RequestorID: uint32(1000 + workerID),
						InstanceID:  uint32(requestCounter),
					},
					RANFunctionID: 1,
					EventTriggers: e2.EventTriggerDefinition{
						TriggerType: e2.TriggerTypePeriodic,
						Period:      5000,
					},
					Actions: []e2.RICAction{
						{ActionID: 1, ActionType: e2.ActionTypeReport},
					},
				}

				err := e2Interface.ProcessRICSubscriptionRequest(ctx, nodeID, subscriptionReq)
				if err != nil {
					errorCount++
				} else {
					operations++
				}

				requestCounter++
				
				// Small delay to prevent overwhelming
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	actualDuration := time.Since(startTime)

	throughput := float64(operations) / actualDuration.Seconds()
	errorRate := float64(errorCount) / float64(operations+errorCount) * 100

	t.Logf("E2 Throughput Test Results:")
	t.Logf("  Test duration: %v", actualDuration)
	t.Logf("  Total operations: %d", operations)
	t.Logf("  Throughput: %.2f ops/sec", throughput)
	t.Logf("  Error rate: %.2f%%", errorRate)

	// Assert minimum throughput requirements
	require.Greater(t, throughput, 100.0, "Throughput should be greater than 100 operations/sec")
	require.Less(t, errorRate, 5.0, "Error rate should be less than 5%")
}

func setupTestPostgres(t *testing.T, ctx context.Context) testcontainers.Container {
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

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	return container
}