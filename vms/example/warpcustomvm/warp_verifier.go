// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warpcustomvm

import (
	"context"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/network/p2p/acp118"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"go.uber.org/zap"
)

var _ acp118.Verifier = (*warpVerifier)(nil)

// warpVerifier implements the ACP-118 Verifier interface to determine which
// Warp messages this VM's validators should sign.
//
// This is CRITICAL for ICM relayers to work! Without this:
// - Validators won't sign your Warp messages
// - Relayers can't collect signatures
// - Messages can't be delivered cross-chain
type warpVerifier struct {
	vm *VM
}

// Verify checks if a Warp message should be signed by this validator.
// This is called when an ICM relayer requests a signature via ACP-118 P2P protocol.
//
// Verification steps:
// 1. Check message is from this chain
// 2. Check message exists in accepted state
// 3. Optionally: verify justification (block hash)
func (v *warpVerifier) Verify(
	_ context.Context,
	msg *warp.UnsignedMessage,
	justification []byte,
) *common.AppError {
	v.vm.chainContext.Log.Info("ACP-118 signature request received",
		zap.String("sourceChainID", msg.SourceChainID.String()),
		zap.Int("payloadSize", len(msg.Payload)),
		zap.Int("justificationSize", len(justification)),
	)

	// Verify the message is from this blockchain
	if msg.SourceChainID != v.vm.chainContext.ChainID {
		v.vm.chainContext.Log.Error("rejecting message: wrong source chain",
			zap.String("expected", v.vm.chainContext.ChainID.String()),
			zap.String("got", msg.SourceChainID.String()),
		)
		return &common.AppError{
			Code:    common.ErrUndefined.Code,
			Message: "warp message not from this chain",
		}
	}

	// Compute message ID from UNSIGNED message bytes
	messageID := ids.ID(hashing.ComputeHash256Array(msg.Bytes()))

	v.vm.chainContext.Log.Info("computed message ID from unsigned bytes",
		zap.String("messageID", messageID.String()),
	)

	// Verify the message exists in our accepted state
	storedMsg, err := state.GetWarpMessage(v.vm.acceptedState, messageID)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			v.vm.chainContext.Log.Error("rejecting message: not found in state",
				zap.String("messageID", messageID.String()),
			)
			return &common.AppError{
				Code:    common.ErrUndefined.Code,
				Message: "warp message not found in accepted state",
			}
		}
		v.vm.chainContext.Log.Error("rejecting message: database error",
			zap.String("messageID", messageID.String()),
			zap.Error(err),
		)
		return &common.AppError{
			Code:    common.ErrUndefined.Code,
			Message: "failed to query warp message: " + err.Error(),
		}
	}

	// Verify the message bytes match what we have stored
	if string(storedMsg.Bytes()) != string(msg.Bytes()) {
		v.vm.chainContext.Log.Error("rejecting message: bytes mismatch",
			zap.String("messageID", messageID.String()),
			zap.Int("storedSize", len(storedMsg.Bytes())),
			zap.Int("requestedSize", len(msg.Bytes())),
		)
		return &common.AppError{
			Code:    common.ErrUndefined.Code,
			Message: "warp message bytes mismatch",
		}
	}

	// Message is valid - allow signing
	v.vm.chainContext.Log.Info("✓ VERIFIED: Will sign warp message",
		zap.String("messageID", messageID.String()),
		zap.String("sourceChainID", msg.SourceChainID.String()),
	)

	return nil
}
