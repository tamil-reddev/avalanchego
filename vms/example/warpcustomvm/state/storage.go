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
	warpMessagePrefix     = []byte("warp")
	blockHeaderPrefix     = []byte("blockheader")
	blockHeightPrefix     = []byte("blockheight")
	lastAcceptedPrefix    = []byte("lastaccepted")
	lastMessageIDPrefix   = []byte("lastmsgid")
	receivedMessagePrefix = []byte("received")       // Stores received messages from other chains
	receivedMessageIDsKey = []byte("receivedmsgids") // Stores list of all received message IDs

	ErrNotFound = errors.New("not found")
)

// KeyValueWriter is an alias for database.KeyValueWriter
type KeyValueWriter = database.KeyValueWriter

// BlockHeader represents minimal block metadata
type BlockHeader struct {
	Number       uint64            `json:"number"`
	Hash         ids.ID            `json:"hash"`
	ParentHash   ids.ID            `json:"parentHash"`
	Timestamp    int64             `json:"timestamp"`
	Messages     []ids.ID          `json:"messages"`
	WarpMessages map[string][]byte `json:"warpMessages"` // Full message bytes
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

	// Initialize WarpMessages if nil (backward compatibility with old blocks)
	if header.WarpMessages == nil {
		header.WarpMessages = make(map[string][]byte)
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
			Number:       0,
			Hash:         ids.Empty,
			ParentHash:   ids.Empty,
			Timestamp:    0,
			Messages:     []ids.ID{},
			WarpMessages: make(map[string][]byte),
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

// SetLastMessageID stores the last allocated Teleporter message ID
func SetLastMessageID(db database.KeyValueWriter, messageID uint64) error {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, messageID)
	return db.Put(lastMessageIDPrefix, data)
}

// GetLastMessageID retrieves the last allocated Teleporter message ID
func GetLastMessageID(db database.KeyValueReader) (uint64, error) {
	data, err := db.Get(lastMessageIDPrefix)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return 0, nil // Start from 0 if not found
		}
		return 0, err
	}

	if len(data) != 8 {
		return 0, errors.New("invalid last message ID length")
	}

	return binary.BigEndian.Uint64(data), nil
}

// ReceivedMessage represents a Warp message received from another chain
type ReceivedMessage struct {
	MessageID       ids.ID `json:"messageID"`
	SourceChainID   ids.ID `json:"sourceChainID"`
	SourceAddress   []byte `json:"sourceAddress"`
	Payload         []byte `json:"payload"`
	ReceivedAt      int64  `json:"receivedAt"`      // Unix timestamp
	BlockHeight     uint64 `json:"blockHeight"`     // Block height when received
	SignedMessage   []byte `json:"signedMessage"`   // Full signed Warp message bytes
	UnsignedMessage []byte `json:"unsignedMessage"` // Unsigned Warp message bytes
}

// SetReceivedMessage stores a received Warp message in the database
func SetReceivedMessage(db database.Database, msg *ReceivedMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	key := append(receivedMessagePrefix, msg.MessageID[:]...)
	if err := db.Put(key, data); err != nil {
		return err
	}

	// Add message ID to the list of received message IDs
	return appendReceivedMessageID(db, msg.MessageID)
}

// GetReceivedMessage retrieves a received Warp message by ID
func GetReceivedMessage(db database.KeyValueReader, messageID ids.ID) (*ReceivedMessage, error) {
	key := append(receivedMessagePrefix, messageID[:]...)
	data, err := db.Get(key)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var msg ReceivedMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// appendReceivedMessageID adds a message ID to the list of received message IDs
func appendReceivedMessageID(db database.Database, messageID ids.ID) error {
	// Get existing list
	data, err := db.Get(receivedMessageIDsKey)
	var messageIDs []ids.ID
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			return err
		}
		// List doesn't exist yet, start fresh
		messageIDs = []ids.ID{}
	} else {
		// Unmarshal existing list
		if err := json.Unmarshal(data, &messageIDs); err != nil {
			return err
		}
	}

	// Append new message ID
	messageIDs = append(messageIDs, messageID)

	// Marshal and save
	data, err = json.Marshal(messageIDs)
	if err != nil {
		return err
	}

	return db.Put(receivedMessageIDsKey, data)
}

// GetAllReceivedMessageIDs retrieves all received message IDs
func GetAllReceivedMessageIDs(db database.KeyValueReader) ([]ids.ID, error) {
	data, err := db.Get(receivedMessageIDsKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return []ids.ID{}, nil
		}
		return nil, err
	}

	var messageIDs []ids.ID
	if err := json.Unmarshal(data, &messageIDs); err != nil {
		return nil, err
	}

	return messageIDs, nil
}
