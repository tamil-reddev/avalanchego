# Warp Custom VM

A custom Avalanche VM implementing **bidirectional** Avalanche Warp Messaging (AWM) for cross-chain communication. Features consensus-based message propagation with full Snowman consensus integration.

## Features

-  **Send messages** from warpcustomvm to C-Chain (or any other chain)
-  **Receive messages** from C-Chain (or any other chain) to warpcustomvm
-  Full Teleporter protocol support
-  ACP-118 Warp signature aggregation
-  ICM relayer compatibility
-  Consensus-based message propagation

## Quick Start

### Send a Message (WarpCustomVM → C-Chain)
```bash
curl -X POST http://localhost:9650/ext/bc/YOUR_BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "destinationChain": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
      "destinationAddress": "0xYourCChainReceiverContract",
      "message": "Hello from WarpCustomVM!"
    }
  }'
```

### Receive a Message (C-Chain → WarpCustomVM)

**Step 1: Send from C-Chain (Solidity):**
```solidity
// Deploy WarpMessageSender.sol then call:
senderContract.sendMessage(
  0x<WARPCUSTOMVM_BLOCKCHAIN_ID>,
  0x0200000000000000000000000000000000000005, // Warp precompile
  "Hello from C-Chain!"
);
```

**Step 2: ICM Relayer automatically delivers** (no manual action needed!)

**Step 3: Query received messages:**
```bash
curl -X POST http://localhost:9650/ext/bc/YOUR_BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getAllReceivedMessages",
    "params": {}
  }'
```

 **For detailed instructions:**
- [ICM_RELAYER_INTEGRATION.md](./ICM_RELAYER_INTEGRATION.md) - **How ICM relayer works with warpcustomvm**
- [ICM_RELAYER_CONFIG.md](./ICM_RELAYER_CONFIG.md) - Complete relayer setup & configuration
- [RECEIVING_MESSAGES.md](./RECEIVING_MESSAGES.md) - Detailed receiving guide

## Architecture Overview

### Components

1. **Warp Message Format** (`api/teleporter/`)
   - **Warp → AddressedCall → Teleporter** three-layer structure
   - Warp: Network ID, Source Chain ID, unsigned message payload
   - AddressedCall: Source address, destination address, payload wrapper
   - Teleporter: Message ID, sender, destination blockchain/address, gas limit, relayer config, receipts, message
   - Message ID computed via Warp message hash

2. **State Management** (`state/`)
   - Database persistence for messages, blocks, and height mappings
   - **Global message ID counter** synchronized across validators via consensus
   - Block headers include both message IDs and full message bytes (`WarpMessages` map)
   - Key prefixes for organized storage

3. **Consensus Engine** (`chain/`)
   - Implements Snowman consensus protocol
   - Block verification with timestamp, parent, and height validation
   - **Extracts and stores messages from accepted blocks** (block-embedded propagation)
   - **Increments global message ID counter** atomically on block acceptance
   - VersionDB for staging uncommitted state changes
   - Accept/Reject lifecycle management with proper cleanup

4. **Block Builder** (`builder/`)
   - Manages pending message queue with condition variable pattern
   - **Embeds full WarpMessages bytes in blocks** for consensus propagation
   - Constructs blocks from pending messages (max 100 per block)
   - WaitForEvent() signals consensus engine when messages are ready
   - Proper mutex protection for concurrent access

5. **P2P Network Handler** (`network/`)
   - **ACP-118 Warp signature handler** for aggregate signature requests
   - Routes AppRequest/AppResponse/AppGossip messages
   - Forwards signature requests to Warp backend
   - Enables ICM relayer to collect validator signatures

6. **HTTP API** (`api/`)
   - JSON-RPC 2.0 endpoints for message submission and querying
   - **Sending**: `submitMessage` - Send Warp messages to other chains
   - **Receiving**: `receiveWarpMessage` - Accept signed messages from other chains
   - **Querying**: `getReceivedMessage`, `getAllReceivedMessages` - Query received messages
   - Block retrieval by height or "latest"
   - **Message allocation from consensus state** (race-condition aware)
   - Backward compatible with old blocks (nil map handling)

