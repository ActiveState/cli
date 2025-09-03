#!/bin/bash

# Quick verification script to test the Docker setup

set -e

echo "=== Verifying Docker Test Setup ==="

# Check if we're in the right directory
if [ ! -f "Dockerfile" ]; then
    echo "Error: Please run this script from the docker-test directory"
    exit 1
fi

# Check if build directory exists and has required binaries
echo "Checking build directory..."
if [ ! -d "../build" ]; then
    echo "Error: ../build directory not found. Please build the project first."
    exit 1
fi

required_binaries=("state-installer" "state" "state-exec" "state-svc" "state-mcp" "state-remote-installer")
for binary in "${required_binaries[@]}"; do
    if [ ! -f "../build/$binary" ]; then
        echo "Error: Required binary ../build/$binary not found"
        exit 1
    fi
done

echo "✓ All required binaries found in build directory"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

echo "✓ Docker is running"

# Build the Docker image
echo "Building Docker image..."
if docker-compose build; then
    echo "✓ Docker image built successfully"
else
    echo "Error: Failed to build Docker image"
    exit 1
fi

# Test the container
echo "Testing container startup..."
if docker-compose up -d; then
    echo "✓ Container started successfully"
    
    # Wait a moment for container to be ready
    sleep 2
    
    # Check if container is running
    if docker-compose ps | grep -q "Up"; then
        echo "✓ Container is running"
        
        # Test the payload structure
        echo "Verifying payload structure..."
        if docker-compose exec state-installer-test test -f state-installer; then
            echo "✓ state-installer found in root"
        else
            echo "✗ state-installer not found in root"
        fi
        
        if docker-compose exec state-installer-test test -d bin; then
            echo "✓ bin directory exists"
        else
            echo "✗ bin directory not found"
        fi
        
        if docker-compose exec state-installer-test test -f .state_install_root; then
            echo "✓ install marker file exists"
        else
            echo "✗ install marker file not found"
        fi
        
        # List the structure
        echo ""
        echo "Payload structure in container:"
        docker-compose exec state-installer-test ls -la
        echo ""
        docker-compose exec state-installer-test ls -la bin/
        
    else
        echo "✗ Container is not running"
        docker-compose logs
        exit 1
    fi
else
    echo "Error: Failed to start container"
    exit 1
fi

echo ""
echo "=== Setup Verification Complete ==="
echo "You can now run tests with:"
echo "  docker-compose exec state-installer-test ./test-installer.sh"
echo "  docker-compose exec state-installer-test ./monitor-network.sh"
echo "  docker-compose exec state-installer-test bash"
