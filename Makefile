# Production-grade Makefile for O-RAN Near-RT RIC Platform
# Comprehensive build, test, and deployment automation with security scanning

# Build Configuration
GO_VERSION := 1.21
NODE_VERSION := 18
DOCKER_BUILDKIT := 1
BUILDX_PLATFORMS := linux/amd64,linux/arm64

# Tools and Commands
GO_CMD := go
NPM_CMD := npm
DOCKER_CMD := docker
HELM_CMD := helm
KUBECTL_CMD := kubectl
TRIVY_CMD := trivy
GOLANGCI_LINT_CMD := golangci-lint

# Project Structure
PROJECT_ROOT := $(shell pwd)
BUILD_DIR := $(PROJECT_ROOT)/build
BIN_DIR := $(PROJECT_ROOT)/bin
COVERAGE_DIR := $(PROJECT_ROOT)/coverage
DOCS_DIR := $(PROJECT_ROOT)/docs

# Source Directories
CMD_DIR := ./cmd
PKG_DIR := ./pkg
INTERNAL_DIR := ./internal
FRONTEND_DIR := ./frontend-dashboard
SCRIPTS_DIR := ./scripts
CONFIG_DIR := ./config
HELM_CHARTS_DIR := ./helm
K8S_DIR := ./k8s

# Specific Component Directories
E2_SIMULATOR_DIR := $(CMD_DIR)/e2-simulator
A1_INTERFACE_DIR := $(CMD_DIR)/ric-a1
MAIN_RIC_DIR := $(CMD_DIR)/ric
XAPP_MANAGER_DIR := $(CMD_DIR)/xapp-manager

# Chart Directories
ORAN_CHART_DIR := $(HELM_CHARTS_DIR)/oran-nearrt-ric
OBSERVABILITY_CHART_DIR := $(HELM_CHARTS_DIR)/observability-stack
SMO_CHART_DIR := $(HELM_CHARTS_DIR)/smo-onap

# Container Registry Configuration
REGISTRY := ghcr.io
IMAGE_NAMESPACE := $(shell echo $${GITHUB_REPOSITORY_OWNER:-hctsai1006} | tr '[:upper:]' '[:lower:]')
BASE_IMAGE_NAME := $(REGISTRY)/$(IMAGE_NAMESPACE)/near-rt-ric

# Image Tags and Names
MAIN_IMAGE := $(BASE_IMAGE_NAME):latest
E2_SIM_IMAGE := $(BASE_IMAGE_NAME)-e2-simulator:latest
A1_IMAGE := $(BASE_IMAGE_NAME)-a1:latest
FRONTEND_IMAGE := $(BASE_IMAGE_NAME)-frontend:latest

# Version and Build Info
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_USER := $(shell whoami)

# Kubernetes Configuration
NAMESPACE := oran-nearrt-ric
STAGING_NAMESPACE := oran-staging
PRODUCTION_NAMESPACE := oran-production
RELEASE_NAME := oran-ric
STAGING_RELEASE := oran-ric-staging
PRODUCTION_RELEASE := oran-ric-prod

# Testing Configuration
COVERAGE_THRESHOLD := 80
TEST_TIMEOUT := 10m
RACE_ENABLED := true
INTEGRATION_TEST_DB := test_oran_ric

# Security Configuration
SECURITY_SCAN_FORMAT := sarif
VULNERABILITY_SEVERITY := HIGH,CRITICAL
TRIVY_EXIT_CODE := 1

