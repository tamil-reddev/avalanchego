// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

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
	s := &server{
		ctx:           ctx,
		chain:         chain,
		builder:       builder,
		acceptedState: acceptedState,
	}
	// Initialize counter to 7 so next message ID will be 8
	s.nextTeleporterMsgID.Store(7)
	return s
}

// NewEthCompatServer creates an EVM-compatible JSON-RPC server
func NewEthCompatServer(ctx *snow.Context) EthCompatServer {
	return &ethCompatServer{ctx: ctx}
}

type server struct {
	ctx                 *snow.Context
	chain               chain.Chain
	builder             builder.Builder
	acceptedState       database.Database
	nextTeleporterMsgID atomic.Uint64
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
	s.ctx.Log.Info("📥 [API Server] Step 1: Received submitMessage request",
		zap.String("destinationChain", args.DestinationChain),
		zap.String("destinationAddress", args.DestinationAddress),
		zap.String("message", args.Message),
	)

	// Parse destination chain ID from request (supports both hex and cb58 formats)
	var destinationChainID ids.ID
	var err error

	s.ctx.Log.Info("📥 [API Server] Step 2: Parsing destination chain ID")
	// Try parsing as hex first (with or without 0x prefix)
	destChainStr := args.DestinationChain
	if len(destChainStr) > 2 && destChainStr[:2] == "0x" {
		// Hex format with 0x prefix - strip 0x and decode
		s.ctx.Log.Info("   Detected hex format (0x prefix)")
		hexStr := destChainStr[2:]
		if len(hexStr) == 64 { // 32 bytes = 64 hex chars
			for i := 0; i < 32; i++ {
				var b byte
				_, err := fmt.Sscanf(hexStr[i*2:i*2+2], "%02x", &b)
				if err != nil {
					s.ctx.Log.Error("   ❌ Invalid hex character", zap.Error(err))
					return fmt.Errorf("invalid hex in destination chain ID: %w", err)
				}
				destinationChainID[i] = b
			}
			s.ctx.Log.Info("   ✓ Parsed hex to chain ID", zap.String("chainID", destinationChainID.String()))
		} else {
			s.ctx.Log.Error("   ❌ Invalid hex length", zap.Int("got", len(hexStr)), zap.Int("expected", 64))
			return fmt.Errorf("invalid destination chain ID hex length: expected 64 hex chars, got %d", len(hexStr))
		}
	} else {
		// Try CB58 format (e.g., yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp)
		s.ctx.Log.Info("   Detected CB58 format")
		destinationChainID, err = ids.FromString(destChainStr)
		if err != nil {
			s.ctx.Log.Error("   ❌ Failed to parse CB58", zap.Error(err))
			return fmt.Errorf("invalid destination chain ID: %w", err)
		}
		s.ctx.Log.Info("   ✓ Parsed CB58 to chain ID", zap.String("chainID", destinationChainID.String()))
	}

	// ABI-encode the message as string (your contract expects abi.decode(message, (string)))
	s.ctx.Log.Info("📥 [API Server] Step 3: ABI-encoding message as string")
	stringType, _ := abi.NewType("string", "", nil)
	abiArgs := abi.Arguments{{Type: stringType}}
	userMessage, err := abiArgs.Pack(args.Message)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to ABI-encode message", zap.Error(err))
		return fmt.Errorf("failed to encode message as string: %w", err)
	}
	s.ctx.Log.Info("   ✓ Message ABI-encoded",
		zap.Int("originalLength", len(args.Message)),
		zap.Int("encodedLength", len(userMessage)),
		zap.String("encodedHex", fmt.Sprintf("0x%x", userMessage)),
	)

	// Try the proper Teleporter message format with destination address from request
	// Increment and get the next Teleporter message ID
	teleporterMsgID := s.nextTeleporterMsgID.Add(1)
	s.ctx.Log.Info("📥 [API Server] Step 4: Encoding Teleporter message",
		zap.Uint64("teleporterMessageID", teleporterMsgID),
	)
	encodedPayload, err := teleporter.CreateProperTeleporterMessageWithAddress(
		teleporterMsgID,
		destinationChainID,
		args.DestinationAddress,
		userMessage,
	)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to create proper teleporter message, trying minimal format", zap.Error(err))

		// Fallback to minimal format
		encodedPayload, err = teleporter.CreateMinimalTeleporterPayload(destinationChainID, userMessage)
		if err != nil {
			s.ctx.Log.Error("❌ Failed to encode with minimal format too", zap.Error(err))
			return fmt.Errorf("failed to encode Teleporter message: %w", err)
		}
	}

	s.ctx.Log.Info("📥 [API Server] Step 5: ✓ Teleporter payload encoded",
		zap.Int("encodedSize", len(encodedPayload)),
		zap.String("first64Bytes", fmt.Sprintf("0x%x", encodedPayload[:min(64, len(encodedPayload))])),
	)

	// ICM relayer expects: Warp → AddressedCall → Teleporter Message
	// The AddressedCall wraps the Teleporter message with a source address
	sourceAddress := TeleporterPrecompileAddress
	s.ctx.Log.Info("📥 [API Server] Step 6: Creating AddressedCall wrapper",
		zap.String("sourceAddress", fmt.Sprintf("0x%x", sourceAddress)),
		zap.Int("teleporterPayloadSize", len(encodedPayload)),
	)

	// Wrap Teleporter message in AddressedCall
	addressedCall, err := payload.NewAddressedCall(
		sourceAddress,
		encodedPayload, // This is the Teleporter message
	)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to create addressed call", zap.Error(err))
		return fmt.Errorf("failed to create addressed call: %w", err)
	}
	s.ctx.Log.Info("   ✓ AddressedCall created", zap.Int("size", len(addressedCall.Bytes())))

	// Create unsigned Warp message with AddressedCall payload
	s.ctx.Log.Info("📥 [API Server] Step 7: Creating unsigned Warp message",
		zap.Uint32("networkID", s.ctx.NetworkID),
		zap.String("sourceChainID", s.ctx.ChainID.String()),
	)
	unsignedMsg, err := warpmsg.NewUnsignedMessage(
		s.ctx.NetworkID,
		s.ctx.ChainID,
		addressedCall.Bytes(), // AddressedCall wraps the Teleporter message
	)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to create warp message", zap.Error(err))
		return fmt.Errorf("failed to create warp message: %w", err)
	}
	s.ctx.Log.Info("   ✓ Unsigned Warp message created",
		zap.Int("totalSize", len(unsignedMsg.Bytes())),
	)

	// Compute message ID from the unsigned message bytes
	messageIDHash := hashing.ComputeHash256Array(unsignedMsg.Bytes())
	messageID := ids.ID(messageIDHash)
	s.ctx.Log.Info("📥 [API Server] Step 8: Computed message ID",
		zap.String("messageID", messageID.String()),
		zap.String("messageIDHex", fmt.Sprintf("0x%x", messageID[:])),
	)

	// Add to pending messages (builder will store it)
	s.ctx.Log.Info("📥 [API Server] Step 9: Adding message to builder")
	if err := s.builder.AddMessage(context.Background(), messageID, unsignedMsg); err != nil {
		s.ctx.Log.Error("❌ Failed to add message to builder", zap.Error(err))
		return err
	}
	s.ctx.Log.Info("   ✓ Message added to builder successfully")

	s.ctx.Log.Info("📥 [API Server] ✅ Step 10: Teleporter message submitted successfully!",
		zap.String("messageID", messageID.String()),
		zap.String("destinationChain", destinationChainID.String()),
		zap.String("destinationAddress", args.DestinationAddress),
		zap.String("userMessage", args.Message),
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
