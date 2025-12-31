# WarpCustomVM API Reference

Complete API documentation for all JSON-RPC endpoints.

## Table of Contents
- [Connection](#connection)
- [Authentication](#authentication)
- [Message APIs](#message-apis)
- [Block APIs](#block-apis)
- [Query APIs](#query-apis)
- [Error Codes](#error-codes)
- [Code Examples](#code-examples)

---

## Connection

### Base URL
```
http://{NODE_IP}:{PORT}/ext/bc/{BLOCKCHAIN_ID}/rpc
```

**Default Port**: `9650`

**Example**:
```
http://localhost:9650/ext/bc/2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ/rpc
```

### Headers
```
Content-Type: application/json
```

### Request Format
All requests use JSON-RPC 2.0 format:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.methodName",
  "params": {
    "param1": "value1",
    "param2": "value2"
  }
}
```

### Response Format
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

### Error Response
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32000,
    "message": "Error description"
  }
}
```

---

## Authentication

**Current**: No authentication required

**Production**: Implement authentication at reverse proxy level (nginx, API gateway)

---

## Message APIs

### submitMessage

Send a message to another blockchain.

**Method**: `warpcustomvm.submitMessage`

**Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `destinationChainID` | `ids.ID` (hex string) | Yes | Target blockchain ID |
| `destinationAddress` | `string` (hex) | Yes | Recipient address (0x prefixed) |
| `payload` | `string` | Yes | Message content (plain text) |

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `messageID` | `ids.ID` | Unique message identifier |
| `unsignedMessage` | `string` (hex) | Unsigned Warp message bytes |

**Example Request**:
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

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ",
    "unsignedMessage": "0x000000017fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5..."
  }
}
```

**Process Flow**:
1. API creates unsigned Warp message
2. Wraps payload in AddressedCall format
3. Adds message to builder queue
4. Returns immediately with message ID
5. Message included in next block (~1 second)
6. Validators sign message (ACP-118)
7. ICM relayer delivers to destination

**Error Cases**:
- Invalid destination chain ID
- Invalid destination address format
- Payload too large (> 1MB)
- Builder queue full

---

### receiveWarpMessage

Receive a message from another blockchain (typically called by ICM relayer).

**Method**: `warpcustomvm.receiveWarpMessage`

**Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `signedMessage` | `string` (hex) | Yes | Signed Warp message (with or without 0x prefix) |

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `messageID` | `ids.ID` | Unique message identifier |
| `sourceChainID` | `ids.ID` | Source blockchain ID |
| `txId` | `ids.ID` | Transaction ID (same as messageID) |
| `success` | `boolean` | Whether message was accepted |

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.receiveWarpMessage",
  "params": {
    "signedMessage": "0x000000017fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5..."
  }
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u",
    "sourceChainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
    "txId": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u",
    "success": true
  }
}
```

**Process Flow**:
1. API parses signed message
2. Verifies BLS signatures
3. Extracts AddressedCall payload
4. Adds to builder queue
5. Returns txId immediately
6. Message included in next block (~1 second)
7. All validators store message during block acceptance

**Error Cases**:
- Invalid message format
- Signature verification failed
- Message already received
- Builder queue full

---

## Query APIs

### getAllReceivedMessages

Get all messages received from other blockchains.

**Method**: `warpcustomvm.getAllReceivedMessages`

**Parameters**: None

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `messages` | `Array<ReceivedMessage>` | List of all received messages |

**ReceivedMessage Structure**:

| Field | Type | Description |
|-------|------|-------------|
| `messageID` | `ids.ID` | Unique message identifier |
| `sourceChainID` | `ids.ID` | Source blockchain ID |
| `sourceAddress` | `string` (hex) | Sender address |
| `payload` | `string` | Message content (plain text) |
| `receivedAt` | `int64` | Unix timestamp (seconds) |
| `blockHeight` | `uint64` | Block number when received |
| `signedMessage` | `string` (hex) | Full signed Warp message |
| `unsignedMessage` | `string` (hex) | Unsigned Warp message |

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getAllReceivedMessages",
  "params": {}
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messages": [
      {
        "messageID": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u",
        "sourceChainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
        "sourceAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
        "payload": "Hello from C-Chain!",
        "receivedAt": 1701234567,
        "blockHeight": 42,
        "signedMessage": "0x000000017fc93d85...",
        "unsignedMessage": "0x000000017fc93d85..."
      },
      {
        "messageID": "4Zw8Q7mK6oN3RvS5uC8kD9fE2hG3jJ4lK5nM6pO7qR8sT9uV0",
        "sourceChainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
        "sourceAddress": "0x8f3Cf7ad23Cd3CaDbD9735AFf958023239c6A063",
        "payload": "Another message!",
        "receivedAt": 1701234600,
        "blockHeight": 45,
        "signedMessage": "0x000000017fc93d85...",
        "unsignedMessage": "0x000000017fc93d85..."
      }
    ]
  }
}
```

**Notes**:
- Messages ordered by received time (oldest first)
- Same result on all validator nodes (consensus-synced)
- Payload is plain text (no decoding needed)

---

### getReceivedMessage

Get a specific received message by ID.

**Method**: `warpcustomvm.getReceivedMessage`

**Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `messageID` | `ids.ID` | Yes | Message identifier |

**Returns**: Single `ReceivedMessage` (same structure as `getAllReceivedMessages`)

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getReceivedMessage",
  "params": {
    "messageID": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u"
  }
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u",
    "sourceChainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
    "sourceAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
    "payload": "Hello from C-Chain!",
    "receivedAt": 1701234567,
    "blockHeight": 42,
    "signedMessage": "0x000000017fc93d85...",
    "unsignedMessage": "0x000000017fc93d85..."
  }
}
```

**Error Cases**:
- Message ID not found

---

## Block APIs

### getLatestBlock

Get the most recently accepted block with full message details.

**Method**: `warpcustomvm.getLatestBlock`

**Parameters**: None

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `blockID` | `ids.ID` | Block hash |
| `parentID` | `ids.ID` | Previous block hash |
| `height` | `uint64` | Block number |
| `timestamp` | `int64` | Unix timestamp (seconds) |
| `messages` | `Array<MessageDetail>` | Messages in this block |

**MessageDetail Structure**:

| Field | Type | Description |
|-------|------|-------------|
| `messageID` | `ids.ID` | Message identifier |
| `networkID` | `uint32` | Avalanche network ID |
| `sourceChainID` | `ids.ID` | Source blockchain ID |
| `sourceAddress` | `string` (hex) | Sender address |
| `payload` | `string` | Message content (plain text) |
| `unsignedMessage` | `string` (hex) | Unsigned Warp message |
| `metadata` | `MessageMetadata` | Block context |

**MessageMetadata Structure**:

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | `int64` | Block timestamp |
| `blockNumber` | `uint64` | Block height |
| `blockHash` | `ids.ID` | Block hash |

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getLatestBlock",
  "params": {}
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "blockID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ",
    "parentID": "1Jp7M6wXt3GKJ3pVu9Qyz2F3N4r8E5h6Y7k8L9m0N1o2P3q4R5",
    "height": 42,
    "timestamp": 1701234567,
    "messages": [
      {
        "messageID": "3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u",
        "networkID": 1,
        "sourceChainID": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
        "sourceAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
        "payload": "Hello World!",
        "unsignedMessage": "0x000000017fc93d85...",
        "metadata": {
          "timestamp": 1701234567,
          "blockNumber": 42,
          "blockHash": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ"
        }
      }
    ]
  }
}
```

---

### getBlock

Get a specific block by height.

**Method**: `warpcustomvm.getBlock`

**Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `height` | `uint64` | Yes | Block number (0-indexed) |

**Returns**: Same as `getLatestBlock`

**Example Request**:
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

**Error Cases**:
- Block height not found (too high)

---

### getChainID

Get the blockchain ID and network ID.

**Method**: `warpcustomvm.getChainID`

**Parameters**: None

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `chainID` | `ids.ID` | Blockchain identifier |
| `networkID` | `uint32` | Network identifier (1=mainnet, 5=fuji) |

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getChainID",
  "params": {}
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "chainID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ",
    "networkID": 1
  }
}
```

