#!/bin/bash
# =============================================================================
# Integration Test Setup Script
# =============================================================================
# Common setup steps for integration tests (proxy, tools, etc.)
# Used by both sharded and non-sharded test runs.
#
# Usage: source ./integration_setup.sh
#        OR
#        ./integration_setup.sh [--proxy-only|--tools-only|--all]
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
PROXY_PORT="${PROXY_PORT:-3128}"
PROXY_CONTAINER_NAME="${PROXY_CONTAINER_NAME:-squid}"
SCARESOLVER_PATH="${SCARESOLVER_PATH:-/tmp}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[SETUP]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SETUP]${NC} $1"
}

log_error() {
    echo -e "${RED}[SETUP]${NC} $1"
}

# =============================================================================
# Setup Squid Proxy
# =============================================================================
setup_proxy() {
    log_info "Setting up Squid proxy..."

    # Check if already running
    if docker ps -q -f name="$PROXY_CONTAINER_NAME" | grep -q .; then
        log_info "Squid proxy already running"
        return 0
    fi

    # Remove if exists but stopped
    docker rm -f "$PROXY_CONTAINER_NAME" 2>/dev/null || true

    # Start proxy
    docker run \
        --name "$PROXY_CONTAINER_NAME" \
        -d \
        -p "${PROXY_PORT}:3128" \
        -v "${SCRIPT_DIR}/squid/squid.conf:/etc/squid/squid.conf" \
        -v "${SCRIPT_DIR}/squid/passwords:/etc/squid/passwords" \
        ubuntu/squid:5.2-22.04_beta

    # Wait for proxy to be ready
    log_info "Waiting for proxy to be ready..."
    for i in {1..30}; do
        if docker exec "$PROXY_CONTAINER_NAME" squid -k check 2>/dev/null; then
            log_success "Squid proxy is ready"
            return 0
        fi
        sleep 1
    done

    log_error "Proxy failed to start"
    return 1
}

# =============================================================================
# Stop Squid Proxy
# =============================================================================
stop_proxy() {
    log_info "Stopping Squid proxy..."
    docker rm -f "$PROXY_CONTAINER_NAME" 2>/dev/null || true
    log_success "Squid proxy stopped"
}

# =============================================================================
# Download ScaResolver
# =============================================================================
setup_scaresolver() {
    log_info "Setting up ScaResolver..."

    if [ -f "${SCARESOLVER_PATH}/ScaResolver" ]; then
        log_info "ScaResolver already exists"
        return 0
    fi

    log_info "Downloading ScaResolver..."
    wget -q https://sca-downloads.s3.amazonaws.com/cli/latest/ScaResolver-linux64.tar.gz -O /tmp/ScaResolver-linux64.tar.gz
    tar -xzf /tmp/ScaResolver-linux64.tar.gz -C "$SCARESOLVER_PATH"
    rm -f /tmp/ScaResolver-linux64.tar.gz

    log_success "ScaResolver installed to ${SCARESOLVER_PATH}"
}

# =============================================================================
# Install Go Tools
# =============================================================================
setup_go_tools() {
    log_info "Setting up Go tools..."

    if ! command -v gocovmerge &> /dev/null; then
        log_info "Installing gocovmerge..."
        go install github.com/wadey/gocovmerge@latest
    fi

    log_success "Go tools ready"
}

# =============================================================================
# Full Setup
# =============================================================================
setup_all() {
    setup_proxy
    setup_scaresolver
    setup_go_tools
}

# =============================================================================
# Main
# =============================================================================
main() {
    case "${1:-all}" in
        --proxy-only)
            setup_proxy
            ;;
        --tools-only)
            setup_scaresolver
            setup_go_tools
            ;;
        --stop-proxy)
            stop_proxy
            ;;
        --all|*)
            setup_all
            ;;
    esac
}

# Only run main if script is executed directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
