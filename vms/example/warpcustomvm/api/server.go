// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"fmt"
	"net/http"

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
}

// EthCompatServer provides EVM-compatible JSON-RPC methods
type EthCompatServer interface {
	Chainid(r *http.Request, args *struct{}, reply *string) error
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

// NewEthCompatServer creates an EVM-compatible JSON-RPC server
func NewEthCompatServer(ctx *snow.Context) EthCompatServer {
	return &ethCompatServer{ctx: ctx}
}

type server struct {
	ctx           *snow.Context
	chain         chain.Chain
	builder       builder.Builder
	acceptedState database.Database
}

type ethCompatServer struct {
	ctx *snow.Context
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetChainID handles the warpcustomvm.getChainID JSON-RPC method
func (s *server) GetChainID(_ *http.Request, _ *struct{}, reply *GetChainIDReply) error {
	reply.ChainID = s.ctx.ChainID
	reply.NetworkID = s.ctx.NetworkID
	return nil
}

// ChainId handles the eth_chainId JSON-RPC method (EVM-compatible)
// Returns a hardcoded chain ID for testing (0x539 = 1337 in decimal)
// Note: Gorilla RPC converts eth_chainId -> eth.Chainid (lowercase 'id')
func (e *ethCompatServer) Chainid(_ *http.Request, _ *struct{}, reply *string) error {
	//*reply = "0x539"
	*reply = fmt.Sprintf("0x%x", e.ctx.ChainID)
	return nil
}

// SubmitMessage handles the warpcustomvm.submitMessage JSON-RPC method
func (s *server) SubmitMessage(_ *http.Request, args *SubmitMessageArgs, reply *SubmitMessageReply) error {
	// Hardcoded destination for testing - C-Chain (Fuji testnet)
	destinationChainID := ids.ID{}
	// This is the C-Chain ID on Fuji: yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp
	destID, err := ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
	if err == nil {
		destinationChainID = destID
	}

	s.ctx.Log.Info("Creating Teleporter message with hardcoded values",
		zap.String("destinationChain", destinationChainID.String()),
		zap.Int("payloadSize", len(args.Payload)),
	)

	// Try the proper Teleporter message format
	encodedPayload, err := teleporter.CreateProperTeleporterMessage(destinationChainID, args.Payload)
	if err != nil {
		s.ctx.Log.Error("Failed to create proper teleporter message, trying minimal format", zap.Error(err))

		// Fallback to minimal format
		encodedPayload, err = teleporter.CreateMinimalTeleporterPayload(destinationChainID, args.Payload)
		if err != nil {
			return fmt.Errorf("failed to encode Teleporter message: %w", err)
		}
	}

	s.ctx.Log.Info("Teleporter message encoded successfully",
		zap.Int("encodedSize", len(encodedPayload)),
		zap.String("encodedHex", fmt.Sprintf("0x%x", encodedPayload[:min(64, len(encodedPayload))])),
	)

	// Always use the Teleporter precompile address as the source address
	sourceAddress := TeleporterPrecompileAddress

	// Create AddressedCall with source address and Teleporter-encoded payload
	addressedCall, err := payload.NewAddressedCall(
		sourceAddress,
		encodedPayload,
	)
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

	// Compute message ID from the unsigned message bytes
	messageIDHash := hashing.ComputeHash256Array(unsignedMsg.Bytes())
	messageID := ids.ID(messageIDHash)

	// Add to pending messages (builder will store it)
	if err := s.builder.AddMessage(context.Background(), messageID, unsignedMsg); err != nil {
		return err
	}

	s.ctx.Log.Info("Warp message submitted via JSON-RPC",
		zap.Stringer("messageID", messageID),
		zap.Int("payloadSize", len(args.Payload)),
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

	// Parse the addressed call
	addressedCall, err := payload.ParseAddressedCall(unsignedMsg.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse addressed call: %w", err)
	}

	reply.MessageID = args.MessageID
	reply.NetworkID = unsignedMsg.NetworkID
	reply.SourceChainID = unsignedMsg.SourceChainID
	reply.SourceAddress = addressedCall.SourceAddress
	reply.Payload = addressedCall.Payload
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

	// Fetch full Warp message details for each message ID
	reply.Messages = make([]MessageDetail, 0, len(blockHeader.Messages))
	for _, msgID := range blockHeader.Messages {
		unsignedMsg, err := state.GetWarpMessage(s.acceptedState, msgID)
		if err != nil {
			s.ctx.Log.Warn("failed to get warp message details",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		// Parse addressed call
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
			Payload:              addressedCall.Payload,
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

	// Fetch full Warp message details for each message ID
	reply.Messages = make([]MessageDetail, 0, len(blockHeader.Messages))
	for _, msgID := range blockHeader.Messages {
		unsignedMsg, err := state.GetWarpMessage(s.acceptedState, msgID)
		if err != nil {
			s.ctx.Log.Warn("failed to get warp message details",
				zap.Stringer("messageID", msgID),
				zap.Error(err),
			)
			continue
		}

		// Parse addressed call
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
			Payload:              addressedCall.Payload,
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
