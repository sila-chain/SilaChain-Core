// Copyright 2024 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

package simulated

import (
	"math/big"

	"github.com/sila-org/sila/sila/silaconfig"
	"github.com/sila-org/sila/node"
)

// WithBlockGasLimit configures the simulated backend to target a specific gas limit
// when producing blocks.
func WithBlockGasLimit(gaslimit uint64) func(nodeConf *node.Config, silaConf *silaconfig.Config) {
	return func(nodeConf *node.Config, silaConf *silaconfig.Config) {
		silaConf.Genesis.GasLimit = gaslimit
		silaConf.Miner.GasCeil = gaslimit
	}
}

// WithCallGasLimit configures the simulated backend to cap sila_calls to a specific
// gas limit when running client operations.
func WithCallGasLimit(gaslimit uint64) func(nodeConf *node.Config, silaConf *silaconfig.Config) {
	return func(nodeConf *node.Config, silaConf *silaconfig.Config) {
		silaConf.RPCGasCap = gaslimit
	}
}

// WithMinerMinTip configures the simulated backend to require a specific minimum
// gas tip for a transaction to be included.
//
// 0 is not possible as a live Sila node would reject that due to DoS protection,
// so the simulated backend will replicate that behavior for consistency.
func WithMinerMinTip(tip *big.Int) func(nodeConf *node.Config, silaConf *silaconfig.Config) {
	if tip == nil || tip.Sign() <= 0 {
		panic("invalid miner minimum tip")
	}
	return func(nodeConf *node.Config, silaConf *silaconfig.Config) {
		silaConf.Miner.GasPrice = tip
	}
}
