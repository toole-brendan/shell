#!/bin/bash
# Shell Reserve - Automated Test Runner
# Executes tests from TESTING_PLAN.md systematically

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results tracking
PASSED=0
FAILED=0
SKIPPED=0

echo "========================================="
echo "Shell Reserve - Comprehensive Test Suite"
echo "========================================="

# Function to run a test category
run_test_category() {
    local category=$1
    local test_cmd=$2
    
    echo -e "\n${YELLOW}Running: $category${NC}"
    
    if eval "$test_cmd"; then
        echo -e "${GREEN}✓ $category PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ $category FAILED${NC}"
        ((FAILED++))
    fi
}

# Phase 1: Core Testing
echo -e "\n${YELLOW}=== PHASE 1: Core Blockchain Testing ===${NC}"

run_test_category "Consensus Tests" "go test ./blockchain/... -run TestRandomX -v"
run_test_category "UTXO Tests" "go test ./blockchain/... -run TestUTXO -v"
run_test_category "Confidential Transactions" "go test ./privacy/confidential/... -v"
run_test_category "Block Propagation" "go test ./test/... -run TestBlockPropagation -timeout 5m"
run_test_category "Fee Structure" "go test ./mempool/... -run TestFee -v"

# Phase 2: Institutional Features
echo -e "\n${YELLOW}=== PHASE 2: Institutional Features Testing ===${NC}"

run_test_category "Claimable Balances" "go test ./settlement/claimable/... -v -count=3"
run_test_category "Document Hashes" "go test ./txscript/... -run TestDocHash -v"
run_test_category "Bilateral Channels" "go test ./settlement/channels/... -v"
run_test_category "ISO 20022 Integration" "go test ./settlement/iso20022/... -v"
run_test_category "Atomic Swaps" "go test ./settlement/swaps/... -v"

# Phase 3: Performance Testing
echo -e "\n${YELLOW}=== PHASE 3: Performance Testing ===${NC}"

run_test_category "Benchmarks" "go test -bench=. ./... -run=^$ -benchtime=10s | tee bench.txt"
run_test_category "Memory Profiling" "go test -run=TestMemory -memprofile=mem.prof ./test/..."
run_test_category "CPU Profiling" "go test -run=TestCPU -cpuprofile=cpu.prof ./test/..."

# Phase 4: Security Testing
echo -e "\n${YELLOW}=== PHASE 4: Security Testing ===${NC}"

run_test_category "Security Scan" "make security-scan"
run_test_category "Vulnerability Check" "make vuln-check"
run_test_category "Fuzzing" "go test -fuzz=Fuzz -fuzztime=30s ./..."

# Phase 5: Integration Testing
echo -e "\n${YELLOW}=== PHASE 5: Integration Testing ===${NC}"

run_test_category "Full Integration" "go test ./test/... -run Integration -timeout 30m"
run_test_category "Network Simulation" "go test ./test/... -run TestNetwork -timeout 1h"

# Phase 6: Chaos Testing (if enabled)
if [ "${RUN_CHAOS_TESTS}" = "true" ]; then
    echo -e "\n${YELLOW}=== PHASE 6: Chaos Testing ===${NC}"
    run_test_category "Chaos Monkey" "go test ./test/... -run TestChaos -timeout 2h"
else
    echo -e "\n${YELLOW}=== PHASE 6: Chaos Testing (SKIPPED) ===${NC}"
    echo "Set RUN_CHAOS_TESTS=true to enable"
    ((SKIPPED++))
fi

# Generate coverage report
echo -e "\n${YELLOW}=== Generating Coverage Report ===${NC}"
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "Total Coverage: ${GREEN}${COVERAGE}${NC}"

# Summary
echo -e "\n========================================="
echo -e "Test Results Summary"
echo -e "========================================="
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
echo -e "Coverage: ${COVERAGE}"
echo -e "========================================="

# Create test report
cat > test-report-$(date +%Y%m%d-%H%M%S).json <<EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "passed": $PASSED,
    "failed": $FAILED,
    "skipped": $SKIPPED,
    "coverage": "${COVERAGE}",
    "details": {
        "benchmark_results": "bench.txt",
        "coverage_report": "coverage.html",
        "memory_profile": "mem.prof",
        "cpu_profile": "cpu.prof"
    }
}
EOF

# Exit with appropriate code
if [ $FAILED -gt 0 ]; then
    echo -e "\n${RED}TESTS FAILED! Fix issues before proceeding.${NC}"
    exit 1
else
    echo -e "\n${GREEN}ALL TESTS PASSED!${NC}"
    exit 0
fi 