#!/bin/bash
# Integration test script for BIG SKIES Framework

set -e

echo "🧪 BIG SKIES Framework - Integration Tests"
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0

# Helper functions
pass_test() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

fail_test() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED++))
}

info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Test 1: Check if all binaries exist
test_binaries() {
    echo ""
    info "Testing compiled binaries..."
    
    local binaries=(
        "bin/message-coordinator"
        "bin/datastore-coordinator"
        "bin/application-coordinator"
        "bin/plugin-coordinator"
        "bin/uielement-coordinator"
    )
    
    for bin in "${binaries[@]}"; do
        if [ -f "$bin" ]; then
            pass_test "Binary exists: $bin"
        else
            fail_test "Binary missing: $bin"
        fi
    done
}

# Test 2: Check binary executability
test_binary_execution() {
    echo ""
    info "Testing binary execution..."
    
    local binaries=(
        "bin/message-coordinator"
        "bin/datastore-coordinator"
        "bin/application-coordinator"
        "bin/plugin-coordinator"
        "bin/uielement-coordinator"
    )
    
    for bin in "${binaries[@]}"; do
        if [ -f "$bin" ]; then
            if $bin --help >/dev/null 2>&1 || [ $? -eq 2 ]; then
                pass_test "Binary executable: $bin"
            else
                fail_test "Binary not executable: $bin"
            fi
        fi
    done
}

# Test 3: Verify Go module
test_go_module() {
    echo ""
    info "Testing Go module..."
    
    if go mod verify >/dev/null 2>&1; then
        pass_test "Go module verified"
    else
        fail_test "Go module verification failed"
    fi
    
    if go mod tidy -v >/dev/null 2>&1; then
        pass_test "Go module tidy succeeded"
    else
        fail_test "Go module tidy failed"
    fi
}

# Test 4: Run unit tests
test_unit_tests() {
    echo ""
    info "Running unit tests..."
    
    if go test ./... -v > /tmp/test-output.log 2>&1; then
        pass_test "Unit tests passed"
    else
        fail_test "Unit tests failed (see /tmp/test-output.log)"
    fi
}

# Test 5: Check Docker files
test_docker_files() {
    echo ""
    info "Testing Docker configuration..."
    
    if [ -f "deployments/docker/Dockerfile.coordinator" ]; then
        pass_test "Dockerfile exists"
    else
        fail_test "Dockerfile missing"
    fi
    
    if [ -f "deployments/docker-compose/docker-compose.yml" ]; then
        pass_test "docker-compose.yml exists"
    else
        fail_test "docker-compose.yml missing"
    fi
    
    if [ -f "deployments/docker-compose/.env" ]; then
        pass_test ".env file exists"
    else
        fail_test ".env file missing"
    fi
}

# Test 6: Validate docker-compose
test_docker_compose_validation() {
    echo ""
    info "Validating docker-compose configuration..."
    
    if command -v docker-compose &> /dev/null; then
        cd deployments/docker-compose
        if docker-compose config >/dev/null 2>&1; then
            pass_test "docker-compose config valid"
        else
            fail_test "docker-compose config invalid"
        fi
        cd ../..
    else
        info "docker-compose not installed, skipping validation"
    fi
}

# Test 7: Check documentation
test_documentation() {
    echo ""
    info "Testing documentation..."
    
    local docs=(
        "README.md"
        "WARP.md"
        "next_steps.txt"
        "deployments/README.md"
    )
    
    for doc in "${docs[@]}"; do
        if [ -f "$doc" ]; then
            pass_test "Documentation exists: $doc"
        else
            fail_test "Documentation missing: $doc"
        fi
    done
}

# Test 8: Check code formatting
test_code_formatting() {
    echo ""
    info "Testing code formatting..."
    
    # Check if any files need formatting
    UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor | wc -l)
    if [ "$UNFORMATTED" -eq 0 ]; then
        pass_test "All Go code is formatted"
    else
        fail_test "$UNFORMATTED files need formatting"
    fi
}

# Test 9: Check directory structure
test_directory_structure() {
    echo ""
    info "Testing directory structure..."
    
    local dirs=(
        "cmd"
        "internal/coordinators"
        "pkg/mqtt"
        "pkg/healthcheck"
        "pkg/api"
        "deployments/docker"
        "deployments/docker-compose"
        "configs"
        "scripts"
        "test"
    )
    
    for dir in "${dirs[@]}"; do
        if [ -d "$dir" ]; then
            pass_test "Directory exists: $dir"
        else
            fail_test "Directory missing: $dir"
        fi
    done
}

# Run all tests
main() {
    test_binaries
    test_binary_execution
    test_go_module
    test_unit_tests
    test_docker_files
    test_docker_compose_validation
    test_documentation
    test_code_formatting
    test_directory_structure
    
    # Summary
    echo ""
    echo "=========================================="
    echo "Test Results:"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    echo -e "${RED}Failed: $FAILED${NC}"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        exit 1
    fi
}

# Run tests
main
