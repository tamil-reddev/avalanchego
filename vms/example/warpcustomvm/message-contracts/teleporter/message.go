// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package teleporter

import (
	"encoding/json"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

// TeleporterMessage represents an unsigned Teleporter-style message
// This follows the Teleporter protocol format for cross-chain messaging
type TeleporterMessage struct {
	// Sender is the address that created the message on the source chain
	Sender string `json:"sender"`

	// DestinationBlockchainID is the target chain ID (can be empty on source)
	DestinationBlockchainID ids.ID `json:"destinationBlockchainID"`

	// DestinationAddress is the recipient contract address on destination chain
	DestinationAddress string `json:"destinationAddress"`

	// Nonce for replay protection and ordering
	Nonce uint64 `json:"nonce"`

	// Payload is the actual message body/data to be delivered
	Payload []byte `json:"payload"`

	// Metadata contains additional context about the message
	Metadata MessageMetadata `json:"metadata"`
}

// MessageMetadata contains contextual information
type MessageMetadata struct {
	Timestamp   int64  `json:"timestamp"`   // Unix timestamp when message was created
	BlockNumber uint64 `json:"blockNumber"` // Block number where message was included
	BlockHash   ids.ID `json:"blockHash"`   // Block hash where message was included
}

// ID computes the unique identifier for this message
func (m *TeleporterMessage) ID() (ids.ID, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return ids.ID{}, err
	}
	return hashing.ComputeHash256Array(bytes), nil
}

// Bytes returns the JSON encoding of the message
func (m *TeleporterMessage) Bytes() ([]byte, error) {
	return json.Marshal(m)
}

// ParseTeleporterMessage parses a TeleporterMessage from bytes
func ParseTeleporterMessage(data []byte) (*TeleporterMessage, error) {
	var msg TeleporterMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// AddressedCallPayload represents the payload for an AddressedCall event
// This mimics the Teleporter precompile's AddressedCall(address,bytes) event
type AddressedCallPayload struct {
	// DestinationAddress is the target contract address
	DestinationAddress string `json:"destinationAddress"`

	// Payload contains the serialized TeleporterMessage
	Payload []byte `json:"payload"`
}

// NewAddressedCallPayload creates an AddressedCall payload from a Teleporter message
func NewAddressedCallPayload(destinationAddress string, message *TeleporterMessage) (*AddressedCallPayload, error) {
	payload, err := message.Bytes()
	if err != nil {
		return nil, err
	}

	return &AddressedCallPayload{
		DestinationAddress: destinationAddress,
		Payload:            payload,
	}, nil
}

// Bytes returns the JSON encoding of the payload
func (p *AddressedCallPayload) Bytes() ([]byte, error) {
	return json.Marshal(p)
}
