// Copyright 2015 The sila Authors
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

package runtime

import (
	"github.com/holiman/uint256"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/params"
)

func NewEnv(cfg *Config) *vm.EVM {
	txContext := vm.TxContext{
		Origin:     cfg.Origin,
		GasPrice:   uint256.MustFromBig(cfg.GasPrice),
		BlobHashes: cfg.BlobHashes,
	}
	blockContext := vm.BlockContext{
		CanTransfer:      core.CanTransfer,
		Transfer:         core.Transfer,
		GetHash:          cfg.GetHashFn,
		Coinbase:         cfg.Coinbase,
		BlockNumber:      cfg.BlockNumber,
		Time:             cfg.Time,
		Difficulty:       cfg.Difficulty,
		GasLimit:         cfg.GasLimit,
		BaseFee:          cfg.BaseFee,
		BlobBaseFee:      cfg.BlobBaseFee,
		Random:           cfg.Random,
		CostPerStateByte: params.CostPerStateByte,
	}

	evm := vm.NewEVM(blockContext, cfg.State, cfg.ChainConfig, cfg.EVMConfig)
	evm.SetTxContext(txContext)
	return evm
}
