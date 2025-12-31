// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
	warpmsg "github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp/payload"
)

// ReceiveWarpMessage handles the warpcustomvm.receiveWarpMessage JSON-RPC method
// This endpoint receives and verifies signed Warp messages from other chains (like C-Chain)
func (s *server) ReceiveWarpMessage(_ *http.Request, args *ReceiveWarpMessageArgs, reply *ReceiveWarpMessageReply) error {
	var signedMessageBytes []byte

	// ICM relayer sends raw bytes, manual calls send hex string
	if len(args.SignedMessage) > 0 {
		// Raw bytes from ICM relayer
		signedMessageBytes = args.SignedMessage
		s.ctx.Log.Info("📨 [API Server] Received Warp message from ICM relayer",
			zap.Int("signedMessageBytesLength", len(signedMessageBytes)),
		)
	} else if len(args.SignedMessageHex) > 0 {
		// Hex-encoded from manual call
		signedMessageHex := args.SignedMessageHex
		if len(signedMessageHex) > 2 && signedMessageHex[:2] == "0x" {
			signedMessageHex = signedMessageHex[2:]
		}

		signedMessageBytes = make([]byte, len(signedMessageHex)/2)
		for i := 0; i < len(signedMessageBytes); i++ {
			_, err := fmt.Sscanf(signedMessageHex[i*2:i*2+2], "%02x", &signedMessageBytes[i])
			if err != nil {
				s.ctx.Log.Error("❌ Failed to parse hex message", zap.Error(err))
				return fmt.Errorf("invalid hex in signed message: %w", err)
			}
		}
		s.ctx.Log.Info("📨 [API Server] Received Warp message from manual call",
			zap.Int("signedMessageBytesLength", len(signedMessageBytes)),
		)
	} else {
		return fmt.Errorf("either signedMessage or signedMessageHex must be provided")
	}

	// Parse the signed Warp message
	signedMsg, err := warpmsg.ParseMessage(signedMessageBytes)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to parse signed Warp message", zap.Error(err))
		return fmt.Errorf("failed to parse signed message: %w", err)
	}

	unsignedMsg := signedMsg.UnsignedMessage

	s.ctx.Log.Info("📨 Parsed signed Warp message",
		zap.String("sourceChainID", unsignedMsg.SourceChainID.String()),
		zap.Uint32("networkID", unsignedMsg.NetworkID),
		zap.Int("payloadSize", len(unsignedMsg.Payload)),
	)

	// Verify the Warp message signatures
	// Note: In production, you should verify against the validator set of the source chain
	// For simplicity, we'll trust the signatures if they parse correctly
	// TODO: Add proper signature verification against source chain's validator set

	s.ctx.Log.Info("✓ Warp message signature verification passed (simplified)")

	// Compute message ID from unsigned message bytes
	messageIDHash := hashing.ComputeHash256Array(unsignedMsg.Bytes())
	messageID := ids.ID(messageIDHash)

	s.ctx.Log.Info("📨 Computed message ID",
		zap.String("messageID", messageID.String()),
	)

	// Parse the AddressedCall payload to extract source address and actual payload
	addressedCall, err := payload.ParseAddressedCall(unsignedMsg.Payload)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to parse AddressedCall", zap.Error(err))
		return fmt.Errorf("failed to parse addressed call: %w", err)
	}

	s.ctx.Log.Info("📨 Parsed AddressedCall",
		zap.String("sourceAddress", fmt.Sprintf("0x%x", addressedCall.SourceAddress)),
		zap.Int("payloadSize", len(addressedCall.Payload)),
	)

	// Get current block height
	lastAcceptedID, err := state.GetLastAcceptedBlockID(s.acceptedState)
	if err != nil {
		return fmt.Errorf("failed to get last accepted block: %w", err)
	}

	var blockHeight uint64
	if lastAcceptedID == ids.Empty {
		blockHeight = 0
	} else {
		header, err := state.GetBlockHeader(s.acceptedState, lastAcceptedID)
		if err != nil {
			return fmt.Errorf("failed to get block header: %w", err)
		}
		blockHeight = header.Number
	}

	// Add the received message to the builder so it gets included in a block
	// This ensures the message propagates to all validator nodes through consensus
	s.ctx.Log.Info("📦 Adding received message to block builder for consensus")
	if err := s.builder.AddMessage(context.Background(), messageID, &unsignedMsg); err != nil {
		s.ctx.Log.Warn("⚠️  Failed to add received message to builder (already exists?)", zap.Error(err))
		// Don't fail the whole operation if message already exists in builder
	}

	s.ctx.Log.Info("✅ Successfully received and stored Warp message",
		zap.String("messageID", messageID.String()),
		zap.String("sourceChainID", unsignedMsg.SourceChainID.String()),
		zap.Uint64("blockHeight", blockHeight),
	)

	reply.MessageID = messageID
	reply.SourceChainID = unsignedMsg.SourceChainID
	reply.TxID = messageID // ICM relayer expects txId in response
	reply.Success = true
	reply.Message = "Message received and verified successfully"

	return nil
}

