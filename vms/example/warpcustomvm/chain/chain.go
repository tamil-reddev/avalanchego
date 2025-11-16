// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/block"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"

	smblock "github.com/ava-labs/avalanchego/snow/engine/snowman/block"
)

const maxClockSkew = 10 * time.Second

var (
	_ Block = (*blockWrapper)(nil)

	errMissingParent         = errors.New("missing parent block")
	errMissingChild          = errors.New("missing child block")
	errParentNotVerified     = errors.New("parent block has not been verified")
	errFutureTimestamp       = errors.New("future timestamp")
	errTimestampBeforeParent = errors.New("timestamp before parent")
	errWrongHeight           = errors.New("wrong height")
)

// Block extends snowman.Block with additional functionality
type Block interface {
	snowman.Block
	smblock.WithVerifyContext
	State() (database.Database, error)
}

// Chain manages block lifecycle and consensus
type Chain interface {
	LastAccepted() ids.ID
	SetChainState(state snow.State)
	GetBlock(blkID ids.ID) (Block, error)
	NewBlock(blk *block.Block) (Block, error)
}

type chain struct {
	chainContext  *snow.Context
	acceptedState database.Database
	chainState    snow.State

	lastAcceptedID ids.ID
	verifiedBlocks map[ids.ID]*blockWrapper
}

// New creates a new chain manager
func New(ctx *snow.Context, db database.Database) (Chain, error) {
	ctx.Log.Info("DEBUG CHAIN: Getting last accepted block ID")
	lastAcceptedID, err := state.GetLastAcceptedBlockID(db)
	if err != nil {
		ctx.Log.Error("DEBUG CHAIN: Failed to get last accepted block ID", zap.Error(err))
		return nil, err
	}
	ctx.Log.Info("DEBUG CHAIN: Last accepted block ID", zap.Stringer("blockID", lastAcceptedID))

	c := &chain{
		chainContext:   ctx,
		acceptedState:  db,
		lastAcceptedID: lastAcceptedID,
		verifiedBlocks: make(map[ids.ID]*blockWrapper),
	}

	// If we have a genesis block (empty ID), create it
	if lastAcceptedID == ids.Empty {
		ctx.Log.Info("DEBUG CHAIN: Last accepted is genesis (empty ID), loading genesis block header")
		// Get genesis block header
		genesisHeader, err := state.GetBlockHeader(db, ids.Empty)
		if err != nil {
			ctx.Log.Error("DEBUG CHAIN: Failed to get genesis block header", zap.Error(err))
			return nil, err
		}
		ctx.Log.Info("DEBUG CHAIN: Genesis block header loaded successfully",
			zap.Uint64("height", genesisHeader.Number),
			zap.Int64("timestamp", genesisHeader.Timestamp))

		// Create genesis block
		genesisBlock := &block.Block{
			ParentID:  ids.Empty,
			Height:    0,
			Timestamp: genesisHeader.Timestamp,
			Messages:  []ids.ID{},
		}

		genesisBytes, err := genesisBlock.Bytes()
		if err != nil {
			ctx.Log.Error("DEBUG CHAIN: Failed to serialize genesis block", zap.Error(err))
			return nil, err
		}

		c.verifiedBlocks[ids.Empty] = &blockWrapper{
			Block: genesisBlock,
			chain: c,
			id:    ids.Empty,
			bytes: genesisBytes,
		}
		ctx.Log.Info("DEBUG CHAIN: Genesis block wrapper created successfully")
	} else {
		ctx.Log.Info("DEBUG CHAIN: Loading last accepted block (non-genesis)")
		// Load the last accepted block
		lastAccepted, err := c.getBlock(lastAcceptedID)
		if err != nil {
			ctx.Log.Error("DEBUG CHAIN: Failed to get last accepted block", zap.Error(err))
			return nil, err
		}
		c.verifiedBlocks[lastAcceptedID] = lastAccepted
		ctx.Log.Info("DEBUG CHAIN: Last accepted block loaded successfully")
	}

	ctx.Log.Info("DEBUG CHAIN: Chain initialization complete")
	return c, nil
}

func (c *chain) LastAccepted() ids.ID {
	return c.lastAcceptedID
}

func (c *chain) SetChainState(state snow.State) {
	c.chainState = state
}

func (c *chain) GetBlock(blkID ids.ID) (Block, error) {
	return c.getBlock(blkID)
}

func (c *chain) NewBlock(blk *block.Block) (Block, error) {
	blkID, err := blk.ID()
	if err != nil {
		return nil, err
	}

	if wrapper, exists := c.verifiedBlocks[blkID]; exists {
		return wrapper, nil
	}

	blkBytes, err := blk.Bytes()
	if err != nil {
		return nil, err
	}

	return &blockWrapper{
		Block: blk,
		chain: c,
		id:    blkID,
		bytes: blkBytes,
	}, nil
}

