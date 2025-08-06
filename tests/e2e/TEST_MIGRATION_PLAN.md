# Test Migration Plan from Avalanche CLI

## Priority 1: Critical Missing Tests

### Validator Management Tests
```go
// tests/e2e/testcases/validators/poa/add_validator_test.go
package poa

import (
    "testing"
    "github.com/luxfi/sdk"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("POA Validator Management", func() {
    var (
        sdk     *sdk.Client
        network *sdk.Network
    )
    
    BeforeEach(func() {
        sdk = setupTestSDK()
        network = sdk.LaunchLocalNetwork(5)
    })
    
    AfterEach(func() {
        network.Cleanup()
    })
    
    Context("Adding validators", func() {
        It("should add a validator to POA network", func() {
            validator := &sdk.Validator{
                NodeID: "NodeID-test123",
                Weight: 1000,
            }
            
            err := network.AddValidator(validator)
            Expect(err).NotTo(HaveOccurred())
            
            validators := network.GetValidators()
            Expect(validators).To(ContainElement(validator))
        })
    })
})
```

### Cross-Chain Messaging Tests
```go
// tests/e2e/testcases/crosschain/teleport_test.go
package crosschain

var _ = Describe("Teleport Bridge", func() {
    It("should transfer assets between L1 and L2", func() {
        l1 := sdk.GetNetwork("lux")
        l2 := sdk.GetNetwork("zoo")
        
        // Transfer 100 LUX from L1 to L2
        tx := &sdk.CrossChainTx{
            From:   l1,
            To:     l2,
            Amount: big.NewInt(100e18),
            Asset:  "LUX",
        }
        
        receipt, err := sdk.ExecuteCrossChainTransfer(tx)
        Expect(err).NotTo(HaveOccurred())
        Expect(receipt.Status).To(Equal("success"))
        
        // Verify balance on L2
        balance := l2.GetBalance(tx.Recipient)
        Expect(balance).To(Equal(tx.Amount))
    })
})
```

### Monitoring Integration Tests
```go
// tests/e2e/testcases/monitoring/prometheus_test.go
package monitoring

var _ = Describe("Prometheus Integration", func() {
    It("should expose metrics endpoint", func() {
        network := sdk.LaunchNetwork()
        
        metrics, err := network.GetMetrics()
        Expect(err).NotTo(HaveOccurred())
        
        // Check key metrics exist
        Expect(metrics).To(HaveKey("lux_network_validators_total"))
        Expect(metrics).To(HaveKey("lux_network_tps"))
        Expect(metrics).To(HaveKey("lux_network_block_height"))
    })
})
```

## Priority 2: Network Upgrade Tests

### L2 to L1 Migration Tests
```go
// tests/e2e/testcases/migration/subnet_to_l1_test.go
var _ = Describe("Subnet to L1 Migration", func() {
    It("should migrate subnet to sovereign L1", func() {
        subnet := sdk.CreateSubnet("test-subnet")
        
        // Add validators
        subnet.AddValidator(validator1)
        subnet.AddValidator(validator2)
        
        // Perform migration
        l1, err := subnet.MigrateToL1()
        Expect(err).NotTo(HaveOccurred())
        
        // Verify L1 properties
        Expect(l1.Type).To(Equal("sovereign"))
        Expect(l1.Validators).To(HaveLen(2))
        
        // Verify state preservation
        originalBalance := subnet.GetBalance(testAddr)
        migratedBalance := l1.GetBalance(testAddr)
        Expect(migratedBalance).To(Equal(originalBalance))
    })
})
```

## Priority 3: Performance Tests

### Load Testing
```go
// tests/e2e/testcases/performance/load_test.go
var _ = Describe("Network Load Testing", func() {
    It("should handle 10,000 TPS", func() {
        network := sdk.LaunchNetwork()
        
        // Generate load
        results := sdk.GenerateLoad(sdk.LoadConfig{
            TPS:      10000,
            Duration: 60 * time.Second,
            TxType:   "transfer",
        })
        
        // Verify performance
        Expect(results.AverageTPS).To(BeNumerically(">", 9500))
        Expect(results.FailedTxs).To(BeZero())
        Expect(results.AverageLatency).To(BeNumerically("<", 2*time.Second))
    })
})
```

## Test Infrastructure Setup

### 1. Install Ginkgo
```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

### 2. Create Test Suite
```go
// tests/e2e/suite_test.go
package e2e

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Lux E2E Test Suite")
}
```

### 3. Test Helpers
```go
// tests/e2e/helpers/network.go
package helpers

func SetupTestNetwork() *sdk.Network {
    config := sdk.TestConfig()
    sdk := sdk.New(config)
    return sdk.LaunchLocalNetwork(5)
}

func CleanupNetwork(network *sdk.Network) {
    network.Stop()
    network.RemoveData()
}
```

## Coverage Goals

| Test Category | Current | Target | Priority |
|--------------|---------|---------|----------|
| Validators | 0% | 90% | HIGH |
| Cross-chain | 20% | 95% | HIGH |
| Monitoring | 0% | 80% | MEDIUM |
| Migration | 50% | 90% | HIGH |
| Performance | 10% | 85% | MEDIUM |
| Upgrades | 0% | 90% | HIGH |

## Implementation Timeline

- **Week 1**: Validator management tests
- **Week 2**: Cross-chain messaging tests
- **Week 3**: Monitoring integration
- **Week 4**: Migration tests
- **Week 5**: Performance tests
- **Week 6**: Integration and CI/CD