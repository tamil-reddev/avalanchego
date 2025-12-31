# WarpCustomVM Quick Start Guide

This guide will help you understand and use WarpCustomVM in 15 minutes.

## What is WarpCustomVM?

WarpCustomVM is a **custom Avalanche blockchain** that can **send and receive messages** to/from other blockchains (like C-Chain, X-Chain, or other Subnets) using Avalanche Warp Messaging.

**Think of it like email for blockchains**: Your blockchain can send messages to other blockchains, and they can send messages back!

## Key Concepts (Simple Explanations)

### 1. Avalanche Warp Messaging (AWM)
- **What**: Native Avalanche protocol for cross-chain communication
- **How**: Messages are signed by validators and delivered automatically
- **Why**: Secure, fast, and built into Avalanche

### 2. ICM Relayer
- **What**: A background service that delivers messages between blockchains
- **How**: Monitors blockchains, collects signatures, delivers messages
- **Why**: You don't need to manually deliver messages!

### 3. Consensus Synchronization
- **What**: All validator nodes have identical data
- **How**: Messages are stored during block acceptance
- **Why**: No matter which node you query, you get the same answer

## Architecture in One Picture

```
┌──────────────┐                    ┌──────────────┐
│   C-Chain    │                    │ WarpCustomVM │
│              │                    │              │
│  Contract    │  ──── Message ──►  │  Receives    │
│  Sends Msg   │                    │  & Stores    │
│              │                    │              │
│              │  ◄─── Message ────  │  Can Send    │
│              │                    │  Messages    │
└──────────────┘                    └──────────────┘
       ▲                                    ▲
       │                                    │
       └──────── ICM Relayer ──────────────┘
                (Auto-delivers)
```

## How It Works (Step by Step)

### Sending a Message (WarpCustomVM → C-Chain)

1. **You call API**: `submitMessage(destination, address, payload)`
2. **VM creates message**: Wraps your payload in Warp format
3. **Message goes in block**: Block builder includes message
4. **Validators agree**: Consensus accepts the block
5. **Validators sign**: Each validator signs the message
6. **Relayer delivers**: ICM relayer sends to C-Chain
7. **Done!** 

**Time**: ~5-10 seconds total

### Receiving a Message (C-Chain → WarpCustomVM)

1. **C-Chain sends**: Contract calls Warp precompile
2. **C-Chain validators sign**: Message gets signatures
3. **Relayer detects**: ICM relayer sees the message
4. **Relayer delivers**: Calls `receiveWarpMessage` API
5. **VM creates block**: Message included in new block
6. **All validators store**: Every node saves the message
7. **Done!**  Message queryable on all nodes

**Time**: ~5-10 seconds total

## Core APIs (5 Methods)

### 1. submitMessage - Send a message

```bash
curl -X POST http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "warpcustomvm.submitMessage",
    "params": {
      "destinationChainID": "0x7fc93d85...",
      "destinationAddress": "0x742d35Cc...",
      "payload": "Hello C-Chain!"
    },
    "id": 1
  }'
```

**Returns**: `messageID` and `unsignedMessage` (hex)

### 2. receiveWarpMessage - Receive a message (called by relayer)

```bash
curl -X POST http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "warpcustomvm.receiveWarpMessage",
    "params": {
      "signedMessage": "0x000000017fc93d85..."
    },
    "id": 1
  }'
```

**Returns**: `messageID`, `sourceChainID`, `txId`, `success`

### 3. getAllReceivedMessages - Get all messages

```bash
curl -X POST http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "warpcustomvm.getAllReceivedMessages",
    "params": {},
    "id": 1
  }'
```

**Returns**: Array of messages with:
- `messageID`
- `sourceChainID`
- `sourceAddress` (hex string)
- `payload` (plain text!)
- `receivedAt` (timestamp)
- `blockHeight`
- `signedMessage` (hex)
- `unsignedMessage` (hex)

### 4. getLatestBlock - Get latest block

```bash
curl -X POST http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "warpcustomvm.getLatestBlock",
    "params": {},
    "id": 1
  }'
```

**Returns**: Block with all messages (full details)

### 5. getBlock - Get specific block

```bash
curl -X POST http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "warpcustomvm.getBlock",
    "params": {
      "height": 42
    },
    "id": 1
  }'
```

**Returns**: Block at height 42 with all messages

## Data Format (Simple)

### Message Payload
**Plain text** - No encoding! Just send "Hello World!" and it stays "Hello World!"

### Addresses
**Hex strings** - Example: `0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb`

### Unsigned/Signed Messages
**Hex strings** - These are the raw Warp message bytes for validators to sign

### Block Height
**Simple number** - Block 1, Block 2, Block 3...

## Common Use Cases

### 1. Cross-Chain Notifications
```
C-Chain: "Transaction completed!"
   │
   └──► WarpCustomVM receives notification
        └──► Your app queries and displays it
```

### 2. Subnet Coordination
```
Subnet A: "Job started"
   │
   └──► Subnet B receives and processes
```

### 3. Oracle Data
```
Oracle Chain: "BTC price = $45,000"
   │
   └──► WarpCustomVM receives and stores
```

### 4. Multi-Chain Voting
```
Governance Chain: "Proposal #5 passed"
   │
   └──► All subnets receive and execute
```

## File Structure (What's What)