7. **VM Implementation** (`vm.go`)
   - ChainVM interface implementation with full Warp integration
   - Integrates all components (chain, builder, API, P2P)
   - Manages VM lifecycle (Initialize, Shutdown, State transitions)
   - ParseBlock logging for debugging consensus propagation

## JSON-RPC API Methods

The warpcustomvm uses JSON-RPC 2.0 format similar to xsvm. All methods are called via POST to a single endpoint.

### warpcustomvm.submitMessage
Submit a new Warp message for cross-chain delivery. The message is wrapped in Warp → AddressedCall → Teleporter layers automatically.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.submitMessage",
  "params": {
    "destinationChain": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
    "destinationAddress": "0x772eb420B677F0c42Dc1aC503D03E02E92ae1502",
    "message": "hello world"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
  }
}
```

**Notes:**
- `destinationChain`: Destination blockchain ID in hex format (e.g., C-Chain)
- `destinationAddress`: EVM contract address to receive the message
- `message`: Plain text message (automatically encoded into Teleporter format)
- Teleporter message ID allocated from consensus state counter (synchronized across validators)

### warpcustomvm.getMessage
Retrieve a message by its ID.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getMessage",
  "params": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1",
    "networkID": 5,
    "sourceChainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
    "sourceAddress": "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf",
    "payload": "0x...",
    "unsignedMessageBytes": "0x..."
  }
}
```

### warpcustomvm.getLatestBlock
Get the latest accepted block with all embedded messages.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getLatestBlock",
  "params": {}
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockNumber": 5,
    "blockHash": "23Handze1mfJApZvC7aeLxiXP6ZDMdCxE5vyE8GKyke9MscwcH",
    "timestamp": 1732001234,
    "messages": [
      {
        "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1",
        "networkID": 5,
        "sourceChainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
        "sourceAddress": "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf",
        "payload": "0x...",
        "unsignedMessageBytes": "0x...",
        "metadata": {
          "timestamp": 1732001234,
          "blockNumber": 5,
          "blockHash": "23Handze1mfJApZvC7aeLxiXP6ZDMdCxE5vyE8GKyke9MscwcH"
        }
      }
    ]
  }
}
```

**Notes:**
- Messages are extracted from the block's `WarpMessages` map (embedded during block building)
- Includes full Warp message structure with metadata for relayer consumption
- Backward compatible with old blocks (empty messages array if WarpMessages is nil)

### warpcustomvm.getBlock
Get a block by its height.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getBlock",
  "params": {
    "height": 10
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockID": "BlockID-...",
    "parentID": "BlockID-...",
    "height": 10,
    "timestamp": 1703001234,
    "messages": ["MessageID-..."]
  }
}
```

## ICM Relayer Integration

### Query-Based Message Detection

The ICM relayer queries the VM's JSON-RPC API to detect new Warp messages (no event-based detection required).

**Relayer Workflow:**
1. **Poll** `warpcustomvm.getLatestBlock` periodically
2. **Parse** embedded Warp messages from block's `messages` array
3. **Request** aggregate signatures via ACP-118 P2P handler
4. **Submit** signed messages to destination chain

**Message Structure in Block:**
```json
{
  "blockNumber": 5,
  "messages": [
    {
      "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1",
      "networkID": 5,
      "sourceChainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
      "sourceAddress": "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf",
      "payload": "0x...",
      "unsignedMessageBytes": "0x...",
      "metadata": {
        "timestamp": 1732001234,
        "blockNumber": 5,
        "blockHash": "..."
      }
    }
  ]
}
```

### Relayer Configuration

