# Local API Deployment and CLI Testing Guide

This guide provides step-by-step instructions for deploying the HuskyCI API server locally and testing it with the CLI.

## Architecture Overview

The HuskyCI system consists of:
- **API Server** (`api/server.go`): Main API server that orchestrates security tests
- **MongoDB**: Database for storing analysis results
- **Docker API**: Docker-in-Docker service for running security test containers
- **CLI** (`cli/`): Command-line tool for interacting with the API

## Deployment Options

### Option 1: Docker Compose (Recommended)

This is the easiest way to deploy the full environment with all dependencies.

**Steps:**

1. **Generate certificates** (required for Docker API communication):
   ```bash
   make create-certs
   ```

2. **Start the full environment**:
   ```bash
   make compose
   ```
   This starts:
   - MongoDB on port 27017
   - Docker API on port 2376
   - HuskyCI API on port 8888

3. **Verify services are running**:
   ```bash
   # Check MongoDB (MongoDB 4.4 uses 'mongo' command, not 'mongosh')
   # Simple check - just verify the container is responding
   docker exec huskyCI_MongoDB mongo --eval "db.adminCommand('ping')" --quiet
   
   # Or with authentication (if needed):
   # docker exec huskyCI_MongoDB mongo huskyCIDB --eval "db.adminCommand('ping')" -u huskyCIUser -p huskyCIPassword --authenticationDatabase admin
   
   # Check API health
   curl http://localhost:8888/healthcheck
   ```

4. **Stop services** (when done):
   ```bash
   make compose-down
   ```

**Default Credentials** (from `deployments/docker-compose.yml`):
- API Username: `huskyCIUser`
- API Password: `huskyCIPassword`
- MongoDB: Same credentials
- API Endpoint: `http://localhost:8888`

### Option 2: Direct Go Execution

For development/testing without Docker Compose, you can run the API server directly.

**Prerequisites:**
- MongoDB running locally or accessible
- Docker daemon accessible (for running security test containers)
- Go 1.24+ installed

**Steps:**

1. **Set required environment variables** (see `api/util/api/api.go` for full list):
   ```bash
   export HUSKYCI_DATABASE_DB_ADDR="localhost"
   export HUSKYCI_DATABASE_DB_NAME="huskyCIDB"
   export HUSKYCI_DATABASE_DB_USERNAME="huskyCIUser"
   export HUSKYCI_DATABASE_DB_PASSWORD="huskyCIPassword"
   export HUSKYCI_API_DEFAULT_USERNAME="huskyCIUser"
   export HUSKYCI_API_DEFAULT_PASSWORD="huskyCIPassword"
   export HUSKYCI_API_ALLOW_ORIGIN_CORS="*"
   export HUSKYCI_INFRASTRUCTURE_USE="docker"
   export HUSKYCI_DOCKERAPI_ADDR="localhost:2376"  # or unix:///var/run/docker.sock
   export HUSKYCI_DOCKERAPI_TLS_VERIFY="0"
   export HUSKYCI_DOCKERAPI_CERT_PATH="/path/to/certs"  # if using TLS
   ```

2. **Build the API server**:
   ```bash
   make build-api
   ```

3. **Run the API server**:
   ```bash
   cd api
   ./huskyci-api-bin
   ```

   Or run directly with Go:
   ```bash
   cd api
   go run server.go
   ```

## CLI Configuration and Testing

### Configure CLI

**Option A: Using CLI setup command** (interactive):
```bash
make build-cli
./cli/huskyci-cli-bin setup
```

**Option B: Using config file** (`~/.huskyci/config.yaml`):
Create `~/.huskyci/config.yaml` based on `examples/cli-config.yaml.example`:
```yaml
targets:
  local:
    current: true
    endpoint: "http://localhost:8888"
    token-storage: "keychain"
```

**Option C: Using environment variables**:
```bash
export HUSKYCI_CLIENT_API_ADDR="http://localhost:8888"
export HUSKYCI_CLI_TOKEN="your-token-here"
```

**Note**: The API server can also read tokens from environment variables if the `Husky-Token` header is not provided:
- `HUSKYCI_CLI_TOKEN` - Used when requests come from the CLI (detected via User-Agent header)
- `HUSKYCI_CLIENT_TOKEN` - Used when requests come from non-human clients (detected via User-Agent header)

The API will first check the `Husky-Token` header, and if empty, it will check the appropriate environment variable based on the request source.

### Generate API Token

Before running CLI tests, you need to generate a token:

