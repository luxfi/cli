#!/bin/bash

# Q-Chain Implementation Test Script
# Tests all Q-Chain functionality in the Lux CLI

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test configuration
CLI_DIR="/home/z/work/lux/cli"
TEST_DIR="${HOME}/.lux/qchain-test"
PASSED=0
FAILED=0

echo -e "${CYAN}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║          Q-Chain Implementation Test Suite            ║${NC}"
echo -e "${CYAN}╚════════════════════════════════════════════════════════╝${NC}"
echo

# Setup test environment
setup_test_env() {
    echo -e "${YELLOW}Setting up test environment...${NC}"
    mkdir -p "${TEST_DIR}"
    cd "${CLI_DIR}"
    echo -e "${GREEN}✓ Test environment ready${NC}"
    echo
}

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_cmd="$2"

    echo -e "${BLUE}Testing: ${test_name}${NC}"

    if eval "${test_cmd}" > "${TEST_DIR}/test_output.log" 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo -e "${RED}  Error output:${NC}"
        tail -3 "${TEST_DIR}/test_output.log"
        ((FAILED++))
    fi
    echo
}

# Test 1: Check if Q-chain command exists
test_qchain_command() {
    run_test "Q-Chain command exists" \
        "./bin/lux qchain --help"
}

# Test 2: Test Q-chain describe command
test_qchain_describe() {
    run_test "Q-Chain describe command" \
        "./bin/lux qchain describe"
}

# Test 3: Test quantum key generation
test_quantum_key_generation() {
    run_test "Quantum key generation" \
        "./generate-quantum-keys.sh ringtail-256 1"
}

# Test 4: Test Q-chain verification
test_qchain_verify() {
    run_test "Q-Chain verify command" \
        "./bin/lux qchain verify --benchmark"
}

# Test 5: Test transaction commands
test_transaction_commands() {
    run_test "Q-Chain transaction help" \
        "./bin/lux qchain transaction --help"
}

# Test 6: Check Q-Chain constants in node
test_node_constants() {
    echo -e "${BLUE}Testing: Q-Chain constants in node${NC}"

    if grep -q "QChainID\|QChainMainnetID\|QChainTestnetID" "/home/z/work/lux/node/utils/constants/network_ids.go"; then
        echo -e "${GREEN}✓ PASSED - Q-Chain constants found${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED - Q-Chain constants not found${NC}"
        ((FAILED++))
    fi
    echo
}

# Test 7: Check VM type support
test_vm_support() {
    echo -e "${BLUE}Testing: Quantum VM support${NC}"

    if grep -q "QuantumVM" "/home/z/work/lux/cli/pkg/models/vm.go"; then
        echo -e "${GREEN}✓ PASSED - Quantum VM type supported${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED - Quantum VM type not found${NC}"
        ((FAILED++))
    fi
    echo
}

# Test 8: Check deployment script
test_deployment_script() {
    echo -e "${BLUE}Testing: Q-Chain deployment script${NC}"

    if [ -f "${CLI_DIR}/deploy-qchain.sh" ] && [ -x "${CLI_DIR}/deploy-qchain.sh" ]; then
        echo -e "${GREEN}✓ PASSED - Deployment script exists and is executable${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED - Deployment script missing or not executable${NC}"
        ((FAILED++))
    fi
    echo
}

# Test 9: Check if CLI builds successfully
test_cli_build() {
    echo -e "${BLUE}Testing: CLI build with Q-Chain support${NC}"

    cd "${CLI_DIR}"
    if go build -o "${TEST_DIR}/test-lux" ./main.go 2>&1 | grep -q "error"; then
        echo -e "${RED}✗ FAILED - CLI build failed${NC}"
        ((FAILED++))
    else
        echo -e "${GREEN}✓ PASSED - CLI builds successfully${NC}"
        ((PASSED++))
    fi
    echo
}

# Test 10: Integration test - Q-Chain workflow
test_integration() {
    echo -e "${BLUE}Testing: Q-Chain integration workflow${NC}"

    # Create a test script that simulates Q-Chain workflow
    cat > "${TEST_DIR}/integration_test.sh" <<'EOF'
#!/bin/bash
set -e

# Test Q-Chain workflow
echo "1. Generating quantum keys..."
./generate-quantum-keys.sh ringtail-256 1 >/dev/null 2>&1

echo "2. Checking Q-Chain status..."
./bin/lux qchain describe >/dev/null 2>&1

echo "3. Verifying quantum safety..."
./bin/lux qchain verify >/dev/null 2>&1

echo "Integration test completed successfully"
EOF

    chmod +x "${TEST_DIR}/integration_test.sh"

    if cd "${CLI_DIR}" && "${TEST_DIR}/integration_test.sh" > "${TEST_DIR}/integration.log" 2>&1; then
        echo -e "${GREEN}✓ PASSED - Integration test successful${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED - Integration test failed${NC}"
        cat "${TEST_DIR}/integration.log"
        ((FAILED++))
    fi
    echo
}

# Display test results
display_results() {
    echo -e "${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}                   Test Results                       ${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════${NC}"
    echo
    echo -e "Total Tests: $((PASSED + FAILED))"
    echo -e "${GREEN}Passed: ${PASSED}${NC}"
    echo -e "${RED}Failed: ${FAILED}${NC}"
    echo

    if [ ${FAILED} -eq 0 ]; then
        echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║     All Q-Chain tests passed successfully! 🎉         ║${NC}"
        echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
        echo
        echo -e "${BLUE}Q-Chain Features Verified:${NC}"
        echo -e "  ✓ Q-Chain command structure"
        echo -e "  ✓ Quantum key generation"
        echo -e "  ✓ Network constants integration"
        echo -e "  ✓ Quantum VM support"
        echo -e "  ✓ Transaction commands"
        echo -e "  ✓ Deployment scripts"
        echo -e "  ✓ CLI integration"
        return 0
    else
        echo -e "${RED}╔════════════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║     Some tests failed. Please review the output.      ║${NC}"
        echo -e "${RED}╚════════════════════════════════════════════════════════╝${NC}"
        return 1
    fi
}

# Main test execution
main() {
    setup_test_env

    echo -e "${YELLOW}Running Q-Chain Implementation Tests...${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════════════${NC}"
    echo

    # Build CLI first if not already built
    if [ ! -f "${CLI_DIR}/bin/lux" ]; then
        echo -e "${YELLOW}Building CLI...${NC}"
        cd "${CLI_DIR}"
        go build -o bin/lux ./main.go || echo -e "${YELLOW}Note: CLI build skipped${NC}"
        echo
    fi

    # Make scripts executable
    chmod +x "${CLI_DIR}/generate-quantum-keys.sh" 2>/dev/null || true
    chmod +x "${CLI_DIR}/deploy-qchain.sh" 2>/dev/null || true

    # Run all tests
    test_node_constants
    test_vm_support
    test_deployment_script
    test_qchain_command
    test_qchain_describe
    test_quantum_key_generation
    test_qchain_verify
    test_transaction_commands
    test_cli_build
    test_integration

    # Display results
    display_results
}

# Run tests
main "$@"