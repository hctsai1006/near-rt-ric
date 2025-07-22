# O-RAN Near-RT RIC CI/CD Pipeline

This document describes the comprehensive CI/CD pipeline implemented for the O-RAN Near-RT RIC project.

## Overview

The CI/CD pipeline ensures:
- **Code Quality**: Automated linting, security scanning, and compliance checks
- **Testing**: Unit, integration, performance, and compliance testing
- **Security**: Vulnerability scanning, SBOM generation, and security audits
- **Deployment**: Automated staging and production deployments with blue-green strategy
- **Monitoring**: Post-deployment verification and health checks

## Workflows

### 1. Continuous Integration (`ci.yml`)

**Triggers**: Push to `main`/`develop`, Pull Requests

**Jobs**:
- **Code Quality & Security**: golangci-lint, gosec, nancy vulnerability scanner
- **Unit Tests**: Multi-version Go testing (1.19, 1.20, 1.21) with coverage reporting
- **Integration Tests**: E2, A1, O1, xApp interface testing with real databases
- **Performance Tests**: Latency and throughput testing (main branch only)
- **Compliance Tests**: O-RAN Alliance specification compliance validation
- **Frontend Tests**: Angular unit tests, e2e tests, and production builds
- **Docker Build**: Multi-component image builds with Trivy security scanning
- **K8s Validation**: Kubernetes manifest and Helm chart validation
- **Release Preparation**: Multi-platform binary builds and artifact generation

**Key Features**:
- Parallel execution for optimal performance
- Comprehensive test coverage (80%+ requirement)
- Security scanning with SARIF uploads
- Multi-architecture support (amd64, arm64)
- Automatic dependency caching

### 2. Continuous Deployment (`cd.yml`)

**Triggers**: Version tags (`v*`), Manual workflow dispatch

**Jobs**:
- **Build & Push Images**: Multi-component Docker builds to GitHub Container Registry
- **Security Scanning**: Trivy and Grype vulnerability scanning with SBOM generation
- **Deploy Staging**: Automated staging deployment with smoke tests
- **Load Testing**: Staging environment load testing
- **Deploy Production**: Blue-green production deployment with traffic switching
- **Post-deployment Monitoring**: Automated health checks and alerting setup
- **GitHub Release**: Automated release creation with artifacts
- **Rollback**: Automatic rollback on deployment failure

**Key Features**:
- Blue-green deployment strategy
- Automated smoke and load testing
- Multi-environment support (staging, production)
- Rollback capabilities
- Container image signing and SBOM attestation

### 3. Nightly Operations (`nightly.yml`)

**Triggers**: Daily at 2 AM UTC, Manual execution

**Jobs**:
- **Security Audit**: Comprehensive security scanning and secret detection
- **Performance Regression**: Baseline performance comparison
- **Dependency Check**: Automated dependency update analysis
- **Code Quality Metrics**: Cyclomatic complexity, code coverage, LOC analysis
- **Database Maintenance**: Migration testing and performance validation
- **Container Security**: Deep security scanning of all images
- **IaC Validation**: Infrastructure as Code validation with Polaris/Checkov
- **Documentation Check**: Documentation freshness and TODO tracking
- **Comprehensive Reporting**: Consolidated nightly health report
- **Cleanup**: Automated cleanup of old artifacts and workflow runs

**Key Features**:
- Proactive security monitoring
- Performance regression detection
- Comprehensive health reporting
- Automated maintenance tasks
- Slack notifications

## Configuration Files

### Dependency Management (`.github/dependabot.yml`)

- **Go Modules**: Weekly updates on Mondays
- **NPM Dependencies**: Weekly updates on Tuesdays
- **Docker Images**: Weekly updates on Wednesdays  
- **GitHub Actions**: Weekly updates on Thursdays
- **Helm Charts**: Monthly updates on first Monday

**Features**:
- Automated pull requests
- Semantic version filtering
- Auto-merge for minor/patch updates
- Review assignment to maintainers

### Code Quality (`.golangci.yml`)

- **Linters**: 30+ enabled linters including security, performance, and style
- **Complexity Limits**: Max cyclomatic complexity of 15
- **Function Limits**: Max 100 lines, 50 statements
- **Line Length**: 120 characters max
- **Security Rules**: gosec with medium confidence threshold
- **Test Exclusions**: Relaxed rules for test files

