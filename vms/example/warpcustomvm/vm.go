// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warpcustomvm

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/network/p2p"
	"github.com/ava-labs/avalanchego/network/p2p/acp118"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/version"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/api"
	xblock "github.com/ava-labs/avalanchego/vms/example/warpcustomvm/block"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/builder"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/chain"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/genesis"
	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm/state"
)

var _ block.ChainVM = (*VM)(nil)

// VM implements the Avalanche ChainVM interface
type VM struct {
	*p2p.Network // P2P network for Warp message signing (ACP-118)

	chainContext *snow.Context

	acceptedState database.Database
	chain         chain.Chain
	builder       builder.Builder
	toEngine      chan<- common.Message
}

// Initialize implements the snowman.ChainVM interface
func (vm *VM) Initialize(
	ctx context.Context,
	chainContext *snow.Context,
	db database.Database,
	genesisBytes []byte,
	_ []byte,
	_ []byte,
	_ []*common.Fx,
	appSender common.AppSender,
) error {
	chainContext.Log.Info("initializing warpCustomVM with Warp message support")

	// Initialize P2P network for Warp message signing
	metrics := prometheus.NewRegistry()
	err := chainContext.Metrics.Register("p2p", metrics)
	if err != nil {
		return fmt.Errorf("failed to register p2p metrics: %w", err)
	}

	vm.Network, err = p2p.NewNetwork(
		chainContext.Log,
		appSender,
		metrics,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to create p2p network: %w", err)
	}

	// Create ACP-118 handler for Warp message signing
	// This is CRITICAL for ICM relayers to work!
	verifier := &warpVerifier{vm: vm}
	acp118Handler := acp118.NewHandler(
		verifier,
		chainContext.WarpSigner,
	)

	// Wrap handler to add logging
	loggingHandler := &loggingACP118Handler{
		Handler: acp118Handler,
		log:     chainContext.Log,
	}

	if err := vm.Network.AddHandler(p2p.SignatureRequestHandlerID, loggingHandler); err != nil {
		return fmt.Errorf("failed to add acp118 handler: %w", err)
	}

	chainContext.Log.Info("ACP-118 Warp signature handler registered successfully",
		zap.String("handlerID", fmt.Sprintf("%d", p2p.SignatureRequestHandlerID)),
	)

	vm.chainContext = chainContext
	vm.acceptedState = db

	// Parse genesis
	var gen *genesis.Genesis
	if len(genesisBytes) > 0 {
		gen = &genesis.Genesis{}
		if err := stdjson.Unmarshal(genesisBytes, gen); err != nil {
			return fmt.Errorf("failed to unmarshal genesis: %w", err)
		}
	} else {
		gen = genesis.Default()
	}

	// Initialize genesis state if this is the first run
	// Check if genesis is already initialized by trying to get the genesis block header
	chainContext.Log.Info("DEBUG: Checking if genesis is initialized")
	_, err = state.GetBlockHeader(vm.acceptedState, ids.Empty)
	if err != nil {
		// Genesis not initialized yet
		chainContext.Log.Info("DEBUG: Genesis not found, initializing...", zap.Error(err))
		if err := genesis.Initialize(vm.acceptedState, gen); err != nil {
			chainContext.Log.Error("DEBUG: Failed to initialize genesis", zap.Error(err))
			return fmt.Errorf("failed to initialize genesis: %w", err)
		}
		chainContext.Log.Info("DEBUG: Genesis initialized successfully")
	} else {
		chainContext.Log.Info("DEBUG: Genesis already initialized")
	}

	chainContext.Log.Info("DEBUG: Getting last accepted block ID")
	lastAcceptedID, err := state.GetLastAcceptedBlockID(vm.acceptedState)
	if err != nil {
		chainContext.Log.Error("DEBUG: Failed to get last accepted block ID", zap.Error(err))
		return fmt.Errorf("failed to get last accepted block ID: %w", err)
	}
	chainContext.Log.Info("DEBUG: Last accepted block ID", zap.Stringer("blockID", lastAcceptedID))

	// Create chain
	chainContext.Log.Info("DEBUG: Creating chain")
	vm.chain, err = chain.New(chainContext, vm.acceptedState)
	if err != nil {
		chainContext.Log.Error("DEBUG: Failed to create chain", zap.Error(err))
		return fmt.Errorf("failed to create chain: %w", err)
	}
	chainContext.Log.Info("DEBUG: Chain created successfully")

	// Create builder with database access for Warp message storage
	vm.builder = builder.New(chainContext, vm.chain, vm.acceptedState)
	vm.builder.SetPreference(lastAcceptedID)

	chainContext.Log.Info("initialized vm",
		zap.Stringer("lastAccepted", lastAcceptedID),
		zap.String("lastAcceptedHex", lastAcceptedID.String()),
	)

	return nil
}

