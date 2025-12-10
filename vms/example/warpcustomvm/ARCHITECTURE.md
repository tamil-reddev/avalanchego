# WarpCustomVM Architecture Documentation

## Table of Contents
1. [Overview](#overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Components](#core-components)
4. [Message Flow](#message-flow)
5. [Data Structures](#data-structures)
6. [API Reference](#api-reference)
7. [Consensus Integration](#consensus-integration)

---

## Overview

**WarpCustomVM** is a custom Avalanche Virtual Machine that implements **bidirectional cross-chain messaging** using Avalanche Warp Messaging (AWM). It enables:

- **Sending messages** from WarpCustomVM to any other Avalanche blockchain (C-Chain, X-Chain, other Subnets)
- **Receiving messages** from any other Avalanche blockchain to WarpCustomVM
- **Consensus-based synchronization** ensuring all validators have identical message state
- **ICM Relayer integration** for automatic message delivery
- **Plain text payloads** with no complex encoding (simplified architecture)

### Key Features
-  Simple, readable code (no encryption, no complex encoding)
-  Full Snowman consensus integration
-  ACP-118 Warp signature aggregation
-  Automatic cross-chain message delivery via ICM Relayer
-  All validators sync received messages through block consensus

---

## Architecture Diagram

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AVALANCHE NETWORK LAYER                               │
│  ┌────────────────┐         ┌────────────────┐         ┌────────────────┐  │
│  │   Validator 1  │◄───────►│   Validator 2  │◄───────►│   Validator 3  │  │
│  │  (Node 9650)   │  P2P    │  (Node 9652)   │  P2P    │  (Node 9654)   │  │
│  └────────┬───────┘         └────────┬───────┘         └────────┬───────┘  │
│           │                          │                          │           │
└───────────┼──────────────────────────┼──────────────────────────┼───────────┘
            │                          │                          │
            ▼                          ▼                          ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                          WARPCUSTOMVM INSTANCES                            │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐       │
│  │   VM Instance 1 │    │   VM Instance 2 │    │   VM Instance 3 │       │
│  │ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │       │
│  │ │   JSON-RPC  │ │    │ │   JSON-RPC  │ │    │ │   JSON-RPC  │ │       │
│  │ │   API Layer │ │    │ │   API Layer │ │    │ │   API Layer │ │       │
│  │ └──────┬──────┘ │    │ └──────┬──────┘ │    │ └──────┬──────┘ │       │
│  │        │        │    │        │        │    │        │        │       │
│  │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │       │
│  │ │   Builder   │ │    │ │   Builder   │ │    │ │   Builder   │ │       │
│  │ │  (Proposes  │ │    │ │  (Proposes  │ │    │ │  (Proposes  │ │       │
│  │ │   Blocks)   │ │    │ │   Blocks)   │ │    │ │   Blocks)   │ │       │
│  │ └──────┬──────┘ │    │ └──────┬──────┘ │    │ └──────┬──────┘ │       │
│  │        │        │    │        │        │    │        │        │       │
│  │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │       │
│  │ │    Chain    │ │    │ │    Chain    │ │    │ │    Chain    │ │       │
│  │ │  (Consensus │ │    │ │  (Consensus │ │    │ │  (Consensus │ │       │
│  │ │   & Accept) │ │    │ │   & Accept) │ │    │ │   & Accept) │ │       │
│  │ └──────┬──────┘ │    │ └──────┬──────┘ │    │ └──────┬──────┘ │       │
│  │        │        │    │        │        │    │        │        │       │
│  │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │       │
│  │ │    State    │ │    │ │    State    │ │    │ │    State    │ │       │
│  │ │  (Database) │ │    │ │  (Database) │ │    │ │  (Database) │ │       │
│  │ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │       │
│  └─────────────────┘    └─────────────────┘    └─────────────────┘       │
│         │                       │                       │                 │
│         └───────────────────────┴───────────────────────┘                 │
│                 │ All instances sync through consensus │                  │
└─────────────────┼───────────────────────────────────────────────────────┘
                  │
         ┌────────▼────────┐
         │   Identical     │
         │  Message State  │
         │  Across All VMs │
         └─────────────────┘
```

### Cross-Chain Message Flow (WarpCustomVM ↔ C-Chain)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     SENDING MESSAGE (VM → C-Chain)                        │
└──────────────────────────────────────────────────────────────────────────┘

     User/Client
         │
         │ 1. submitMessage(destinationChainID, destinationAddress, payload)
         ▼
   ┌─────────────┐
   │  JSON-RPC   │
   │  API Server │
   └──────┬──────┘
          │ 2. Create unsigned Warp message
          │    - Wrap payload in AddressedCall format
          │    - Set source address from node ID
          ▼
   ┌─────────────┐
   │   Builder   │
   │             │ 3. AddMessage(unsigned message)
   └──────┬──────┘
          │
          │ 4. Propose new block with message
          ▼
   ┌─────────────┐
   │    Chain    │
   │ (Consensus) │ 5. Block accepted by all validators
   └──────┬──────┘
          │
          │ 6. Store in block header
          ▼
   ┌─────────────┐
   │    State    │
   │ (Database)  │ 7. Message persisted with block
   └──────┬──────┘
          │
          │ 8. Validators sign via ACP-118
          ▼
   ┌─────────────┐
   │ ICM Relayer │ 9. Collects signatures
   │             │ 10. Delivers to C-Chain
   └──────┬──────┘
          │
          ▼
   ┌─────────────┐
   │   C-Chain   │ 11. Message received
   │  Contract   │
   └─────────────┘


┌──────────────────────────────────────────────────────────────────────────┐
│                   RECEIVING MESSAGE (C-Chain → VM)                        │
└──────────────────────────────────────────────────────────────────────────┘

   C-Chain Contract
         │
         │ 1. sendWarpMessage(payload) via Warp Precompile
         ▼
   ┌─────────────┐
   │   C-Chain   │
   │Warp Precom. │ 2. Create unsigned Warp message
   └──────┬──────┘
          │
          │ 3. Validators sign message
          ▼
   ┌─────────────┐
   │ ICM Relayer │ 4. Collects signatures (aggregated BLS)
   │             │ 5. Calls receiveWarpMessage API
   └──────┬──────┘
          │
          │ 6. POST /rpc receiveWarpMessage(signedMessage)
          ▼
   ┌─────────────┐
   │  JSON-RPC   │
   │  API Server │ 7. Parse signed message
   └──────┬──────┘    - Verify signatures
          │           - Extract payload
          │
          │ 8. AddMessage(parsed message)
          ▼
   ┌─────────────┐
   │   Builder   │ 9. Propose new block with message
   └──────┬──────┘
          │
          │ 10. Block propagates to all validators
          ▼
   ┌─────────────┐
   │    Chain    │ 11. Accept() called on ALL nodes
   │ (Consensus) │     - Detects external message
   └──────┬──────┘     - Stores as received message
          │
          │ 12. SetReceivedMessage() on ALL nodes
          ▼
   ┌─────────────┐
   │    State    │ 13. Message synced across all validators
   │ (Database)  │
   └─────────────┘
```

---

## Core Components

### 1. VM (vm.go)

**Purpose**: Main entry point and coordinator for the Virtual Machine.

**Responsibilities**:
- Initialize P2P network for Warp message signing (ACP-118)
- Register ACP-118 signature handler for ICM relayer compatibility
- Coordinate between API, Builder, Chain, and State layers
- Manage VM lifecycle (Initialize, SetState, Shutdown)

**Key Code**:
```go
type VM struct {
    *p2p.Network      // P2P network for Warp signing
    chainContext      *snow.Context
    acceptedState     database.Database
    chain             chain.Chain
    builder           builder.Builder
    toEngine          chan<- common.Message
}
```

### 2. API Layer (api/server.go, api/client.go)

**Purpose**: Exposes JSON-RPC endpoints for external interaction.

**Key Methods**:

#### `submitMessage`
- **Input**: `destinationChainID`, `destinationAddress`, `payload` (plain text)
- **Process**: 
  1. Create unsigned Warp message
  2. Wrap payload in AddressedCall format
  3. Send to Builder
  4. Return message ID and unsigned message (hex)
- **Output**: `messageID`, `unsignedMessage`

#### `receiveWarpMessage`
- **Input**: `signedMessage` (hex-encoded signed Warp message)
- **Process**:
  1. Parse signed message
  2. Verify signatures
  3. Extract AddressedCall payload
  4. Send to Builder for block inclusion
- **Output**: `messageID`, `sourceChainID`, `txId`, `success`

#### `getAllReceivedMessages`
- **Input**: None
- **Process**: Query database for all received messages
- **Output**: Array of messages with plain text payloads

#### `getLatestBlock`
- **Input**: None
- **Process**: Retrieve latest block with full message details
- **Output**: Block header + messages (payload as plain string, addresses as hex)

#### `getBlock`
- **Input**: `height` (block number)
- **Process**: Retrieve specific block with full message details
- **Output**: Block header + messages (payload as plain string, addresses as hex)

### 3. Builder (builder/)

**Purpose**: Constructs new blocks containing messages and transactions.

**Responsibilities**:
- Queue incoming messages (both outgoing and incoming Warp messages)
- Build blocks at regular intervals (1 second default)
- Propose blocks to the consensus engine

**Key Flow**:
```
User submits message
     │
     ▼
Builder.AddMessage()
     │
     ▼
Queued in pendingMessages
     │
     ▼
BuildBlock() (every 1 second)
     │
     ▼
Create block with messages
     │
     ▼
Send to consensus engine
```

### 4. Chain (chain/chain.go)

**Purpose**: Manages blockchain state and block acceptance.

**Critical Function**: `Accept()`
- Called on **ALL validator nodes** when a block is accepted
- Detects external messages (sourceChainID ≠ our chainID)
- Stores received messages in database
- **This ensures consensus synchronization**

**Key Code**:
```go
func (c *chain) Accept(ctx context.Context, b *xblock.Block) error {
    for _, msg := range b.Messages {
        if msg.SourceChainID != c.ctx.ChainID {
            // External message - store as received
            receivedMsg := &state.ReceivedMessage{
                MessageID:       msg.ID(),
                SourceChainID:   msg.SourceChainID,
                SourceAddress:   addressedCall.SourceAddress[:],
                Payload:         addressedCall.Payload,
                ReceivedAt:      time.Now().Unix(),
                BlockHeight:     b.Height,
                SignedMessage:   signedBytes,
                UnsignedMessage: msg.Bytes(),
            }
            state.SetReceivedMessage(c.db, receivedMsg)
        }
    }
}
```

### 5. State (state/storage.go)

**Purpose**: Persistence layer for blocks and messages.

**Data Structures**:

```go
type ReceivedMessage struct {
    MessageID       ids.ID    // Unique message identifier
    SourceChainID   ids.ID    // Origin blockchain ID
    SourceAddress   []byte    // Sender address (32 bytes)
    Payload         []byte    // Plain text message content
    ReceivedAt      int64     // Unix timestamp
    BlockHeight     uint64    // Block where message was accepted
    SignedMessage   []byte    // Full signed Warp message
    UnsignedMessage []byte    // Unsigned Warp message
}
```

**Key Functions**:
- `SetReceivedMessage(db, msg)` - Store received message
- `GetReceivedMessage(db, msgID)` - Retrieve message by ID
- `GetAllReceivedMessageIDs(db)` - List all received message IDs
- `GetBlockHeader(db, blockID)` - Retrieve block header
- `SetLatestBlockHeader(db, header)` - Update latest block

### 6. Warp Verifier (warp_verifier.go)

**Purpose**: Implements ACP-118 signature verification for ICM relayer.

**Responsibilities**:
- Verify unsigned Warp messages exist in blockchain
- Return message bytes for relayer signature collection
- Required for ICM relayer to function

---

## Message Flow

### Outgoing Message Flow (WarpCustomVM → C-Chain)

1. **User Action**: Client calls `submitMessage` via JSON-RPC
   ```bash
   curl -X POST http://localhost:9650/ext/bc/CHAIN_ID/rpc \
     -d '{"method":"warpcustomvm.submitMessage","params":{...}}'
   ```

2. **API Processing**: 
   - Parse request arguments
   - Create unsigned Warp message
   - Wrap payload in AddressedCall format:
     ```
     [CodecID(2 bytes)][TypeID(4 bytes)][SourceAddress(32 bytes)][Payload(variable)]
     ```

3. **Builder Queuing**: 
   - `builder.AddMessage(unsignedMsg)`
   - Message queued in `pendingMessages`

4. **Block Creation**: 
   - Builder creates block (every 1 second)
   - Block includes message in header

5. **Consensus**: 
   - Block proposed to validators
   - All validators vote on block
   - Block accepted when majority agrees

6. **State Persistence**: 
   - `chain.Accept()` stores block in database
   - Block header includes message bytes

7. **Signature Collection** (ACP-118):
   - ICM relayer queries `getWarpMessage(messageID)`
   - Relayer requests signatures from validators
   - Validators sign via P2P network
   - Relayer aggregates BLS signatures

8. **Delivery**: 
   - ICM relayer sends signed message to C-Chain
   - C-Chain contract receives message

### Incoming Message Flow (C-Chain → WarpCustomVM)

1. **C-Chain Sender**: 
   - Solidity contract calls Warp Precompile
   ```solidity
   WARP_PRECOMPILE.sendWarpMessage(payload);
   ```

2. **C-Chain Consensus**: 
   - Message included in C-Chain block
   - Validators sign message

3. **ICM Relayer Detection**: 
   - Relayer monitors C-Chain for Warp messages
   - Collects validator signatures
   - Aggregates into single signed message

4. **API Delivery**: 
   - Relayer calls `receiveWarpMessage(signedMessage)`
   - API parses and verifies signed message

5. **Builder Queuing**: 
   - Parsed message sent to builder
   - `builder.AddMessage(unsignedMsg)`

6. **Block Creation**: 
   - Builder creates block with message
   - Block proposed to all validators

7. **Consensus Synchronization**: 
   - **ALL validators** receive block
   - **ALL validators** call `chain.Accept()`
   - Each validator detects external message
   - Each validator stores in local database

8. **Database Storage**: 
   - `SetReceivedMessage()` called on every node
   - Message now queryable via `getAllReceivedMessages()`

---

## Data Structures

### AddressedCall Format

```
Byte Layout:
┌────────────┬────────────┬─────────────────┬──────────────────┐
│  CodecID   │  TypeID    │  SourceAddress  │     Payload      │
│  (2 bytes) │ (4 bytes)  │   (32 bytes)    │   (variable)     │
└────────────┴────────────┴─────────────────┴──────────────────┘
  0x0000       0x00000000    0x00...(32)       "Hello World!"

Example:
0x0000                           // CodecID
  00000000                       // TypeID (AddressedCall)
  1234567890abcdef...            // Source address (32 bytes)
  48656c6c6f20576f726c6421       // Payload: "Hello World!"
```

### Warp Message Structure

```
Unsigned Warp Message:
┌─────────────┬─────────────┬──────────────┬──────────────┐
│  NetworkID  │ SourceChain │ AddressedCall│   Payload    │
│  (4 bytes)  │  (32 bytes) │   Format     │  (variable)  │
└─────────────┴─────────────┴──────────────┴──────────────┘

Signed Warp Message:
┌──────────────────┬─────────────────────────┐
│ Unsigned Message │  BLS Signature (96 B)   │
└──────────────────┴─────────────────────────┘
```

### Block Structure

```go
type Block struct {
    ParentHash   ids.ID              // Previous block hash
    Height       uint64              // Block number
    Timestamp    int64               // Unix timestamp
    Messages     []ids.ID            // Message IDs in this block
    WarpMessages map[string][]byte   // MessageID -> unsigned bytes
}
```

---

## API Reference

### JSON-RPC Endpoints

Base URL: `http://localhost:9650/ext/bc/{BLOCKCHAIN_ID}/rpc`

#### 1. submitMessage

**Description**: Send a message to another blockchain.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.submitMessage",
  "params": {
    "destinationChainID": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
    "destinationAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
    "payload": "Hello from WarpCustomVM!"
  }
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2Kq8N...",
    "unsignedMessage": "0x00000001..."
  }
}
```

#### 2. receiveWarpMessage

**Description**: Receive a message from another blockchain (called by ICM relayer).

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.receiveWarpMessage",
  "params": {
    "signedMessage": "0x000000017fc93d85..."
  }
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "3Xy9P...",
    "sourceChainID": "yH8D7...",
    "txId": "3Xy9P...",
    "success": true
  }
}
```

#### 3. getAllReceivedMessages

**Description**: Get all messages received from other blockchains.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getAllReceivedMessages",
  "params": {}
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messages": [
      {
        "messageID": "3Xy9P...",
        "sourceChainID": "yH8D7...",
        "sourceAddress": "0x742d35Cc...",
        "payload": "Hello from C-Chain!",
        "receivedAt": 1701234567,
        "blockHeight": 42,
        "signedMessage": "0x00000001...",
        "unsignedMessage": "0x00000001..."
      }
    ]
  }
}
```

#### 4. getLatestBlock

**Description**: Get the latest accepted block with full message details.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getLatestBlock",
  "params": {}
}
```

**Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockID": "2Kq8N...",
    "parentID": "1Jp7M...",
    "height": 42,
    "timestamp": 1701234567,
    "messages": [
      {
        "messageID": "3Xy9P...",
        "networkID": 1,
        "sourceChainID": "yH8D7...",
        "sourceAddress": "0x742d35Cc...",
        "payload": "Hello World!",
        "unsignedMessage": "0x00000001...",
        "metadata": {
          "timestamp": 1701234567,
          "blockNumber": 42,
          "blockHash": "2Kq8N..."
        }
      }
    ]
  }
}
```

#### 5. getBlock

**Description**: Get a specific block by height.

**Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getBlock",
  "params": {
    "height": 42
  }
}
```

**Response**: Same as `getLatestBlock`

---

## Consensus Integration

### Why Consensus Matters for Message Synchronization

In a distributed system with multiple validators, **consensus ensures all nodes have identical state**. For cross-chain messaging, this means:

1. **Outgoing messages**: All validators must agree which messages were sent
2. **Incoming messages**: All validators must store the same received messages
3. **Block consistency**: All validators must have identical block content

### How Consensus Works in WarpCustomVM

#### Snowman Consensus

WarpCustomVM uses **Snowman consensus** (linear blockchain):
- Validators vote on proposed blocks
- Blocks are accepted when majority agrees
- All validators execute `Accept()` on accepted blocks

