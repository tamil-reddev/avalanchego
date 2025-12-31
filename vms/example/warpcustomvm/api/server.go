// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/builder"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/chain"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/teleporter"
	warpmsg "github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp/payload"
)

// Server defines the warpcustomvm JSON-RPC API server
type Server interface {
	GetChainID(r *http.Request, args *struct{}, reply *GetChainIDReply) error
	SubmitMessage(r *http.Request, args *SubmitMessageArgs, reply *SubmitMessageReply) error
	GetMessage(r *http.Request, args *GetMessageArgs, reply *GetMessageReply) error
	GetLatestBlock(r *http.Request, args *struct{}, reply *GetBlockReply) error
	GetBlock(r *http.Request, args *GetBlockArgs, reply *GetBlockReply) error
	ReceiveWarpMessage(r *http.Request, args *ReceiveWarpMessageArgs, reply *ReceiveWarpMessageReply) error
	GetReceivedMessage(r *http.Request, args *GetReceivedMessageArgs, reply *GetReceivedMessageReply) error
	GetAllReceivedMessages(r *http.Request, args *struct{}, reply *GetAllReceivedMessagesReply) error
}

// NewServer creates a new JSON-RPC API server
func NewServer(
	ctx *snow.Context,
	chain chain.Chain,
	builder builder.Builder,
	acceptedState database.Database,
) Server {
	return &server{
		ctx:           ctx,
		chain:         chain,
		builder:       builder,
		acceptedState: acceptedState,
	}
}

type server struct {
	ctx           *snow.Context
	chain         chain.Chain
	builder       builder.Builder
	acceptedState database.Database
	msgIDMutex    sync.Mutex // Protects message ID allocation within same block height
}

// parseChainID parses a chain ID from hex (with 0x prefix) or CB58 format
func parseChainID(s string) (ids.ID, error) {
	s = strings.TrimPrefix(s, "0x")
	if len(s) == 64 { // 32 bytes = 64 hex chars
		bytes, err := hex.DecodeString(s)
		if err != nil {
			return ids.ID{}, fmt.Errorf("invalid hex chain ID: %w", err)
		}
		var id ids.ID
		copy(id[:], bytes)
		return id, nil
	}
	return ids.FromString(s)
}

// GetChainID handles the warpcustomvm.getChainID JSON-RPC method
func (s *server) GetChainID(_ *http.Request, _ *struct{}, reply *GetChainIDReply) error {
	reply.ChainID = s.ctx.ChainID
	reply.NetworkID = s.ctx.NetworkID
	return nil
}

// SubmitMessage handles the warpcustomvm.submitMessage JSON-RPC method
func (s *server) SubmitMessage(_ *http.Request, args *SubmitMessageArgs, reply *SubmitMessageReply) error {
	s.ctx.Log.Debug("submitMessage request",
		zap.String("destinationChain", args.DestinationChain),
		zap.String("destinationAddress", args.DestinationAddress),
	)

	// Parse destination chain ID (supports hex with 0x prefix or CB58)
	destinationChainID, err := parseChainID(args.DestinationChain)
	if err != nil {
		return fmt.Errorf("invalid destination chain ID: %w", err)
	}

	// ABI-encode the message as string (contract expects abi.decode(message, (string)))
	stringType, _ := abi.NewType("string", "", nil)
	abiArgs := abi.Arguments{{Type: stringType}}
	userMessage, err := abiArgs.Pack(args.Message)
	if err != nil {
		return fmt.Errorf("failed to encode message as string: %w", err)
	}

	// Allocate Teleporter message ID from consensus counter
	s.msgIDMutex.Lock()
	lastMessageID, err := state.GetLastMessageID(s.acceptedState)
	if err != nil {
		s.msgIDMutex.Unlock()
		return fmt.Errorf("failed to get last message ID: %w", err)
	}
	teleporterMsgID := lastMessageID + 1
	s.msgIDMutex.Unlock()

	// Encode Teleporter message
	encodedPayload, err := teleporter.CreateTeleporterMessage(
		teleporterMsgID,
		destinationChainID,
		args.DestinationAddress,
		userMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to encode Teleporter message: %w", err)
	}

	// Create AddressedCall wrapper (ICM expects: Warp → AddressedCall → Teleporter)
	sourceAddress := teleporter.TeleporterContractAddress.Bytes()
	addressedCall, err := payload.NewAddressedCall(sourceAddress, encodedPayload)
	if err != nil {
		return fmt.Errorf("failed to create addressed call: %w", err)
	}

	// Create unsigned Warp message
	unsignedMsg, err := warpmsg.NewUnsignedMessage(
		s.ctx.NetworkID,
		s.ctx.ChainID,
		addressedCall.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("failed to create warp message: %w", err)
	}

	// Compute message ID from unsigned message bytes
	messageID := ids.ID(hashing.ComputeHash256Array(unsignedMsg.Bytes()))

	// Add to builder for inclusion in next block
	if err := s.builder.AddMessage(context.Background(), messageID, unsignedMsg); err != nil {
		return err
	}

	s.ctx.Log.Info("teleporter message submitted",
		zap.Stringer("messageID", messageID),
		zap.Stringer("destinationChain", destinationChainID),
	)

	reply.MessageID = messageID
	return nil
}

