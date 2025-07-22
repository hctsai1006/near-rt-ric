# O-RAN Near-RT RIC Testing Makefile

.PHONY: help test test-unit test-integration test-performance test-compliance test-coverage \
        test-e2e test-setup test-cleanup test-containers test-lint test-security \
        test-benchmark test-load test-stress test-all

# Default target
help:
	@echo "Available test targets:"
	@echo "  test-all          - Run all tests (unit, integration, performance, compliance)"
	@echo "  test-unit         - Run unit tests"
	@echo "  test-integration  - Run integration tests"
	@echo "  test-performance  - Run performance tests"
	@echo "  test-compliance   - Run O-RAN compliance tests"
	@echo "  test-e2e          - Run end-to-end tests"
	@echo "  test-coverage     - Generate test coverage report"
	@echo "  test-benchmark    - Run Go benchmarks"
	@echo "  test-load         - Run load tests"
	@echo "  test-stress       - Run stress tests"
	@echo "  test-security     - Run security tests"
	@echo "  test-lint         - Run code linting"
	@echo "  test-setup        - Setup test environment"
	@echo "  test-cleanup      - Cleanup test environment"

# Variables
GOTEST := go test
GOTESTFLAGS := -v -race -timeout=30m
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html
TEST_PACKAGES := ./...

# Test environment variables
export TEST_TIMEOUT := 30m
export TEST_VERBOSE := 1
export TEST_PARALLEL := 4

# Database configuration for tests
export TEST_DB_URL := postgres://test_user:test_password@localhost:5432/near_rt_ric_test?sslmode=disable
export TEST_REDIS_URL := redis://localhost:6379/0
export TEST_KAFKA_BROKERS := localhost:9092

# Docker and container settings
DOCKER_COMPOSE := docker-compose
COMPOSE_FILE := ../docker/docker-compose.test.yml

# Setup test environment
test-setup:
	@echo "Setting up test environment..."
	@mkdir -p reports/coverage reports/performance reports/compliance
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) up -d postgres redis kafka
	@sleep 10  # Wait for containers to be ready
	@echo "Test environment ready"

# Cleanup test environment
test-cleanup:
	@echo "Cleaning up test environment..."
	@$(DOCKER_COMPOSE) -f $(COMPOSE_FILE) down -v
	@docker system prune -f
	@echo "Test environment cleaned up"

# Run all tests
test-all: test-lint test-unit test-integration test-performance test-compliance test-coverage
	@echo "All tests completed successfully!"

# Unit tests
test-unit:
	@echo "Running unit tests..."
	@$(GOTEST) $(GOTESTFLAGS) -short -coverprofile=$(COVERAGE_FILE) $(TEST_PACKAGES)
	@echo "Unit tests completed"

# Integration tests
test-integration: test-setup
	@echo "Running integration tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=integration ./test/integration/...
	@echo "Integration tests completed"

# Performance tests
test-performance: test-setup
	@echo "Running performance tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=performance ./test/performance/...
	@echo "Performance tests completed"

# Compliance tests
test-compliance: test-setup
	@echo "Running O-RAN compliance tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=compliance ./test/compliance/...
	@echo "Compliance tests completed"

# End-to-end tests
test-e2e: test-setup
	@echo "Running end-to-end tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=e2e ./test/e2e/...
	@echo "End-to-end tests completed"

# Coverage report
test-coverage:
	@echo "Generating test coverage report..."
	@$(GOTEST) -coverprofile=$(COVERAGE_FILE) $(TEST_PACKAGES)
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Benchmarks
test-benchmark:
	@echo "Running Go benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...
	@echo "Benchmarks completed"

# Load tests
test-load: test-setup
	@echo "Running load tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=load -run=TestLoad ./test/performance/...
	@echo "Load tests completed"

# Stress tests
test-stress: test-setup
	@echo "Running stress tests..."
	@$(GOTEST) $(GOTESTFLAGS) -tags=stress -run=TestStress ./test/performance/...
	@echo "Stress tests completed"

# Security tests
test-security:
	@echo "Running security tests..."
	@gosec ./...
	@nancy sleuth
	@echo "Security tests completed"

# Linting
test-lint:
	@echo "Running code linting..."
	@golangci-lint run --timeout=5m
	@echo "Linting completed"

# Individual interface tests
test-e2:
	@echo "Running E2 interface tests..."
	@$(GOTEST) $(GOTESTFLAGS) ./test/integration/e2/...
	@echo "E2 interface tests completed"

test-a1:
	@echo "Running A1 interface tests..."
	@$(GOTEST) $(GOTESTFLAGS) ./test/integration/a1/...
	@echo "A1 interface tests completed"

