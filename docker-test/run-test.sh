#!/bin/bash

# Convenience script to run the state-installer test environment
# This script provides easy commands for common testing scenarios

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  build                    Build the Docker image"
    echo "  run                      Run the test environment"
    echo "  test [CONFIG_FLAGS]      Run installer with specific config flags"
    echo "  shell                    Open shell in running container"
    echo "  logs                     Show container logs"
    echo "  monitor                  Start network monitoring"
    echo "  clean                    Clean up containers and images"
    echo "  help                     Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 build"
    echo "  $0 run"
    echo "  $0 test '--config-set analytics.enabled=false --config-set output.level=debug'"
    echo "  $0 shell"
    echo "  $0 monitor"
    echo ""
}

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker and try again."
        exit 1
    fi
}

# Function to build the Docker image
build_image() {
    print_status "Building Docker image..."
    docker-compose build
    print_success "Docker image built successfully"
}

# Function to run the test environment
run_environment() {
    print_status "Starting test environment..."
    docker-compose up -d
    print_success "Test environment started"
    print_status "Container name: state-installer-test"
    print_status "To access: docker-compose exec state-installer-test bash"
}

# Function to run installer with specific config flags
run_test() {
    local config_flags="$1"
    
    if [ -z "$config_flags" ]; then
        config_flags="--config-set analytics.enabled=false --config-set output.level=debug"
        print_warning "No config flags provided, using defaults: $config_flags"
    fi
    
    print_status "Running installer with config flags: $config_flags"
    
    # Run the installer in the container
    docker-compose exec -e CONFIG_FLAGS="$config_flags" state-installer-test ./test-installer.sh
}

# Function to open shell in container
open_shell() {
    print_status "Opening shell in container..."
    docker-compose exec state-installer-test bash
}

# Function to show logs
show_logs() {
    print_status "Showing container logs..."
    docker-compose logs -f state-installer-test
}

# Function to start network monitoring
start_monitoring() {
    print_status "Starting network monitoring..."
    docker-compose exec state-installer-test ./monitor-network.sh
}

# Function to clean up
cleanup() {
    print_status "Cleaning up containers and images..."
    docker-compose down --rmi all -v
    print_success "Cleanup completed"
}

# Main script logic
main() {
    check_docker
    
    case "${1:-help}" in
        build)
            build_image
            ;;
        run)
            run_environment
            ;;
        test)
            run_test "$2"
            ;;
        shell)
            open_shell
            ;;
        logs)
            show_logs
            ;;
        monitor)
            start_monitoring
            ;;
        clean)
            cleanup
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            echo ""
            show_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
