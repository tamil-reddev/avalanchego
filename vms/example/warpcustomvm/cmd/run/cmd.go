// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package run

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/ava-labs/avalanchego/vms/example/warpcustomvm"
	"github.com/ava-labs/avalanchego/vms/rpcchainvm"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "warpcustomvm",
		Short: "Runs a WarpCustomVM plugin",
		RunE:  runFunc,
	}
}

func runFunc(*cobra.Command, []string) error {
	return rpcchainvm.Serve(context.Background(), &warpcustomvm.VM{})
}
