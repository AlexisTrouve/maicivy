#!/bin/bash

# ============================================
# maicivy - Build Docker Images Script
# ============================================
# Usage: ./scripts/build-images.sh [OPTIONS]
# Options:
#   --tag TAG      Tag for images (default: latest)
#   --push         Push images to registry after build
#   --no-cache     Build without using cache
#   --platform     Target platform (default: linux/amd64)
# ============================================

set -e  # Exit on error
set -u  # Exit on undefined variable

# ============================================
# Configuration
# ============================================
REGISTRY="ghcr.io"
IMAGE_PREFIX="${GITHUB_REPOSITORY_OWNER:-yourusername}/maicivy"
TAG="latest"
PUSH=false
NO_CACHE=false
PLATFORM="linux/amd64"
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# Functions
# ============================================

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

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running!"
        exit 1
    fi

    # Check if buildx is available
    if ! docker buildx version > /dev/null 2>&1; then
        log_warning "Docker buildx not available, using standard build"
    fi

    log_success "Prerequisites check passed"
}

build_backend_image() {
    log_info "Building backend image..."

    local image_name="${REGISTRY}/${IMAGE_PREFIX}-backend:${TAG}"
    local cache_args=""

    if [ "$NO_CACHE" = false ]; then
        cache_args="--cache-from type=registry,ref=${image_name}"
    else
        cache_args="--no-cache"
    fi

    docker buildx build \
        --platform "$PLATFORM" \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VCS_REF="$VCS_REF" \
        $cache_args \
        -t "$image_name" \
        -f backend/Dockerfile \
        backend/

    log_success "Backend image built: $image_name"
}

build_frontend_image() {
    log_info "Building frontend image..."

    local image_name="${REGISTRY}/${IMAGE_PREFIX}-frontend:${TAG}"
    local cache_args=""

    if [ "$NO_CACHE" = false ]; then
        cache_args="--cache-from type=registry,ref=${image_name}"
    else
        cache_args="--no-cache"
    fi

    docker buildx build \
        --platform "$PLATFORM" \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VCS_REF="$VCS_REF" \
        $cache_args \
        -t "$image_name" \
        -f frontend/Dockerfile \
        frontend/

    log_success "Frontend image built: $image_name"
}

build_nginx_image() {
    log_info "Building nginx image..."

    local image_name="${REGISTRY}/${IMAGE_PREFIX}-nginx:${TAG}"
    local cache_args=""

    if [ "$NO_CACHE" = false ]; then
        cache_args="--cache-from type=registry,ref=${image_name}"
    else
        cache_args="--no-cache"
    fi

    docker buildx build \
        --platform "$PLATFORM" \
        $cache_args \
        -t "$image_name" \
        -f docker/nginx/Dockerfile \
        docker/nginx/

    log_success "Nginx image built: $image_name"
}

push_images() {
    log_info "Pushing images to registry..."

    docker push "${REGISTRY}/${IMAGE_PREFIX}-backend:${TAG}"
    log_success "Backend image pushed"

    docker push "${REGISTRY}/${IMAGE_PREFIX}-frontend:${TAG}"
    log_success "Frontend image pushed"

    docker push "${REGISTRY}/${IMAGE_PREFIX}-nginx:${TAG}"
    log_success "Nginx image pushed"
}

show_images_info() {
    log_info "Built images:"
    echo ""
    docker images | grep "${IMAGE_PREFIX}" | grep "${TAG}"
    echo ""
}

# ============================================
# Parse Arguments
# ============================================

parse_args() {
    while [ $# -gt 0 ]; do
        case $1 in
            --tag)
                shift
                TAG=$1
                ;;
            --push)
                PUSH=true
                ;;
            --no-cache)
                NO_CACHE=true
                ;;
            --platform)
                shift
                PLATFORM=$1
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --tag TAG      Tag for images (default: latest)"
                echo "  --push         Push images to registry after build"
                echo "  --no-cache     Build without using cache"
                echo "  --platform     Target platform (default: linux/amd64)"
                echo "  --help         Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
        shift
    done
}

# ============================================
# Main Build Flow
# ============================================

main() {
    parse_args "$@"

    log_info "=== maicivy Docker Image Builder ==="
    echo ""
    log_info "Registry: $REGISTRY"
    log_info "Image prefix: $IMAGE_PREFIX"
    log_info "Tag: $TAG"
    log_info "Platform: $PLATFORM"
    log_info "Build date: $BUILD_DATE"
    log_info "VCS ref: $VCS_REF"
    echo ""

    check_prerequisites

    # Build images
    build_backend_image
    build_frontend_image
    build_nginx_image

    echo ""
    show_images_info

    # Push if requested
    if [ "$PUSH" = true ]; then
        echo ""
        push_images
    fi

    echo ""
    log_success "✅ All images built successfully!"

    if [ "$PUSH" = false ]; then
        echo ""
        log_info "To push images to registry, run:"
        log_info "  $0 --tag $TAG --push"
    fi
}

# ============================================
# Execute
# ============================================
main "$@"