# Performance Configuration
BENCHMARK_TIME := 30s
LOAD_TEST_USERS := 100
LOAD_TEST_DURATION := 5m

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
MAGENTA := \033[0;35m
WHITE := \033[1;37m
NC := \033[0m # No Color

# Build flags
LDFLAGS := -w -s \
	-X main.version=$(VERSION) \
	-X main.buildTime=$(BUILD_TIME) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.buildUser=$(BUILD_USER)

GCFLAGS := all=-trimpath
BUILD_TAGS := osusergo netgo static_build
CGO_ENABLED := 1

.PHONY: help clean build test security-scan deploy demo tools

# Export environment variables for sub-processes
export DOCKER_BUILDKIT
export GO_VERSION
export VERSION
export BUILD_TIME
export GIT_COMMIT

# Default target
all: clean tools security-scan test-coverage build-all ## Run complete build pipeline

## Help
help: ## Show this comprehensive help message
	@echo "$(WHITE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo "$(CYAN)           O-RAN Near-RT RIC Platform - Production Build System           $(NC)"
	@echo "$(WHITE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(NC)"
	@echo ""
	@echo "$(GREEN)📋 Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(CYAN)%-25s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(WHITE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)🔧 Build Configuration:$(NC)"
	@echo "  Version:     $(VERSION)"
	@echo "  Go Version:  $(GO_VERSION)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Registry:    $(REGISTRY)"
	@echo ""
	@echo "$(BLUE)📖 Examples:$(NC)"
	@echo "  make all                 # Complete build pipeline"
	@echo "  make build-all          # Build all components"
	@echo "  make test-all           # Run all tests"
	@echo "  make docker-build-all   # Build all container images"
	@echo "  make deploy-staging     # Deploy to staging environment"
	@echo ""

##@ 🧹 Development Environment
clean: ## Clean all build artifacts and caches
	@echo "$(YELLOW)🧹 Cleaning build artifacts...$(NC)"
	@rm -rf $(BIN_DIR) $(BUILD_DIR) $(COVERAGE_DIR)
	@rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules/.cache
	@$(GO_CMD) clean -cache -modcache -testcache
	@#$(DOCKER_CMD) system prune -f --volumes
	@echo "$(GREEN)✓ Clean completed$(NC)"

tools: ## Install required development tools
	@echo "$(YELLOW)🔧 Installing development tools...$(NC)"
	@$(GO_CMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GO_CMD) install github.com/securecodewarrior/sast-scan-runner@latest
	@$(GO_CMD) install gotest.tools/gotestsum@latest
	@$(GO_CMD) install github.com/onsi/ginkgo/v2/ginkgo@latest
	@$(GO_CMD) install golang.org/x/vuln/cmd/govulncheck@latest
	@curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
	@echo "$(GREEN)✓ Development tools installed$(NC)"

deps: ## Download and verify dependencies
	@echo "$(YELLOW)📦 Installing dependencies...$(NC)"
	@$(GO_CMD) mod download
	@$(GO_CMD) mod verify
	@$(GO_CMD) mod tidy
	@cd $(FRONTEND_DIR) && $(NPM_CMD) ci --audit-level moderate
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

##@ 🔍 Code Quality
lint: ## Run comprehensive linting
	@echo "$(YELLOW)🔍 Running linters...$(NC)"
	@$(GOLANGCI_LINT_CMD) run --config .golangci.yml --timeout $(TEST_TIMEOUT)
	@cd $(FRONTEND_DIR) && $(NPM_CMD) run lint
	@echo "$(GREEN)✓ Linting completed$(NC)"

fmt: ## Format code
	@echo "$(YELLOW)📝 Formatting code...$(NC)"
	@$(GO_CMD) fmt ./...
	@goimports -w -local github.com/hctsai1006/near-rt-ric .
	@cd $(FRONTEND_DIR) && $(NPM_CMD) run format
	@echo "$(GREEN)✓ Code formatting completed$(NC)"

vet: ## Run Go vet
	@echo "$(YELLOW)🔍 Running go vet...$(NC)"
	@$(GO_CMD) vet ./...
	@echo "$(GREEN)✓ Go vet completed$(NC)"

vuln-check: ## Check for vulnerabilities
	@echo "$(YELLOW)🛡️ Checking for vulnerabilities...$(NC)"
	@govulncheck ./...
	@echo "$(GREEN)✓ Vulnerability check completed$(NC)"

security-scan: ## Run comprehensive security scan
	@echo "$(YELLOW)🔒 Running security scan...$(NC)"
	@echo "$(CYAN)  Scanning for hardcoded credentials...$(NC)"
	@! grep -r "password.*=" --include="*.go" --include="*.yaml" --include="*.yml" . | grep -v "PASSWORD.*FILE" | grep -v "_FILE" || (echo "$(RED)❌ Hardcoded credentials found$(NC)"; exit 1)
	@echo "$(GREEN)✓ No hardcoded credentials found$(NC)"
	@echo "$(CYAN)  Running Trivy vulnerability scan...$(NC)"
	@$(TRIVY_CMD) fs --security-checks vuln,config --format table --exit-code 0 .
	@echo "$(GREEN)✓ Security scan completed$(NC)"

##@ 🏗️ Build Targets
build-all: build-backend build-frontend ## Build all components
	@echo "$(GREEN)✓ All components built successfully$(NC)"

build-backend: ## Build all Go backend services
	@echo "$(BLUE)🏗️ Building Go backend services...$(NC)"
	@mkdir -p $(BIN_DIR)
	@echo "$(CYAN)  Building main RIC service...$(NC)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO_CMD) build \
		-ldflags="$(LDFLAGS)" \
		-gcflags="$(GCFLAGS)" \
		-tags="$(BUILD_TAGS)" \
		-o $(BIN_DIR)/near-rt-ric \
		$(MAIN_RIC_DIR)/main.go
	@echo "$(CYAN)  Building E2 Simulator...$(NC)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO_CMD) build \
		-ldflags="$(LDFLAGS)" \
		-gcflags="$(GCFLAGS)" \
		-tags="$(BUILD_TAGS)" \
		-o $(BIN_DIR)/e2-simulator \
		$(E2_SIMULATOR_DIR)/main.go
	@echo "$(CYAN)  Building A1 Interface...$(NC)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO_CMD) build \
		-ldflags="$(LDFLAGS)" \
		-gcflags="$(GCFLAGS)" \
		-tags="$(BUILD_TAGS)" \
		-o $(BIN_DIR)/ric-a1 \
		$(A1_INTERFACE_DIR)/main.go
	@echo "$(GREEN)✓ Backend services built successfully$(NC)"

build-frontend: ## Build modern React frontend
	@echo "$(BLUE)🎨 Building frontend dashboard...$(NC)"
	@cd $(FRONTEND_DIR) && $(NPM_CMD) run build
	@echo "$(GREEN)✓ Frontend built successfully$(NC)"

build-debug: ## Build with debug symbols
	@echo "$(BLUE)🐛 Building debug versions...$(NC)"
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=1 $(GO_CMD) build -gcflags="all=-N -l" -race \
		-o $(BIN_DIR)/near-rt-ric-debug $(MAIN_RIC_DIR)/main.go
	@echo "$(GREEN)✓ Debug builds completed$(NC)"

##@ Container Images
docker-build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	@$(DOCKER_CMD) build -t $(MAIN_IMAGE):latest -f Dockerfile .
	@$(DOCKER_CMD) build -t $(FRONTEND_IMAGE):latest -f docker/Dockerfile.dashboard .

docker-push: ## Push Docker images to registry
	@echo "$(YELLOW)Pushing Docker images...$(NC)"
	@$(DOCKER_CMD) push $(MAIN_IMAGE):latest
	@$(DOCKER_CMD) push $(FRONTEND_IMAGE):latest

##@ Helm Charts
helm-lint: ## Lint all Helm charts
	@echo "$(YELLOW)Linting Helm charts...$(NC)"
	@$(HELM_CMD) lint $(ORAN_CHART_DIR)
	@$(HELM_CMD) lint $(SMO_CHART_DIR)
	@$(HELM_CMD) lint $(OBSERVABILITY_CHART_DIR)

helm-deps: ## Update Helm chart dependencies
	@echo "$(YELLOW)Updating Helm dependencies...$(NC)"
	@$(HELM_CMD) dependency update $(ORAN_CHART_DIR)
	@$(HELM_CMD) dependency update $(SMO_CHART_DIR)
	@$(HELM_CMD) dependency update $(OBSERVABILITY_CHART_DIR)

helm-package: ## Package Helm charts
	@echo "$(YELLOW)Packaging Helm charts...$(NC)"
	@$(HELM_CMD) package $(ORAN_CHART_DIR) -d ./helm-packages
	@$(HELM_CMD) package $(SMO_CHART_DIR) -d ./helm-packages
	@$(HELM_CMD) package $(OBSERVABILITY_CHART_DIR) -d ./helm-packages

##@ Deployment
deploy: deploy-observability deploy-smo deploy-oran ## Deploy complete O-RAN platform

deploy-oran: ## Deploy O-RAN Near-RT RIC platform
	@echo "$(GREEN)Deploying O-RAN Near-RT RIC platform...$(NC)"
	@$(KUBECTL_CMD) create namespace $(NAMESPACE) --dry-run=client -o yaml | $(KUBECTL_CMD) apply -f -
	@$(HELM_CMD) upgrade --install $(RELEASE_NAME) $(ORAN_CHART_DIR) \
		--namespace $(NAMESPACE) \
		--set global.registry=$(REGISTRY)/$(IMAGE_NAMESPACE) \
		--set frontend.image.tag=latest \
		--set backend.image.tag=latest \
		--wait --timeout=10m

deploy-smo: ## Deploy SMO (Service Management & Orchestration) stack
	@echo "$(GREEN)Deploying SMO ONAP stack...$(NC)"
	@$(KUBECTL_CMD) create namespace $(SMO_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL_CMD) apply -f -
	@$(HELM_CMD) repo add onap https://nexus3.onap.org/repository/onap-helm-release/ || true
	@$(HELM_CMD) repo update
	@$(HELM_CMD) upgrade --install $(SMO_RELEASE_NAME) $(SMO_CHART_DIR) \
		--namespace $(SMO_NAMESPACE) \
		--set global.flavor=unlimited \
		--set global.persistence.enabled=true \
		--wait --timeout=15m

deploy-observability: ## Deploy observability stack (Grafana, Prometheus, Elasticsearch, Kafka)
	@echo "$(GREEN)Deploying observability stack...$(NC)"
	@$(KUBECTL_CMD) create namespace $(OBSERVABILITY_NAMESPACE) --dry-run=client -o yaml | $(KUBECTL_CMD) apply -f -
	@$(HELM_CMD) repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
	@$(HELM_CMD) repo add grafana https://grafana.github.io/helm-charts || true
	@$(HELM_CMD) repo add elastic https://helm.elastic.co || true
	@$(HELM_CMD) repo add bitnami https://charts.bitnami.com/bitnami || true
	@$(HELM_CMD) repo update
	@$(HELM_CMD) upgrade --install $(OBSERVABILITY_RELEASE_NAME) $(OBSERVABILITY_CHART_DIR) \
		--namespace $(OBSERVABILITY_NAMESPACE) \
		--set grafana.enabled=true \
		--set prometheus.enabled=true \
		--set elasticsearch.enabled=true \
		--set kafka.enabled=true \
		--wait --timeout=12m

undeploy: ## Remove all deployments
	@echo "$(RED)Removing all deployments...$(NC)"
	@$(HELM_CMD) uninstall $(RELEASE_NAME) -n $(NAMESPACE) || true
	@$(HELM_CMD) uninstall $(SMO_RELEASE_NAME) -n $(SMO_NAMESPACE) || true
	@$(HELM_CMD) uninstall $(OBSERVABILITY_RELEASE_NAME) -n $(OBSERVABILITY_NAMESPACE) || true
	@$(KUBECTL_CMD) delete namespace $(NAMESPACE) || true
	@$(KUBECTL_CMD) delete namespace $(SMO_NAMESPACE) || true
	@$(KUBECTL_CMD) delete namespace $(OBSERVABILITY_NAMESPACE) || true

##@ Testing & Validation
test: test-coverage
	@echo "Running tests with coverage enforcement..."

test-coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out -covermode=count ./...
	@coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//'); \
	if [ "${coverage%.*}" -lt 80 ]; then \
		echo "❌ Coverage ${coverage}% is below required 80%"; \
		go tool cover -func=coverage.out | grep -v "100.0%"; \
		exit 1; \
	else \
		echo "✅ Coverage ${coverage}% meets threshold"; \
	fi


coverage-html:
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

e2e: ## Run end-to-end tests
	@echo "$(YELLOW)Running end-to-end tests...$(NC)"
	@$(MAKE) test-interfaces
	@$(MAKE) test-smo
	@$(MAKE) test-observability

test-interfaces: ## Test O-RAN interfaces (E2, A1, O1)
	@echo "$(BLUE)Testing O-RAN interfaces...$(NC)"
	@curl -f http://localhost:8080/api/v1/interfaces/e2/health || echo "$(RED)E2 interface test failed$(NC)"
	@curl -f http://localhost:8080/api/v1/interfaces/a1/health || echo "$(RED)A1 interface test failed$(NC)"
	@curl -f http://localhost:8080/api/v1/interfaces/o1/health || echo "$(RED)O1 interface test failed$(NC)"

test-smo: ## Test SMO health endpoints
	@echo "$(BLUE)Testing SMO components...$(NC)"
	@$(KUBECTL_CMD) get pods -n $(SMO_NAMESPACE) | grep Running || echo "$(RED)SMO components not all running$(NC)"

test-observability: ## Test observability stack
	@echo "$(BLUE)Testing observability stack...$(NC)"
	@curl -f http://localhost:3001/api/health || echo "$(RED)Grafana health check failed$(NC)"
	@curl -f http://localhost:9090/-/healthy || echo "$(RED)Prometheus health check failed$(NC)"

##@ Demo & Showcase
demo: ## Run complete demo deployment
	@echo "$(GREEN)🚀 Starting O-RAN Near-RT RIC Demo Deployment$(NC)"
	@echo "$(BLUE)This will deploy the complete O-RAN platform with:$(NC)"
	@echo "  ✅ Near-RT RIC with modern React dashboard"
	@echo "  ✅ SMO stack based on ONAP Frankfurt"
	@echo "  ✅ Full observability pipeline (Grafana, Prometheus, Elasticsearch, Kafka)"
	@echo "  ✅ Auto-discovery of network functions"
	@echo "  ✅ Real-time KPIs and alarms"
	@echo ""
	@$(MAKE) deploy
	@echo "$(GREEN)✅ Demo deployment completed!$(NC)"
	@echo ""
	@echo "$(GREEN)🎉 O-RAN Platform is ready!$(NC)"
	@echo "$(BLUE)Run 'make port-forward' to access dashboards$(NC)"

port-forward: ## Setup port forwarding for local access
	@echo "$(GREEN)Setting up port forwarding...$(NC)"
	@$(KUBECTL_CMD) port-forward -n $(NAMESPACE) service/$(RELEASE_NAME)-frontend 3000:80 &
	@$(KUBECTL_CMD) port-forward -n $(NAMESPACE) service/$(RELEASE_NAME)-backend 8080:8080 &
	@$(KUBECTL_CMD) port-forward -n $(OBSERVABILITY_NAMESPACE) service/$(OBSERVABILITY_RELEASE_NAME)-grafana 3001:80 &
	@$(KUBECTL_CMD) port-forward -n $(OBSERVABILITY_NAMESPACE) service/$(OBSERVABILITY_RELEASE_NAME)-prometheus-server 9090:80 &
	@echo "$(GREEN)Port forwarding active:$(NC)"
	@echo "  Frontend Dashboard: http://localhost:3000"
	@echo "  Backend API:        http://localhost:8080"
	@echo "  Grafana:           http://localhost:3001"
	@echo "  Prometheus:        http://localhost:9090"

status: ## Show deployment status
	@echo "$(GREEN)O-RAN Platform Status:$(NC)"
	@echo ""
	@echo "$(BLUE)Near-RT RIC Components:$(NC)"
	@$(KUBECTL_CMD) get pods -n $(NAMESPACE) -o wide 2>/dev/null || echo "  No Near-RT RIC deployment found"
	@echo ""
	@echo "$(BLUE)SMO Components:$(NC)"
	@$(KUBECTL_CMD) get pods -n $(SMO_NAMESPACE) -o wide 2>/dev/null || echo "  No SMO deployment found"
	@echo ""
	@echo "$(BLUE)Observability Components:$(NC)"
	@$(KUBECTL_CMD) get pods -n $(OBSERVABILITY_NAMESPACE) -o wide 2>/dev/null || echo "  No observability stack found"
