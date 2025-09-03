# State Tool Installer Docker Test Environment

This Docker setup allows you to test the state-installer with configuration flags in a controlled environment that mimics the behavior of `install.sh`.

## Quick Start

1. **Build and run the test environment:**
   ```bash
   cd docker-test
   docker-compose up --build -d
   ```

2. **Access the container:**
   ```bash
   docker-compose exec state-installer-test bash
   ```

3. **Run the installer with custom config flags:**
   ```bash
   # Set your custom config flags
   export CONFIG_FLAGS="--config-set analytics.enabled=false --config-set output.level=debug --config-set your.key=value"
   
   # Run the test installer
   ./test-installer.sh
   ```

## Quick Reference for Custom Endpoints

**Test KSA endpoints with network monitoring:**
```bash
# 1. Clean install
docker-compose exec state-installer-test rm -rf /opt/state-install

# 2. Install with KSA endpoints
docker-compose exec -e CONFIG_FLAGS="--config-set api.host=ksa.activestate.build --config-set report.analytics.endpoint=https://ksa.state-tool.s3.amazonaws.com/pixel --config-set update.endpoint=https://ksa.activestate.com/sv/state-update/api/v1 --config-set notifications.endpoint=https://ksa.state-tool.s3.amazonaws.com/messages.json --config-set analytics.enabled=false --config-set output.level=debug" state-installer-test ./test-installer.sh

# 3. Monitor network traffic (in new terminal)
docker-compose exec state-installer-test bash -c "tcpdump -i any -n host ksa.activestate.build or host ksa.state-tool.s3.amazonaws.com"

# 4. Trigger network activity (in another terminal)
docker-compose exec state-installer-test /opt/state-install/bin/state auth --prompt
```

## Configuration

### Environment Variables

You can customize the test environment using these environment variables:

- `CONFIG_FLAGS`: Configuration flags to pass to the installer (default: `--config-set analytics.enabled=false --config-set output.level=debug`)
- `INSTALL_PATH`: Installation directory (default: `/opt/state-install`)
- `INSTALLER_FLAGS`: Additional installer flags (default: `--force --non-interactive`)

### Example Usage

```bash
# Test with custom configuration
docker-compose run --rm -e CONFIG_FLAGS="--config-set analytics.enabled=true --config-set output.level=info" state-installer-test ./test-installer.sh

# Test with different installation path
docker-compose run --rm -e INSTALL_PATH="/tmp/state-test" state-installer-test ./test-installer.sh
```

## Network Monitoring

The environment includes network monitoring capabilities:

1. **Start network monitoring:**
   ```bash
   docker-compose exec state-installer-test ./monitor-network.sh
   ```

2. **Monitor with external tool:**
   ```bash
   # Start the monitoring service
   docker-compose --profile monitoring up -d
   
   # Access the network monitor
   docker-compose exec network-monitor bash
   ```

## Available Scripts

### `test-installer.sh`
- Runs the state-installer with configurable flags
- Shows installation results and configuration
- Provides an interactive shell for inspection

### `monitor-network.sh`
- Monitors network connections and processes
- Captures network traffic (if tcpdump is available)
- Tests connectivity to State Tool endpoints
- Provides DNS monitoring capabilities

## File Structure

```
docker-test/
├── Dockerfile              # Main Docker image definition
├── docker-compose.yml      # Docker Compose configuration
├── test-installer.sh       # Test script for running installer
├── monitor-network.sh      # Network monitoring script
├── run-test.sh            # Convenience script for common operations
├── README.md              # This file
└── test-results/          # Volume mount for persistent results
```

## Payload Structure

The Docker image creates the proper payload structure that the state-installer expects:

```
/opt/state-test/
├── state-installer         # Main installer executable
├── .state_install_root     # Install marker file
└── bin/                    # Directory containing all other binaries
    ├── state
    ├── state-exec
    ├── state-svc
    ├── state-mcp
    └── state-remote-installer
```

This structure matches what the installer expects when it copies files from the payload directory to the installation directory.

## Testing Different Scenarios

### Test Configuration Flags
```bash
# Test analytics configuration
export CONFIG_FLAGS="--config-set analytics.enabled=false"
./test-installer.sh

# Test output level configuration
export CONFIG_FLAGS="--config-set output.level=debug"
./test-installer.sh

# Test multiple configuration options
export CONFIG_FLAGS="--config-set analytics.enabled=false --config-set output.level=info --config-set your.custom.setting=value"
./test-installer.sh
```

### Test Custom API Endpoints and Network Monitoring