func (c *chain) getBlock(blkID ids.ID) (*blockWrapper, error) {
	if wrapper, exists := c.verifiedBlocks[blkID]; exists {
		return wrapper, nil
	}

	// Get block header from state
	header, err := state.GetBlockHeader(c.acceptedState, blkID)
	if err != nil {
		return nil, err
	}

	// Reconstruct block from header
	blk := &block.Block{
		ParentID:  header.ParentHash,
		Height:    header.Number,
		Timestamp: header.Timestamp,
		Messages:  header.Messages,
	}

	blkBytes, err := blk.Bytes()
	if err != nil {
		return nil, err
	}

	return &blockWrapper{
		Block: blk,
		chain: c,
		id:    blkID,
		bytes: blkBytes,
	}, nil
}

// blockWrapper wraps a block with chain-specific functionality
type blockWrapper struct {
	*block.Block

	chain *chain

	id    ids.ID
	bytes []byte

	state               *versiondb.Database
	verifiedChildrenIDs set.Set[ids.ID]
}

func (b *blockWrapper) ID() ids.ID {
	return b.id
}

func (b *blockWrapper) Parent() ids.ID {
	return b.ParentID
}

func (b *blockWrapper) Bytes() []byte {
	return b.bytes
}

func (b *blockWrapper) Height() uint64 {
	return b.Block.Height
}

func (b *blockWrapper) Timestamp() time.Time {
	return b.Block.Time()
}

func (b *blockWrapper) Verify(ctx context.Context) error {
	return b.VerifyWithContext(ctx, nil)
}

// Accept implements the snowman.Block interface
func (b *blockWrapper) Accept(ctx context.Context) error {
	// Commit the state changes to the database
	if b.state != nil {
		if err := b.state.Commit(); err != nil {
			return err
		}
	}

	// Update block header in state
	header := &state.BlockHeader{
		Number:     b.Block.Height,
		Hash:       b.id,
		ParentHash: b.ParentID,
		Timestamp:  b.Block.Timestamp,
		Messages:   b.Block.Messages,
	}

	if err := state.SetBlockHeader(b.chain.acceptedState, header); err != nil {
		return err
	}

	// Update last accepted
	if err := state.SetLastAcceptedBlockID(b.chain.acceptedState, b.id); err != nil {
		return err
	}

	// Update children to point to base state
	for childID := range b.verifiedChildrenIDs {
		child, exists := b.chain.verifiedBlocks[childID]
		if !exists {
			return errMissingChild
		}
		if child.state != nil {
			if err := child.state.SetDatabase(b.chain.acceptedState); err != nil {
				return err
			}
		}
	}

	b.chain.lastAcceptedID = b.id
	delete(b.chain.verifiedBlocks, b.ParentID)
	b.state = nil

	b.chain.chainContext.Log.Info("accepted block",
		zap.Uint64("height", b.Height()),
		zap.Stringer("id", b.id),
		zap.Stringer("parent", b.ParentID),
	)

	return nil
}

// Reject implements the snowman.Block interface
func (b *blockWrapper) Reject(context.Context) error {
	delete(b.chain.verifiedBlocks, b.id)
	b.state = nil

	b.chain.chainContext.Log.Info("rejected block",
		zap.Uint64("height", b.Height()),
		zap.Stringer("id", b.id),
	)

	return nil
}

func (b *blockWrapper) ShouldVerifyWithContext(context.Context) (bool, error) {
	// For this simple VM, we don't need block context
	return false, nil
}

// VerifyWithContext implements the smblock.WithVerifyContext interface
func (b *blockWrapper) VerifyWithContext(ctx context.Context, blockContext *smblock.Context) error {
	timestamp := b.Time()
	if time.Until(timestamp) > maxClockSkew {
		return errFutureTimestamp
	}

	// Parent block must be verified or accepted
	parent, exists := b.chain.verifiedBlocks[b.ParentID]
	if !exists {
		return errMissingParent
	}

	if b.Block.Height != parent.Block.Height+1 {
		return errWrongHeight
	}

	parentTimestamp := parent.Time()
	if timestamp.Before(parentTimestamp) {
		return errTimestampBeforeParent
	}

	parentState, err := parent.State()
	if err != nil {
		return err
	}

	// Create versioned database on top of parent state
	blkState := versiondb.New(parentState)

	// For this simple VM, we just verify the block structure is valid
	// In a full implementation, you would verify:
	// - Message signatures
	// - State transitions
	// - Nonces
	// - Fees
	// etc.

	// Store the state for this block
	if b.state == nil {
		b.state = blkState
		parent.verifiedChildrenIDs.Add(b.id)
		b.chain.verifiedBlocks[b.id] = b
	}

	b.chain.chainContext.Log.Info("verified block",
		zap.Uint64("height", b.Height()),
		zap.Stringer("id", b.id),
		zap.Stringer("parent", b.ParentID),
		zap.Time("timestamp", timestamp),
	)

	return nil
}

// State returns the database state for this block
func (b *blockWrapper) State() (database.Database, error) {
	if b.id == b.chain.lastAcceptedID {
		return b.chain.acceptedState, nil
	}

	if b.state == nil {
		return nil, errParentNotVerified
	}

	return b.state, nil
}
