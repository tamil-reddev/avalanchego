# Warp Custom VM

A custom Avalanche VM for sending Warp messages (TeleporterMessages) from this VM to C-Chain, designed for use with the ICM (Interchain Messaging) relayer.

## Architecture Overview

### Components

1. **TeleporterMessage Format** (`message-contracts/teleporter/`)
   - Standard format for cross-chain messages
   - Fields: Sender, DestinationBlockchainID, DestinationAddress, Nonce, Payload, Metadata
   - Message ID computed via hashing

2. **State Management** (`state/`)
   - Database persistence for messages, blocks, and height mappings
   - Key prefixes for organized storage

3. **Consensus Engine** (`chain/`)
   - Implements Snowman consensus protocol
   - Block verification with timestamp, parent, and height validation
   - VersionDB for staging uncommitted state changes
   - Accept/Reject lifecycle management

4. **Block Builder** (`builder/`)
   - Manages pending message queue
   - Constructs blocks from pending messages
   - Notifies engine when ready to build

5. **Event Emitter** (`events/`)
   - Emits `AddressedCall` events for ICM relayers
   - Event signature: `keccak256("AddressedCall(bytes32,bytes32)")`
   - Log format with Topics and Data for relayer parsing

6. **HTTP API** (`api/`)
   - RESTful endpoints for message submission and querying
   - Block retrieval by height or "latest"

7. **VM Implementation** (`vm.go`)
   - ChainVM interface implementation
   - Integrates all components
   - Manages VM lifecycle (Initialize, Shutdown, State transitions)

## JSON-RPC API Methods

The warpcustomvm uses JSON-RPC 2.0 format similar to xsvm. All methods are called via POST to a single endpoint.

### warpcustomvm.submitMessage
Submit a new Warp message for cross-chain delivery.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.submitMessage",
  "params": {
    "sender": "P-fuji1...",
    "destinationBlockchainID": "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
    "destinationAddress": "0x1234567890123456789012345678901234567890",
    "nonce": 1,
    "payload": "base64_encoded_payload",
    "metadata": "base64_encoded_metadata"
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "MessageID-..."
  }
}
```

### warpcustomvm.getMessage
Retrieve a message by its ID.

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getMessage",
  "params": {
    "messageID": "MessageID-..."
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "sender": "P-fuji1...",
    "destinationBlockchainID": "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
    "destinationAddress": "0x1234567890123456789012345678901234567890",
    "nonce": 1,
    "payload": "base64_encoded_payload"
  }
}
```

### warpcustomvm.getLatestBlock
Get the latest accepted block.

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
    "blockID": "BlockID-...",
    "parentID": "BlockID-...",
    "height": 42,
    "timestamp": 1703001234,
    "messages": ["MessageID-...", "MessageID-..."]
  }
}
```

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

### AddressedCall Event Format

When a message is submitted via POST /message, the VM emits an `AddressedCall` event that ICM relayers can detect.

**Event Signature:**
```
keccak256("AddressedCall(bytes32,bytes32)") = 0x...
```

**Log Structure:**
```json
{
  "topics": [
    "0x<event_signature_hash>",
    "0x<destination_blockchain_id>"
  ],
  "data": "0x<hex_encoded_addressed_call_payload>"
}
```

**AddressedCallPayload:**
```json
{
  "destinationBlockchainID": "BlockchainID-...",
  "destinationAddress": "Address-...",
  "payload": {...}
}
```

### Relayer Configuration

Configure your ICM relayer to:
1. Listen to this VM's blockchain for `AddressedCall` events
2. Parse the event Topics to extract destination blockchain ID
3. Decode the Data field to get the AddressedCallPayload
4. Submit the message to the destination chain (e.g., C-Chain)

**Example Relayer Config:**
```json
{
  "source-blockchains": [
    {
      "blockchainID": "<warpcustomvm-blockchain-id>",
      "vm": "warpcustomvm",
      "rpc-endpoint": "http://localhost:9650/ext/bc/warpcustomvm",
      "event-signatures": [
        "AddressedCall(bytes32,bytes32)"
      ]
    }
  ],
  "destination-blockchains": [
    {
      "blockchainID": "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
      "vm": "evm",
      "rpc-endpoint": "https://api.avax-test.network/ext/bc/C/rpc"
    }
  ]
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
curl -X POST http://localhost:9650/ext/bc/<blockchain-id> \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "sender": "P-fuji1abcdef...",
      "destinationBlockchainID": "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
      "destinationAddress": "0x1234567890123456789012345678901234567890",
      "nonce": 1,
      "payload": "'$(echo -n '{"action":"transfer","amount":"1000"}' | base64)'",
      "metadata": ""
    }
  }'
```

#### Check Latest Block
```bash
curl -X POST http://localhost:9650/ext/bc/<blockchain-id> \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getLatestBlock",
    "params": {}
  }'
```

#### Retrieve Message
```bash
curl -X POST http://localhost:9650/ext/bc/<blockchain-id> \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getMessage",
    "params": {
      "messageID": "MessageID-..."
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

This VM implements the **Snowman consensus protocol**:

1. **Block Building**: Builder collects pending messages and creates blocks
2. **Block Verification**: 
   - Timestamp validation (not >10s in future)
   - Parent block existence check
   - Height validation (parent.height + 1)
   - Timestamp ordering (≥ parent.timestamp)
3. **State Staging**: Uses VersionDB for uncommitted changes during verification
4. **Block Acceptance**: 
   - Commits staged state
   - Stores block header
   - Updates last accepted block ID
   - Repoints child blocks to accepted state
5. **Block Rejection**: Cleans up staged state and removes from verified blocks map

## Key Features

- ✅ **Snowman Consensus**: Full implementation of Avalanche consensus protocol
- ✅ **TeleporterMessage Support**: Standard format for cross-chain messages
- ✅ **ICM Relayer Compatible**: Emits AddressedCall events for relayers
- ✅ **REST API**: HTTP endpoints for message submission and querying
- ✅ **Go Client SDK**: Complete JSON-RPC client library for easy integration
- ✅ **State Persistence**: Database-backed storage for messages and blocks
- ✅ **Height Indexing**: Query blocks by height
- ✅ **Event Emission**: Logs for relayer integration

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
- Check genesis file format
- Verify database permissions
- Review AvalancheGo logs

### Messages not included in blocks
- Check builder is receiving messages (logs)
- Verify engine is being notified (listenForBuildEvents)
- Check pending message queue

### Relayer not detecting events
- Verify AddressedCall event is being emitted (check logs)
- Confirm event signature hash matches relayer config
- Check relayer is connected to correct RPC endpoint

## References

- [Avalanche Documentation](https://docs.avax.network/)
- [Avalanche Interchain Messaging (ICM)](https://github.com/ava-labs/teleporter)
- [XSVM Example](https://github.com/ava-labs/avalanchego/tree/master/vms/example/xsvm)
- [Snowman Consensus](https://docs.avax.network/learn/avalanche/avalanche-consensus)

## License

Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
See the file LICENSE for licensing terms.
