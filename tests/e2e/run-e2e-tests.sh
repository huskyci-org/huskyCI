#!/bin/bash
#
# E2E Test Script for HuskyCI
# This script tests the complete flow of HuskyCI:
# 1. Health check
# 2. Token generation
# 3. Analysis submission
# 4. Analysis monitoring
# 5. Results verification

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_URL="http://localhost:8888"
API_USER="huskyCIUser"
API_PASS="huskyCIPassword"
TEST_REPO_URL="https://github.com/huskyci-org/huskyCI.git"
TEST_REPO_BRANCH="main"
MAX_WAIT_TIME=600  # 10 minutes max wait for analysis
POLL_INTERVAL=5    # Check every 5 seconds

# Base64 encode credentials for Basic Auth
AUTH_HEADER=$(echo -n "${API_USER}:${API_PASS}" | base64)

echo_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

echo_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test 1: Health Check
test_health_check() {
    echo_info "Test 1: Checking API health..."
    
    response=$(curl -s -w "\n%{http_code}" "${API_URL}/healthcheck")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" -eq 200 ] && [ "$body" = "WORKING" ]; then
        echo_info "✓ Health check passed"
        return 0
    else
        echo_error "✗ Health check failed. HTTP Code: $http_code, Body: $body"
        return 1
    fi
}

# Test 2: Version Check
test_version_check() {
    echo_info "Test 2: Checking API version..."
    
    response=$(curl -s -w "\n%{http_code}" "${API_URL}/version")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" -eq 200 ]; then
        echo_info "✓ Version check passed: $body"
        return 0
    else
        echo_error "✗ Version check failed. HTTP Code: $http_code"
        return 1
    fi
}

# Test 3: Token Generation
test_token_generation() {
    echo_info "Test 3: Generating access token..."
    
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Basic ${AUTH_HEADER}" \
        -d "{\"repositoryURL\": \"${TEST_REPO_URL}\"}" \
        "${API_URL}/api/1.0/token")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" -eq 201 ]; then
        # Extract token from JSON response
        TOKEN=$(echo "$body" | grep -o '"huskytoken":"[^"]*' | cut -d'"' -f4)
        if [ -n "$TOKEN" ]; then
            echo_info "✓ Token generation passed. Token: ${TOKEN:0:20}..."
            export HUSKY_TOKEN="$TOKEN"
            return 0
        else
            echo_error "✗ Token generation failed: Could not extract token from response"
            return 1
        fi
    else
        echo_error "✗ Token generation failed. HTTP Code: $http_code, Body: $body"
        return 1
    fi
}