#### Message Synchronization Flow

```
Validator 1 (Proposer):
  1. Receives message via API
  2. Builder creates block with message
  3. Proposes block to network
  
Validator 2 & 3 (Voters):
  4. Receive block proposal
  5. Verify block is valid
  6. Vote to accept block
  
All Validators (After Consensus):
  7. Call chain.Accept(block)
  8. Detect external messages
  9. Store in local database
  10. Now all have identical message state
```

#### Critical Code in chain.go

```go
func (c *chain) Accept(ctx context.Context, b *xblock.Block) error {
    // This runs on ALL validators when block is accepted
    
    for _, msg := range b.Messages {
        // Check if message is from external chain
        if msg.SourceChainID != c.ctx.ChainID {
            // This is a received message - store it!
            receivedMsg := &state.ReceivedMessage{...}
            state.SetReceivedMessage(c.db, receivedMsg)
        }
    }
    
    // Update block header in database
    state.SetLatestBlockHeader(c.db, blockHeader)
}
```

**Why this matters**:
- Without this logic, only the node receiving the API call would have the message
- With this logic, **every validator** stores the message during block acceptance
- Result: `getAllReceivedMessages()` returns same data on all nodes

---

## Security Considerations

### 1. Signature Verification
- All incoming Warp messages are signed by source chain validators
- BLS signature aggregation ensures authenticity
- WarpCustomVM verifies signatures before accepting messages

