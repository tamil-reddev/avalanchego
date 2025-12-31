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

const (
	// TeleporterContractAddressHex is the deployed Teleporter contract address
	TeleporterContractAddressHex = "0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf"

	// DefaultRequiredGasLimit is the default gas limit for Teleporter messages
	DefaultRequiredGasLimit = 2000000
)

// TeleporterContractAddress is the parsed Teleporter contract address
var TeleporterContractAddress = common.HexToAddress(TeleporterContractAddressHex)

// CreateTeleporterMessage creates a full TeleporterMessage that ICM can unpack.
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
func CreateTeleporterMessage(messageID uint64, destinationChainID ids.ID, destinationAddress string, payload []byte) ([]byte, error) {
	// Validate destination address format
	if len(destinationAddress) < 2 || destinationAddress[:2] != "0x" {
		return nil, fmt.Errorf("destination address must be hex string with 0x prefix")
	}
	destAddr := common.HexToAddress(destinationAddress)

	// Create TeleporterMessage tuple type for ABI encoding
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

	// Convert destination chain ID to bytes32
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
		MessageID:               big.NewInt(int64(messageID)),
		SenderAddress:           TeleporterContractAddress,
		DestinationBlockchainID: destChainID,
		DestinationAddress:      destAddr,
		RequiredGasLimit:        big.NewInt(DefaultRequiredGasLimit),
		AllowedRelayerAddresses: []common.Address{},
		Receipts: []struct {
			ReceivedMessageID    *big.Int
			RelayerRewardAddress common.Address
		}{},
		Message: payload,
	}

	// Pack as ABI-encoded tuple
	packed, err := abi.Arguments{{Type: teleporterMessageType}}.Pack(messageStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to pack teleporter message: %w", err)
	}

	return packed, nil
}
