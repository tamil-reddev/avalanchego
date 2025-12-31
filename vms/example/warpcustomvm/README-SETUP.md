# Warp Custom VM Setup & Operations Guide

Complete guide for deploying and operating a Warp-enabled Custom VM on Avalanche Fuji Testnet with cross-chain messaging capabilities.

---

## Table of Contents
1. [Network Information](#1-network-information)
2. [Validator Nodes](#2-validator-nodes)
3. [C-Chain Contracts](#3-c-chain-contracts-fuji)
4. [Operational Commands](#4-operational-commands)
5. [Warp Message Operations](#5-warp-message-operations)
6. [ICM Relayer](#6-icm-relayer)
7. [Verification & Testing](#7-verification--testing)
8. [Troubleshooting](#8-troubleshooting)
9. [Architecture Notes](#9-architecture-notes)

---

## 1. Network Information

### Subnet Details
| Property | Value |
|----------|-------|
| **Subnet ID** | `2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND` |
| **Blockchain ID** | `2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ` |
| **VM ID** | `v3m4wPxaHpvGr8qfMeyK6PRW3idZrPHmYcMTt7oXdK47yurVH` |
| **Network** | Fuji Testnet |

### L1 Conversion
```
L1 Conversion Hash: rJUvxBaJexGUffBXb4mSWEqo33duGNFP69or97eoPx1mGw5Co
```

### Genesis Configuration
```json
{
  "timestamp": 0
}
```

---

## 2. Validator Nodes

### Node 1 (Primary Validator)

**Node Information:**
- **Node ID**: `NodeID-JbWCEn5JJLazVNrGwWhGVuzDAtZwBSPEA`
- **HTTP Port**: `9650`
- **Staking Port**: `9651`
- **Data Directory**: `~/.avalanchego`
- **Validator Weight**: 102
- **Public Key**: 
  ```
  0x92de6acef7599d82aea274db0f174981b48e9c669dc6ad281dbe1bce54cbc01ec3aa9fdf518c0384c3f88ce8600487e5
  ```

**Docker Deployment:**
```bash
docker run -it -d \
    --name avago1 \
    -p 9650:9650 -p 9651:9651 \
    -v ~/.avalanchego:/root/.avalanchego \
    -e AVAGO_PUBLIC_IP_RESOLUTION_SERVICE=opendns \
    -e AVAGO_HTTP_HOST=0.0.0.0 \
    -e AVAGO_PARTIAL_SYNC_PRIMARY_NETWORK=true \
    -e AVAGO_TRACK_SUBNETS=2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND \
    -e AVAGO_NETWORK_ID=fuji \
    -e AVAGO_HTTP_ALLOWED_HOSTS="*" \
    -e AVAGO_CHAIN_CONFIG_CONTENT=eyIyRWN4UG82QkhyOXhQY0hyakxTZ0hMenpNdFBWdUthS3pXdlNQREdKUWdMa0Z4VXpSUSI6eyJDb25maWciOiJleUp3Y25WdWFXNW5MV1Z1WVdKc1pXUWlPbVpoYkhObExDSnNiMmN0YkdWMlpXd2lPaUprWldKMVp5SXNJbmRoY25BdFlYQnBMV1Z1WVdKc1pXUWlPblJ5ZFdVc0ltVjBhQzFoY0dseklqcGJJbVYwYUNJc0ltVjBhQzFtYVd4MFpYSWlMQ0p1WlhRaUxDSmhaRzFwYmlJc0luZGxZak1pTENKcGJuUmxjbTVoYkMxbGRHZ2lMQ0pwYm5SbGNtNWhiQzFpYkc5amEyTm9ZV2x1SWl3aWFXNTBaWEp1WVd3dGRISmhibk5oWTNScGIyNGlMQ0pwYm5SbGNtNWhiQzFrWldKMVp5SXNJbWx1ZEdWeWJtRnNMV0ZqWTI5MWJuUWlMQ0pwYm5SbGNtNWhiQzF3WlhKemIyNWhiQ0lzSW1SbFluVm5JaXdpWkdWaWRXY3RkSEpoWTJWeUlpd2laR1ZpZFdjdFptbHNaUzEwY21GalpYSWlMQ0prWldKMVp5MW9ZVzVrYkdWeUlsMTkiLCJVcGdyYWRlIjpudWxsfX0= \
    -e AVAGO_VM_ALIASES_FILE_CONTENT=ewogICJ2M200d1B4YUhwdkdyOHFmTWV5SzZQUlczaWRaclBIbVljTVR0N29YZEs0N3l1clZIIjogWwogICAgInNyRVhpV2FIdWhOeUd3UFVpNDQ0VHU0N1pFRHd4VFdyYlFpdUQ3Rm1nU0FRNlg3RHkiCiAgXQp9 \
    avaplatform/avalanchego:v1.14.0
```

### Node 2 (Secondary Validator)

**Node Information:**
- **Node ID**: `NodeID-G6aEDpPB9H6Yvnw1TxrrBfoC63q4FqxgN`
- **HTTP Port**: `9652` (mapped to container port 9650)
- **Staking Port**: `9653` (mapped to container port 9651)
- **Data Directory**: `~/.avalanchego-node2`
- **Validator Weight**: 48
- **Public Key**: 
  ```
  0x8caa1ed629c26f9629692efe950f3e247dd51734e660d754e9054eb8dd64f9908babaee8734324e8ae5aa9dc61814998
  ```

**Docker Deployment:**
```bash
docker run -it -d \
    --name avago2 \
    -p 9652:9650 -p 9653:9651 \
    -v ~/.avalanchego-node2:/root/.avalanchego \
    -e AVAGO_PUBLIC_IP_RESOLUTION_SERVICE=opendns \
    -e AVAGO_HTTP_PORT=9650 \
    -e AVAGO_STAKING_PORT=9651 \
    -e AVAGO_HTTP_HOST=0.0.0.0 \
    -e AVAGO_PARTIAL_SYNC_PRIMARY_NETWORK=true \
    -e AVAGO_TRACK_SUBNETS=2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND \
    -e AVAGO_NETWORK_ID=fuji \
    -e AVAGO_HTTP_ALLOWED_HOSTS="*" \
    -e AVAGO_CHAIN_CONFIG_CONTENT=eyIyRWN4UG82QkhyOXhQY0hyakxTZ0hMenpNdFBWdUthS3pXdlNQREdKUWdMa0Z4VXpSUSI6eyJDb25maWciOiJleUp3Y25WdWFXNW5MV1Z1WVdKc1pXUWlPbVpoYkhObExDSnNiMmN0YkdWMlpXd2lPaUprWldKMVp5SXNJbmRoY25BdFlYQnBMV1Z1WVdKc1pXUWlPblJ5ZFdVc0ltVjBhQzFoY0dseklqcGJJbVYwYUNJc0ltVjBhQzFtYVd4MFpYSWlMQ0p1WlhRaUxDSmhaRzFwYmlJc0luZGxZak1pTENKcGJuUmxjbTVoYkMxbGRHZ2lMQ0pwYm5SbGNtNWhiQzFpYkc5amEyTm9ZV2x1SWl3aWFXNTBaWEp1WVd3dGRISmhibk5oWTNScGIyNGlMQ0pwYm5SbGNtNWhiQzFrWldKMVp5SXNJbWx1ZEdWeWJtRnNMV0ZqWTI5MWJuUWlMQ0pwYm5SbGNtNWhiQzF3WlhKemIyNWhiQ0lzSW1SbFluVm5JaXdpWkdWaWRXY3RkSEpoWTJWeUlpd2laR1ZpZFdjdFptbHNaUzEwY21GalpYSWlMQ0prWldKMVp5MW9ZVzVrYkdWeUlsMTkiLCJVcGdyYWRlIjpudWxsfX0= \
    -e AVAGO_VM_ALIASES_FILE_CONTENT=ewogICJ2M200d1B4YUhwdkdyOHFmTWV5SzZQUlczaWRaclBIbVljTVR0N29YZEs0N3l1clZIIjogWwogICAgInNyRVhpV2FIdWhOeUd3UFVpNDQ0VHU0N1pFRHd4VFdyYlFpdUQ3Rm1nU0FRNlg3RHkiCiAgXQp9 \
    avaplatform/avalanchego:v1.14.0
```

### Validator Summary
- **Total Stake Weight**: 150
- **Consensus Threshold**: >50% (76 required)
- **Both validators must participate** for block acceptance

---

## 3. C-Chain Contracts (Fuji)

### Teleporter Infrastructure

| Contract | Address |
|----------|---------|
| **VM Library** | `0x7ce725e835447f347ffe4f5e422d8e3c5f327e00` |
| **VMC (Controller)** | `0xa80d3621f40f70ae9513983344630dd395ab1b8d` |
| **Proxy Admin** | `0xe14c72720b5a2aa51974325cd55bf91aa5a72026` |
| **Transparent Proxy** | `0xc47234a3906c1052f97c423bb36902e27358caaf` |
| **Teleporter Precompile** | `0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf` |

### Message Receiver Contracts

| Contract | Address |
|----------|---------|
| **ReceiverOnSubnet** | `0x772eb420B677F0c42Dc1aC503D03E02E92ae1502` |
| **C-Chain Blockchain ID** | `0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5` |

### Contract Deployment Commands

#### Deploy ReceiverOnSubnet
```bash
forge create --broadcast --rpc-url fuji-c --private-key $PK \
  contracts/interchain-messaging/send-receive/receiverOnSubnet.sol:ReceiverOnSubnet \
  --constructor-args $FUJI_C_CHAIN_BLOCKCHAIN_ID_HEX
```

**Expected Output:**
```
Deployed to: 0x772eb420B677F0c42Dc1aC503D03E02E92ae1502
```

#### Deploy WarpMessageReceiver
```bash
forge create --broadcast --rpc-url fuji-c --private-key $PK \
  contracts/interchain-messaging/send-receive/WarpMessageReceiver.sol:WarpMessageReceiver \
  --constructor-args $FUJI_C_CHAIN_BLOCKCHAIN_ID_HEX
```

---

## 4. Operational Commands

### Check Node ID

**Node 1:**
```bash
curl -X POST --data '{
  "jsonrpc":"2.0",
  "id":1,
  "method":"info.getNodeID"
}' -H "content-type:application/json;" http://127.0.0.1:9650/ext/info
```

**Node 2:**
```bash
curl -X POST --data '{
  "jsonrpc":"2.0",
  "id":1,
  "method":"info.getNodeID"
}' -H "content-type:application/json;" http://127.0.0.1:9652/ext/info
```

### Check Bootstrap Status

```bash
curl -k -X POST --data '{
  "jsonrpc": "2.0",
  "method": "info.isBootstrapped",
  "params": {
    "chain": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ"
  },
  "id": 1
}' -H 'content-type:application/json;' http://localhost:9650/ext/info
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "isBootstrapped": true
  },
  "id": 1
}
```

### Get Current Validators

```bash
curl -X POST --data '{
  "jsonrpc":"2.0",
  "id": 1,
  "method": "platform.getCurrentValidators",
  "params": {
    "subnetID": "2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND"
  }
}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/bc/P
```

---

## 5. Warp Message Operations

### Submit Warp Message

#### Via Node 1 (Primary)
```bash
curl -X POST --data '{
  "jsonrpc": "2.0",
  "method": "warpcustomvm.submitMessage",
  "params": {
    "destinationChain": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
    "destinationAddress": "0x772eb420B677F0c42Dc1aC503D03E02E92ae1502",
    "message": "hello world 9650 1"
  },
  "id": 1
}' -H 'content-type:application/json;' \
http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
```

#### Via Node 2 (Secondary)
```bash
curl -X POST --data '{
  "jsonrpc": "2.0",
  "method": "warpcustomvm.submitMessage",
  "params": {
    "destinationChain": "0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5",
    "destinationAddress": "0x772eb420B677F0c42Dc1aC503D03E02E92ae1502",
    "message": "hello world 9652 1"
  },
  "id": 1
}' -H 'content-type:application/json;' \
http://127.0.0.1:9652/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
  },
  "id": 1
}
```

### Query Latest Block

#### From Node 1
```bash
curl -k -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getLatestBlock",
  "params": {}
}' http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
```

#### From Node 2
```bash
curl -k -X POST -H "Content-Type: application/json" -d '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "warpcustomvm.getLatestBlock",
  "params": {}
}' http://127.0.0.1:9652/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
```

**Expected Response:**
```json
{
  "jsonrpc": "2.0",
  "result": {
    "blockNumber": 5,
    "blockHash": "...",
    "timestamp": 1700000000,
    "messages": [
      {
        "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1",
        "networkID": 5,
        "sourceChainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
        "sourceAddress": "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf",
        "payload": "0x...",
        "metadata": {
          "timestamp": 1700000000,
          "blockNumber": 5,
          "blockHash": "..."
        }
      }
    ]
  },
  "id": 1
}
```

### Query Specific Message

```bash
curl -X POST --data '{
  "jsonrpc": "2.0",
  "method": "warpcustomvm.getMessage",
  "params": {
    "messageID": "2HgU68fXLffrsXimkrSP4rmtSgEsbaCCNAvnH25JKCkQVXU1N1"
  },
  "id": 1
}' -H 'content-type:application/json;' \
http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
```

---

## 6. ICM Relayer

### Configuration File
Create `/home/tamil/pocWarp/icm-relayer-config-custom-cchain.json`:

```json
{
  "source-blockchains": [
    {
      "subnetID": "2U4eQ6cBdihtYCqSxaGfkJ65f6g4ScNEW5gjgHCDAqvSTMeWND",
      "blockchainID": "2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ",
      "vm": "srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy",
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

### Start Relayer

```bash
/home/tamil/pocWarp/reddev-icm-services/build/icm-relayer \
  --config-file /home/tamil/pocWarp/icm-relayer-config-custom-cchain.json
```

**Expected Output:**
```
INFO Starting ICM Relayer
INFO Monitoring source blockchain 2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ
INFO Connected to destination blockchain yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp
```

---

## 7. Verification & Testing

### Check Message on C-Chain

```bash
cast call --rpc-url fuji-c \
  0x772eb420B677F0c42Dc1aC503D03E02E92ae1502 \
  "lastMessage()(string)"
```

**Expected Output:**
```
hello world 9650 1
```

### Full End-to-End Test Flow

1. **Submit message on Custom VM:**
   ```bash
   curl -X POST --data '{"jsonrpc":"2.0","method":"warpcustomvm.submitMessage","params":{"destinationChain":"0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5","destinationAddress":"0x772eb420B677F0c42Dc1aC503D03E02E92ae1502","message":"test message 1"},"id":1}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
   ```

2. **Wait for block acceptance** (check both validator logs)

3. **Verify block on both nodes:**
   ```bash
   # Node 1
   curl -k -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"warpcustomvm.getLatestBlock","params":{}}' http://127.0.0.1:9650/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
   
   # Node 2
   curl -k -X POST -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","id":1,"method":"warpcustomvm.getLatestBlock","params":{}}' http://127.0.0.1:9652/ext/bc/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ/rpc
   ```

4. **Check relayer logs** for message delivery

5. **Verify on C-Chain:**
   ```bash
   cast call --rpc-url fuji-c 0x772eb420B677F0c42Dc1aC503D03E02E92ae1502 "lastMessage()(string)"
   ```

---

## 8. Troubleshooting

### Reset Blockchain Database

If you need to start fresh after code changes or corruption:

```bash
# Stop validators
docker stop avago1 avago2

# Delete blockchain databases (keeps node identity)
rm -rf ~/.avalanchego/db/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ
rm -rf ~/.avalanchego-node2/db/2EcxPo6BHr9xPcHrjLSgHLzzMtPVuKaKzWvSPDGJQgLkFxUzRQ

# Restart validators
docker start avago1 avago2
```

### Common Issues

#### Issue: "BuildBlock failed to get preferred block"
**Cause:** Old blocks in database without `WarpMessages` field  
**Solution:** Delete blockchain database and restart (see above)

#### Issue: Duplicate Teleporter Message IDs
**Cause:** Race condition when submitting to multiple nodes simultaneously  
**Solution:** 
- Submit messages to only ONE node (preferably Node 1)
- Or accept that Teleporter protocol handles duplicates gracefully on destination

#### Issue: Messages not syncing between validators
**Cause:** Consensus not reaching threshold or network partition  
**Solution:**
- Check validator logs for consensus messages
- Verify both validators are connected: `curl http://127.0.0.1:9650/ext/info -d '{"jsonrpc":"2.0","method":"info.peers","params":{},"id":1}'`
- Ensure total stake >= threshold (150 >= 76)

#### Issue: Relayer not picking up messages
**Cause:** Configuration mismatch or endpoint unavailable  
**Solution:**
- Verify RPC endpoints are accessible
- Check blockchain ID matches in config
- Ensure message contract address is correct
- Review relayer logs for specific errors

### Debug Logging

Enable debug logging by setting chain config:
```json
{
  "log-level": "debug",
  "warp-api-enabled": true
}
```

**Key log entries to monitor:**
- ` [API Server] Step X` - Message submission flow
- ` BuildBlock called` - Block building triggered
- ` WaitForEvent returning PendingTxs` - Consensus engine activated
- ` stored warp message from accepted block` - Message accepted
- ` updated global message ID counter` - Counter synchronized

---

## 9. Architecture Notes

### Consensus Architecture
- **Mechanism**: Snowman consensus
- **Validators**: 2 nodes
  - Node 1: Weight 102
  - Node 2: Weight 48
  - Total: 150
- **Threshold**: >50% (76+ required for finality)
- **Block Time**: Dynamic (triggered by pending messages)

### Message Flow Architecture

```
┌─────────────────┐
│   API Client    │
└────────┬────────┘
         │ submitMessage
         ▼
┌─────────────────┐
│  Custom VM API  │ ← Allocates Teleporter ID from consensus state
└────────┬────────┘
         │ AddMessage (to pending pool)
         ▼
┌─────────────────┐
│     Builder     │ ← WaitForEvent() returns PendingTxs
└────────┬────────┘
         │ BuildBlock (embeds WarpMessages in block)
         ▼
┌─────────────────┐
│   Consensus     │ ← Snowman: Verify → Accept
└────────┬────────┘
         │ Block propagates to all validators
         ▼
┌─────────────────┐
│  Chain.Accept() │ ← Extracts messages, updates counter
└────────┬────────┘
         │ Messages stored in acceptedState
         ▼
┌─────────────────┐
│   ICM Relayer   │ ← Queries via getLatestBlock
└────────┬────────┘
         │ Submits to destination chain
         ▼
┌─────────────────┐
│    C-Chain      │ ← ReceiverOnSubnet contract
└─────────────────┘
```

### Message Structure

**Warp Message Layers:**
1. **Warp Unsigned Message** (outermost)
   - Network ID: 5 (Fuji)
   - Source Chain ID: Custom VM blockchain ID
   - Payload: AddressedCall

2. **AddressedCall** (middle layer)
   - Source Address: Teleporter precompile (0x253b2784...)
   - Destination Address: Custom destination
   - Payload: Teleporter Message

3. **Teleporter Message** (innermost)
   - Message ID: Sequential counter
   - Sender Address: Source address
   - Destination Blockchain ID: C-Chain
   - Destination Address: Receiver contract
   - Required Gas Limit: 0
   - Allowed Relayer Addresses: []
   - Receipts: []
   - Message: User message bytes

### Storage Architecture

- **WarpMessages Map**: Embedded in `Block` struct, propagates via consensus
- **Message ID Counter**: Stored in acceptedState, increments atomically on block acceptance
- **Accepted Messages**: Stored in state database after block acceptance
- **Block Headers**: Include `Messages` (IDs) and `WarpMessages` (full bytes)

### Key Components

| Component | File | Responsibility |
|-----------|------|----------------|
| **API Server** | `api/server.go` | Message submission, ID allocation, queries |
| **Builder** | `builder/builder.go` | Block construction, pending message management |
| **Chain** | `chain/chain.go` | Block lifecycle (Verify/Accept/Reject), message extraction |
| **State** | `state/storage.go` | Database operations, counter management |
| **VM** | `vm.go` | Main VM interface, consensus integration |
| **Teleporter** | `api/teleporter/` | Message encoding (Warp/AddressedCall/Teleporter) |

### Race Condition Handling

**Problem:** Multiple nodes can allocate the same Teleporter message ID if they submit simultaneously.

**Solution:** 
- Message ID counter stored in consensus state (acceptedState)
- Counter increments only during `Block.Accept()`
- All validators see same counter after block acceptance
- Duplicate IDs handled by Teleporter protocol on destination (idempotent)

**Best Practice:** Submit messages to only one validator to avoid temporary duplicates.

---

## 10. Additional Resources

### Useful Endpoints

- **Fuji C-Chain RPC**: `https://api.avax-test.network/ext/bc/C/rpc`
- **Fuji P-Chain**: `https://api.avax-test.network/ext/bc/P`
- **Fuji Info API**: `https://api.avax-test.network/ext/info`

### Environment Variables

```bash
# Required for contract deployment
export PK="<your-private-key>"
export FUJI_C_CHAIN_BLOCKCHAIN_ID_HEX="0x7fc93d85c6d62c5b2ac0b519c87010ea5294012d1e407030d6acd0021cac10d5"

# Optional for relayer
export RELAYER_PRIVATE_KEY="<relayer-private-key>"
```

### Related Documentation

- [Avalanche Warp Messaging](https://docs.avax.network/cross-chain/avalanche-warp-messaging/overview)
- [ICM (Interchain Messaging)](https://github.com/ava-labs/icm-contracts)
- [Teleporter Protocol](https://github.com/ava-labs/teleporter)
- [Custom VM Development](https://docs.avax.network/build/vm/intro)

---

**Document Version**: 1.0  
**Last Updated**: November 19, 2025  
**Maintainer**: Red Bridge Development Team
