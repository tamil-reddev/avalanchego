// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var (
	// Prefixes for different data types in the database
	warpMessagePrefix  = []byte("warp")
	blockHeaderPrefix  = []byte("blockheader")
	blockHeightPrefix  = []byte("blockheight")
	lastAcceptedPrefix = []byte("lastaccepted")

	ErrNotFound = errors.New("not found")
)

// KeyValueWriter is an alias for database.KeyValueWriter
type KeyValueWriter = database.KeyValueWriter

// BlockHeader represents minimal block metadata
type BlockHeader struct {
	Number     uint64   `json:"number"`
	Hash       ids.ID   `json:"hash"`
	ParentHash ids.ID   `json:"parentHash"`
	Timestamp  int64    `json:"timestamp"`
	Messages   []ids.ID `json:"messages"`
}

// SetBlockHeader stores a block header in the database
func SetBlockHeader(db database.KeyValueWriter, header *BlockHeader) error {
	data, err := json.Marshal(header)
	if err != nil {
		return err
	}

	// Store by block hash
	key := append(blockHeaderPrefix, header.Hash[:]...)
	if err := db.Put(key, data); err != nil {
		return err
	}

	// Store height -> hash mapping
	heightKey := make([]byte, len(blockHeightPrefix)+8)
	copy(heightKey, blockHeightPrefix)
	binary.BigEndian.PutUint64(heightKey[len(blockHeightPrefix):], header.Number)
	return db.Put(heightKey, header.Hash[:])
}

// GetBlockHeader retrieves a block header by hash
func GetBlockHeader(db database.KeyValueReader, blockHash ids.ID) (*BlockHeader, error) {
	key := append(blockHeaderPrefix, blockHash[:]...)
	data, err := db.Get(key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var header BlockHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}

	return &header, nil
}

// GetBlockIDByHeight retrieves a block ID by height
func GetBlockIDByHeight(db database.KeyValueReader, height uint64) (ids.ID, error) {
	heightKey := make([]byte, len(blockHeightPrefix)+8)
	copy(heightKey, blockHeightPrefix)
	binary.BigEndian.PutUint64(heightKey[len(blockHeightPrefix):], height)

	data, err := db.Get(heightKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ids.Empty, ErrNotFound
		}
		return ids.Empty, err
	}

	var blockID ids.ID
	copy(blockID[:], data)
	return blockID, nil
}

// SetLastAcceptedBlockID stores the last accepted block ID
func SetLastAcceptedBlockID(db database.KeyValueWriter, blockID ids.ID) error {
	return db.Put(lastAcceptedPrefix, blockID[:])
}

// GetLastAcceptedBlockID retrieves the last accepted block ID
func GetLastAcceptedBlockID(db database.KeyValueReader) (ids.ID, error) {
	data, err := db.Get(lastAcceptedPrefix)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ids.Empty, nil // Return empty ID for genesis
		}
		return ids.Empty, err
	}

	var blockID ids.ID
	copy(blockID[:], data)
	return blockID, nil
}

// GetLatestBlockHeader retrieves the latest accepted block header
func GetLatestBlockHeader(db database.KeyValueReader) (*BlockHeader, error) {
	blockID, err := GetLastAcceptedBlockID(db)
	if err != nil {
		return nil, err
	}

	if blockID == ids.Empty {
		// Return genesis block header
		return &BlockHeader{
			Number:     0,
			Hash:       ids.Empty,
			ParentHash: ids.Empty,
			Timestamp:  0,
		}, nil
	}

	return GetBlockHeader(db, blockID)
}

// SetWarpMessage stores an unsigned Warp message in the database
func SetWarpMessage(db database.KeyValueWriter, messageID ids.ID, message *warp.UnsignedMessage) error {
	key := append(warpMessagePrefix, messageID[:]...)
	return db.Put(key, message.Bytes())
}

// GetWarpMessage retrieves an unsigned Warp message by ID
func GetWarpMessage(db database.KeyValueReader, messageID ids.ID) (*warp.UnsignedMessage, error) {
	key := append(warpMessagePrefix, messageID[:]...)
	data, err := db.Get(key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return warp.ParseUnsignedMessage(data)
}