This is particularly useful for testing custom API endpoints (like KSA environments) and verifying that the correct endpoints are being used.

#### 1. Clean Previous Installation
```bash
docker-compose exec state-installer-test rm -rf /opt/state-install
```

#### 2. Install with Custom Endpoints
```bash
# Example: KSA environment endpoints
docker-compose exec -e CONFIG_FLAGS="--config-set api.host=ksa.activestate.build --config-set report.analytics.endpoint=https://ksa.state-tool.s3.amazonaws.com/pixel --config-set update.endpoint=https://ksa.activestate.com/sv/state-update/api/v1 --config-set notifications.endpoint=https://ksa.state-tool.s3.amazonaws.com/messages.json --config-set analytics.enabled=false --config-set output.level=debug" state-installer-test ./test-installer.sh
```

#### 3. Verify Configuration Applied
```bash
# Check API host
docker-compose exec state-installer-test /opt/state-install/bin/state config get api.host

# Check analytics endpoint
docker-compose exec state-installer-test /opt/state-install/bin/state config get report.analytics.endpoint

# Check update endpoint
docker-compose exec state-installer-test /opt/state-install/bin/state config get update.endpoint
```

#### 4. Monitor Network Traffic
```bash
# Start network monitoring for custom endpoints
docker-compose exec state-installer-test bash -c "tcpdump -i any -n host ksa.activestate.build or host ksa.state-tool.s3.amazonaws.com"
```

#### 5. Trigger Network Activity
In a new terminal, run commands that will make network requests:
```bash
# This will trigger network requests to your custom endpoints
docker-compose exec state-installer-test /opt/state-install/bin/state auth --prompt

# Or try other commands that make API calls
docker-compose exec state-installer-test /opt/state-install/bin/state projects
```

#### 6. Verify Custom Endpoints Are Used
Look for these indicators in the debug output:
- `Using host override: ksa.activestate.build`
- `Using build planner at: https://ksa.activestate.build/sv/buildplanner/graphql`
- `secrets-api scheme=https host=ksa.activestate.build`

And in the network traffic, you should see connections to your custom endpoint IPs instead of the default ActiveState endpoints.

#### Example Custom Endpoint Configurations

**KSA Environment:**
```bash
CONFIG_FLAGS="--config-set api.host=ksa.activestate.build --config-set report.analytics.endpoint=https://ksa.state-tool.s3.amazonaws.com/pixel --config-set update.endpoint=https://ksa.activestate.com/sv/state-update/api/v1 --config-set notifications.endpoint=https://ksa.state-tool.s3.amazonaws.com/messages.json"
```

**Staging Environment:**
```bash
CONFIG_FLAGS="--config-set api.host=staging-api.activestate.com --config-set analytics.enabled=false"
```

**Local Development:**
```bash
CONFIG_FLAGS="--config-set api.host=localhost:8080 --config-set api.endpoint=http://localhost:8080/api --config-set analytics.enabled=false"
```

### Test Installation Paths
```bash
# Test different installation directory
export INSTALL_PATH="/tmp/state-custom"
./test-installer.sh
```

### Test Installer Flags
```bash
# Test with update flag
export INSTALLER_FLAGS="--update --force"
./test-installer.sh

# Test with activation
export INSTALLER_FLAGS="--force --activate your/project"
./test-installer.sh
```

## Debugging

### View Logs
```bash
# View container logs
docker-compose logs state-installer-test

# Follow logs in real-time
docker-compose logs -f state-installer-test
```

### Inspect Installation
```bash
# Check installation directory
ls -la /opt/state-install/

# Check configuration
/opt/state-install/bin/state config list

# Check version
/opt/state-install/bin/state --version
```

### Network Debugging
```bash
# Check network connections
netstat -tuln

# Monitor network traffic
tcpdump -i any host platform.activestate.com

# Check DNS resolution
nslookup platform.activestate.com
```

## Cleanup

```bash
# Stop and remove containers
docker-compose down

# Remove volumes (this will delete test results)
docker-compose down -v

# Remove images
docker-compose down --rmi all
```

## Troubleshooting

### Container Won't Start
- Check if the build directory exists and contains the required binaries
- Ensure Docker has sufficient resources allocated

### Installer Fails
- Check the logs: `docker-compose logs state-installer-test`
- Verify the binaries are executable: `ls -la /opt/state-test/state*`
- Check network connectivity: `./monitor-network.sh`

### Network Monitoring Issues
- Ensure the container has the necessary capabilities (`NET_ADMIN`, `NET_RAW`)
- Check if tcpdump is available: `which tcpdump`
- Verify network interface access: `ip addr show`
