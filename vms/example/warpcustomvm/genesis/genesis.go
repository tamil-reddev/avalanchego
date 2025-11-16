// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"encoding/json"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
)

// Genesis represents the genesis state of the VM
type Genesis struct {
	// Timestamp is the genesis block timestamp
	Timestamp int64 `json:"timestamp"`
}

// Default returns a default genesis configuration
func Default() *Genesis {
	return &Genesis{
		Timestamp: 0,
	}
}

// Bytes returns the JSON encoding of the genesis
func (g *Genesis) Bytes() ([]byte, error) {
	return json.Marshal(g)
}

// Initialize initializes the genesis state in the database
func Initialize(db state.KeyValueWriter, genesis *Genesis) error {
	// Create genesis block header
	genesisHeader := &state.BlockHeader{
		Number:     0,
		Hash:       ids.Empty,
		ParentHash: ids.Empty,
		Timestamp:  genesis.Timestamp,
		Messages:   []ids.ID{}, // Genesis block has no messages
	}

	// Store genesis block header
	if err := state.SetBlockHeader(db, genesisHeader); err != nil {
		return err
	}

	// Set genesis as last accepted
	if err := state.SetLastAcceptedBlockID(db, ids.Empty); err != nil {
		return err
	}

	return nil
}