test-o1:
	@echo "Running O1 interface tests..."
	@$(GOTEST) $(GOTESTFLAGS) ./test/integration/o1/...
	@echo "O1 interface tests completed"

test-xapp:
	@echo "Running xApp Manager tests..."
	@$(GOTEST) $(GOTESTFLAGS) ./test/integration/xapp/...
	@echo "xApp Manager tests completed"

# Performance profiling
profile-cpu:
	@echo "Running CPU profiling..."
	@$(GOTEST) -cpuprofile=cpu.prof -bench=. ./test/performance/...
	@go tool pprof cpu.prof

profile-memory:
	@echo "Running memory profiling..."
	@$(GOTEST) -memprofile=mem.prof -bench=. ./test/performance/...
	@go tool pprof mem.prof

# Test data generation
generate-test-data:
	@echo "Generating test data..."
	@go run ./test/utils/data_generator.go
	@echo "Test data generated"

# Test reports
test-report:
	@echo "Generating test reports..."
	@mkdir -p reports
	@$(GOTEST) -json $(TEST_PACKAGES) > reports/test-results.json
	@go-junit-report < reports/test-results.json > reports/test-results.xml
	@echo "Test reports generated in reports/ directory"

# Continuous testing (watch mode)
test-watch:
	@echo "Starting continuous testing (watch mode)..."
	@reflex -r '\.go$$' -s -- make test-unit

# Test specific packages
test-pkg:
	@echo "Running tests for specific package: $(PKG)"
	@$(GOTEST) $(GOTESTFLAGS) ./$(PKG)/...

# Test with race detection
test-race:
	@echo "Running tests with race detection..."
	@$(GOTEST) -race $(TEST_PACKAGES)

# Test with memory sanitizer
test-msan:
	@echo "Running tests with memory sanitizer..."
	@$(GOTEST) -msan $(TEST_PACKAGES)

# Test database migrations
test-migrations:
	@echo "Testing database migrations..."
	@go run ./scripts/migrate.go --test
	@echo "Migration tests completed"

# Validate test fixtures
validate-fixtures:
	@echo "Validating test fixtures..."
	@go run ./test/utils/validate_fixtures.go
	@echo "Fixtures validated"

# Clean test artifacts
clean-test:
	@echo "Cleaning test artifacts..."
	@rm -rf reports/ *.prof $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Test artifacts cleaned"

# Test container health
test-containers:
	@echo "Testing container health..."
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo "Container health checked"

# Test network connectivity
test-network:
	@echo "Testing network connectivity..."
	@nc -zv localhost 5432 || echo "PostgreSQL connection failed"
	@nc -zv localhost 6379 || echo "Redis connection failed"
	@nc -zv localhost 9092 || echo "Kafka connection failed"

# Test with different Go versions
test-go-versions:
	@echo "Testing with different Go versions..."
	@docker run --rm -v $(PWD):/app -w /app golang:1.19 go test ./...
	@docker run --rm -v $(PWD):/app -w /app golang:1.20 go test ./...
	@docker run --rm -v $(PWD):/app -w /app golang:1.21 go test ./...

# Test in different environments
test-environments:
	@echo "Testing in different environments..."
	@TEST_ENV=development make test-unit
	@TEST_ENV=staging make test-integration
	@TEST_ENV=production make test-performance

# Parallel test execution
test-parallel:
	@echo "Running tests in parallel..."
	@$(GOTEST) -parallel $(TEST_PARALLEL) $(TEST_PACKAGES)

# Test timeout scenarios
test-timeouts:
	@echo "Testing timeout scenarios..."
	@$(GOTEST) -timeout=5s -short $(TEST_PACKAGES)

# Memory leak detection
test-memory-leaks:
	@echo "Detecting memory leaks..."
	@$(GOTEST) -memprofile=mem.prof -run=TestMemoryLeak ./test/performance/...
	@go tool pprof -alloc_space mem.prof

# Test configuration validation
test-config:
	@echo "Testing configuration validation..."
	@go run ./cmd/config-validator/main.go --config=test/fixtures/config/

# API compatibility tests
test-api-compatibility:
	@echo "Running API compatibility tests..."
	@$(GOTEST) -tags=compatibility ./test/compatibility/...

# Contract tests
test-contracts:
	@echo "Running contract tests..."
	@$(GOTEST) -tags=contracts ./test/contracts/...

# Chaos engineering tests
test-chaos:
	@echo "Running chaos engineering tests..."
	@$(GOTEST) -tags=chaos ./test/chaos/...

# Generate test documentation
test-docs:
	@echo "Generating test documentation..."
	@go doc -all ./test/... > docs/test-documentation.md
	@echo "Test documentation generated"