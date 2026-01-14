# Lux CLI Commands Documentation

The Lux CLI v2 is a unified toolchain for creating and managing sovereign L1s, based rollups (L2s), and app-specific chains (L3s). This document provides comprehensive documentation for all available commands.

## Table of Contents

1. [Overview](#overview)
2. [Global Flags](#global-flags)
3. [L1 Commands](#l1-commands)
4. [L2 Commands (Subnet)](#l2-commands-subnet)
5. [L3 Commands](#l3-commands)
6. [Network Commands](#network-commands)
7. [Node Commands](#node-commands)
8. [Key Commands](#key-commands)
9. [Transaction Commands](#transaction-commands)
10. [Configuration Commands](#configuration-commands)
11. [Update Commands](#update-commands)
12. [Migration Commands](#migration-commands)

## Overview

The Lux CLI supports multiple blockchain architectures:
- **L1**: Sovereign chains with independent validation
- **L2**: Based rollups or OP Stack compatible (formerly subnets)
- **L3**: App-specific chains on L2s

### Features
- EIP-4844 blob support
- Pre-confirmations (<100ms acknowledgment)
- IBC/Teleport cross-chain messaging
- Ringtail post-quantum signatures

## Global Flags

These flags are available for all commands:

```bash
--config <file>        # Config file (default: $HOME/.lux/cli.json)
--log-level <level>    # Log level for the application (default: ERROR)
--skip-update-check    # Skip check for new versions
```

## L1 Commands

Manage sovereign L1 blockchains with their own validator sets, tokenomics, and consensus mechanisms.

### lux l1 create

Create a new L1 blockchain configuration.

```bash
lux l1 create [l1Name] [flags]
```

**Flags:**
- `--proof-of-authority`: Use Proof of Authority validator management
- `--proof-of-stake`: Use Proof of Stake validator management
- `--evm-chain-id <uint>`: EVM chain ID for the L1
- `--token-name <string>`: Native token name
- `--token-symbol <string>`: Native token symbol
- `-f, --force`: Overwrite existing configuration

**Examples:**
```bash
# Create PoA L1 interactively
lux l1 create myL1

# Create PoS L1 with specific parameters
lux l1 create myL1 --proof-of-stake --evm-chain-id 96369 --token-name "MyToken" --token-symbol "MTK"
```

### lux l1 deploy

Deploy a sovereign L1 blockchain to local, testnet, or mainnet.

```bash
lux l1 deploy [l1Name] [flags]
```

**Flags:**
- `-l, --local`: Deploy to local network
- `-t, --testnet`: Deploy to testnet
- `-m, --mainnet`: Deploy to mainnet
- `--use-existing`: Use existing blockchain data
- `--protocol <string>`: Protocol to use (lux, lux-compat) (default: "lux")

**Examples:**
```bash
# Deploy to local network
lux l1 deploy myL1 --local

# Deploy to testnet with existing data
lux l1 deploy myL1 --testnet --use-existing
```

### lux l1 list

List all configured L1 blockchains.

```bash
lux l1 list
```

### lux l1 describe

Show detailed information about an L1 blockchain.

```bash
lux l1 describe [l1Name]
```

### lux l1 validator

Manage validators for an L1 blockchain.

```bash
lux l1 validator [subcommand] [l1Name] [flags]
```

**Subcommands:**
- `add`: Add a validator to the L1
- `remove`: Remove a validator from the L1
- `list`: List all validators

### lux l1 upgrade

Upgrade an L1 blockchain configuration.

```bash
lux l1 upgrade [l1Name] [flags]
```

### lux l1 import

Import an existing L1 configuration.

```bash
lux l1 import [l1Name] [flags]
```

### lux l1 migrate

Migrate L1 data between networks.

```bash
lux l1 migrate [l1Name] [flags]
```

## L2 Commands (Subnet)

Create and manage L2s (formerly subnets) with multiple sequencing models. The `l2` command has an alias `subnet` for backward compatibility.

### lux l2 create (subnet create)

Create a new L2/subnet configuration.

```bash
lux l2 create [subnetName] [flags]
lux subnet create [subnetName] [flags]  # alias
```

**Flags:**
- `--evm`: Use the Lux EVM as the base template
- `--custom`: Use a custom VM template
- `--genesis <file>`: File path of genesis to use
- `--vm <file>`: File path of custom VM to use
- `--vm-version <string>`: Version of VM template to use
- `--latest`: Use latest VM version
- `-f, --force`: Overwrite existing configuration
- `--sequencer <string>`: Sequencer for the L2 (lux, ethereum, lux, op, external)
- `--enable-preconfirm`: Enable pre-confirmations for fast UX

**Sequencing Models:**
- **lux**: Based rollup, 100ms blocks, lowest cost
- **ethereum**: Based rollup, 12s blocks, highest security
- **lux**: Based rollup, 2s blocks, fast finality
- **op**: OP Stack compatible for Optimism ecosystem
- **external**: Traditional centralized sequencer

**Examples:**
```bash
# Create L2 with interactive wizard
lux l2 create myL2

# Create L2 with Ethereum-based sequencing
lux l2 create myL2 --evm --sequencer ethereum --enable-preconfirm

# Create L2 with custom genesis
lux l2 create myL2 --genesis ./genesis.json --force
```

### lux l2 deploy (subnet deploy)

Deploy an L2/subnet.

```bash
lux l2 deploy [subnetName] [flags]
```

**Similar flags to L1 deploy**

### lux l2 list (subnet list)

List all configured L2s/subnets.

```bash
lux l2 list
```

### lux l2 describe (subnet describe)

Show detailed information about an L2/subnet.

```bash
lux l2 describe [subnetName]
```

### lux l2 join (subnet join)

Join a validator to an L2/subnet.

```bash
lux l2 join [subnetName] [flags]
```

### lux l2 addValidator (subnet addValidator)

Add a validator to an L2/subnet.

```bash
lux l2 addValidator [subnetName] [flags]
```

### lux l2 removeValidator (subnet removeValidator)

Remove a validator from an L2/subnet.

```bash
lux l2 removeValidator [subnetName] [flags]
```

### lux l2 export (subnet export)

Export L2/subnet configuration.

```bash
lux l2 export [subnetName] [flags]
```

### lux l2 import (subnet import)

Import L2/subnet configuration.

```bash
lux l2 import [flags]
```

**Subcommands:**
- `file`: Import from a file
- `running`: Import from a running network
- `historic`: Import historic subnet data

### lux l2 publish (subnet publish)

Publish L2/subnet to a repository.

```bash
lux l2 publish [subnetName] [flags]
```

### lux l2 upgrade (subnet upgrade)

Upgrade L2/subnet VM or configuration.

```bash
lux l2 upgrade [subcommand]
```

**Subcommands:**
- `apply`: Apply an upgrade
- `export`: Export upgrade configuration
- `generate`: Generate upgrade configuration
- `import`: Import upgrade configuration
- `print`: Print upgrade information
- `vm`: Upgrade the VM

### lux l2 stats (subnet stats)

Show statistics for an L2/subnet.

```bash
lux l2 stats [subnetName]
```

### lux l2 configure (subnet configure)

Configure L2/subnet parameters.

```bash
lux l2 configure [subnetName] [flags]
```

### lux l2 elastic (subnet elastic)

Manage elastic subnets (dynamic validator sets).

```bash
lux l2 elastic [subcommand] [subnetName]
```

### lux l2 validators (subnet validators)

List all validators for an L2/subnet.

```bash
lux l2 validators [subnetName]
```

### lux l2 vmid (subnet vmid)

Get the VMID of an L2/subnet.

```bash
lux l2 vmid [subnetName]
```

## L3 Commands

Create and manage L3s (app-specific chains) that run on top of L2s.

### lux l3 create

Create a new L3 configuration.

```bash
lux l3 create [l3Name] [flags]
```

**Flags:**
- `--l2 <string>`: Parent L2 for this L3
- `--vm <string>`: VM type (evm, custom)
- Additional flags similar to L2 create

**Examples:**
```bash
# Create L3 on top of an L2
lux l3 create gaming-chain --l2 myL2 --vm custom

# Interactive creation
lux l3 create myL3
```

### lux l3 deploy

Deploy an L3.

```bash
lux l3 deploy [l3Name] [flags]
```

**Flags:**
- `--preconfirm`: Enable pre-confirmations
- Additional deployment flags

### lux l3 list

List all configured L3s.

```bash
lux l3 list
```

### lux l3 describe

Show detailed information about an L3.

```bash
lux l3 describe [l3Name]
```

### lux l3 bridge

Manage cross-chain bridges for L3s.

```bash
lux l3 bridge [subcommand] [l3Name]
```

## Network Commands

Manage locally deployed networks.

### lux network start

Start a local network.

```bash
lux network start [flags]
```

**Flags:**
- `--node-version <string>`: Use this version of node (default: "latest")
- `--snapshot-name <string>`: Name of snapshot to use (default: "default-1654102509")

**Examples:**
```bash
# Start with default settings
lux network start

# Start with specific node version
lux network start --node-version v1.17.12

# Start from custom snapshot
lux network start --snapshot-name my-snapshot
```

### lux network stop

Stop the local network.

```bash
lux network stop
```

### lux network clean

Clean the local network state.

```bash
lux network clean [flags]
```

**Flags:**
- `--force`: Force clean without confirmation

### lux network status

Show the status of the local network.

```bash
lux network status
```

### lux network quickstart

Quick start a local network with pre-configured settings.

```bash
lux network quickstart
```

## Node Commands

Manage Lux node operations including development mode and automining.

### lux node dev

Start a node in development mode.

```bash
lux node dev [flags]
```

### lux node start

Start a node with specific configuration.

```bash
lux node start [flags]
```

### lux node automine

Control automining on a running node.

```bash
lux node automine [subcommand]
```

**Subcommands:**

#### lux node automine start

Start automining.

```bash
lux node automine start [flags]
```

**Flags:**
- `--rpc-url <string>`: RPC URL of the node (default: "http://localhost:9630/ext/bc/C/rpc")
- `--account <string>`: Mining account address (default: "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")
- `--private-key <string>`: Private key for the mining account
- `--threads <int>`: Number of mining threads (default: 1)
- `--monitor`: Monitor block production

**Examples:**
```bash
# Start automining with default account
lux node automine start

# Start with custom account and monitor
lux node automine start --account 0x123... --monitor

# Start with multiple threads
lux node automine start --threads 4
```

#### lux node automine stop

Stop automining.

```bash
lux node automine stop [flags]
```

**Flags:**
- `--rpc-url <string>`: RPC URL of the node

#### lux node automine status

Check automining status.

```bash
lux node automine status [flags]
```

**Flags:**
- `--rpc-url <string>`: RPC URL of the node

### lux node validator

Manage node validators.

```bash
lux node validator [subcommand]
```

## Key Commands

Create and manage testnet signing keys. **WARNING**: These keys are NOT suitable for production/mainnet use.

### lux key create

Create a new signing key.

```bash
lux key create [keyName] [flags]
```

**Flags:**
- `--file <string>`: Import the key from an existing key file
- `-f, --force`: Overwrite an existing key with the same name

**Examples:**
```bash
# Generate new key
lux key create myKey

# Import existing key
lux key create myKey --file ./existing-key.pk

# Force overwrite
lux key create myKey --force
```

### lux key list

List all available keys.

```bash
lux key list
```

### lux key delete

Delete a key.

```bash
lux key delete [keyName]
```

### lux key export

Export a key to a file.

```bash
lux key export [keyName] [flags]
```

## Transaction Commands

Sign and execute multisig transactions.

### lux transaction sign

Sign a multisig transaction.

```bash
lux transaction sign [subnetName] [flags]
```

**Flags:**
- `--input-tx-filepath <string>`: Path to the transaction file for signing
- `-k, --key <string>`: Select the key to use (testnet only)
- `-g, --ledger`: Use ledger instead of key
- `--ledger-addrs <strings>`: Use the given ledger addresses

**Examples:**
```bash
# Sign with key
lux transaction sign mySubnet --key myKey --input-tx-filepath tx.json

# Sign with ledger
lux transaction sign mySubnet --ledger --ledger-addrs addr1,addr2
```

### lux transaction commit

Commit a signed transaction.

```bash
lux transaction commit [subnetName] [flags]
```

## Configuration Commands

Manage CLI configuration.

### lux config

Show or modify configuration.

```bash
lux config [subcommand]
```

**Subcommands:**
- `metrics`: Configure metrics settings

## Update Commands

Update the CLI to the latest version.

### lux update

Check for and install updates.

```bash
lux update
```

## Migration Commands

Migrate configurations between versions or networks.

### lux migrate

Run migrations.

```bash
lux migrate [flags]
```

## Configuration Files

The CLI uses several configuration files:

1. **CLI Config**: `~/.lux/cli.json` - General CLI settings
2. **Sidecar Files**: `~/.cli/subnets/<name>/sidecar.json` - Subnet/L1/L2/L3 configurations
3. **Genesis Files**: `~/.cli/subnets/<name>/genesis.json` - Genesis configurations
4. **Node Config**: Custom node configurations for local networks

## Environment Variables

- `RUN_E2E`: Skip user metrics preference when set
- Standard Go proxy variables for downloading binaries

## Common Workflows

### Creating and Deploying an L1

```bash
# 1. Create L1 configuration
lux l1 create myL1 --proof-of-authority --token-name "MyToken" --token-symbol "MTK"

# 2. Deploy to local network
lux l1 deploy myL1 --local

# 3. Add validators
lux l1 validator add myL1 --node-id NodeID-...
```

### Creating and Deploying an L2

```bash
# 1. Create L2 with Ethereum-based sequencing
lux l2 create myL2 --evm --sequencer ethereum

# 2. Deploy to testnet
lux l2 deploy myL2 --testnet

# 3. Add validators
lux l2 addValidator myL2
```

### Setting Up Automining for Development

```bash
# 1. Start local network
lux network start

# 2. Start automining
lux node automine start --monitor

# 3. Check status
lux node automine status
```

## Best Practices

1. **Key Management**: Never use CLI-generated keys on mainnet
2. **Network Selection**: Always verify the target network before deployment
3. **Backups**: Keep backups of important configurations and genesis files
4. **Version Control**: Track subnet configurations in version control
5. **Testing**: Always test on local network before deploying to testnet/mainnet

## Troubleshooting

### Common Issues

1. **Network not running**: Start with `lux network start`
2. **Port conflicts**: Check for services using required ports
3. **Missing dependencies**: Ensure all required tools are installed
4. **Permission errors**: Check file permissions in `~/.cli/`

### Debug Mode

Enable debug logging:
```bash
lux --log-level DEBUG [command]
```

### Getting Help

Each command has built-in help:
```bash
lux [command] --help
lux [command] [subcommand] --help
```
