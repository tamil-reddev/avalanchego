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

// CreateProperTeleporterMessageWithAddress creates a full TeleporterMessage that ICM can unpack
// This matches the TeleporterMessage struct that the relayer expects:
//
//	struct TeleporterMessage {
//	    uint256 messageID;
//	    address senderAddress;
//	    bytes32 destinationBlockchainID;
//	    address destinationAddress;
//	    uint256 requiredGasLimit;
//	    address[] allowedRelayerAddresses;
//	    TeleporterMessageReceipt[] receipts;
//	    bytes message;
//	}
func CreateProperTeleporterMessageWithAddress(teleporterMessageID uint64, destinationChainID ids.ID, destinationAddress string, userMessage []byte) ([]byte, error) {
	fmt.Printf("🔧 [Teleporter Encoder] Step 1: Input parameters\n")
	fmt.Printf("   teleporterMessageID: %d\n", teleporterMessageID)
	fmt.Printf("   destinationChainID: %s (hex: 0x%x)\n", destinationChainID.String(), destinationChainID[:])
	fmt.Printf("   destinationAddress: %s\n", destinationAddress)
	fmt.Printf("   userMessage length: %d bytes (0x%x)\n", len(userMessage), userMessage)

	// Parse destination address from hex string
	if len(destinationAddress) < 2 || destinationAddress[:2] != "0x" {
		return nil, fmt.Errorf("destination address must be hex string with 0x prefix")
	}
	destAddr := common.HexToAddress(destinationAddress)
	fmt.Printf("🔧 [Teleporter Encoder] Step 2: Parsed destination address: %s\n", destAddr.Hex())

	// Create TeleporterMessage as a tuple type (this is what Unpack expects)
	fmt.Printf("🔧 [Teleporter Encoder] Step 4: Creating TeleporterMessage tuple type\n")
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
		fmt.Printf("   ❌ Failed to create tuple type: %v\n", err)
		return nil, fmt.Errorf("failed to create teleporter message type: %w", err)
	}
	fmt.Printf("   ✓ TeleporterMessage tuple type created\n")

	// Convert destination chain ID
	var destChainID [32]byte
	copy(destChainID[:], destinationChainID[:])
	fmt.Printf("🔧 [Teleporter Encoder] Step 5: Preparing message struct\n")
	fmt.Printf("   messageID: %d\n", teleporterMessageID)
	fmt.Printf("   senderAddress: 0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf (Teleporter contract)\n")
	fmt.Printf("   destinationBlockchainID: 0x%x\n", destChainID)
	fmt.Printf("   destinationAddress: %s\n", destAddr.Hex())
	fmt.Printf("   requiredGasLimit: 100000\n")
	fmt.Printf("   allowedRelayerAddresses: []\n")
	fmt.Printf("   receipts: []\n")
	fmt.Printf("   message: %d bytes\n", len(userMessage))

	// Create the struct matching the tuple
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
		MessageID:               big.NewInt(int64(teleporterMessageID)),
		SenderAddress:           common.HexToAddress("0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf"),
		DestinationBlockchainID: destChainID,
		DestinationAddress:      destAddr,
		RequiredGasLimit:        big.NewInt(100000),
		AllowedRelayerAddresses: []common.Address{},
		Receipts: []struct {
			ReceivedMessageID    *big.Int
			RelayerRewardAddress common.Address
		}{},
		Message: userMessage,
	}

	// Pack the struct as a tuple
	fmt.Printf("🔧 [Teleporter Encoder] Step 6: Packing TeleporterMessage tuple\n")
	arguments := abi.Arguments{{Type: teleporterMessageType}}
	packed, err := arguments.Pack(messageStruct)

	if err != nil {
		fmt.Printf("   ❌ Packing failed: %v\n", err)
		return nil, fmt.Errorf("failed to pack teleporter message: %w", err)
	}

	fmt.Printf("🔧 [Teleporter Encoder] Step 7: ✓ Successfully packed!\n")
	fmt.Printf("   Packed size: %d bytes\n", len(packed))
	fmt.Printf("   First 128 bytes: 0x%x\n", packed[:min(128, len(packed))])
	fmt.Printf("🔧 [Teleporter Encoder] ✅ Encoding complete!\n\n")

	return packed, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateProperTeleporterMessage creates a message matching the actual Teleporter contract format
// Deprecated: Use CreateProperTeleporterMessageWithAddress instead
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
