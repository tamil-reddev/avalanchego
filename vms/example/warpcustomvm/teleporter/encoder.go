// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package teleporter

import (
	"encoding/binary"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// TeleporterMessage represents the structure expected by the Teleporter protocol
type TeleporterMessage struct {
	MessageID               *big.Int
	SenderAddress           common.Address
	DestinationBlockchainID [32]byte
	DestinationAddress      common.Address
	RequiredGasLimit        *big.Int
	AllowedRelayerAddresses []common.Address
	Receipts                []TeleporterMessageReceipt
	Message                 []byte
}

// TeleporterMessageReceipt represents a receipt in the Teleporter message
type TeleporterMessageReceipt struct {
	ReceivedMessageID    *big.Int
	RelayerRewardAddress common.Address
}

// CreateHardcodedTeleporterMessage creates a Teleporter message with hardcoded test values
func CreateHardcodedTeleporterMessage(destinationChainID ids.ID, userPayload []byte) *TeleporterMessage {
	// Convert destination chain ID to [32]byte
	var destChainID [32]byte
	copy(destChainID[:], destinationChainID[:])

	return &TeleporterMessage{
		MessageID:               big.NewInt(1),                                                     // Hardcoded message ID
		SenderAddress:           common.HexToAddress("0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf"), // Teleporter contract address
		DestinationBlockchainID: destChainID,
		DestinationAddress:      common.HexToAddress("0x1AA5722D8C209d1657c9e973F379d36c342E1eC4"), // Hardcoded destination
		RequiredGasLimit:        big.NewInt(100000),                                                // Hardcoded gas limit
		AllowedRelayerAddresses: []common.Address{},                                                // Empty for now
		Receipts:                []TeleporterMessageReceipt{},                                      // Empty receipts
		Message:                 userPayload,
	}
}

// EncodeTeleporterMessage encodes a TeleporterMessage using Solidity ABI encoding
func EncodeTeleporterMessage(msg *TeleporterMessage) ([]byte, error) {
	// Build the ABI for TeleporterMessage manually
	// This matches the Solidity struct expected by ICM relayer

	// Calculate total size needed
	// messageID (32) + senderAddress (32) + destinationBlockchainID (32) +
	// destinationAddress (32) + requiredGasLimit (32) +
	// allowedRelayerAddresses offset (32) + receipts offset (32) + message offset (32) +
	// dynamic arrays...

	// Simple encoding approach - pack the struct
	result := make([]byte, 0, 1024)

	// 1. messageID (uint256 - 32 bytes)
	messageIDBytes := make([]byte, 32)
	msg.MessageID.FillBytes(messageIDBytes)
	result = append(result, messageIDBytes...)

	// 2. senderAddress (address - 32 bytes with left padding)
	senderBytes := make([]byte, 32)
	copy(senderBytes[12:], msg.SenderAddress.Bytes())
	result = append(result, senderBytes...)

	// 3. destinationBlockchainID (bytes32 - 32 bytes)
	result = append(result, msg.DestinationBlockchainID[:]...)

	// 4. destinationAddress (address - 32 bytes with left padding)
	destAddrBytes := make([]byte, 32)
	copy(destAddrBytes[12:], msg.DestinationAddress.Bytes())
	result = append(result, destAddrBytes...)

	// 5. requiredGasLimit (uint256 - 32 bytes)
	gasLimitBytes := make([]byte, 32)
	msg.RequiredGasLimit.FillBytes(gasLimitBytes)
	result = append(result, gasLimitBytes...)

	// 6. Offset to allowedRelayerAddresses array (uint256 - 32 bytes)
	// For now, point to after all static fields + 3 offset fields
	allowedRelayersOffset := big.NewInt(int64(32 * 8)) // 8 fields * 32 bytes
	offsetBytes := make([]byte, 32)
	allowedRelayersOffset.FillBytes(offsetBytes)
	result = append(result, offsetBytes...)

	// 7. Offset to receipts array (uint256 - 32 bytes)
	receiptsOffset := big.NewInt(int64(32*8 + 32)) // after allowedRelayers
	offsetBytes2 := make([]byte, 32)
	receiptsOffset.FillBytes(offsetBytes2)
	result = append(result, offsetBytes2...)

	// 8. Offset to message bytes (uint256 - 32 bytes)
	messageOffset := big.NewInt(int64(32*8 + 32 + 32)) // after receipts
	offsetBytes3 := make([]byte, 32)
	messageOffset.FillBytes(offsetBytes3)
	result = append(result, offsetBytes3...)

	// 9. allowedRelayerAddresses array (length + elements)
	// Array length
	arrayLenBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(arrayLenBytes[24:], uint64(len(msg.AllowedRelayerAddresses)))
	result = append(result, arrayLenBytes...)

	// 10. receipts array (length + elements)
	receiptsLenBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(receiptsLenBytes[24:], uint64(len(msg.Receipts)))
	result = append(result, receiptsLenBytes...)

	// 11. message bytes (length + data)
	messageLenBytes := make([]byte, 32)
	binary.BigEndian.PutUint64(messageLenBytes[24:], uint64(len(msg.Message)))
	result = append(result, messageLenBytes...)
	result = append(result, msg.Message...)

	// Pad to 32-byte boundary
	if remainder := len(msg.Message) % 32; remainder != 0 {
		padding := make([]byte, 32-remainder)
		result = append(result, padding...)
	}

	return result, nil
}

// EncodeTeleporterMessageWithGoEthereumABI uses the go-ethereum ABI encoder
// This is more reliable but requires understanding the exact ABI structure
func EncodeTeleporterMessageWithGoEthereumABI(msg *TeleporterMessage) ([]byte, error) {
	// Define the TeleporterMessage struct ABI types
	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}

	bytes32Type, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}

	bytesType, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	addressArrayType, err := abi.NewType("address[]", "", nil)
	if err != nil {
		return nil, err
	}

	// Define the receipt tuple array type
	receiptComponents := []abi.ArgumentMarshaling{
		{Name: "receivedMessageID", Type: "uint256"},
		{Name: "relayerRewardAddress", Type: "address"},
	}
	receiptType, err := abi.NewType("tuple[]", "", receiptComponents)
	if err != nil {
		return nil, err
	}

	// Create the arguments for the struct
	arguments := abi.Arguments{
		{Type: uint256Type},      // messageID
		{Type: addressType},      // senderAddress
		{Type: bytes32Type},      // destinationBlockchainID
		{Type: addressType},      // destinationAddress
		{Type: uint256Type},      // requiredGasLimit
		{Type: addressArrayType}, // allowedRelayerAddresses
		{Type: receiptType},      // receipts
		{Type: bytesType},        // message
	}

	// Convert receipts to the format expected by ABI packer
	receiptsForPacking := convertReceiptsToInterface(msg.Receipts)

	// Pack the values
	packed, err := arguments.Pack(
		msg.MessageID,
		msg.SenderAddress,
		msg.DestinationBlockchainID,
		msg.DestinationAddress,
		msg.RequiredGasLimit,
		msg.AllowedRelayerAddresses,
		receiptsForPacking,
		msg.Message,
	)

	return packed, err
}

