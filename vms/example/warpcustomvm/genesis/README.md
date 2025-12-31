# WarpCustomVM Genesis Configuration

This directory contains the genesis configuration for WarpCustomVM.

## Genesis Format

The genesis file is a simple JSON structure:

```json
{
  "timestamp": 0
}
```

### Fields

- **`timestamp`** (int64): Unix timestamp (in seconds) for the genesis block creation time
  - Use `0` for development/testing
  - Use actual Unix timestamp for production deployments

## Usage

### When Creating a Subnet

When creating a new subnet with WarpCustomVM, you'll need to provide the genesis file:

```bash
# Using avalanche-cli
avalanche subnet create mysubnet \
  --vm warpcustomvm \
  --genesis genesis.json
```

### Genesis Initialization

The genesis is initialized in the VM's state database with:

1. **Genesis Block Header**: Block number 0, no parent, no messages
2. **Last Accepted Block**: Set to the genesis block ID (empty ID)

This is handled automatically by the `genesis.Initialize()` function when the VM starts for the first time.

## Default Genesis

The `genesis.Default()` function provides a default configuration:

```go
&Genesis{
    Timestamp: 0,
}
```

## Comparison with XSVM

Unlike XSVM which includes account allocations in genesis:

```json
{
  "timestamp": 1699574400,
  "allocations": [
    {"address": "...", "balance": 1000000000}
  ]
}
```

WarpCustomVM has a simpler genesis because:
- It focuses on **cross-subnet message passing** (Teleporter protocol)
- It doesn't manage native token balances
- Initial state is minimal (just the genesis block header)

## State Initialization

When the genesis is applied, the following state is initialized:

1. **Genesis Block Header**:
   ```json
   {
     "number": 0,
     "hash": "11111111111111111111111111111111LpoYY",
     "parentHash": "11111111111111111111111111111111LpoYY",
     "timestamp": <from genesis>,
     "messages": []
   }
   ```

2. **Last Accepted Block ID**: Set to empty ID (genesis block)

## Example Genesis Files

### Development/Testing
```json
{
  "timestamp": 0
}
```

### Production (with real timestamp)
```json
{
  "timestamp": 1699574400
}
```

### Generate Current Timestamp

```bash
# Unix/Linux/Mac
date +%s

# PowerShell
[int][double]::Parse((Get-Date -UFormat %s))
```

## See Also

- [Genesis Go Implementation](genesis.go)
- [State Storage](../state/storage.go)
- [VM Initialization](../vm.go)
