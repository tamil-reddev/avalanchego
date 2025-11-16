// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package teleporter

import (
	"fmt"
	"math/big"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// SimpleTeleporterPayload creates a minimal valid Teleporter message
// The ICM relayer expects messages in a specific format emitted by SendCrossChainMessage event
func CreateMinimalTeleporterPayload(destinationChainID ids.ID, userMessage []byte) ([]byte, error) {
	// The Teleporter protocol expects this structure in the warp message payload:
	// It's actually the ABI-encoded parameters of the SendCrossChainMessage event

	// Event SendCrossChainMessage(
	//     bytes32 indexed destinationBlockchainID,
	//     uint256 indexed messageID,
	//     TeleporterMessage message,
	//     TeleporterFeeInfo feeInfo
	// )

	// For minimal testing, we'll create the TeleporterMessage struct only

	// Define types
	uint256Ty, _ := abi.NewType("uint256", "uint256", nil)
	addressTy, _ := abi.NewType("address", "address", nil)
	bytes32Ty, _ := abi.NewType("bytes32", "bytes32", nil)
	bytesTy, _ := abi.NewType("bytes", "bytes", nil)

	// TeleporterMessage struct components
	teleporterMessageArgs := abi.Arguments{
		{Type: uint256Ty, Name: "messageID"},
		{Type: addressTy, Name: "senderAddress"},
		{Type: bytes32Ty, Name: "destinationBlockchainID"},
		{Type: addressTy, Name: "destinationAddress"},
		{Type: uint256Ty, Name: "requiredGasLimit"},
		{Type: bytesTy, Name: "allowedRelayerAddresses"}, // encoded as empty dynamic array
		{Type: bytesTy, Name: "receipts"},                // encoded as empty dynamic array
		{Type: bytesTy, Name: "message"},
	}

	// Convert destination chain ID
	var destChainID [32]byte
	copy(destChainID[:], destinationChainID[:])

	// Prepare empty arrays (ABI-encoded)
	emptyArrayArgs := abi.Arguments{{Type: addressTy}}
	emptyAddressArray, _ := emptyArrayArgs.Pack()

	emptyReceiptArgs := abi.Arguments{{Type: uint256Ty}}
	emptyReceipts, _ := emptyReceiptArgs.Pack()

	// Pack the TeleporterMessage
	packed, err := teleporterMessageArgs.Pack(
		big.NewInt(1), // messageID
		common.HexToAddress("0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf"), // senderAddress (Teleporter contract)
		destChainID, // destinationBlockchainID
		common.HexToAddress("0x1AA5722D8C209d1657c9e973F379d36c342E1eC4"), // destinationAddress
		big.NewInt(100000), // requiredGasLimit
		emptyAddressArray,  // allowedRelayerAddresses
		emptyReceipts,      // receipts
		userMessage,        // message
	)

	if err != nil {
		return nil, fmt.Errorf("failed to pack teleporter message: %w", err)
	}

	return packed, nil
}

// CreateProperTeleporterMessage creates a message matching the actual Teleporter contract format
func CreateProperTeleporterMessage(destinationChainID ids.ID, userMessage []byte) ([]byte, error) {
	// The actual structure the relayer expects is the TeleporterMessage struct
	// as defined in the Teleporter smart contract

	// Define the TeleporterMessage struct type
	teleporterMessageType, err := abi.NewType("tuple", "TeleporterMessage", []abi.ArgumentMarshaling{
		{Name: "messageID", Type: "uint256"},
		{Name: "senderAddress", Type: "address"},
		{Name: "destinationBlockchainID", Type: "bytes32"},
		{Name: "destinationAddress", Type: "address"},
		{Name: "requiredGasLimit", Type: "uint256"},
		{Name: "allowedRelayerAddresses", Type: "address[]"},
		{Name: "receipts", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
			{Name: "receivedMessageID", Type: "uint256"},
			{Name: "relayerRewardAddress", Type: "address"},
		}},
		{Name: "message", Type: "bytes"},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create teleporter message type: %w", err)
	}

	// Convert destination chain ID
	var destChainID [32]byte
	copy(destChainID[:], destinationChainID[:])

	// Create the message struct
	messageStruct := struct {
		MessageID               *big.Int
		SenderAddress           common.Address
		DestinationBlockchainID [32]byte
		DestinationAddress      common.Address
		RequiredGasLimit        *big.Int
		AllowedRelayerAddresses []common.Address
		Receipts                []struct {
			ReceivedMessageID    *big.Int
			RelayerRewardAddress common.Address
		}
		Message []byte
	}{
		MessageID:               big.NewInt(1),
		SenderAddress:           common.HexToAddress("0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf"), // Teleporter contract address
		DestinationBlockchainID: destChainID,
		DestinationAddress:      common.HexToAddress("0x1AA5722D8C209d1657c9e973F379d36c342E1eC4"),
		RequiredGasLimit:        big.NewInt(100000),
		AllowedRelayerAddresses: []common.Address{},
		Receipts: []struct {
			ReceivedMessageID    *big.Int
			RelayerRewardAddress common.Address
		}{},
		Message: userMessage,
	}

	// Pack the struct
	arguments := abi.Arguments{{Type: teleporterMessageType}}
	packed, err := arguments.Pack(messageStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to pack message struct: %w", err)
	}

	return packed, nil
}
