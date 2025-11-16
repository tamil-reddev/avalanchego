// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/rpc"
)

// WarpPrecompileAddress is the low-level Warp messenger precompile
var WarpPrecompileAddress = []byte{
	0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
}

// TeleporterContractAddress is the deployed Teleporter contract address
// This is the address that the ICM relayer expects for Teleporter protocol messages
// Address: 0x253b2784c75e510dD0fF1da844684a1aC0aa5fcf
var TeleporterContractAddress = []byte{
	0x25, 0x3b, 0x27, 0x84, 0xc7, 0x5e, 0x51, 0x0d, 0xD0, 0xfF,
	0x1d, 0xa8, 0x44, 0x68, 0x4a, 0x1a, 0xC0, 0xaa, 0x5f, 0xcf,
}

// TeleporterPrecompileAddress is an alias - kept for backward compatibility
// Now points to the Teleporter contract address instead of Warp precompile
var TeleporterPrecompileAddress = TeleporterContractAddress

// NewClient creates a new JSON-RPC client for warpcustomvm
func NewClient(uri, chain string) *Client {
	path := fmt.Sprintf(
		"%s/ext/%s/%s",
		uri,
		constants.ChainAliasPrefix,
		chain,
	)
	return &Client{
		Req: rpc.NewEndpointRequester(path),
	}
}

// Client provides JSON-RPC client for warpcustomvm API
type Client struct {
	Req rpc.EndpointRequester
}

// GetChainIDReply is the response for getting the blockchain ID
type GetChainIDReply struct {
	ChainID   ids.ID `json:"chainID"`
	NetworkID uint32 `json:"networkID"`
}

// SubmitMessageArgs is the args for submitting a new Warp message
type SubmitMessageArgs struct {
	Payload []byte `json:"payload"` // Actual message payload (can contain destination info)
}

// SubmitMessageReply is the response from submitting a message
type SubmitMessageReply struct {
	MessageID ids.ID `json:"messageID"`
}

// GetMessageArgs is the args for getting a message
type GetMessageArgs struct {
	MessageID ids.ID `json:"messageID"`
}

// GetMessageReply is the response for getting a Warp message
type GetMessageReply struct {
	MessageID            ids.ID `json:"messageID"`            // TxID for relayer
	NetworkID            uint32 `json:"networkID"`            // Network ID
	SourceChainID        ids.ID `json:"sourceChainID"`        // Source blockchain ID
	SourceAddress        []byte `json:"sourceAddress"`        // Address in source VM
	Payload              []byte `json:"payload"`              // Message payload
	UnsignedMessageBytes []byte `json:"unsignedMessageBytes"` // Full unsigned Warp message bytes
}

// GetBlockArgs is the args for getting a block
type GetBlockArgs struct {
	Height uint64 `json:"height,omitempty"`
}

// MessageMetadata contains contextual information about a message
type MessageMetadata struct {
	Timestamp   int64  `json:"timestamp"`
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   ids.ID `json:"blockHash"`
}

// MessageDetail contains full details of a Warp message in a block
type MessageDetail struct {
	MessageID            ids.ID          `json:"messageID"`            // TxID for relayer
	NetworkID            uint32          `json:"networkID"`            // Network ID
	SourceChainID        ids.ID          `json:"sourceChainID"`        // Source blockchain ID
	SourceAddress        []byte          `json:"sourceAddress"`        // Address in source VM
	Payload              []byte          `json:"payload"`              // Message payload
	UnsignedMessageBytes []byte          `json:"unsignedMessageBytes"` // Full unsigned Warp message bytes
	Metadata             MessageMetadata `json:"metadata"`             // Block metadata
}

// GetBlockReply is the response for getting a block
type GetBlockReply struct {
	BlockID   ids.ID          `json:"blockID"`
	ParentID  ids.ID          `json:"parentID"`
	Height    uint64          `json:"height"`
	Timestamp int64           `json:"timestamp"`
	Messages  []MessageDetail `json:"messages"`
}

// GetWarpMessageArgs is the args for getting an unsigned Warp message
type GetWarpMessageArgs struct {
	MessageID ids.ID `json:"messageID"`
}

// GetWarpMessageReply is the response for getting an unsigned Warp message
type GetWarpMessageReply struct {
	MessageID        ids.ID `json:"messageID"`
	UnsignedMessage  string `json:"unsignedMessage"` // Hex-encoded
	SourceChainID    ids.ID `json:"sourceChainID"`
	DestinationChain ids.ID `json:"destinationChain"`
	DestinationAddr  string `json:"destinationAddress"`
}

// GetChainID retrieves the blockchain ID and network ID via JSON-RPC
func (c *Client) GetChainID(
	ctx context.Context,
	options ...rpc.Option,
) (*GetChainIDReply, error) {
	resp := new(GetChainIDReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.getChainID",
		&struct{}{},
		resp,
		options...,
	)
	return resp, err
}

// SubmitMessage submits a new Warp message to the VM via JSON-RPC
// Note: The source address is automatically set to the Teleporter precompile address on the server
func (c *Client) SubmitMessage(
	ctx context.Context,
	payload []byte,
	options ...rpc.Option,
) (ids.ID, error) {
	resp := new(SubmitMessageReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.submitMessage",
		&SubmitMessageArgs{
			Payload: payload,
		},
		resp,
		options...,
	)
	return resp.MessageID, err
}

// GetMessage retrieves a message by its ID via JSON-RPC
func (c *Client) GetMessage(
	ctx context.Context,
	messageID ids.ID,
	options ...rpc.Option,
) (*GetMessageReply, error) {
	resp := new(GetMessageReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.getMessage",
		&GetMessageArgs{
			MessageID: messageID,
		},
		resp,
		options...,
	)
	return resp, err
}

// GetLatestBlock retrieves the latest accepted block via JSON-RPC
func (c *Client) GetLatestBlock(
	ctx context.Context,
	options ...rpc.Option,
) (*GetBlockReply, error) {
	resp := new(GetBlockReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.getLatestBlock",
		&struct{}{},
		resp,
		options...,
	)
	return resp, err
}

// GetBlock retrieves a block by its height via JSON-RPC
func (c *Client) GetBlock(
	ctx context.Context,
	height uint64,
	options ...rpc.Option,
) (*GetBlockReply, error) {
	resp := new(GetBlockReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.getBlock",
		&GetBlockArgs{
			Height: height,
		},
		resp,
		options...,
	)
	return resp, err
}

// GetWarpMessage retrieves an unsigned Warp message by its ID via JSON-RPC
// This is used by relayers to fetch messages for cross-chain delivery
func (c *Client) GetWarpMessage(
	ctx context.Context,
	messageID ids.ID,
	options ...rpc.Option,
) (*GetWarpMessageReply, error) {
	resp := new(GetWarpMessageReply)
	err := c.Req.SendRequest(
		ctx,
		"warpcustomvm.getWarpMessage",
		&GetWarpMessageArgs{
			MessageID: messageID,
		},
		resp,
		options...,
	)
	return resp, err
}
