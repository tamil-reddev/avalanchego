// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/linked"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/block"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/chain"
	warpmsg "github.com/ava-labs/avalanchego/vms/platformvm/warp"

	smblock "github.com/ava-labs/avalanchego/snow/engine/snowman/block"
)

const MaxMessagesPerBlock = 100

var _ Builder = (*builder)(nil)

// Builder builds new blocks
type Builder interface {
	SetPreference(preferred ids.ID)
	AddMessage(ctx context.Context, messageID ids.ID, message *warpmsg.UnsignedMessage) error
	WaitForEvent(ctx context.Context) (common.Message, error)
	BuildBlock(ctx context.Context, blockContext *smblock.Context) (chain.Block, error)
}

type builder struct {
	chainContext *snow.Context
	chain        chain.Chain
	db           database.Database

	preference ids.ID

	// pendingMessagesCond is awoken when there's at least one pending message
	pendingMessagesMu   sync.Mutex
	pendingMessagesCond *sync.Cond
	pendingMessages     *linked.Hashmap[ids.ID, *warpmsg.UnsignedMessage]
}

// New creates a new block builder
func New(chainContext *snow.Context, chain chain.Chain, db database.Database) Builder {
	b := &builder{
		chainContext:    chainContext,
		chain:           chain,
		db:              db,
		pendingMessages: linked.NewHashmap[ids.ID, *warpmsg.UnsignedMessage](),
	}
	b.pendingMessagesCond = sync.NewCond(&b.pendingMessagesMu)
	return b
}

func (b *builder) SetPreference(preferred ids.ID) {
	b.chainContext.Log.Info("builder preference updated", zap.Stringer("preferred", preferred))
	b.preference = preferred
}

// AddMessage adds a Warp message to the pending pool
// Note: The message should already be stored in accepted state by the API server
func (b *builder) AddMessage(_ context.Context, messageID ids.ID, message *warpmsg.UnsignedMessage) error {
	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	b.chainContext.Log.Info("added Warp message to pending pool",
		zap.Stringer("messageID", messageID),
		zap.Int("payloadSize", len(message.Payload)),
	)

	b.pendingMessages.Put(messageID, message)
	b.pendingMessagesCond.Broadcast()

	return nil
}

// WaitForEvent waits for pending messages or context cancellation
func (b *builder) WaitForEvent(ctx context.Context) (common.Message, error) {
	b.chainContext.Log.Debug("🔍 WaitForEvent called")
	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	for b.pendingMessages.Len() == 0 {
		b.chainContext.Log.Debug("⏳ waiting for pending messages...")
		// Check context before waiting
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		b.pendingMessagesCond.Wait()
	}

	b.chainContext.Log.Info("✅ WaitForEvent returning PendingTxs", zap.Int("pendingCount", b.pendingMessages.Len()))
	return common.PendingTxs, nil
}

// BuildBlock builds a new block from pending messages
func (b *builder) BuildBlock(ctx context.Context, blockContext *smblock.Context) (chain.Block, error) {
	b.chainContext.Log.Info("🔨 BuildBlock called")

	// Get the preferred block
	preferredBlk, err := b.chain.GetBlock(b.preference)
	if err != nil {
		b.chainContext.Log.Error("❌ BuildBlock failed to get preferred block", zap.Error(err))
		return nil, err
	}

	parentTimestamp := preferredBlk.Timestamp()
	timestamp := time.Now().Unix()
	if timestamp < parentTimestamp.Unix() {
		timestamp = parentTimestamp.Unix()
	}

	// Create new block
	wipBlock := &block.Block{
		ParentID:     b.preference,
		Timestamp:    timestamp,
		Height:       preferredBlk.Height() + 1,
		Messages:     []ids.ID{},
		WarpMessages: make(map[string][]byte),
	}

	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	// Add pending messages to the block with full message bytes
	for len(wipBlock.Messages) < MaxMessagesPerBlock {
		messageID, unsignedMsg, exists := b.pendingMessages.Oldest()
		if !exists {
			break
		}
		b.pendingMessages.Delete(messageID)

		wipBlock.Messages = append(wipBlock.Messages, messageID)
		// Store full message bytes in block for consensus propagation
		wipBlock.WarpMessages[messageID.String()] = unsignedMsg.Bytes()

		b.chainContext.Log.Info("  → added message to block",
			zap.Stringer("messageID", messageID),
			zap.Int("messageBytes", len(unsignedMsg.Bytes())),
		)
	}

	// Create block through chain
	newBlock, err := b.chain.NewBlock(wipBlock)
	if err != nil {
		return nil, err
	}

	b.chainContext.Log.Info("🔨 built block",
		zap.Uint64("height", wipBlock.Height),
		zap.Int("messageIDs", len(wipBlock.Messages)),
		zap.Int("warpMessages", len(wipBlock.WarpMessages)),
		zap.Stringer("parent", wipBlock.ParentID),
	)

	return newBlock, nil
}
