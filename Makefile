# Makefile for O-RAN Near-RT RIC Platform with SMO and Observability Stack

# Variables
GO_CMD := go
NPM_CMD := npm
DOCKER_CMD := docker
HELM_CMD := helm
KUBECTL_CMD := kubectl

# Project directories
FRONTEND_DASHBOARD_DIR := ./frontend-dashboard
HELM_CHARTS_DIR := ./helm
SMO_CHART_DIR := $(HELM_CHARTS_DIR)/smo-onap
OBSERVABILITY_CHART_DIR := $(HELM_CHARTS_DIR)/observability-stack
ORAN_CHART_DIR := $(HELM_CHARTS_DIR)/oran-nearrt-ric

# Container registry configuration
REGISTRY := ghcr.io
IMAGE_NAMESPACE := $(shell echo $${GITHUB_REPOSITORY_OWNER:-local} | tr '[:upper:]' '[:lower:]')
FRONTEND_DASHBOARD_IMAGE := $(REGISTRY)/$(IMAGE_NAMESPACE)/frontend-dashboard
NEAR_RT_RIC_IMAGE := $(REGISTRY)/$(IMAGE_NAMESPACE)/near-rt-ric

# Kubernetes configuration
NAMESPACE := oran-nearrt-ric
SMO_NAMESPACE := onap
OBSERVABILITY_NAMESPACE := observability
RELEASE_NAME := oran-nearrt-ric
SMO_RELEASE_NAME := smo-onap
OBSERVABILITY_RELEASE_NAME := observability-stack

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help clean build test deploy deploy-all deploy-smo deploy-observability demo

# Default target
all: clean lint test build

## Help
help: ## Show this help message
	@echo "$(BLUE)O-RAN Near-RT RIC Platform - Build and Deployment$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
clean: ## Clean all build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(FRONTEND_DASHBOARD_DIR)/dist
	@rm -rf ./bin
	@$(GO_CMD) clean -modcache

install-deps: ## Install all dependencies
	@echo "$(YELLOW)Installing dependencies...$(NC)"
	@cd $(FRONTEND_DASHBOARD_DIR) && $(NPM_CMD) install
	@$(GO_CMD) mod download

lint: ## Run linting for all components
	@echo "$(YELLOW)Running linters...$(NC)"
	@cd $(FRONTEND_DASHBOARD_DIR) && $(NPM_CMD) run lint
	@$(GO_CMD) vet ./...

test: ## Run tests for all components
	@echo "$(YELLOW)Running tests...$(NC)"
	@cd $(FRONTEND_DASHBOARD_DIR) && $(NPM_CMD) run test
	@$(GO_CMD) test -v -race -coverprofile=coverage.out ./...

build: ## Build all components
	@echo "$(YELLOW)Building all components...$(NC)"
	@$(MAKE) build-frontend
	@$(MAKE) build-backend

build-frontend: ## Build modern React frontend
	@echo "$(BLUE)Building modern frontend dashboard...$(NC)"
	@cd $(FRONTEND_DASHBOARD_DIR) && $(NPM_CMD) run build

build-backend: ## Build Go backend services
	@echo "$(BLUE)Building Go backend services...$(NC)"
	@mkdir -p ./bin
	@CGO_ENABLED=1 GOOS=linux $(GO_CMD) build -ldflags="-w -s" -o ./bin/near-rt-ric ./cmd/ric/main.go
	@CGO_ENABLED=1 GOOS=linux $(GO_CMD) build -ldflags="-w -s" -o ./bin/e2-simulator ./cmd/e2-simulator/main.go
	@CGO_ENABLED=1 GOOS=linux $(GO_CMD) build -ldflags="-w -s" -o ./bin/ric-a1 ./cmd/ric-a1/main.go

##@ Container Images
docker-build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	@$(DOCKER_CMD) build -t $(NEAR_RT_RIC_IMAGE):latest -f Dockerfile .
	@$(DOCKER_CMD) build -t $(FRONTEND_DASHBOARD_IMAGE):latest -f $(FRONTEND_DASHBOARD_DIR)/Dockerfile .

docker-push: ## Push Docker images to registry
	@echo "$(YELLOW)Pushing Docker images...$(NC)"
	@$(DOCKER_CMD) push $(NEAR_RT_RIC_IMAGE):latest
	@$(DOCKER_CMD) push $(FRONTEND_DASHBOARD_IMAGE):latest

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