## Security Features

### Vulnerability Scanning
- **Code**: gosec, nancy, TruffleHog secret detection
- **Containers**: Trivy, Grype multi-scanner approach
- **Dependencies**: Automated vulnerability detection and reporting
- **Infrastructure**: Checkov IaC scanning, Polaris policy validation

### Supply Chain Security
- **SBOM Generation**: Software Bill of Materials for all images
- **Image Signing**: Container image attestation
- **Dependency Verification**: Go module integrity checks
- **License Compliance**: Automated license scanning

### Access Control
- **Environment Protection**: Required reviewers for production
- **Secret Management**: GitHub secrets for sensitive data
- **RBAC**: Role-based access control for deployments
- **Audit Logging**: Comprehensive audit trail

## Performance Testing

### E2 Interface Performance
- **Latency**: <10ms requirement validation
- **Throughput**: 100+ operations/second
- **Concurrency**: 100 concurrent E2 nodes
- **Load Testing**: Sustained load testing

### Integration Testing
- **Real Protocols**: SCTP, NETCONF SSH, HTTP REST
- **Database Integration**: PostgreSQL with TestContainers
- **Message Validation**: ASN.1, JSON schema validation
- **Error Handling**: Comprehensive error scenario testing

## Deployment Strategy

### Blue-Green Deployment
1. Deploy new version to "green" environment
2. Run comprehensive health checks
3. Switch traffic from "blue" to "green"
4. Monitor for issues
5. Cleanup old "blue" deployment

### Environment Progression
- **Development**: Continuous integration testing
- **Staging**: Integration and load testing
- **Production**: Blue-green deployment with monitoring

### Rollback Strategy
- Automatic rollback on deployment failure
- Health check validation
- Traffic switching capabilities
- Data consistency verification

## Monitoring and Alerting

### Health Checks
- Application health endpoints
- Database connectivity
- Interface availability
- Performance metrics

### Notifications
- Slack integration for pipeline status
- GitHub issue creation on failures
- Email alerts for critical issues
- Dashboard integration

### Metrics Collection
- Deployment success rates
- Test execution times
- Security scan results
- Performance benchmarks

## Usage

### Running Tests Locally
```bash
# Unit tests
make test-unit

# Integration tests  
make test-integration

# Performance tests
make test-performance

# All tests
make test-all
```

### Manual Deployment
```bash
# Deploy to staging
gh workflow run cd.yml -f environment=staging -f version=v1.0.0

# Deploy to production
gh workflow run cd.yml -f environment=production -f version=v1.0.0
```

### Quality Gates

All pull requests must pass:
- ✅ Code quality checks (golangci-lint)
- ✅ Security scans (gosec, trivy)
- ✅ Unit tests with 80%+ coverage
- ✅ Integration tests
- ✅ Docker image builds
- ✅ Kubernetes manifest validation

Production deployments require:
- ✅ All CI checks passing
- ✅ Staging deployment successful
- ✅ Load testing passed
- ✅ Security scans clear
- ✅ Manual approval (for tagged releases)

## Maintenance

### Regular Tasks
- **Weekly**: Dependency updates via Dependabot
- **Daily**: Nightly comprehensive health checks
- **Monthly**: Security audit reviews
- **Quarterly**: Performance baseline updates

### Troubleshooting
- Check workflow run logs in GitHub Actions
- Review security scan results in Security tab
- Monitor deployment status in Environments
- Check notification channels for alerts

## Best Practices

1. **Version Tags**: Use semantic versioning for releases
2. **Branch Protection**: Require PR reviews and status checks
3. **Secret Rotation**: Regular rotation of deployment secrets
4. **Documentation**: Keep deployment docs up to date
5. **Testing**: Write tests for new features and bug fixes

## Contributing

1. Create feature branch from `develop`
2. Implement changes with tests
3. Ensure CI pipeline passes
4. Request code review
5. Merge to `develop` after approval
6. Release from `main` with version tag

This CI/CD pipeline ensures production-ready, secure, and compliant O-RAN Near-RT RIC deployments with comprehensive quality gates and automated operations.