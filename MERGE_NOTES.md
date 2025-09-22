# Merge Notes: avalanche-cli v1.9.2 into lux/cli

## Date: September 22, 2025

## Summary
Successfully merged critical upstream changes from avalanche-cli v1.9.2 into lux/cli while maintaining Lux branding and package structure.

## Key Changes Merged

### Version Update
- Updated VERSION from 1.9.0 to 1.9.2

### Critical Bug Fixes Applied
1. **Wizard Command Fixes**:
   - Skip subnet validation for sovereign subnets
   - Fix nil pointer checks in vmcAtL1 flag lookup
   - Improved error handling in relayer start/stop commands

2. **Signature Aggregator Improvements**:
   - Updated signature aggregator version
   - Enhanced signature aggregator list and stop commands

3. **VMC on Different Blockchain Support**:
   - Added support for validator manager on different blockchain
   - New RPC endpoint and blockchain ID parameters

### API Changes Adapted

1. **Type Renames**:
   - `txs.SubnetValidator` → `txs.NetValidator`
   - `Subnet` field → `Net` field in validator structs

2. **Wallet Method Renames**:
   - `IssueCreateSubnetTx` → `IssueCreateNetTx`
   - `IssueTransformSubnetTx` → `IssueTransformNetTx`
   - `IssueRemoveSubnetValidatorTx` → `IssueRemoveNetValidatorTx`
   - `IssueAddSubnetValidatorTx` → `IssueAddNetValidatorTx`

3. **Builder Method Renames**:
   - `NewAddSubnetValidatorTx` → `NewAddNetValidatorTx`
   - `NewRemoveSubnetValidatorTx` → `NewRemoveNetValidatorTx`
   - `NewTransformSubnetTx` → `NewTransformNetTx`

### Infrastructure Fixes

1. **Set Package Standardization**:
   - Migrated from `github.com/luxfi/node/utils/set` to `github.com/luxfi/math/set`
   - Fixed set.Set[T] interface compatibility issues

2. **Keychain Interface Adapters**:
   - Created `CryptoToWalletWrapper` to convert between keychain interfaces
   - Fixed mismatches between crypto, wallet, and ledger keychain types
   - Simplified logger adapter since luxfi/log.Logger already implements needed interface

3. **Import Fixes**:
   - Fixed logging imports to use `github.com/luxfi/node/utils/logging`
   - Removed unused secp256k1fx imports
   - Added wallet keychain imports where needed

## Files Modified

### Core Files:
- `VERSION` - Version bump to 1.9.2
- `pkg/keychain/keychain.go` - Fixed keychain wrapper usage and imports
- `pkg/keychain/wrapper.go` - Updated set package import
- `pkg/keychain/wallet_wrapper.go` - New file for wallet keychain adapter
- `pkg/binutils/logger_adapter.go` - Simplified logger adapter
- `pkg/metrics/metrics.go` - Fixed logging import
- `pkg/subnet/local.go` - Updated validator types, wallet methods, set imports
- `pkg/subnet/public.go` - Updated validator types, wallet methods, set imports

### Remaining Work (Build Errors):
Several packages still have build errors that need attention:
- `pkg/localnet/*.go` - Logger and API compatibility issues
- `cmd/subnetcmd/*.go` - Keychain interface mismatches

## Important Notes

1. **No avalanche-tooling-sdk-go**: The upstream introduced a new SDK repository, but we're using our own luxfi/sdk instead.

2. **Package Structure Maintained**: All ava-labs imports remain replaced with luxfi equivalents:
   - `github.com/ava-labs/avalanche-cli` → `github.com/luxfi/cli`
   - `github.com/ava-labs/avalanchego` → `github.com/luxfi/node`
   - `github.com/ava-labs/subnet-evm` → `github.com/luxfi/evm`
   - `github.com/ava-labs/coreth` → `github.com/luxfi/geth`

3. **Branding**: Lux branding maintained throughout (no Avalanche/AVAX references added)

## Testing Required

Once build errors are resolved:
1. Test local network deployment
2. Test subnet creation and deployment
3. Test validator management commands
4. Test signature aggregator functionality
5. Test VMC on different blockchain feature

## Next Steps

1. Fix remaining build errors in localnet and subnetcmd packages
2. Update dependencies to match upstream versions where applicable
3. Comprehensive testing of merged functionality
4. Update documentation for new features