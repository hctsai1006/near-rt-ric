#!/bin/bash
# Docker build script for O-RAN Near-RT RIC
# Builds all Docker images with proper tagging and security scanning

set -e

# Configuration
PROJECT_NAME="o-ran-near-rt-ric"
REGISTRY=${DOCKER_REGISTRY:-"localhost:5000"}
VERSION=${BUILD_VERSION:-"latest"}
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Build function for individual components
build_image() {
    local component=$1
    local dockerfile=$2
    local context=${3:-"."}
    local target=${4:-"runtime"}
    
    local image_name="${REGISTRY}/${PROJECT_NAME}-${component}:${VERSION}"
    local image_name_latest="${REGISTRY}/${PROJECT_NAME}-${component}:latest"
    
    log_info "Building ${component} image..."
    
    # Build arguments
    BUILD_ARGS=(
        "--build-arg" "BUILD_DATE=${BUILD_DATE}"
        "--build-arg" "GIT_COMMIT=${GIT_COMMIT}"
        "--build-arg" "VERSION=${VERSION}"
        "--target" "${target}"
        "--tag" "${image_name}"
        "--tag" "${image_name_latest}"
        "--file" "${dockerfile}"
    )
    
    # Add BuildKit features for better caching and security
    export DOCKER_BUILDKIT=1
    
    if docker build "${BUILD_ARGS[@]}" "${context}"; then
        log_success "Successfully built ${component} image"
        
        # Security scan if trivy is available
        if command -v trivy &> /dev/null; then
            log_info "Running security scan for ${component}..."
            trivy image --exit-code 1 --severity HIGH,CRITICAL "${image_name}" || {
                log_warning "Security vulnerabilities found in ${component}, but continuing build"
            }
        fi
        
        return 0
    else
        log_error "Failed to build ${component} image"
        return 1
    fi
}

# Build all images
build_all() {
    log_info "Starting build process for O-RAN Near-RT RIC"
    log_info "Registry: ${REGISTRY}"
    log_info "Version: ${VERSION}"
    log_info "Build Date: ${BUILD_DATE}"
    log_info "Git Commit: ${GIT_COMMIT}"
    
    # Create buildx builder if it doesn't exist (for multi-platform builds)
    if ! docker buildx ls | grep -q "oran-builder"; then
        log_info "Creating buildx builder for multi-platform builds..."
        docker buildx create --name oran-builder --use --bootstrap
    else
        docker buildx use oran-builder
    fi
    
    # Build main application (production and development variants)
    log_info "Building main Near-RT RIC application..."
    build_image "main" "Dockerfile" "." "runtime"
    build_image "main-dev" "Dockerfile" "." "development"
    
    # Build specialized interface components
    build_image "e2-interface" "docker/Dockerfile.e2-interface" "." "runtime"
    build_image "a1-interface" "docker/Dockerfile.a1-interface" "." "runtime"
    build_image "o1-interface" "docker/Dockerfile.o1-interface" "." "runtime"
    build_image "xapp-manager" "docker/Dockerfile.xapp-manager" "." "runtime"
    
    log_success "All images built successfully!"
}

# Push images to registry
push_images() {
    if [ -z "$SKIP_PUSH" ]; then
        log_info "Pushing images to registry ${REGISTRY}..."
        
        for component in main main-dev e2-interface a1-interface o1-interface xapp-manager; do
            local image_name="${REGISTRY}/${PROJECT_NAME}-${component}:${VERSION}"
            local image_name_latest="${REGISTRY}/${PROJECT_NAME}-${component}:latest"
            
            log_info "Pushing ${component}..."
            docker push "${image_name}"
            docker push "${image_name_latest}"
        done
        
        log_success "All images pushed successfully!"
    else
        log_info "Skipping push (SKIP_PUSH is set)"
    fi
}