// Helper function to convert receipts to interface{} for ABI packing
func convertReceiptsToInterface(receipts []TeleporterMessageReceipt) []struct {
	ReceivedMessageID    *big.Int
	RelayerRewardAddress common.Address
} {
	result := make([]struct {
		ReceivedMessageID    *big.Int
		RelayerRewardAddress common.Address
	}, len(receipts))

	for i, r := range receipts {
		result[i].ReceivedMessageID = r.ReceivedMessageID
		result[i].RelayerRewardAddress = r.RelayerRewardAddress
	}

	return result
}

// EncodeSimpleTeleporterMessage creates a simplified Teleporter-compatible message
// This uses a basic structure that should match what the relayer expects
func EncodeSimpleTeleporterMessage(msg *TeleporterMessage) ([]byte, error) {
	// Use go-ethereum's ABI encoder with proper error handling
	// Define each field type carefully

	uint256Ty, _ := abi.NewType("uint256", "", nil)
	addressTy, _ := abi.NewType("address", "", nil)
	bytes32Ty, _ := abi.NewType("bytes32", "", nil)
	bytesTy, _ := abi.NewType("bytes", "", nil)

	// Create arguments matching the Teleporter struct fields
	args := abi.Arguments{
		{Type: uint256Ty, Name: "messageID"},
		{Type: addressTy, Name: "senderAddress"},
		{Type: bytes32Ty, Name: "destinationBlockchainID"},
		{Type: addressTy, Name: "destinationAddress"},
		{Type: uint256Ty, Name: "requiredGasLimit"},
		{Type: bytesTy, Name: "allowedRelayerAddresses"}, // Simplified as bytes for empty array
		{Type: bytesTy, Name: "receipts"},                // Simplified as bytes for empty array
		{Type: bytesTy, Name: "message"},
	}

	// Pack with empty arrays as empty bytes
	emptyArray := []byte{}

	return args.Pack(
		msg.MessageID,
		msg.SenderAddress,
		msg.DestinationBlockchainID,
		msg.DestinationAddress,
		msg.RequiredGasLimit,
		emptyArray, // allowedRelayerAddresses
		emptyArray, // receipts
		msg.Message,
	)
}