# Test 4: Analysis Submission
test_analysis_submission() {
    echo_info "Test 4: Submitting analysis request..."
    
    if [ -z "$HUSKY_TOKEN" ]; then
        echo_error "✗ No token available for analysis submission"
        return 1
    fi
    
    # Use curl with -i to get headers, -s for silent, -w for status code
    full_response=$(curl -s -i -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Husky-Token: ${HUSKY_TOKEN}" \
        -d "{\"repositoryURL\": \"${TEST_REPO_URL}\", \"repositoryBranch\": \"${TEST_REPO_BRANCH}\"}" \
        "${API_URL}/analysis")
    
    http_code=$(echo "$full_response" | tail -n1)
    response_body=$(echo "$full_response" | sed '$d' | sed -n '/^$/,$p' | tail -n +2)
    response_headers=$(echo "$full_response" | sed '$d' | sed -n '1,/^$/p')
    
    if [ "$http_code" -eq 201 ]; then
        # Extract RID from X-Request-Id header (Echo framework sets this)
        RID=$(echo "$response_headers" | grep -i "X-Request-Id" | head -n1 | sed 's/.*X-Request-Id: *//' | tr -d '\r\n' | xargs)
        
        if [ -z "$RID" ]; then
            # Fallback: try to extract from response body if available
            RID=$(echo "$response_body" | grep -o '"RID":"[^"]*' | cut -d'"' -f4 || echo "$response_body" | grep -o '"rid":"[^"]*' | cut -d'"' -f4)
        fi
        
        if [ -n "$RID" ]; then
            echo_info "✓ Analysis submission passed. RID: $RID"
            export ANALYSIS_RID="$RID"
            return 0
        else
            echo_error "✗ Analysis submission failed: Could not extract RID from headers or body"
            echo_error "Response headers: $response_headers"
            echo_error "Response body: $response_body"
            return 1
        fi
    elif [ "$http_code" -eq 409 ]; then
        # Analysis already running - this is acceptable, try to get the existing RID
        echo_warn "Analysis already running for this repository/branch"
        # We can't easily get the RID from a 409 response, so we'll skip this test
        echo_warn "Skipping analysis submission test (analysis already in progress)"
        return 0
    else
        echo_error "✗ Analysis submission failed. HTTP Code: $http_code"
        echo_error "Response: $response_body"
        return 1
    fi
}

# Test 5: Analysis Monitoring
test_analysis_monitoring() {
    echo_info "Test 5: Monitoring analysis progress..."
    
    if [ -z "$ANALYSIS_RID" ]; then
        echo_warn "No RID available for monitoring (analysis may already be running)"
        echo_warn "Skipping analysis monitoring test"
        return 0
    fi
    
    elapsed=0
    while [ $elapsed -lt $MAX_WAIT_TIME ]; do
        response=$(curl -s -w "\n%{http_code}" \
            -H "Husky-Token: ${HUSKY_TOKEN}" \
            "${API_URL}/analysis/${ANALYSIS_RID}")
        
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | head -n-1)
        
        if [ "$http_code" -eq 200 ]; then
            # Check if analysis is complete
            # Analysis is complete when status is "finished" or containers have results
            status=$(echo "$body" | grep -o '"status":"[^"]*' | cut -d'"' -f4 || echo "")
            
            if echo "$body" | grep -q '"containers"' && [ -n "$status" ]; then
                echo_info "✓ Analysis completed. Status: $status"
                echo_info "Analysis results:"
                echo "$body" | jq '.' 2>/dev/null || echo "$body"
                export ANALYSIS_RESULTS="$body"
                return 0
            fi
            
            echo_info "Analysis in progress... (${elapsed}s/${MAX_WAIT_TIME}s)"
        elif [ "$http_code" -eq 404 ]; then
            echo_warn "Analysis not found yet, waiting..."
        else
            echo_error "✗ Error monitoring analysis. HTTP Code: $http_code"
            return 1
        fi
        
        sleep $POLL_INTERVAL
        elapsed=$((elapsed + POLL_INTERVAL))
    done
    
    echo_error "✗ Analysis monitoring timeout after ${MAX_WAIT_TIME}s"
    return 1
}

# Test 6: Verify Analysis Results
test_verify_results() {
    echo_info "Test 6: Verifying analysis results..."
    
    if [ -z "$ANALYSIS_RESULTS" ]; then
        echo_error "✗ No analysis results to verify"
        return 1
    fi
    
    # Check if jq is available for JSON parsing
    if command -v jq &> /dev/null; then
        # Verify the structure of the results
        has_rid=$(echo "$ANALYSIS_RESULTS" | jq -r '.RID // empty')
        has_containers=$(echo "$ANALYSIS_RESULTS" | jq -r '.containers // empty')
        
        if [ -n "$has_rid" ] && [ -n "$has_containers" ]; then
            container_count=$(echo "$ANALYSIS_RESULTS" | jq '.containers | length')
            echo_info "✓ Analysis results verified. Found $container_count security test containers."
            return 0
        else
            echo_error "✗ Analysis results structure invalid"
            return 1
        fi
    else
        # Basic verification without jq
        if echo "$ANALYSIS_RESULTS" | grep -q "RID" && echo "$ANALYSIS_RESULTS" | grep -q "containers"; then
            echo_info "✓ Analysis results contain expected fields"
            return 0
        else
            echo_error "✗ Analysis results missing expected fields"
            return 1
        fi
    fi
}

# Main test execution
main() {
    echo_info "Starting HuskyCI E2E Tests..."
    echo_info "API URL: ${API_URL}"
    echo_info "Test Repository: ${TEST_REPO_URL} (branch: ${TEST_REPO_BRANCH})"
    echo ""
    
    failed_tests=0
    
    test_health_check || failed_tests=$((failed_tests + 1))
    test_version_check || failed_tests=$((failed_tests + 1))
    test_token_generation || failed_tests=$((failed_tests + 1))
    test_analysis_submission || failed_tests=$((failed_tests + 1))
    test_analysis_monitoring || failed_tests=$((failed_tests + 1))
    test_verify_results || failed_tests=$((failed_tests + 1))
    
    echo ""
    if [ $failed_tests -eq 0 ]; then
        echo_info "========================================="
        echo_info "All E2E tests passed! ✓"
        echo_info "========================================="
        exit 0
    else
        echo_error "========================================="
        echo_error "$failed_tests test(s) failed! ✗"
        echo_error "========================================="
        exit 1
    fi
}

# Run main function
main
