// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package warpcustomvm

import (
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms"
)

var _ vms.Factory = (*Factory)(nil)

// Factory implements the vms.Factory interface
type Factory struct{}

// New returns a new instance of the VM
func (*Factory) New(logging.Logger) (interface{}, error) {
	return &VM{}, nil
}