# Multi-platform build
build_multiplatform() {
    log_info "Building multi-platform images for linux/amd64 and linux/arm64..."
    
    # Platforms to build for
    PLATFORMS="linux/amd64,linux/arm64"
    
    for component in main e2-interface a1-interface o1-interface xapp-manager; do
        local dockerfile="Dockerfile"
        case $component in
            e2-interface) dockerfile="docker/Dockerfile.e2-interface" ;;
            a1-interface) dockerfile="docker/Dockerfile.a1-interface" ;;
            o1-interface) dockerfile="docker/Dockerfile.o1-interface" ;;
            xapp-manager) dockerfile="docker/Dockerfile.xapp-manager" ;;
        esac
        
        local image_name="${REGISTRY}/${PROJECT_NAME}-${component}:${VERSION}"
        local image_name_latest="${REGISTRY}/${PROJECT_NAME}-${component}:latest"
        
        log_info "Building multi-platform ${component} image..."
        
        docker buildx build \
            --platform "${PLATFORMS}" \
            --build-arg "BUILD_DATE=${BUILD_DATE}" \
            --build-arg "GIT_COMMIT=${GIT_COMMIT}" \
            --build-arg "VERSION=${VERSION}" \
            --tag "${image_name}" \
            --tag "${image_name_latest}" \
            --file "${dockerfile}" \
            --push \
            .
    done
    
    log_success "Multi-platform build completed!"
}

# Clean up build artifacts
cleanup() {
    log_info "Cleaning up build artifacts..."
    
    # Remove dangling images
    docker image prune -f
    
    # Remove buildx cache if it gets too large
    docker buildx du --verbose
    
    log_success "Cleanup completed!"
}

# Show help
show_help() {
    cat << EOF
O-RAN Near-RT RIC Docker Build Script

Usage: $0 [OPTIONS] [COMMAND]

Commands:
    build           Build all Docker images (default)
    multiplatform   Build multi-platform images for AMD64 and ARM64
    push            Push images to registry
    clean           Clean up build artifacts
    help            Show this help message

Options:
    --registry REGISTRY     Set Docker registry (default: localhost:5000)
    --version VERSION       Set build version (default: latest)
    --skip-push            Skip pushing images to registry
    --skip-scan            Skip security scanning with trivy

Environment Variables:
    DOCKER_REGISTRY        Docker registry URL
    BUILD_VERSION          Version tag for images
    SKIP_PUSH             Set to skip pushing images
    SKIP_SCAN             Set to skip security scanning

Examples:
    # Build all images for local development
    $0 build

    # Build and push to production registry
    DOCKER_REGISTRY=registry.example.com BUILD_VERSION=v1.0.0 $0 build push

    # Build multi-platform images
    $0 multiplatform

    # Clean up after build
    $0 clean
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --skip-push)
            SKIP_PUSH=1
            shift
            ;;
        --skip-scan)
            SKIP_SCAN=1
            shift
            ;;
        build)
            COMMAND="build"
            shift
            ;;
        multiplatform)
            COMMAND="multiplatform"
            shift
            ;;
        push)
            COMMAND="push"
            shift
            ;;
        clean)
            COMMAND="clean"
            shift
            ;;
        help|--help|-h)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Default command
COMMAND=${COMMAND:-"build"}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed and running
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker is not running"
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -f "Dockerfile" ]; then
        log_error "This script must be run from the project root directory"
        exit 1
    fi
    
    # Check if git is available for commit info
    if ! command -v git &> /dev/null; then
        log_warning "Git not available, using 'unknown' for commit hash"
        GIT_COMMIT="unknown"
    fi
    
    # Check if trivy is available for security scanning
    if ! command -v trivy &> /dev/null && [ -z "$SKIP_SCAN" ]; then
        log_warning "Trivy not found, security scanning will be skipped"
        log_warning "Install trivy for security scanning: https://aquasecurity.github.io/trivy/"
    fi
    
    log_success "Prerequisites check passed"
}

# Main execution
main() {
    check_prerequisites
    
    case $COMMAND in
        build)
            build_all
            if [ -z "$SKIP_PUSH" ]; then
                push_images
            fi
            ;;
        multiplatform)
            build_multiplatform
            ;;
        push)
            push_images
            ;;
        clean)
            cleanup
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
    
    log_success "Build process completed successfully!"
}

# Handle interrupts
trap 'log_error "Build interrupted by user"; exit 1' INT TERM

# Run main function
main "$@"