**Example ICM Relayer Config:**
```json
{
  "log-level": "info",
  "source-blockchains": [
    {
      "subnetID": "2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND",
      "blockchainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
      "rpc-endpoint": "http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc",
      "ws-endpoint": "ws://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/ws",
      "message-contracts": {
        "0x772eb420B677F0c42Dc1aC503D03E02E92ae1502": {
          "message-format": "teleporter"
        }
      }
    }
  ],
  "destination-blockchains": [
    {
      "subnetID": "11111111111111111111111111111111LpoYY",
      "blockchainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
      "vm": "evm",
      "rpc-endpoint": "https://api.avax-test.network/ext/bc/C/rpc",
      "account-private-key": "${RELAYER_PRIVATE_KEY}"
    }
  ]
}
```

**Key Configuration Points:**
- `vm`: VM ID (base58 encoded, e.g., `srEXi...`)
- `rpc-endpoint`: Full path including `/rpc`
- `ws-endpoint`: WebSocket endpoint for real-time updates
- `message-contracts`: Destination contract addresses expecting Teleporter messages
- Relayer automatically queries `getLatestBlock` for new messages

### Signature Aggregation (ACP-118)

The VM implements ACP-118 for Warp signature aggregation:

1. **Relayer requests signatures** via P2P AppRequest
2. **VM forwards request** to Warp backend
3. **Validators sign** if message exists and threshold met
4. **Relayer collects** signatures from >50% stake
5. **Submits to destination** with aggregate signature

**Enable Warp API:**
```json
{
  "warp-api-enabled": true,
  "log-level": "debug"
}
```

## Building the VM

### Prerequisites
- Go 1.21 or higher
- AvalancheGo node

### Build
```bash
cd vms/example/warpcustomvm
go build -o warpcustomvm ./...
```

Or build from the root:
```bash
cd avalanchego
go build -o bin/warpcustomvm ./vms/example/warpcustomvm/...
```

## Running the VM

### 1. Build AvalancheGo with the VM
```bash
cd avalanchego
go build -o avalanchego ./main
```

### 2. Create VM Configuration

Create `~/.avalanchego/configs/chains/<blockchain-id>/config.json`:
```json
{
  "log-level": "info"
}
```

### 3. Create Genesis File

Create `genesis.json`:
```json
{
  "timestamp": 0
}
```

### 4. Start AvalancheGo Node
```bash
./avalanchego --network-id=fuji \
  --http-host=0.0.0.0 \
  --http-port=9650 \
  --staking-tls-cert-file=<path-to-cert> \
  --staking-tls-key-file=<path-to-key>
```

### 5. Create Subnet and Blockchain

Use the Avalanche CLI or API to:
1. Create a subnet
2. Create a blockchain with `warpcustomvm` as the VM
3. Note the blockchain ID for API calls

## Testing the VM

### Using JSON-RPC (cURL)

#### Submit a Message
```bash
# Example: Send message from Custom VM to C-Chain
curl -X POST http://localhost:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "destinationChain": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
      "destinationAddress": "0x772eb420B677F0c42Dc1aC503D03E02E92ae1502",
      "message": "hello world from custom VM"
    }
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
  }
}
```

#### Check Latest Block
```bash
curl -X POST http://localhost:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getLatestBlock",
    "params": {}
  }'
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockNumber": 5,
    "blockHash": "23Handze1mfJApZvC7aeLxiXP6ZDMdCxE5vyE8GKyke9MscwcH",
    "timestamp": 1732001234,
    "messages": [
      {
        "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1",
        "networkID": 5,
        "sourceChainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
        "sourceAddress": "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf",
        "payload": "0x...",
        "metadata": {
          "timestamp": 1732001234,
          "blockNumber": 5,
          "blockHash": "23Handze1mfJApZvC7aeLxiXP6ZDMdCxE5vyE8GKyke9MscwcH"
        }
      }
    ]
  }
}
```

#### Retrieve Message by ID
```bash
curl -X POST http://localhost:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getMessage",
    "params": {
      "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
    }
  }'
```

