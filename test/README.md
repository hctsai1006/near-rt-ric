# O-RAN Near-RT RIC Testing Suite

This directory contains the comprehensive testing suite for the O-RAN Near-RT RIC implementation. The testing strategy covers multiple levels of testing to ensure reliability, performance, and O-RAN compliance.

## Testing Strategy

### 1. Unit Tests
- **Location**: `*_test.go` files alongside source code
- **Coverage**: Individual functions and methods
- **Framework**: Go testing package with testify
- **Target Coverage**: ≥ 80%

### 2. Integration Tests
- **Location**: `test/integration/`
- **Coverage**: Component interactions and API endpoints
- **Framework**: Go testing with Docker containers
- **Database**: TestContainers for isolated testing

### 3. End-to-End Tests
- **Location**: `test/e2e/`
- **Coverage**: Complete O-RAN interface workflows
- **Framework**: Ginkgo and Gomega
- **Environment**: Kubernetes-based testing

### 4. Performance Tests
- **Location**: `test/performance/`
- **Coverage**: Load testing and benchmarks
- **Framework**: Go benchmarking with custom load generators
- **Metrics**: Latency, throughput, resource usage

### 5. Compliance Tests
- **Location**: `test/compliance/`
- **Coverage**: O-RAN Alliance specification compliance
- **Framework**: Custom test framework
- **Standards**: E2AP v3.0, A1 v2.1, O1 v8.0

## Running Tests

### Prerequisites
```bash
# Install test dependencies
go mod download
docker compose up -d  # For integration tests
```

### Unit Tests
```bash
# Run all unit tests
make test-unit

# Run with coverage
make test-coverage

# Run specific package
go test -v ./pkg/e2/...
```

### Integration Tests
```bash
# Run all integration tests
make test-integration

# Run specific interface tests
go test -v ./test/integration/e2/...
go test -v ./test/integration/a1/...
go test -v ./test/integration/o1/...
go test -v ./test/integration/xapp/...
```

### End-to-End Tests
```bash
# Setup test environment
make test-e2e-setup

# Run E2E tests
make test-e2e

# Cleanup
make test-e2e-cleanup
```

### Performance Tests
```bash
# Run benchmarks
make test-benchmark

# Run load tests
make test-load

# Generate performance report
make test-performance-report
```

### Compliance Tests
```bash
# Run O-RAN compliance tests
make test-compliance

# Generate compliance report
make test-compliance-report
```

## Test Configuration

### Environment Variables
- `TEST_DB_URL`: Database connection for integration tests
- `TEST_REDIS_URL`: Redis connection for caching tests
- `TEST_KAFKA_BROKERS`: Kafka brokers for messaging tests
- `TEST_TIMEOUT`: Test timeout duration (default: 30s)
- `TEST_VERBOSE`: Enable verbose test output

### Test Data
- **Fixtures**: `test/fixtures/` - Test data and configurations
- **Mocks**: `test/mocks/` - Mock implementations
- **Schemas**: `test/schemas/` - Validation schemas for testing

## Continuous Integration

Tests are automatically run in GitHub Actions on:
- Pull requests
- Push to main branch
- Nightly for performance regression testing

### Test Reports
- **Coverage**: Generated in `coverage/`
- **Performance**: Generated in `reports/performance/`
- **Compliance**: Generated in `reports/compliance/`

## Test Guidelines

### Writing Tests
1. **Naming**: Test files should end with `_test.go`
2. **Structure**: Use AAA pattern (Arrange, Act, Assert)
3. **Isolation**: Each test should be independent
4. **Cleanup**: Always clean up resources
5. **Documentation**: Add test descriptions and comments

### Best Practices
1. **Mock External Dependencies**: Use mocks for external services
2. **Use TestContainers**: For database and messaging tests
3. **Parallel Execution**: Mark tests as parallel when possible
4. **Flaky Test Prevention**: Use proper timeouts and retries
5. **Resource Management**: Clean up containers and resources

## Troubleshooting

### Common Issues
1. **Port Conflicts**: Ensure test ports don't conflict with running services
2. **Docker Issues**: Check Docker daemon is running for integration tests
3. **Permission Issues**: Ensure proper file permissions for test artifacts
4. **Timeout Issues**: Increase timeout for slow environments

### Test Debugging
```bash
# Run with verbose output
go test -v -timeout=300s ./...

# Run specific test
go test -v -run TestSpecificFunction ./pkg/component/

# Enable race detection
go test -race ./...
```