### 2. Consensus-Based Validation
- Messages only become "official" after block acceptance
- Majority of validators must agree on block content
- Prevents single-node manipulation

### 3. Source Chain Verification
- Each message includes `sourceChainID`
- VM can verify message origin
- Applications can implement allow/deny lists

### 4. Database Integrity
- All state changes go through consensus
- Database writes only happen after block acceptance
- Ensures data consistency across validators

---

## Performance Characteristics

### Block Time
- **Default**: 1 second
- **Configurable**: Can be adjusted in builder configuration

### Message Throughput
- **Per Block**: Unlimited (constrained by block size)
- **Practical Limit**: ~1000 messages per second (network dependent)

### Latency
- **Outgoing Message**: 1-2 seconds (block creation + consensus)
- **Signature Collection**: 2-5 seconds (ACP-118 protocol)
- **Incoming Message**: 1-2 seconds (block creation + consensus)
- **Total Cross-Chain**: 5-10 seconds (source → destination)

### Storage
- **Per Message**: ~200 bytes (header) + payload size
- **Database**: LevelDB or PebbleDB (configurable)
- **Scalability**: Handles millions of messages

---

## Troubleshooting Guide

### Issue: Messages not syncing across validators

**Symptom**: `getAllReceivedMessages()` returns different results on different nodes

