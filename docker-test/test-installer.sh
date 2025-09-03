#!/bin/bash

# Test script for state-installer with config flags
# This mimics what install.sh does but allows for custom configuration

set -e

echo "=== State Tool Installer Test Environment ==="
echo "Current directory: $(pwd)"
echo "Payload structure:"
echo "Root directory:"
ls -la
echo ""
echo "Bin directory:"
ls -la bin/
echo ""
echo "Install marker:"
cat .state_install_root 2>/dev/null || echo "No install marker found"

echo ""
echo "=== Running State Installer with Config Flags ==="

# Default config flags - you can modify these or pass them as environment variables
CONFIG_FLAGS=${CONFIG_FLAGS:-"--config-set analytics.enabled=false --config-set output.level=debug"}

# Installation path
INSTALL_PATH=${INSTALL_PATH:-"/opt/state-install"}

# Additional installer flags
INSTALLER_FLAGS=${INSTALLER_FLAGS:-"--force --non-interactive"}

echo "Config flags: $CONFIG_FLAGS"
echo "Install path: $INSTALL_PATH"
echo "Installer flags: $INSTALLER_FLAGS"
echo ""

# Run the installer
echo "Executing: ./state-installer $INSTALL_PATH $INSTALLER_FLAGS $CONFIG_FLAGS"
./state-installer "$INSTALL_PATH" $INSTALLER_FLAGS $CONFIG_FLAGS

echo ""
echo "=== Installation Complete ==="
echo "Installation directory contents:"
ls -la "$INSTALL_PATH" || echo "Installation directory not found"

echo ""
echo "=== State Tool Configuration ==="
if [ -f "$INSTALL_PATH/bin/state" ]; then
    echo "Running: $INSTALL_PATH/bin/state config list"
    "$INSTALL_PATH/bin/state" config list || echo "Failed to list config"
else
    echo "State tool not found in expected location"
fi

echo ""
echo "=== Log Files ==="
echo "Looking for log files..."
find /tmp -name "*state*" -type f 2>/dev/null || echo "No state log files found in /tmp"
find "$HOME" -name "*state*" -type f 2>/dev/null || echo "No state log files found in home directory"

echo ""
echo "=== Test Environment Ready ==="
echo "You can now inspect the installation and run additional tests."
echo "Available commands:"
echo "  - Check config: $INSTALL_PATH/bin/state config list"
echo "  - Run state tool: $INSTALL_PATH/bin/state --help"
echo "  - Monitor network: ./monitor-network.sh"
echo ""
echo "Starting interactive shell..."
exec /bin/bash
