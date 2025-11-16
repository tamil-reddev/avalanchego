// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package builder

import (
	"context"
	"fmt"
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
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
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
	b.preference = preferred
}

// AddMessage adds a Warp message to the pending pool
func (b *builder) AddMessage(_ context.Context, messageID ids.ID, message *warpmsg.UnsignedMessage) error {
	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	// Store Warp message
	if err := state.SetWarpMessage(b.db, messageID, message); err != nil {
		return fmt.Errorf("failed to store warp message: %w", err)
	}

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
	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	for b.pendingMessages.Len() == 0 {
		// Check context before waiting
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		b.pendingMessagesCond.Wait()
	}

	return common.PendingTxs, nil
}

// BuildBlock builds a new block from pending messages
func (b *builder) BuildBlock(ctx context.Context, blockContext *smblock.Context) (chain.Block, error) {
	// Get the preferred block
	preferredBlk, err := b.chain.GetBlock(b.preference)
	if err != nil {
		return nil, err
	}

	parentTimestamp := preferredBlk.Timestamp()
	timestamp := time.Now().Unix()
	if timestamp < parentTimestamp.Unix() {
		timestamp = parentTimestamp.Unix()
	}

	// Create new block
	wipBlock := &block.Block{
		ParentID:  b.preference,
		Timestamp: timestamp,
		Height:    preferredBlk.Height() + 1,
		Messages:  []ids.ID{},
	}

	b.pendingMessagesMu.Lock()
	defer b.pendingMessagesMu.Unlock()

	// Add pending messages to the block
	for len(wipBlock.Messages) < MaxMessagesPerBlock {
		messageID, _, exists := b.pendingMessages.Oldest()
		if !exists {
			break
		}
		b.pendingMessages.Delete(messageID)

		wipBlock.Messages = append(wipBlock.Messages, messageID)
	}

	// Create block through chain
	newBlock, err := b.chain.NewBlock(wipBlock)
	if err != nil {
		return nil, err
	}

	b.chainContext.Log.Info("built block",
		zap.Uint64("height", wipBlock.Height),
		zap.Int("messages", len(wipBlock.Messages)),
		zap.Stringer("parent", wipBlock.ParentID),
	)

	return newBlock, nil
}