---

### getWarpMessage

Get unsigned Warp message for signature collection (used by ICM relayer).

**Method**: `warpcustomvm.getWarpMessage`

**Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `messageID` | `ids.ID` | Yes | Message identifier |

**Returns**:

| Field | Type | Description |
|-------|------|-------------|
| `messageID` | `ids.ID` | Message identifier |
| `unsignedMessage` | `string` (hex) | Unsigned Warp message bytes |
| `sourceChainID` | `ids.ID` | Source blockchain ID |
| `destinationChain` | `ids.ID` | Destination blockchain ID |
| `destinationAddress` | `string` (hex) | Recipient address |

**Example Request**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getWarpMessage",
  "params": {
    "messageID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ"
  }
}
```

**Example Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "messageID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ",
    "unsignedMessage": "0x000000017fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5...",
    "sourceChainID": "2Kq8N7vYu4FLK4qbXvxKnxFj8VbH4PxZdE3k5hMrg5dYLXwGfZ",
    "destinationChain": "yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp",
    "destinationAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
  }
}
```

**Notes**:
- Required by ICM relayer for signature collection
- Message must be in an accepted block
- Returns raw unsigned bytes for validators to sign

---

## Error Codes

| Code | Message | Description |
|------|---------|-------------|
| `-32700` | Parse error | Invalid JSON |
| `-32600` | Invalid Request | Missing required fields |
| `-32601` | Method not found | Unknown method name |
| `-32602` | Invalid params | Parameter type mismatch |
| `-32603` | Internal error | Server error |
| `-32000` | Message not found | Message ID doesn't exist |
| `-32001` | Invalid signature | Signature verification failed |
| `-32002` | Invalid format | Message format incorrect |
| `-32003` | Block not found | Block height doesn't exist |
| `-32004` | Queue full | Builder queue at capacity |

