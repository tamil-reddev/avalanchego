// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"encoding/json"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

// Block represents a block in the warpcustomvm blockchain
type Block struct {
	// ParentID is the ID of the parent block
	ParentID ids.ID `json:"parentID"`

	// Height is the block number
	Height uint64 `json:"height"`

	// Timestamp is when the block was created (Unix timestamp)
	Timestamp int64 `json:"timestamp"`

	// Messages contains the IDs of Teleporter messages included in this block
	Messages []ids.ID `json:"messages"`
}

// ID computes the unique identifier for this block
func (b *Block) ID() (ids.ID, error) {
	bytes, err := b.Bytes()
	if err != nil {
		return ids.ID{}, err
	}
	return hashing.ComputeHash256Array(bytes), nil
}

// Bytes returns the JSON encoding of the block
func (b *Block) Bytes() ([]byte, error) {
	return json.Marshal(b)
}

// Time returns the block timestamp as a time.Time
func (b *Block) Time() time.Time {
	return time.Unix(b.Timestamp, 0)
}

// Parse parses a block from bytes
func Parse(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	return &block, err
}
