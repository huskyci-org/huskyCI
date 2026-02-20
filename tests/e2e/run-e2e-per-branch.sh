#!/usr/bin/env bash
#
# Runs E2E tests on each feature branch and reports results.
# Requires: Docker daemon running, make, curl.
#
# Note: feat/zip-upload-analysis does not build standalone (depends on types/securitytest
# from main). It is validated when merged into feat/setup-wizard-command or main.
# Default branches: feat/multi-platform-docker-builds feat/setup-wizard-command
#
# Usage: ./tests/e2e/run-e2e-per-branch.sh [branch1 branch2 ...]

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

BRANCHES=("${@:-feat/multi-platform-docker-builds feat/setup-wizard-command}")

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

run_e2e_for_branch() {
    local branch="$1"
    echo -e "\n${YELLOW}========== E2E tests for branch: $branch ==========${NC}"
    git checkout "$branch" || { echo -e "${RED}Failed to checkout $branch${NC}"; return 1; }

    echo "Checking API build..."
    if ! make build-api >/dev/null 2>&1; then
        echo -e "${YELLOW}Branch $branch does not build API (e.g. zip-only branch). Skipping E2E.${NC}"
        echo "Validate this feature by merging into feat/setup-wizard-command and running E2E there."
        return 0
    fi

    echo "Creating certificates (if needed)..."
    make create-certs || true

    echo "Starting compose (API + MongoDB + Docker API)..."
    make compose

    echo "Waiting for API to be ready (up to 120s)..."
    local i=0
    while [ $i -lt 24 ]; do
        if curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/healthcheck 2>/dev/null | grep -q 200; then
            echo "API is ready."
            break
        fi
        sleep 5
        i=$((i + 1))
    done
    if [ $i -eq 24 ]; then
        echo -e "${RED}API did not become ready in time.${NC}"
        make compose-down 2>/dev/null || true
        return 1
    fi

    echo "Running E2E tests..."
    if make test-e2e; then
        echo -e "${GREEN}✓ E2E passed for $branch${NC}"
        make compose-down 2>/dev/null || true
        return 0
    else
        echo -e "${RED}✗ E2E failed for $branch${NC}"
        make compose-down 2>/dev/null || true
        return 1
    fi
}

echo "E2E tests per branch. Repo root: $REPO_ROOT"
echo "Branches: ${BRANCHES[*]}"

if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}Docker daemon is not running. Start Docker and re-run this script.${NC}"
    exit 1
fi

FAILED=()
for b in "${BRANCHES[@]}"; do
    if run_e2e_for_branch "$b"; then
        :
    else
        FAILED+=("$b")
    fi
done

echo -e "\n${YELLOW}========== Summary ==========${NC}"
if [ ${#FAILED[@]} -eq 0 ]; then
    echo -e "${GREEN}All branches passed E2E. Safe to merge to main.${NC}"
    exit 0
else
    echo -e "${RED}Failed branches: ${FAILED[*]}${NC}"
    exit 1
fi