**Example Error Response**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32000,
    "message": "message not found: 3Xy9P8kL5nMz2QvR6tB4jC7wD9eF1gH2iJ3kL4mN5oP6qR7sT8u"
  }
}
```

---

## Code Examples

### JavaScript / Node.js

```javascript
const axios = require('axios');

const RPC_URL = 'http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc';

// Submit a message
async function submitMessage(destChainID, destAddress, payload) {
  const response = await axios.post(RPC_URL, {
    jsonrpc: '2.0',
    id: 1,
    method: 'warpcustomvm.submitMessage',
    params: {
      destinationChainID: destChainID,
      destinationAddress: destAddress,
      payload: payload
    }
  });
  
  return response.data.result;
}

// Get all received messages
async function getAllReceivedMessages() {
  const response = await axios.post(RPC_URL, {
    jsonrpc: '2.0',
    id: 1,
    method: 'warpcustomvm.getAllReceivedMessages',
    params: {}
  });
  
  return response.data.result.messages;
}

// Usage
submitMessage(
  '0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5',
  '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb',
  'Hello from Node.js!'
).then(result => {
  console.log('Message ID:', result.messageID);
});

getAllReceivedMessages().then(messages => {
  messages.forEach(msg => {
    console.log(`From ${msg.sourceChainID}: ${msg.payload}`);
  });
});
```

### Python

```python
import requests
import json

RPC_URL = 'http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc'

def submit_message(dest_chain_id, dest_address, payload):
    response = requests.post(RPC_URL, json={
        'jsonrpc': '2.0',
        'id': 1,
        'method': 'warpcustomvm.submitMessage',
        'params': {
            'destinationChainID': dest_chain_id,
            'destinationAddress': dest_address,
            'payload': payload
        }
    })
    return response.json()['result']

def get_all_received_messages():
    response = requests.post(RPC_URL, json={
        'jsonrpc': '2.0',
        'id': 1,
        'method': 'warpcustomvm.getAllReceivedMessages',
        'params': {}
    })
    return response.json()['result']['messages']

# Usage
result = submit_message(
    '0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5',
    '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb',
    'Hello from Python!'
)
print(f"Message ID: {result['messageID']}")

messages = get_all_received_messages()
for msg in messages:
    print(f"From {msg['sourceChainID']}: {msg['payload']}")
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

const RPC_URL = "http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc"

type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params"`
}

type SubmitMessageParams struct {
    DestinationChainID  string `json:"destinationChainID"`
    DestinationAddress  string `json:"destinationAddress"`
    Payload             string `json:"payload"`
}

func submitMessage(destChainID, destAddress, payload string) (string, error) {
    req := JSONRPCRequest{
        JSONRPC: "2.0",
        ID:      1,
        Method:  "warpcustomvm.submitMessage",
        Params: SubmitMessageParams{
            DestinationChainID: destChainID,
            DestinationAddress: destAddress,
            Payload:            payload,
        },
    }
    
    jsonData, _ := json.Marshal(req)
    resp, err := http.Post(RPC_URL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)
    
    return result["result"].(map[string]interface{})["messageID"].(string), nil
}

func main() {
    messageID, err := submitMessage(
        "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
        "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
        "Hello from Go!",
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("Message ID: %s\n", messageID)
}
```

### cURL (Bash)

```bash
#!/bin/bash

RPC_URL="http://localhost:9650/ext/bc/BLOCKCHAIN_ID/rpc"

# Submit message
curl -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "destinationChainID": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
      "destinationAddress": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
      "payload": "Hello from Bash!"
    }
  }'

# Get all received messages
curl -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.getAllReceivedMessages",
    "params": {}
  }' | jq '.result.messages'
```

---

## Rate Limiting

**Current**: No rate limiting

**Recommended Production Settings**:
- 100 requests per second per IP
- 1000 requests per minute per API key
- Implement at reverse proxy level

---

## Versioning

**Current Version**: 1.0.0

**API Stability**: All methods are stable except where marked "experimental"

**Breaking Changes**: Will increment major version (2.0.0)

---

## Support

- **Issues**: [GitHub Issues](https://github.com/ava-labs/avalanchego/issues)
- **Documentation**: [docs.avax.network](https://docs.avax.network)
- **Discord**: [Avalanche Discord](https://discord.gg/RwXY7P6)