// Bootstrapping implements the block.ChainVM interface
func (vm *VM) Bootstrapping(context.Context) error {
	return nil
}

// Bootstrapped implements the block.ChainVM interface
func (vm *VM) Bootstrapped(context.Context) error {
	return nil
}

// Shutdown implements the block.ChainVM interface
func (vm *VM) Shutdown(context.Context) error {
	if vm.acceptedState != nil {
		return vm.acceptedState.Close()
	}
	return nil
}

// Version implements the block.ChainVM interface
func (vm *VM) Version(context.Context) (string, error) {
	return version.Current.String(), nil
}

// CreateHandlers implements the block.ChainVM interface
func (vm *VM) CreateHandlers(context.Context) (map[string]http.Handler, error) {
	// Create JSON-RPC server for warpcustomvm methods
	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")

	// Create API service
	apiServer := api.NewServer(
		vm.chainContext,
		vm.chain,
		vm.builder,
		vm.acceptedState,
	)

	// Register the API service with the RPC server
	if err := server.RegisterService(apiServer, "warpcustomvm"); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	return map[string]http.Handler{
		"":     server,
		"/rpc": server,
	}, nil
}

// CreateStaticHandlers implements the block.ChainVM interface
func (vm *VM) CreateStaticHandlers(context.Context) (map[string]http.Handler, error) {
	// Add a simple health check handler that deployment scripts can use
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","vm":"warpcustomvm"}`))
	})

	return map[string]http.Handler{
		"/health": handler,
	}, nil
}

// HealthCheck implements the block.ChainVM interface
func (vm *VM) HealthCheck(context.Context) (interface{}, error) {
	return http.StatusOK, nil
}

// SetState implements the block.ChainVM interface
func (vm *VM) SetState(_ context.Context, state snow.State) error {
	switch state {
	case snow.Bootstrapping:
		vm.chainContext.Log.Info("state transition to bootstrapping")
	case snow.NormalOp:
		vm.chainContext.Log.Info("state transition to normal operation")
	default:
		return fmt.Errorf("unknown state: %s", state)
	}

	vm.chain.SetChainState(state)
	return nil
}

// BuildBlock implements the block.ChainVM interface
func (vm *VM) BuildBlock(ctx context.Context) (snowman.Block, error) {
	// Get the current timestamp for block context
	timestamp := time.Now()
	blockContext := &block.Context{
		PChainHeight: 0, // Not used in this VM
	}

	// Build block using builder
	blk, err := vm.builder.BuildBlock(ctx, blockContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build block: %w", err)
	}

	vm.chainContext.Log.Info("built block",
		zap.Stringer("blockID", blk.ID()),
		zap.Uint64("height", blk.Height()),
		zap.Time("timestamp", timestamp),
	)

	return blk, nil
}

// ParseBlock implements the block.ChainVM interface
func (vm *VM) ParseBlock(_ context.Context, blockBytes []byte) (snowman.Block, error) {
	// Parse block bytes
	var blk xblock.Block
	if err := stdjson.Unmarshal(blockBytes, &blk); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	vm.chainContext.Log.Info("📦 ParseBlock received block from network",
		zap.Uint64("height", blk.Height),
		zap.Int("messageIDs", len(blk.Messages)),
		zap.Int("warpMessages", len(blk.WarpMessages)),
	)

	// Create block wrapper through chain
	wrapper, err := vm.chain.NewBlock(&blk)
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	return wrapper, nil
}

// GetBlock implements the block.ChainVM interface
func (vm *VM) GetBlock(_ context.Context, blockID ids.ID) (snowman.Block, error) {
	return vm.chain.GetBlock(blockID)
}

// SetPreference implements the block.ChainVM interface
func (vm *VM) SetPreference(_ context.Context, preferred ids.ID) error {
	vm.chainContext.Log.Debug("SetPreference called", zap.Stringer("preferred", preferred))
	vm.builder.SetPreference(preferred)
	return nil
}

// LastAccepted implements the block.ChainVM interface
func (vm *VM) LastAccepted(context.Context) (ids.ID, error) {
	return vm.chain.LastAccepted(), nil
}

// VerifyHeightIndex implements the block.HeightIndexedChainVM interface
func (vm *VM) VerifyHeightIndex(context.Context) error {
	return nil
}

