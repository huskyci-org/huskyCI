# Local API Deployment and CLI Testing Guide

This guide provides step-by-step instructions for deploying the HuskyCI API server locally and testing it with the CLI.

## Architecture Overview

The HuskyCI system consists of:
- **API Server** (`api/server.go`): Main API server that orchestrates security tests
- **MongoDB**: Database for storing analysis results
- **Container runner**: Abstraction used by the API to run security test and zip-extract containers. By default this is a **Docker runner** that talks to a Docker daemon (TCP or Unix socket).
- **CLI** (`cli/`): Command-line tool for interacting with the API

**Runner abstraction:** The API does not depend on a specific Docker host. It uses a `Runner` interface (`api/runner`). You can choose:

- **Docker runner** (default): The API talks to a Docker daemon (TCP or Unix socket). Config: `HUSKYCI_DOCKERAPI_ADDR`, `HUSKYCI_DOCKERAPI_PORT`, and TLS/cert settings. Set `HUSKYCI_RUNNER_TYPE=docker` or leave it unset.
- **Remote runner**: The API sends run requests to a separate **runner service** over HTTP. Useful when the API runs in a restricted environment and cannot reach a Docker socket. Set `HUSKYCI_RUNNER_TYPE=remote` and `HUSKYCI_RUNNER_ADDR` (e.g. `http://runner:8090`). The runner service must be built with `make build-runner` and run on a host (or container) that has access to the Docker daemon (e.g. mount `/var/run/docker.sock`). See **Remote runner service** below for how to run and deploy it.

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

