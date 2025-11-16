#!/bin/bash

# WarpCustomVM JSON-RPC API Examples using curl
# Blockchain RPC Endpoint: http://localhost:9654/ext/bc/W9gKoUkZw1zAoxfrWQbkU5Bq5HbvBKFZUMVj7hHvYXBfKtr5z

ENDPOINT="http://localhost:9654/ext/bc/W9gKoUkZw1zAoxfrWQbkU5Bq5HbvBKFZUMVj7hHvYXBfKtr5z"

echo "=========================================="
echo "WarpCustomVM JSON-RPC API Examples"
echo "=========================================="
echo ""

# Example 1: Submit a Message
echo "1. Submit Message"
echo "----------------------------------------"
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "sender": "P-fuji1abcdefghijklmnopqrstuvwxyz123456789",
      "destinationBlockchainID": "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
      "destinationAddress": "0x1234567890123456789012345678901234567890",
      "nonce": 1,
      "payload": "SGVsbG8gV29ybGQh",
      "metadata": ""
    }
  }' \
  $ENDPOINT

echo -e "\n\n"

# Example 2: Get Message by ID
echo "2. Get Message by ID"
echo "----------------------------------------"
echo "Note: Replace MESSAGE_ID with actual message ID from submit response"
MESSAGE_ID="11111111111111111111111111111111LpoYY"
curl -X POST \
  -H "Content-Type: application/json" \
  -d "{
    \"jsonrpc\": \"2.0\",
    \"id\": 2,
    \"method\": \"warpcustomvm.getMessage\",
    \"params\": {
      \"messageID\": \"$MESSAGE_ID\"
    }
  }" \
  $ENDPOINT

echo -e "\n\n"

# Example 3: Get Latest Block
echo "3. Get Latest Block"
echo "----------------------------------------"
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "warpcustomvm.getLatestBlock",
    "params": {}
  }' \
  $ENDPOINT

echo -e "\n\n"

# Example 4: Get Block by Height
echo "4. Get Block by Height"
echo "----------------------------------------"
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "warpcustomvm.getBlock",
    "params": {
      "height": 0
    }
  }' \
  $ENDPOINT

echo -e "\n\n"

# Example 5: Submit Message with Metadata
echo "5. Submit Message with Metadata"
echo "----------------------------------------"
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "warpcustomvm.submitMessage",
    "params": {
      "sender": "P-fuji1sender123456789012345678901234567",
      "destinationBlockchainID": "2CFohG9MQCmUc4kh9SwBqJtCMnMBNeZ3nutkdGR2iYnZQs71df",
      "destinationAddress": "0xabcdef1234567890abcdef1234567890abcdef12",
      "nonce": 2,
      "payload": "eyJhY3Rpb24iOiJ0cmFuc2ZlciIsImFtb3VudCI6IjEwMDAifQ==",
      "metadata": "eyJ0eXBlIjoiY3Jvc3MtY2hhaW4ifQ=="
    }
  }' \
  $ENDPOINT

echo -e "\n\n"
echo "=========================================="
echo "Examples Complete"
echo "=========================================="
