#\!/bin/bash
# O-RAN Near-RT RIC Automated Deployment Script
set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
CLUSTER_NAME="near-rt-ric"
NAMESPACE="oran-nearrt-ric"
HELM_RELEASE="oran-nearrt-ric"
KIND_CONFIG="config/kind-config.yaml"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

check_prerequisites() {
    log_info "Checking prerequisites..."
    local missing_deps=()
    
    if \! command_exists docker; then missing_deps+=("docker"); fi
    if \! command_exists kubectl; then missing_deps+=("kubectl"); fi
    if \! command_exists helm; then missing_deps+=("helm"); fi
    if \! command_exists kind; then missing_deps+=("kind"); fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        exit 1
    fi
    log_success "All prerequisites found"
}

setup_kind_cluster() {
    log_info "Setting up KIND cluster..."
    
    if kind get clusters  < /dev/null |  grep -q "$CLUSTER_NAME"; then
        log_info "Using existing cluster"
        return 0
    fi
    
    if [ -f "$KIND_CONFIG" ]; then
        kind create cluster --name "$CLUSTER_NAME" --config "$KIND_CONFIG"
    else
        kind create cluster --name "$CLUSTER_NAME"
    fi
    
    kubectl cluster-info --context kind-"$CLUSTER_NAME"
    log_success "KIND cluster ready"
}

install_dependencies() {
    log_info "Installing Helm dependencies..."
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
    helm repo add grafana https://grafana.github.io/helm-charts || true
    helm repo add bitnami https://charts.bitnami.com/bitnami || true
    helm repo update
    
    cd helm/oran-nearrt-ric
    helm dependency build
    cd -
    log_success "Dependencies installed"
}

deploy_platform() {
    log_info "Deploying O-RAN Near-RT RIC platform..."
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    helm upgrade --install "$HELM_RELEASE" ./helm/oran-nearrt-ric/ \
        --namespace "$NAMESPACE" \
        --set oran.enabled=true \
        --set monitoring.prometheus.enabled=true \
        --set monitoring.grafana.enabled=true \
        --wait --timeout=10m
    
    log_success "Platform deployed"
}

verify_deployment() {
    log_info "Verifying deployment..."
    kubectl get pods -n "$NAMESPACE"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance="$HELM_RELEASE" -n "$NAMESPACE" --timeout=300s || true
    log_success "Deployment verified"
}

setup_port_forwarding() {
    log_info "Setting up port forwarding..."
    pkill -f "kubectl port-forward" || true
    sleep 2
    
    kubectl port-forward -n "$NAMESPACE" service/"$HELM_RELEASE" 8080:8080 &
    log_success "Port forwarding configured"
}

main() {
    echo "🚀 O-RAN Near-RT RIC Automated Deployment"
    echo "========================================="
    
    if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        echo "Usage: $0 [--help]"
        exit 0
    fi
    
    check_prerequisites
    setup_kind_cluster
    install_dependencies
    deploy_platform
    verify_deployment
    setup_port_forwarding
    
    echo "🎉 Deployment completed!"
    echo "📊 Main Dashboard: http://localhost:8080"
    wait
}

main "$@"
