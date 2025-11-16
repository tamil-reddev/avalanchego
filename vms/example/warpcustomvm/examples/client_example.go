// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/api"
)

// Example demonstrates how to use the warpcustomvm JSON-RPC client
func main() {
	// Create client - similar to xsvm
	client := api.NewClient("http://localhost:9650", "<blockchain-id>")

	ctx := context.Background()

	// Example 1: Submit a Warp message
	fmt.Println("=== Example 1: Submit Warp Message ===")

	// Prepare payload (can encode destination + data)
	// Note: The source address is automatically set to Teleporter precompile address on the server
	payload, err := json.Marshal(map[string]interface{}{
		"destination": "0x1234567890123456789012345678901234567890",
		"action":      "transfer",
		"amount":      "1000",
	})
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v", err)
	}

	messageID, err := client.SubmitMessage(
		ctx,
		payload,
	)
	if err != nil {
		log.Fatalf("Failed to submit message: %v", err)
	}

	fmt.Printf("✓ Message submitted successfully!\n")
	fmt.Printf("  Message ID: %s\n\n", messageID)

	// Example 2: Get Warp message by ID
	fmt.Println("=== Example 2: Get Warp Message ===")

	message, err := client.GetMessage(ctx, messageID)
	if err != nil {
		log.Fatalf("Failed to get message: %v", err)
	}

	fmt.Printf("✓ Warp message retrieved successfully!\n")
	fmt.Printf("  Message ID (TxID): %s\n", message.MessageID)
	fmt.Printf("  Network ID: %d\n", message.NetworkID)
	fmt.Printf("  Source Chain: %s\n", message.SourceChainID)
	fmt.Printf("  Source Address: %x\n", message.SourceAddress)
	fmt.Printf("  Payload: %s\n", string(message.Payload))
	fmt.Printf("  UnsignedMessage Size: %d bytes\n\n", len(message.UnsignedMessageBytes))

	// Example 3: Get latest block
	fmt.Println("=== Example 3: Get Latest Block ===")
	latestBlock, err := client.GetLatestBlock(ctx)
	if err != nil {
		log.Fatalf("Failed to get latest block: %v", err)
	}

	fmt.Printf("✓ Latest block retrieved successfully!\n")
	fmt.Printf("  Block ID: %s\n", latestBlock.BlockID)
	fmt.Printf("  Parent ID: %s\n", latestBlock.ParentID)
	fmt.Printf("  Height: %d\n", latestBlock.Height)
	fmt.Printf("  Timestamp: %d\n", latestBlock.Timestamp)
	fmt.Printf("  Messages count: %d\n\n", len(latestBlock.Messages))

	// Example 4: Get block by height
	fmt.Println("=== Example 4: Get Block by Height ===")
	blockAtHeight, err := client.GetBlock(ctx, latestBlock.Height)
	if err != nil {
		log.Fatalf("Failed to get block by height: %v", err)
	}

	fmt.Printf("✓ Block at height %d retrieved successfully!\n", latestBlock.Height)
	fmt.Printf("  Block ID: %s\n", blockAtHeight.BlockID)
	fmt.Printf("  Messages count: %d\n\n", len(blockAtHeight.Messages))
}