```bash
# Using Basic Auth (username:password from docker-compose.yml)
curl -X POST \
  -H "Content-Type: application/json" \
  -u huskyCIUser:huskyCIPassword \
  -d '{"repositoryURL": "https://github.com/huskyci-org/huskyCI.git"}' \
  http://localhost:8888/api/1.0/token
```

Extract the token from the response and set it:
```bash
export HUSKYCI_CLI_TOKEN="your-token-here"
```

### Test CLI Connection

The CLI includes a built-in connection test command:

```bash
# Build CLI if not already built
make build-cli

# Test connection to current target
./cli/huskyci-cli-bin test-connection

# Test connection to specific endpoint
./cli/huskyci-cli-bin test-connection --endpoint http://localhost:8888

# Test with verbose output
./cli/huskyci-cli-bin test-connection --verbose
```

This command tests:
1. Basic connectivity
2. Health check endpoint (`/healthcheck`)
3. Version endpoint (`/version`)
4. Authentication (if token configured)

### Run CLI Analysis Tests

**Test 1: Analyze a local directory**:
```bash
./cli/huskyci-cli-bin run ./path/to/your/project
```

**Test 2: Run E2E tests** (full integration test):
```bash
# Ensure API is running
make compose

# Run E2E tests
make test-e2e
# or directly:
./tests/e2e/run-e2e-tests.sh
```

The E2E test script (`tests/e2e/run-e2e-tests.sh`) performs:
1. Health check
2. Version check
3. Token generation
4. Analysis submission
5. Analysis monitoring
6. Results verification

## Key Files Reference

- **API Server**: [`api/server.go`](api/server.go) - Main entry point
- **Docker Compose**: [`deployments/docker-compose.yml`](deployments/docker-compose.yml) - Full environment setup
- **CLI Config Example**: [`examples/cli-config.yaml.example`](examples/cli-config.yaml.example) - CLI configuration template
- **CLI Test Connection**: [`cli/cmd/testConnection.go`](cli/cmd/testConnection.go) - Connection testing implementation
- **E2E Test Script**: [`tests/e2e/run-e2e-tests.sh`](tests/e2e/run-e2e-tests.sh) - Full integration tests
- **Makefile**: [`Makefile`](Makefile) - Common commands

## Troubleshooting

**API won't start:**
- Check all required environment variables are set
- Verify MongoDB is accessible
- Check Docker daemon is running (if using docker infrastructure)

**CLI can't connect:**
- Verify API is running: `curl http://localhost:8888/healthcheck`
- Check endpoint URL in config matches API address
- Ensure token is valid (regenerate if needed)

**Analysis fails:**
- Check Docker API is accessible
- Verify security test containers are available
- Check API logs for detailed error messages

**CLI "zip file not found" error when running `huskyci run ./`:**
- This error occurs when the zip file upload succeeds but the API cannot find it when starting analysis
- Possible causes:
  1. **API server permissions**: The API server may not have write permissions to `/tmp/huskyci-zips`
  2. **Docker volume issue**: If running in Docker, ensure the `/tmp` directory is writable
  3. **RID mismatch**: There may be a mismatch between the upload RID and analysis RID
- Troubleshooting steps:
  ```bash
  # Check API logs
  docker logs huskyCI_API
  
  # Verify API can write to /tmp (if running in Docker)
  docker exec huskyCI_API ls -la /tmp/huskyci-zips
  
  # Run CLI with verbose mode for detailed logs
  huskyci run ./ --verbose
  ```
- If the issue persists, try:
  1. Ensure the API container has proper volume mounts or tmpfs access
  2. Check API server logs for file system errors
  3. Verify the upload endpoint returns success with the correct RID

## Quick Start Summary

```bash
# 1. Setup environment
make create-certs
make compose

# 2. Wait for services (30-60 seconds)
curl http://localhost:8888/healthcheck

# 3. Generate token
TOKEN=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -u huskyCIUser:huskyCIPassword \
  -d '{"repositoryURL": "https://github.com/huskyci-org/huskyCI.git"}' \
  http://localhost:8888/api/1.0/token | grep -o '"huskytoken":"[^"]*' | cut -d'"' -f4)

# 4. Configure CLI
export HUSKYCI_CLI_TOKEN="$TOKEN"
make build-cli

# 5. Test connection
./cli/huskyci-cli-bin test-connection --endpoint http://localhost:8888

# 6. Run analysis
./cli/huskyci-cli-bin run .

# 7. Cleanup
make compose-down
```
