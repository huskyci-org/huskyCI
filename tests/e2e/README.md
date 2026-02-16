# HuskyCI E2E Tests

This directory contains end-to-end (E2E) tests for HuskyCI that verify the complete flow of the security analysis pipeline.

## Overview

The E2E tests verify:
1. **Health Check** - API is responding correctly
2. **Version Check** - API version endpoint works
3. **Token Generation** - Can create access tokens for repositories
4. **Analysis Submission** - Can submit analysis requests
5. **Analysis Monitoring** - Can monitor analysis progress and retrieve results
6. **Results Verification** - Analysis results are properly structured

## Running E2E Tests Locally

### Prerequisites

- Docker and Docker Compose installed
- `curl` command available
- `jq` (optional, for better JSON parsing)
- HuskyCI environment certificates generated

### Steps

1. **Generate certificates** (if not already done):
   ```bash
   make create-certs
   ```

2. **Start the HuskyCI environment**:
   ```bash
   make compose
   ```

3. **Wait for services to be ready** (usually takes 1-2 minutes):
   ```bash
   # Check MongoDB (MongoDB 4.4 uses 'mongo' command, not 'mongosh')
   # Simple check - just verify the container is responding
   docker exec huskyCI_MongoDB mongo --eval "db.adminCommand('ping')" --quiet
   
   # Check API
   curl http://localhost:8888/healthcheck
   ```

4. **Run the E2E tests**:
   ```bash
   ./tests/e2e/run-e2e-tests.sh
   ```

5. **Clean up** (optional):
   ```bash
   make compose-down
   ```

## Running E2E Tests in CI/CD

The E2E tests are automatically run in GitHub Actions on:
- Push to `main` branch
- Pull requests to `main` branch
- Manual workflow dispatch

See `.github/workflows/e2e-tests.yml` for the CI configuration.

## Test Configuration

You can customize the test behavior by modifying these variables in `run-e2e-tests.sh`:

- `API_URL`: HuskyCI API endpoint (default: `http://localhost:8888`)
- `API_USER`: API username (default: `huskyCIUser`)
- `API_PASS`: API password (default: `huskyCIPassword`)
- `TEST_REPO_URL`: Repository to test (default: `https://github.com/huskyci-org/huskyCI.git`)
- `TEST_REPO_BRANCH`: Branch to analyze (default: `main`)
- `MAX_WAIT_TIME`: Maximum time to wait for analysis (default: 600 seconds)
- `POLL_INTERVAL`: Interval between status checks (default: 5 seconds)

## Troubleshooting

### Tests fail with "Connection refused"

- Ensure Docker Compose services are running: `docker-compose -f deployments/docker-compose.yml ps`
- Check if the API is accessible: `curl http://localhost:8888/healthcheck`
- Verify MongoDB is healthy: `docker exec huskyCI_MongoDB mongo --eval "db.adminCommand('ping')" --quiet`

### Tests timeout waiting for analysis

- Security analysis can take time depending on the repository size
- Increase `MAX_WAIT_TIME` in the test script
- Check Docker API logs: `docker logs huskyCI_Docker_API`
- Check API logs: `docker logs huskyCI_API`

### Token generation fails

- Verify basic auth credentials are correct
- Check API logs for authentication errors
- Ensure MongoDB is properly initialized

## Adding New Tests

To add new E2E tests:

1. Create a new test function in `run-e2e-tests.sh` following the pattern:
   ```bash
   test_new_feature() {
       echo_info "Test X: Testing new feature..."
       # Your test logic here
       if [ condition ]; then
           echo_info "✓ Test passed"
           return 0
       else
           echo_error "✗ Test failed"
           return 1
       fi
   }
   ```

2. Call the test function in the `main()` function

3. Update this README with the new test description
