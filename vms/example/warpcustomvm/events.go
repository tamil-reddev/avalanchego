// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// DEPRECATED: This file contains event emitter code for Teleporter-style events.
// With the refactoring to use only UnsignedWarpMessage, this is no longer needed.
// Relayers can directly fetch Warp messages via the GetMessage API endpoint.
// This file is kept for reference only.

package warpcustomvm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/message-contracts/teleporter"
)

// Event signature for AddressedCall - mimics Teleporter precompile
// AddressedCall(address indexed destinationAddress, bytes payload)
const AddressedCallEventSignature = "AddressedCall(address,bytes)"

// Log represents a VM event log entry that can be detected by relayers
// This follows the Ethereum-style log format used by Teleporter
type Log struct {
	// Topics are indexed parameters (first topic is the event signature hash)
	Topics []string `json:"topics"`

	// Data is the non-indexed event data (contains the AddressedCall payload)
	Data string `json:"data"`

	// BlockNumber where the event was emitted
	BlockNumber uint64 `json:"blockNumber"`

	// BlockHash where the event was emitted
	BlockHash string `json:"blockHash"`

	// TxHash is the transaction/message ID that generated this event
	TxHash string `json:"txHash"`

	// LogIndex is the index of this log in the block
	LogIndex uint64 `json:"logIndex"`
}

// EventEmitter handles emission of Teleporter-style events for relayers
type EventEmitter struct {
	ctx  *snow.Context
	logs []Log
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(ctx *snow.Context) *EventEmitter {
	return &EventEmitter{
		ctx:  ctx,
		logs: make([]Log, 0),
	}
}

// EmitAddressedCall emits an AddressedCall event for a Teleporter message
// This allows ICM relayers to detect and process cross-chain messages
func (e *EventEmitter) EmitAddressedCall(
	messageID ids.ID,
	message *teleporter.TeleporterMessage,
	blockNumber uint64,
	blockHash ids.ID,
) error {
	// Create AddressedCall payload (destinationAddress, messageBytes)
	payload, err := teleporter.NewAddressedCallPayload(
		message.DestinationAddress,
		message,
	)
	if err != nil {
		return err
	}

	payloadBytes, err := payload.Bytes()
	if err != nil {
		return err
	}

	// Compute event signature hash (first topic)
	// In production, this would be keccak256, but we use SHA256 for simplicity
	eventSigHash := computeEventSignatureHash(AddressedCallEventSignature)

	// Second topic is the indexed destination address (for filtering)
	destAddressTopic := hexEncode([]byte(message.DestinationAddress))

	// Create the log entry
	log := Log{
		Topics: []string{
			eventSigHash,
			destAddressTopic,
		},
		Data:        hexEncode(payloadBytes),
		BlockNumber: blockNumber,
		BlockHash:   blockHash.String(),
		TxHash:      messageID.String(),
		LogIndex:    uint64(len(e.logs)),
	}

	e.logs = append(e.logs, log)

	e.ctx.Log.Info("emitted AddressedCall event",
		zap.String("messageID", messageID.String()),
		zap.String("destinationAddress", message.DestinationAddress),
		zap.Uint64("blockNumber", blockNumber),
		zap.Uint64("nonce", message.Nonce),
	)

	return nil
}

// EmitAddressedCallSimple is a simplified version for API use (without block info)
func (e *EventEmitter) EmitAddressedCallSimple(
	destinationBlockchainID ids.ID,
	destinationAddress string,
	message *teleporter.TeleporterMessage,
) {
	e.ctx.Log.Info("emitted AddressedCall event",
		zap.String("destinationBlockchain", destinationBlockchainID.String()),
		zap.String("destinationAddress", destinationAddress),
		zap.Uint64("nonce", message.Nonce),
	)
}

// GetLogs returns all logs emitted by this emitter
func (e *EventEmitter) GetLogs() []Log {
	return e.logs
}

// GetLogsByBlockNumber returns logs for a specific block number
func (e *EventEmitter) GetLogsByBlockNumber(blockNumber uint64) []Log {
	var logs []Log
	for _, log := range e.logs {
		if log.BlockNumber == blockNumber {
			logs = append(logs, log)
		}
	}
	return logs
}

// Helper functions

// computeEventSignatureHash computes the hash of an event signature
// In production Teleporter, this uses keccak256. We use SHA256 for simplicity.
func computeEventSignatureHash(signature string) string {
	hash := sha256.Sum256([]byte(signature))
	return "0x" + hex.EncodeToString(hash[:])
}

// hexEncode encodes bytes to hex string with 0x prefix
func hexEncode(data []byte) string {
	return "0x" + hex.EncodeToString(data)
}

// ParseAddressedCallPayload parses an AddressedCall payload from hex-encoded data
// Relayers use this to extract the Teleporter message
func ParseAddressedCallPayload(hexData string) (*teleporter.AddressedCallPayload, error) {
	// Remove 0x prefix if present
	if len(hexData) >= 2 && hexData[:2] == "0x" {
		hexData = hexData[2:]
	}

	data, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, err
	}

	payload := &teleporter.AddressedCallPayload{}
	err = json.Unmarshal(data, payload)
	return payload, err
}

// ExtractTeleporterMessage extracts the TeleporterMessage from an AddressedCall payload
func ExtractTeleporterMessage(payload *teleporter.AddressedCallPayload) (*teleporter.TeleporterMessage, error) {
	return teleporter.ParseTeleporterMessage(payload.Payload)
}
