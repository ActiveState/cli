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
echo "=== Blocking Default ActiveState Endpoints ==="
echo "Adding entries to /etc/hosts to block default ActiveState URLs..."
echo "127.0.0.1 platform.activestate.com" >> /etc/hosts
echo "127.0.0.1 state-tool.s3.amazonaws.com" >> /etc/hosts
echo "127.0.0.1 s3.ca-central-1.amazonaws.com" >> /etc/hosts
echo "127.0.0.1 www.activestate.com" >> /etc/hosts
echo "127.0.0.1 community.activestate.com" >> /etc/hosts
echo "127.0.0.1 github.com" >> /etc/hosts
echo "127.0.0.1 state-tool.activestate.com" >> /etc/hosts
echo "127.0.0.1 docs.activestate.com" >> /etc/hosts
echo "Blocked default endpoints successfully"

echo ""
echo "=== Running State Installer with Config Flags ==="

# Default config flags - KSA proxy endpoints
CONFIG_FLAGS=${CONFIG_FLAGS:-"--config-set api.host=ksa.activestate.build --config-set report.analytics.endpoint=https://ksa-s3-state-tool.activestate.build/pixel-ksa --config-set update.endpoint=https://ksa-s3-state-tool.activestate.build/update/state --config-set update.info.endpoint=https://ksa.activestate.build/sv/state-update/api/v1 --config-set notifications.endpoint=https://ksa-s3-state-tool.activestate.build/messages.json --config-set analytics.enabled=false --config-set output.level=debug"}

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