### Using Go Client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/ava-labs/avalanchego/vms/example/warpcustomvm/api"
)

func main() {
    // Create client - similar to xsvm
    client := api.NewClient("http://localhost:9650", "<blockchain-id>")
    ctx := context.Background()

    // Prepare payload
    payload, _ := json.Marshal(map[string]interface{}{
        "action": "transfer",
        "amount": "1000",
    })

    // Submit a message
    messageID, err := client.SubmitMessage(
        ctx,
        "P-fuji1abcdef...",
        "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
        "0x1234567890123456789012345678901234567890",
        1,
        payload,
        nil,
    )
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    fmt.Printf("Message submitted! ID: %s\n", messageID)

    // Get latest block
    block, err := client.GetLatestBlock(ctx)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    fmt.Printf("Latest block height: %d\n", block.Height)
}
```

See `examples/client_example.go` for more examples.

## Consensus Protocol

This VM implements the **Snowman consensus protocol** with block-embedded message propagation:

1. **Message Submission**: 
   - API allocates Teleporter message ID from consensus state counter (synchronized across validators)
   - Message added to builder's pending pool
   - Condition variable broadcast wakes up consensus engine

2. **Block Building**: 
   - Builder's `WaitForEvent()` returns `PendingTxs` to consensus engine
   - `BuildBlock()` called to construct new block
   - **Full WarpMessages bytes embedded in block** (up to 100 messages per block)
   - Block propagates to all validators via consensus gossip

3. **Block Verification**: 
   - Timestamp validation (not >10s in future)
   - Parent block existence check
   - Height validation (parent.height + 1)
   - Timestamp ordering (≥ parent.timestamp)
   - WarpMessages map initialized (backward compatibility)

4. **State Staging**: Uses VersionDB for uncommitted changes during verification

5. **Block Acceptance**: 
   - **Extracts messages from block's WarpMessages map**
   - Parses and stores each Warp message in acceptedState
   - **Increments global message ID counter** by number of messages in block
   - Commits staged state
   - Stores block header with WarpMessages
   - Updates last accepted block ID
   - Repoints child blocks to accepted state

6. **Block Rejection**: Cleans up staged state and removes from verified blocks map

### Message ID Synchronization

- **Counter stored in consensus state** (acceptedState database)
- **Incremented atomically on block acceptance** (only when blocks with messages are accepted)
- **All validators maintain same counter** through consensus
- **Race condition handling**: Multiple nodes can temporarily allocate same ID, but only one block wins; Teleporter protocol handles duplicates gracefully
- **Best practice**: Submit messages to one node (primary validator) to avoid temporary duplicates

## Key Features

-  **Snowman Consensus**: Full implementation of Avalanche consensus protocol with threshold voting
-  **Avalanche Warp Messaging (AWM)**: Three-layer structure (Warp → AddressedCall → Teleporter)
-  **ACP-118 P2P Signature Handler**: Aggregate signature collection for Warp messages
-  **Block-Embedded Message Propagation**: Messages propagate through consensus (not gossip)
-  **Consensus-Based Message ID**: Global counter synchronized across all validators
-  **ICM Relayer Compatible**: Standard Warp message format for cross-chain delivery
-  **JSON-RPC 2.0 API**: HTTP endpoints for message submission and querying
-  **State Persistence**: Database-backed storage for messages, blocks, and counters
-  **Height Indexing**: Query blocks by height with full message data
-  **Backward Compatibility**: Handles old blocks without WarpMessages field
-  **Race-Condition Aware**: Proper mutex protection and atomic counter updates
-  **Multi-Validator Support**: Tested with 2-validator setup (weights: 102 + 48 = 150)

## Client SDK Usage

The warpcustomvm includes a complete Go client SDK in the `api` package using JSON-RPC 2.0 (similar to xsvm):

```go
import "github.com/ava-labs/avalanchego/vms/example/warpcustomvm/api"