3. **Load the extract image** (required for file:// zip upload analysis when using Docker Compose). Run once after compose is up:
   ```bash
   make load-extract-image
   ```
   This builds the zip-extraction image on the host and loads it into the Docker API container so the API can extract uploaded zips without container network access.

4. **Verify services are running**:
   ```bash
   # Check MongoDB (MongoDB 4.4 uses 'mongo' command, not 'mongosh')
   # Simple check - just verify the container is responding
   docker exec huskyCI_MongoDB mongo --eval "db.adminCommand('ping')" --quiet
   
   # Or with authentication (if needed):
   # docker exec huskyCI_MongoDB mongo huskyCIDB --eval "db.adminCommand('ping')" -u huskyCIUser -p huskyCIPassword --authenticationDatabase admin
   
   # Check API health
   curl http://localhost:8888/healthcheck
   ```

5. **Stop services** (when done):
   ```bash
   make compose-down
   ```

**Default Credentials** (from `deployments/docker-compose.yml`):
- API Username: `huskyCIUser`
- API Password: `huskyCIPassword`
- MongoDB: Same credentials
- API Endpoint: `http://localhost:8888`

**Docker-in-Docker (Docker Compose):** When the API runs inside a container, it must talk to the Docker API service (e.g. `dockerapi`) over TCP, not the host’s Unix socket. The API uses `HUSKYCI_DOCKERAPI_ADDR` (e.g. `dockerapi`) and `HUSKYCI_DOCKERAPI_PORT` (default `2376`) to build the Docker host URL. If the database has a Unix socket path stored, the API falls back to these environment variables so security test containers run in the correct Docker daemon.

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
   export HUSKYCI_DOCKERAPI_ADDR="localhost"       # or hostname of Docker API (use "dockerapi" when API runs in Docker Compose)
   export HUSKYCI_DOCKERAPI_PORT="2376"            # optional; default 2376
   export HUSKYCI_DOCKERAPI_TLS_VERIFY="0"
   export HUSKYCI_DOCKERAPI_CERT_PATH="/path/to/certs"  # if using TLS
   # Optional: use remote runner instead of Docker
   # export HUSKYCI_RUNNER_TYPE="remote"
   # export HUSKYCI_RUNNER_ADDR="http://localhost:8090"
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

   On macOS, if you see "could not be run by the operating system" or an interpreter-directive error, the binary may have quarantine attributes; clear them with: `xattr -c api/huskyci-api-bin` (or run `make build-api` again, which clears them on Darwin).

   Or run directly with Go:
   ```bash
   cd api
   go run server.go
   ```

### Remote runner service (optional)

When `HUSKYCI_RUNNER_TYPE=remote`, the API does not talk to Docker directly; it sends container run requests to a **runner service** over HTTP. The runner service runs on a host (or container) that has access to the Docker daemon and executes the same run/health contract.

**Environment variables (API):**

| Variable | Description | Default |
|----------|-------------|---------|
| `HUSKYCI_RUNNER_TYPE` | `docker` or `remote` | `docker` |
| `HUSKYCI_RUNNER_ADDR` | Base URL of the runner service (e.g. `http://runner:8090`) | Required when type is `remote` |

**Build and run the runner service:**

1. Build the binary:
   ```bash
   make build-runner
   ```

2. Run the runner (with Docker socket access):
   ```bash
   # On the host (Docker socket default)
   ./cmd/runner/huskyci-runner-bin

   # Or set port (default 8090)
   RUNNER_PORT=8090 ./cmd/runner/huskyci-runner-bin
   ```

3. In a container (e.g. for Docker Compose), mount the Docker socket and expose the port:
   ```yaml
   runner:
     image: ...  # or build from cmd/runner
     volumes:
       - /var/run/docker.sock:/var/run/docker.sock
     ports:
       - "8090:8090"
   ```
   Then point the API at it: `HUSKYCI_RUNNER_TYPE=remote` and `HUSKYCI_RUNNER_ADDR=http://runner:8090` (or `http://localhost:8090` if the API runs on the host).

**Endpoints:**

- `GET /health` — Returns 200 if the Docker daemon is reachable.
- `POST /run` — Runs a container. Body: JSON `{ "image", "cmd", "volumePath", "timeoutSeconds", "readWriteVolume" }`, or multipart with part `request` (same JSON) and optional part `stdin` (raw bytes). Response: JSON `{ "stdout", "stderr", "exitCode", "error?" }`.

## CLI Configuration and Testing

### Configure CLI

**Option A: Using CLI setup command** (interactive):
```bash
make build-cli
./cli/huskyci-cli-bin setup
```

**Installing the CLI on macOS:** If you see "Failed to execute process" or "interpreter directive (#!) is broken?" when running the CLI, the binary in use is likely a **Linux** build (ELF). Build a native macOS binary and install it, e.g.:

```bash
make build-cli
cp cli/huskyci-cli-bin ~/.local/bin/huskyci
# If macOS blocks execution, clear quarantine:
xattr -c ~/.local/bin/huskyci
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

**Test 1: Analyze a local directory** (file upload / file:// analysis):
```bash
./cli/huskyci-cli-bin run ./path/to/your/project
```

The CLI compresses the directory, uploads it to the API, and starts an analysis with a `file://` URL. The API extracts the zip in the API container and in the Docker API container so security test containers can read the code. Gitauthors is skipped for file:// (no git history). Enry language detection can be done locally by the CLI and sent in the request to avoid running Enry in Docker.

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

### Local (file://) vs remote – test matrix

For **file://** (zip upload), the clone line is replaced with a copy from `/workspace`; the container sees the extracted dir at `/workspace`. **Trivy** is special-cased to run `trivy fs /workspace/` and always emit valid JSON (or `{}`). For **remote** (git URL), the flow is unchanged: clone then run the tool on `./code`.

After running one file:// analysis (`./cli/huskyci-cli-bin run ./path/to/project`), record pass/fail per test from API logs or the analysis result. Example below from a **Go-only** file:// run (Enry from CLI; only Generic + Go language tests run):

| Test       | file:// (zip) | Remote (git) | Notes |
|------------|----------------|-------------|-------|
| Enry       | N/A (CLI)      | pass        | For file://, CLI provides Enry output; API skips Enry container. |
| Gitauthors | skipped        | pass        | Skipped for file:// (no git history in extracted dir). |
| Gitleaks   | pass           | —           | Clone line replaced with copy from `/workspace`. |
| Trivy      | pass           | —           | Special-cased: runs on `/workspace/`, emits JSON or `{}`. |
| Gosec      | pass           | —           | Clone → copy, then same cmd on `./code`. |
| Bandit     | —              | —           | Run with a Python project to verify. |
| Others     | —              | —           | Per language (Brakeman/Ruby, npmaudit/JS, etc.). |

If a test fails only for file://, fix its command in `api/util/util.go` (same pattern: replace clone with copy, then existing cmd) or document the exception.

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
- Check Docker API is accessible (with Docker Compose, the API uses `HUSKYCI_DOCKERAPI_ADDR=dockerapi` to reach the Docker-in-Docker service).
- Verify security test containers are available (images are pulled on first use).
- Check API logs for detailed error messages.

**"lookup /var/run/docker.sock: no such host" or "unable to parse docker host":**
- With Docker Compose, the API must use the `dockerapi` service, not the host’s Unix socket. Ensure `HUSKYCI_DOCKERAPI_ADDR` is set (e.g. `dockerapi`) in the API container. The API prefers this over the database value when it’s a TCP hostname.
- If you run the API outside Docker, set `HUSKYCI_DOCKERAPI_ADDR` to your Docker host (e.g. `localhost` or a TCP address) and optionally `HUSKYCI_DOCKERAPI_PORT` (default `2376`).

**"unexpected end of JSON input" or "security tool produced no valid JSON":**
- For **file://** (zip upload) analyses, the zip is extracted in both the API container and the Docker API container so security tests can read the code. If extraction in the Docker API fails (e.g. zip not visible yet), tools may run on an empty workspace and produce no valid JSON. The API retries extraction for up to ~15 seconds. For very large uploads, ensure the shared volume (e.g. `/tmp/huskyci-zips-host`) is the same for API and Docker API.
- **Gitauthors** for file:// URLs: the API skips gitauthors when the repository URL is file:// (no git history). If gitauthors still runs and returns empty/invalid JSON, it is treated as “no authors” and does not fail the analysis.

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

# 2. Wait for services (30-60 seconds), then load extract image for zip analysis
curl http://localhost:8888/healthcheck
make load-extract-image

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

# 6. Run analysis (file:// zip upload requires load-extract-image from step 2)
./cli/huskyci-cli-bin run .

# 7. Cleanup
make compose-down
```
