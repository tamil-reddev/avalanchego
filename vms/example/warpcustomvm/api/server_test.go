// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/utils/logging"
)

func TestServer_GetChainID(t *testing.T) {
	// Create a test context
	chainID := ids.GenerateTestID()
	networkID := uint32(1337)

	ctx := &snow.Context{
		ChainID:   chainID,
		NetworkID: networkID,
		Log:       logging.NoLog{},
	}

	// Create a minimal server (we only need ctx for GetChainID)
	server := &server{
		ctx: ctx,
	}

	// Call the GetChainID method
	var result GetChainIDReply
	err := server.GetChainID(&http.Request{}, &struct{}{}, &result)
	require.NoError(t, err)

	// Verify the results
	require.Equal(t, chainID, result.ChainID)
	require.Equal(t, networkID, result.NetworkID)

	t.Logf("Chain ID: %s", result.ChainID)
	t.Logf("Network ID: %d", result.NetworkID)
}
