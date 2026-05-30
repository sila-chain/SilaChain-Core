// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package blockapi

import (
	"context"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

var RPCMarshalBlock = ethapi.RPCMarshalBlock

// BlockChainBackend is the minimal backend required by Sila blockchain helpers.
type BlockChainBackend interface {
	ChainConfig() *params.ChainConfig
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
}

// ChainId returns the replay-protection chain id for the current SilaChain config.
func ChainId(b BlockChainBackend) *hexutil.Big {
	return (*hexutil.Big)(b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func BlockNumber(b BlockChainBackend) hexutil.Uint64 {
	header, _ := b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber)
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the given block number or hash.
func GetBalance(ctx context.Context, b BlockChainBackend, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	balance := state.GetBalance(address).ToBig()
	return (*hexutil.Big)(balance), state.Error()
}