// Create client
client := api.NewClient("http://localhost:9650", "<blockchain-id>")
```

**Available Methods:**
- `SubmitMessage(ctx, sender, destBlockchainID, destAddress, nonce, payload, metadata, options...)` - Submit a new TeleporterMessage
- `GetMessage(ctx, messageID, options...)` - Retrieve a message by ID
- `GetLatestBlock(ctx, options...)` - Get the latest accepted block
- `GetBlock(ctx, height, options...)` - Get a block at specific height

**All methods use JSON-RPC 2.0 format** and communicate via a single HTTP endpoint.

See `examples/client_example.go` for a complete working example.

## Security Considerations

1. **Message Validation**: Ensure all message fields are validated before submission
2. **Rate Limiting**: Consider adding rate limits to POST /message endpoint
3. **Authentication**: Add authentication/authorization if needed for production
4. **Timestamp Validation**: maxClockSkew is 10 seconds - adjust if needed
5. **Message Size**: MaxMessageSize is 256KB - adjust based on requirements

## Troubleshooting

### VM fails to initialize
- Check genesis file format (should include `"timestamp": 0`)
- Verify database permissions
- Review AvalancheGo logs for initialization errors
- Ensure WarpMessages field initialized in genesis

### BuildBlock fails with "preferred block not found"
**Cause:** Old blocks in database without `WarpMessages` field after code update

**Solution:**
```bash
# Stop validators
docker stop avago1 avago2

# Delete blockchain database (keeps node identity)
rm -rf ~/.avalanchego/db/<blockchain-id>
rm -rf ~/.avalanchego-node2/db/<blockchain-id>

# Restart validators
docker start avago1 avago2
```

### Messages not included in blocks
- Check logs for ` [API Server]` steps - verify message submission flow
- Look for ` WaitForEvent returning PendingTxs` - confirms engine is polling
- Check ` BuildBlock called` - verifies block building triggered
- Verify ` built block` with message count - confirms messages embedded
- Check pending message queue: `added Warp message to pending pool`

### Messages not syncing between validators
**Cause:** Consensus not reaching threshold or WarpMessages not propagating

**Solution:**
- Check validator logs for consensus messages (Verify/Accept)
- Verify both validators are connected via P2P
- Ensure total stake >= threshold (e.g., 150 >= 76 for >50%)
- Check logs for ` stored warp message from accepted block` on all validators
- Verify `ParseBlock` logs show correct `warpMessages` count

### Duplicate Teleporter Message IDs
**Cause:** Race condition when submitting to multiple nodes simultaneously

**Solution:**
- Submit messages to **only one node** (primary validator)
- Or accept that Teleporter protocol handles duplicates gracefully
- Counter synchronizes across validators after block acceptance
- Check logs for ` updated global message ID counter`

### ICM Relayer not picking up messages
**Cause:** Warp message format or signature aggregation issue

**Solution:**
- Verify `warp-api-enabled: true` in chain config
- Check ACP-118 handler logs for signature requests
- Ensure relayer queries `getLatestBlock` (not events)
- Verify message structure: Warp → AddressedCall → Teleporter
- Check relayer config matches blockchain ID and RPC endpoint
- Review relayer logs for specific errors

### "messages array empty" after submission
**Cause:** WarpMessages map not initialized or backward compatibility issue

**Solution:**
- Verify block headers have `WarpMessages` field
- Check `GetLatestBlock` logs for nil map warnings
- Ensure `GetLastMessageID`/`SetLastMessageID` working correctly
- Confirm messages are in `acceptedState` after block acceptance

## References

- [Avalanche Documentation](https://docs.avax.network/)
- [Avalanche Interchain Messaging (ICM)](https://github.com/ava-labs/teleporter)
- [XSVM Example](https://github.com/ava-labs/avalanchego/tree/master/vms/example/xsvm)
- [Snowman Consensus](https://docs.avax.network/learn/avalanche/avalanche-consensus)