**Solution**: 
- Verify `chain.Accept()` logic is storing messages
- Check logs for "External message received" entries
- Ensure all validators are running same VM version

### Issue: ICM Relayer not delivering messages

**Symptom**: Messages sent but never received

**Solution**:
- Verify ACP-118 handler is registered: Check logs for "ACP-118 Warp signature handler registered"
- Ensure `getWarpMessage` API returns unsigned message
- Check relayer configuration for correct source/destination chains

### Issue: "Invalid signature" errors

**Symptom**: Messages rejected with signature errors

**Solution**:
- Verify message format matches AddressedCall structure
- Check that source chain validators are online and signing
- Ensure network IDs match between chains

---

## Development Guide

### Adding New Message Types

1. Define message structure in `state/storage.go`
2. Add parsing logic in `api/receive_handlers.go`
3. Update `chain.Accept()` to handle new type
4. Add API endpoint in `api/server.go`

### Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
cd scripts
./run_test.sh

# Test cross-chain messaging
./test_cross_chain.sh
```

### Building

```bash
# Build VM binary
./scripts/build.sh

# Install to Avalanche node
cp build/warpcustomvm ~/.avalanchego/plugins/

# Restart node
systemctl restart avalanchego
```

---

## Glossary

- **AWM**: Avalanche Warp Messaging - Native cross-chain communication protocol
- **ACP-118**: Avalanche Community Proposal 118 - Warp signature aggregation protocol
- **ICM Relayer**: Interchain Messaging Relayer - Automatic message delivery service
- **AddressedCall**: Message format wrapping payload with source address
- **BLS Signature**: Boneh-Lynn-Shacham signature - Aggregatable signature scheme
- **Snowman**: Linear blockchain consensus protocol used by Avalanche
- **ChainVM**: Avalanche interface for custom virtual machines

---

## Further Reading

- [Avalanche Warp Messaging Documentation](https://docs.avax.network/cross-chain/avalanche-warp-messaging/overview)
- [ACP-118 Specification](https://github.com/avalanche-foundation/ACPs/tree/main/ACPs/118-warp-signature-request)
- [ICM Relayer Setup](https://github.com/ava-labs/awm-relayer)
- [Snowman Consensus](https://docs.avax.network/learn/avalanche/avalanche-consensus)