// GetBlockIDAtHeight implements the block.HeightIndexedChainVM interface
func (vm *VM) GetBlockIDAtHeight(_ context.Context, height uint64) (ids.ID, error) {
	return state.GetBlockIDByHeight(vm.acceptedState, height)
}

// AppGossip implements the block.ChainVM interface
func (vm *VM) AppGossip(ctx context.Context, nodeID ids.NodeID, msg []byte) error {
	vm.chainContext.Log.Debug("📡 AppGossip received",
		zap.String("from", nodeID.String()),
		zap.Int("messageSize", len(msg)),
	)

	return vm.Network.AppGossip(ctx, nodeID, msg)
}

// AppRequest implements the block.ChainVM interface
// This is CRITICAL: Routes P2P requests (including ACP-118 signature requests) to the P2P network handlers
func (vm *VM) AppRequest(ctx context.Context, nodeID ids.NodeID, requestID uint32, deadline time.Time, request []byte) error {
	vm.chainContext.Log.Info("📨 AppRequest received - routing to P2P network",
		zap.String("from", nodeID.String()),
		zap.Uint32("requestID", requestID),
		zap.Int("requestSize", len(request)),
	)

	// Route to P2P network which will dispatch to registered handlers (including ACP-118)
	return vm.Network.AppRequest(ctx, nodeID, requestID, deadline, request)
}

// AppResponse implements the block.ChainVM interface
func (vm *VM) AppResponse(ctx context.Context, nodeID ids.NodeID, requestID uint32, response []byte) error {
	vm.chainContext.Log.Debug("📬 AppResponse received",
		zap.String("from", nodeID.String()),
		zap.Uint32("requestID", requestID),
		zap.Int("responseSize", len(response)),
	)

	return vm.Network.AppResponse(ctx, nodeID, requestID, response)
}

// AppRequestFailed implements the block.ChainVM interface
func (vm *VM) AppRequestFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32, appErr *common.AppError) error {
	vm.chainContext.Log.Warn("❌ AppRequest failed",
		zap.String("from", nodeID.String()),
		zap.Uint32("requestID", requestID),
		zap.Error(appErr),
	)

	return vm.Network.AppRequestFailed(ctx, nodeID, requestID, appErr)
}

// CrossChainAppRequest implements the block.ChainVM interface
func (vm *VM) CrossChainAppRequest(context.Context, ids.ID, uint32, time.Time, []byte) error {
	return nil
}

// CrossChainAppResponse implements the block.ChainVM interface
func (vm *VM) CrossChainAppResponse(context.Context, ids.ID, uint32, []byte) error {
	return nil
}

// CrossChainAppRequestFailed implements the block.ChainVM interface
func (vm *VM) CrossChainAppRequestFailed(context.Context, ids.ID, uint32, *common.AppError) error {
	return nil
}

// Connected implements the block.ChainVM interface
func (vm *VM) Connected(context.Context, ids.NodeID, *version.Application) error {
	return nil
}

// Disconnected implements the block.ChainVM interface
func (vm *VM) Disconnected(context.Context, ids.NodeID) error {
	return nil
}

// loggingACP118Handler wraps the ACP-118 handler to add request logging
type loggingACP118Handler struct {
	*acp118.Handler
	log logging.Logger
}

func (h *loggingACP118Handler) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	deadline time.Time,
	requestBytes []byte,
) ([]byte, *common.AppError) {
	h.log.Info("⚡ ACP-118 AppRequest received",
		zap.String("from", nodeID.String()),
		zap.Int("requestSize", len(requestBytes)),
		zap.Time("deadline", deadline),
	)

	response, err := h.Handler.AppRequest(ctx, nodeID, deadline, requestBytes)

	if err != nil {
		h.log.Error("❌ ACP-118 request failed",
			zap.String("from", nodeID.String()),
			zap.Error(err),
		)
	} else {
		h.log.Info("✓ ACP-118 signature response sent",
			zap.String("to", nodeID.String()),
			zap.Int("responseSize", len(response)),
		)
	}

	return response, err
}

// NewHTTPHandler implements the block.ChainVM interface for Connect RPC
func (vm *VM) NewHTTPHandler(context.Context) (http.Handler, error) {
	// Return nil for now - can be extended for Connect RPC support
	return nil, nil
}

// WaitForEvent implements block.ChainVM
func (vm *VM) WaitForEvent(ctx context.Context) (common.Message, error) {
	return vm.builder.WaitForEvent(ctx)
}
