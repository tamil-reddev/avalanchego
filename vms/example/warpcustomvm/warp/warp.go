// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warp

import (
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp/payload"
)

// CreateWarpMessage creates an unsigned Warp message suitable for Teleporter delivery
// This message can be picked up by relayers and delivered to the destination chain
func CreateWarpMessage(
	networkID uint32,
	sourceChainID ids.ID,
	destinationAddress []byte, // EVM address (20 bytes) or hex string bytes
	messagePayload []byte, // The actual message bytes to deliver
) (*warp.UnsignedMessage, error) {
	// Create AddressedCall payload for Teleporter protocol
	// This tells Teleporter to call receiveTeleporterMessage on destinationAddress
	addressedPayload, err := payload.NewAddressedCall(
		destinationAddress,
		messagePayload,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create addressed payload: %w", err)
	}

	addressedPayloadBytes := addressedPayload.Bytes()

	// Create unsigned Warp message
	unsignedMsg, err := warp.NewUnsignedMessage(
		networkID,
		sourceChainID,
		addressedPayloadBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create unsigned warp message: %w", err)
	}

	return unsignedMsg, nil
}

// WarpMessageInfo contains information about a Warp message for storage
type WarpMessageInfo struct {
	MessageID       ids.ID
	UnsignedMessage *warp.UnsignedMessage
	DestinationID   ids.ID
	DestinationAddr string
	Payload         []byte
	BlockHeight     uint64
	BlockHash       ids.ID
}
