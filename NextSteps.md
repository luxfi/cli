# Migration Rollout Plan

This document outlines the three-phase rollout to migrate Lux‑Geth, Lux‑EVM, and the CLI to compatible versions.

## Phase 1: luxfi/geth → v0.15.38

1. **Branch**: `migration-v0.15.38`
2. Update code:
   - `core/state/state_object.go`: change `Balance` to `*uint256.Int`, add `IsZero()`, fix `Copy()`.
   - `ethclient/ethclient.go`: update `.WithBody(...)` to use `ethtypes.Body` signature.
   - `trie/verkle.go`: decode balance into `*uint256.Int`.
   - `core/state/snapshot/**`: adjust `FullAccountRLP`/`SlimAccountRLP` to new account types.
3. Run tests and formatting:
   ```bash
   go fmt ./core/state/... ./ethclient/... ./trie/...
   go test ./core/state/... ./ethclient/... ./trie/...
   ```
4. Commit, push, open PR, merge, and tag:
   ```bash
   git checkout -b migration-v0.15.38
   # apply changes...
   git commit -am "chore: migrate to geth v0.15.38 (state, ethclient, trie, snapshot)"
   git push origin migration-v0.15.38
   # Open PR on GitHub → merge
   git tag v0.15.38 && git push origin v0.15.38
   ```

## Phase 2: luxfi/evm → v0.6.0

1. **Branch**: `migration-v0.6.0`
2. Update code:
   - `core/state/snapshot/conversion.go`: use `types.FullAccountRLP`/`types.FullAccount`, remove `trie.OnTrieNode`.
3. Bump `go.mod` to require `github.com/luxfi/geth v0.15.38`.
4. Run tests and formatting:
   ```bash
   go fmt ./core/state/snapshot
   go test ./core/state/snapshot
   ```
5. Commit, push, open PR, merge, and tag:
   ```bash
   git checkout -b migration-v0.6.0
   # apply changes...
   git commit -am "chore: migrate EVM snapshot to geth v0.15.38 API"
   git push origin migration-v0.6.0
   # Open PR on GitHub → merge
   git tag v0.6.0 && git push origin v0.6.0
   ```

## Phase 3: luxfi/cli → v1.14.1

1. **Branch**: `bump-geth-evm`
2. Update `go.mod`:
   ```diff
   require (
     github.com/luxfi/geth v0.15.38
     github.com/luxfi/evm  v0.6.0
   )
   ```
3. Tidy and build:
   ```bash
   go mod tidy
   make build
   go test ./...
   ```
4. Commit, push, open PR, merge, and tag:
   ```bash
   git checkout -b bump-geth-evm
   git add go.mod go.sum
   git commit -m "chore: bump geth to v0.15.38, evm to v0.6.0"
   git push origin bump-geth-evm
   # Open PR on GitHub → merge
   git tag v1.14.1 && git push origin v1.14.1
   ```