// GetMessage handles the warpcustomvm.getMessage JSON-RPC method
func (s *server) GetMessage(_ *http.Request, args *GetMessageArgs, reply *GetMessageReply) error {
	// Retrieve Warp message from state
	unsignedMsg, err := state.GetWarpMessage(s.acceptedState, args.MessageID)
	if err != nil {
		return err
	}

	// Parse the AddressedCall to get the Teleporter message inside
	addressedCall, err := payload.ParseAddressedCall(unsignedMsg.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse addressed call: %w", err)
	}

	reply.MessageID = args.MessageID
	reply.NetworkID = unsignedMsg.NetworkID
	reply.SourceChainID = unsignedMsg.SourceChainID
	reply.SourceAddress = addressedCall.SourceAddress
	reply.Payload = addressedCall.Payload // This is the Teleporter message
	reply.UnsignedMessageBytes = unsignedMsg.Bytes()

	return nil
}

// GetLatestBlock handles the warpcustomvm.getLatestBlock JSON-RPC method
func (s *server) GetLatestBlock(_ *http.Request, _ *struct{}, reply *GetBlockReply) error {
	// Get latest block header
	blockHeader, err := state.GetLatestBlockHeader(s.acceptedState)
	if err != nil {
		return err
	}

	reply.BlockID = blockHeader.Hash
	reply.ParentID = blockHeader.ParentHash
	reply.Height = blockHeader.Number
	reply.Timestamp = blockHeader.Timestamp

	// Extract full Warp message details directly from block header
	// Messages are embedded in the block and synced across all validators via consensus
	reply.Messages = make([]MessageDetail, 0, len(blockHeader.Messages))

	// Handle old blocks that don't have WarpMessages field (backward compatibility)
	if blockHeader.WarpMessages == nil {
		blockHeader.WarpMessages = make(map[string][]byte)
	}

	for _, msgID := range blockHeader.Messages {
		// Get message bytes from block header (embedded during block creation)
		msgBytes, exists := blockHeader.WarpMessages[msgID.String()]
		if !exists {
			s.ctx.Log.Warn("message ID in block but bytes not found",
				zap.Stringer("messageID", msgID),
				zap.Uint64("blockHeight", blockHeader.Number),
			)
			continue
		}

		// Parse the unsigned Warp message from bytes
		unsignedMsg, err := warpmsg.ParseUnsignedMessage(msgBytes)
		if err != nil {
			s.ctx.Log.Warn("failed to parse warp message from block",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		// Parse the AddressedCall to get the Teleporter message inside
		addressedCall, err := payload.ParseAddressedCall(unsignedMsg.Payload)
		if err != nil {
			s.ctx.Log.Warn("failed to parse addressed call",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		reply.Messages = append(reply.Messages, MessageDetail{
			MessageID:            msgID,
			NetworkID:            unsignedMsg.NetworkID,
			SourceChainID:        unsignedMsg.SourceChainID,
			SourceAddress:        addressedCall.SourceAddress,
			Payload:              addressedCall.Payload, // Teleporter message
			UnsignedMessageBytes: unsignedMsg.Bytes(),
			Metadata: MessageMetadata{
				Timestamp:   blockHeader.Timestamp,
				BlockNumber: blockHeader.Number,
				BlockHash:   blockHeader.Hash,
			},
		})
	}

	return nil
}

// GetBlock handles the warpcustomvm.getBlock JSON-RPC method
func (s *server) GetBlock(_ *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	// Get block ID by height
	blockID, err := state.GetBlockIDByHeight(s.acceptedState, args.Height)
	if err != nil {
		return err
	}

	// Get block header
	blockHeader, err := state.GetBlockHeader(s.acceptedState, blockID)
	if err != nil {
		return err
	}

	reply.BlockID = blockID
	reply.ParentID = blockHeader.ParentHash
	reply.Height = blockHeader.Number
	reply.Timestamp = blockHeader.Timestamp

	// Extract full Warp message details directly from block header
	// Messages are embedded in the block and synced across all validators via consensus
	reply.Messages = make([]MessageDetail, 0, len(blockHeader.Messages))

	// Handle old blocks that don't have WarpMessages field (backward compatibility)
	if blockHeader.WarpMessages == nil {
		blockHeader.WarpMessages = make(map[string][]byte)
	}

	for _, msgID := range blockHeader.Messages {
		// Get message bytes from block header (embedded during block creation)
		msgBytes, exists := blockHeader.WarpMessages[msgID.String()]
		if !exists {
			s.ctx.Log.Warn("message ID in block but bytes not found",
				zap.Stringer("messageID", msgID),
				zap.Uint64("blockHeight", blockHeader.Number),
			)
			continue
		}

		// Parse the unsigned Warp message from bytes
		unsignedMsg, err := warpmsg.ParseUnsignedMessage(msgBytes)
		if err != nil {
			s.ctx.Log.Warn("failed to parse warp message from block",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		// Parse the AddressedCall to get the Teleporter message inside
		addressedCall, err := payload.ParseAddressedCall(unsignedMsg.Payload)
		if err != nil {
			s.ctx.Log.Warn("failed to parse addressed call",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		reply.Messages = append(reply.Messages, MessageDetail{
			MessageID:            msgID,
			NetworkID:            unsignedMsg.NetworkID,
			SourceChainID:        unsignedMsg.SourceChainID,
			SourceAddress:        addressedCall.SourceAddress,
			Payload:              addressedCall.Payload, // Teleporter message
			UnsignedMessageBytes: unsignedMsg.Bytes(),
			Metadata: MessageMetadata{
				Timestamp:   blockHeader.Timestamp,
				BlockNumber: blockHeader.Number,
				BlockHash:   blockHeader.Hash,
			},
		})
	}

	return nil
}