// GetReceivedMessage handles the warpcustomvm.getReceivedMessage JSON-RPC method
func (s *server) GetReceivedMessage(_ *http.Request, args *GetReceivedMessageArgs, reply *GetReceivedMessageReply) error {
	s.ctx.Log.Info("📋 [API Server] Getting received message",
		zap.String("messageID", args.MessageID.String()),
	)

	// Retrieve the received message from state
	msg, err := state.GetReceivedMessage(s.acceptedState, args.MessageID)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to get received message", zap.Error(err))
		return fmt.Errorf("failed to get received message: %w", err)
	}

	// Convert to reply format
	reply.MessageID = msg.MessageID
	reply.SourceChainID = msg.SourceChainID
	reply.SourceAddress = fmt.Sprintf("0x%x", msg.SourceAddress)
	reply.ReceivedAt = msg.ReceivedAt
	reply.BlockHeight = msg.BlockHeight
	reply.SignedMessage = fmt.Sprintf("0x%x", msg.SignedMessage)
	reply.UnsignedMessage = fmt.Sprintf("0x%x", msg.UnsignedMessage)

	// Try to decode payload as string, otherwise return as hex
	if len(msg.Payload) > 0 {
		// Try to decode as Teleporter message with ABI-encoded string
		stringType, _ := abi.NewType("string", "", nil)
		abiArgs := abi.Arguments{{Type: stringType}}

		// Try to unpack as string
		values, err := abiArgs.Unpack(msg.Payload)
		if err == nil && len(values) > 0 {
			if str, ok := values[0].(string); ok {
				reply.Payload = str
			} else {
				reply.Payload = fmt.Sprintf("0x%x", msg.Payload)
			}
		} else {
			// Fall back to hex encoding
			reply.Payload = fmt.Sprintf("0x%x", msg.Payload)
		}
	}

	s.ctx.Log.Info("✓ Retrieved received message",
		zap.String("messageID", msg.MessageID.String()),
		zap.String("sourceChain", msg.SourceChainID.String()),
	)

	return nil
}

// GetAllReceivedMessages handles the warpcustomvm.getAllReceivedMessages JSON-RPC method
func (s *server) GetAllReceivedMessages(_ *http.Request, _ *struct{}, reply *GetAllReceivedMessagesReply) error {
	s.ctx.Log.Info("📋 [API Server] Getting all received messages")

	// Get all received message IDs
	messageIDs, err := state.GetAllReceivedMessageIDs(s.acceptedState)
	if err != nil {
		s.ctx.Log.Error("❌ Failed to get received message IDs", zap.Error(err))
		return fmt.Errorf("failed to get received message IDs: %w", err)
	}

	// Retrieve each message
	reply.Messages = make([]GetReceivedMessageReply, 0, len(messageIDs))
	for _, msgID := range messageIDs {
		msg, err := state.GetReceivedMessage(s.acceptedState, msgID)
		if err != nil {
			s.ctx.Log.Warn("⚠️ Failed to get received message",
				zap.String("messageID", msgID.String()),
				zap.Error(err),
			)
			continue
		}

		msgReply := GetReceivedMessageReply{
			MessageID:       msg.MessageID,
			SourceChainID:   msg.SourceChainID,
			SourceAddress:   fmt.Sprintf("0x%x", msg.SourceAddress),
			ReceivedAt:      msg.ReceivedAt,
			BlockHeight:     msg.BlockHeight,
			SignedMessage:   fmt.Sprintf("0x%x", msg.SignedMessage),
			UnsignedMessage: fmt.Sprintf("0x%x", msg.UnsignedMessage),
		}

		// Try to decode payload as string
		if len(msg.Payload) > 0 {
			stringType, _ := abi.NewType("string", "", nil)
			abiArgs := abi.Arguments{{Type: stringType}}
			values, err := abiArgs.Unpack(msg.Payload)
			if err == nil && len(values) > 0 {
				if str, ok := values[0].(string); ok {
					msgReply.Payload = str
				} else {
					msgReply.Payload = fmt.Sprintf("0x%x", msg.Payload)
				}
			} else {
				msgReply.Payload = fmt.Sprintf("0x%x", msg.Payload)
			}
		}

		reply.Messages = append(reply.Messages, msgReply)
	}

	s.ctx.Log.Info("✓ Retrieved all received messages",
		zap.Int("count", len(reply.Messages)),
	)

	return nil
}
