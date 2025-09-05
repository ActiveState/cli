# Quick Start Guide

## Prerequisites
- Docker and Docker Compose installed
- Built binaries in the `../build/` directory

## Quick Test

1. **Verify setup:**
   ```bash
   cd ksa-test
   ./verify-setup.sh
   ```

2. **Run installer with default config:**
   ```bash
   docker-compose exec state-installer-test ./test-installer.sh
   ```

3. **Run installer with custom config:**
   ```bash
   docker-compose exec -e CONFIG_FLAGS="--config-set analytics.enabled=false --config-set output.level=debug" state-installer-test ./test-installer.sh
   ```

4. **Access interactive shell:**
   ```bash
   docker-compose exec state-installer-test bash
   ```

## Using the Convenience Script

```bash
# Build the image
./run-test.sh build

# Run with default settings
./run-test.sh run

# Test with custom config flags
./run-test.sh test "--config-set analytics.enabled=false --config-set output.level=debug"

# Open shell
./run-test.sh shell

# Monitor network
./run-test.sh monitor

# Clean up
./run-test.sh clean
```

## What This Tests

This Docker environment:
- ✅ Creates the proper payload structure (`state-installer` in root, other binaries in `bin/`)
- ✅ Includes the required install marker file
- ✅ Runs the installer with configurable flags
- ✅ Provides network monitoring capabilities
- ✅ Gives you an interactive shell to inspect results
- ✅ Mimics the behavior of `install.sh` but in a controlled environment

## Example Config Flags to Test

```bash
# Disable analytics
--config-set analytics.enabled=false

# Set debug output
--config-set output.level=debug

# Set custom settings
--config-set your.custom.key=value

# Multiple settings
--config-set analytics.enabled=false --config-set output.level=info --config-set another.setting=test
```
