// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ava-labs/avalanchego/snow"
)

// EthRPCHandler provides an Ethereum JSON-RPC compatible handler
// that translates eth_* method calls to the appropriate handlers
type EthRPCHandler struct {
	ctx *snow.Context
}

// NewEthRPCHandler creates a new Ethereum-compatible JSON-RPC handler
func NewEthRPCHandler(ctx *snow.Context) *EthRPCHandler {
	return &EthRPCHandler{ctx: ctx}
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      interface{}     `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ServeHTTP handles Ethereum JSON-RPC requests
func (h *EthRPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, nil, -32700, "Parse error")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, nil, -32700, "Parse error")
		return
	}

	// Handle the method
	var result interface{}
	switch req.Method {
	case "eth_chainId":
		result = h.handleChainID()
	default:
		h.writeError(w, req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		return
	}

	// Write successful response
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleChainID handles the eth_chainId method
// Returns a hardcoded chain ID for testing (0x539 = 1337 in decimal)
func (h *EthRPCHandler) handleChainID() string {
	//return "0x539"
	return fmt.Sprintf("0x%x", h.ctx.ChainID)
}

// writeError writes a JSON-RPC error response
func (h *EthRPCHandler) writeError(w http.ResponseWriter, id interface{}, code int, message string) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}

	w.WriteHeader(http.StatusOK) // JSON-RPC errors still return 200 OK
	json.NewEncoder(w).Encode(response)
}