```
warpcustomvm/
├── vm.go                    # Main VM entry point
├── api/
│   ├── server.go           # API method implementations
│   ├── client.go           # Request/response types
│   └── receive_handlers.go # Message receiving logic
├── builder/                # Block creation
├── chain/
│   └── chain.go           # Consensus & block acceptance
├── state/
│   └── storage.go         # Database operations
├── contracts/
│   └── DirectWarpSender.sol # C-Chain sender contract
├── ARCHITECTURE.md         # This documentation!
└── DIAGRAMS.md            # Visual diagrams
```

## Key Code Locations

### Where messages are sent
**File**: `api/server.go` → `SubmitMessage()`
- Creates unsigned Warp message
- Sends to builder

### Where messages are received
**File**: `api/receive_handlers.go` → `ReceiveWarpMessage()`
- Parses signed message
- Sends to builder

### Where consensus happens
**File**: `chain/chain.go` → `Accept()`
- Called on ALL validators
- Stores received messages
- **This is why all nodes sync!**

### Where data is stored
**File**: `state/storage.go`
- `SetReceivedMessage()` - Store message
- `GetReceivedMessage()` - Retrieve message
- `GetAllReceivedMessageIDs()` - List all messages

## Understanding Consensus Sync

**Problem**: If only one node receives the message, how do others get it?

**Solution**: Messages go in blocks, blocks are accepted by all validators!

```
Node 1 receives message
     │
     └──► Creates block with message
            │
            └──► All validators vote
                   │
                   └──► Block accepted
                          │
                          └──► chain.Accept() runs on EVERY node
                                 │
                                 └──► Each node stores message
                                        │
                                        └──► All nodes have it!
```

**Code Location**: `chain/chain.go` line ~100

```go
func (c *chain) Accept(ctx context.Context, b *xblock.Block) error {
    for _, msg := range b.Messages {
        if msg.SourceChainID != c.ctx.ChainID {
            // External message - store it!
            state.SetReceivedMessage(c.db, receivedMsg)
        }
    }
}
```

## Security Features

### 1. BLS Signature Verification
- Every message signed by validators
- Signatures aggregated (efficient!)
- VM verifies before accepting

### 2. Consensus-Based Storage
- Messages only stored after block acceptance
- Majority of validators must agree
- No single point of failure

### 3. Source Chain Verification
- Each message has `sourceChainID`
- You can implement allow/deny lists
- Prevents unauthorized messages

## Performance

### Latency
- **Single blockchain**: ~1-2 seconds (block time)
- **Cross-chain**: ~5-10 seconds (includes signing)

### Throughput
- **Messages per block**: Unlimited (within block size limit)
- **Practical**: ~1000 messages/second

### Storage
- **Per message**: ~200 bytes + payload size
- **Scalable**: Millions of messages supported

## Troubleshooting

### "Messages not syncing across validators"
**Check**: `chain.Accept()` logic
**Look for**: "External message received" in logs
**Solution**: Ensure all nodes run same VM version

### "Relayer not delivering"
**Check**: ACP-118 handler registered
**Look for**: "ACP-118 Warp signature handler registered"
**Solution**: Verify relayer config

### "Invalid signature errors"
**Check**: Message format (AddressedCall)
**Solution**: Ensure source chain validators are online

## Example Workflow

### Complete Send-Receive Cycle

1. **Terminal 1**: Start your blockchain
   ```bash
   avalanche subnet deploy mysubnet
   ```

2. **Terminal 2**: Send a message
   ```bash
   curl -X POST http://localhost:9650/ext/bc/CHAIN/rpc \
     -d '{"method":"warpcustomvm.submitMessage",...}'
   ```

3. **Terminal 3**: Check message sent
   ```bash
   # Message ID returned: 2Kq8N...
   ```

4. **ICM Relayer**: Automatically delivers (background)

5. **Terminal 2**: Check received on other chain
   ```bash
   curl -X POST http://localhost:9650/ext/bc/OTHER_CHAIN/rpc \
     -d '{"method":"warpcustomvm.getAllReceivedMessages",...}'
   ```

6. **Result**: Message appears!
   ```json
   {
     "messages": [{
       "messageID": "2Kq8N...",
       "sourceChainID": "yH8D7...",
       "payload": "Hello!",
       ...
     }]
   }
   ```

## Next Steps

### Learn More
1. Read [ARCHITECTURE.md](./ARCHITECTURE.md) for deep dive
2. View [DIAGRAMS.md](./DIAGRAMS.md) for visual explanations
3. Check [Avalanche Warp Messaging Docs](https://docs.avax.network/cross-chain/avalanche-warp-messaging/overview)

### Try It Yourself
1. Deploy a subnet with WarpCustomVM
2. Send a test message using `submitMessage`
3. Query with `getAllReceivedMessages`
4. Set up ICM relayer for cross-chain messaging

### Customize
1. Modify `api/server.go` to add custom logic
2. Add new message types in `state/storage.go`
3. Create custom handlers in `api/receive_handlers.go`

## Key Takeaways

 **Simple**: 5 core APIs, plain text payloads
 **Secure**: BLS signatures, consensus-based storage
 **Automatic**: ICM relayer handles delivery
 **Synced**: All validators have identical state
 **Fast**: ~5-10 seconds cross-chain

## Support

- **GitHub Issues**: Report bugs or ask questions
- **Avalanche Discord**: Community support
- **Documentation**: [docs.avax.network](https://docs.avax.network)

---

**Ready to build?** Start with `submitMessage` and see your first cross-chain message! 
