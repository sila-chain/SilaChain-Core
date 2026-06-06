// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package ethapi

import (
	"context"

	"github.com/sila-org/sila/internal/silaapi/simapi"
	"github.com/sila-org/sila/rpc"
)

type simBlock = simapi.SimBlock
type simCallResult = simapi.SimCallResult
type simBlockResult = simapi.SimBlockResult
type simOpts = simapi.SimOpts

const errCodeVMError = -32015

func (api *BlockChainAPI) SimulateV1(ctx context.Context, opts simOpts, blockNrOrHash *rpc.BlockNumberOrHash) ([]*simBlockResult, error) {
	return simapi.SimulateV1(ctx, api.b, opts, blockNrOrHash)
}
