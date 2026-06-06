// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package registry

import (
	"context"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/backend"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	"github.com/sila-org/sila/internal/silaapi/proofapi"
	"github.com/sila-org/sila/internal/silaapi/simapi"
	"github.com/sila-org/sila/internal/silaapi/txapi"
	"github.com/sila-org/sila/rpc"
)

type blockChainAPI struct {
	b backend.Backend
	*blockapi.BlockChainAPI
	*callapi.API
	ProofAPI *proofapi.API
}

func NewSilaBlockChainAPI(apiBackend backend.Backend) interface{} {
	return &blockChainAPI{
		b:             apiBackend,
		BlockChainAPI: blockapi.NewBlockChainAPI(apiBackend),
		API:           callapi.NewAPI(apiBackend),
		ProofAPI:      proofapi.NewAPI(apiBackend),
	}
}

func (api *blockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*proofapi.AccountResult, error) {
	return proofapi.GetProof(ctx, api.b, address, storageKeys, blockNrOrHash)
}

func (api *blockChainAPI) SimulateV1(ctx context.Context, opts simapi.SimOpts, blockNrOrHash *rpc.BlockNumberOrHash) ([]*simapi.SimBlockResult, error) {
	return simapi.SimulateV1(ctx, api.b, opts, blockNrOrHash)
}

func NewSilaTransactionAPI(apiBackend backend.Backend, nonceLock *addrlock.AddrLocker) interface{} {
	return txapi.NewTransactionAPI(apiBackend, nonceLock)
}

func GetAPIs(apiBackend backend.Backend) []rpc.API {
	nonceLock := new(addrlock.AddrLocker)
	return []rpc.API{
		{Namespace: "sila", Service: silaapi.NewSilaAPI(apiBackend)},
		{Namespace: "eth", Service: silaapi.NewSilaAPI(apiBackend)},
		{Namespace: "sila", Service: NewSilaBlockChainAPI(apiBackend)},
		{Namespace: "eth", Service: NewSilaBlockChainAPI(apiBackend)},
		{Namespace: "sila", Service: NewSilaTransactionAPI(apiBackend, nonceLock)},
		{Namespace: "eth", Service: NewSilaTransactionAPI(apiBackend, nonceLock)},
		{Namespace: "txpool", Service: silaapi.NewTxPoolAPI(apiBackend)},
		{Namespace: "debug", Service: silaapi.NewDebugAPI(apiBackend)},
		{Namespace: "sila", Service: silaapi.NewSilaAccountAPI(apiBackend.AccountManager())},
		{Namespace: "eth", Service: silaapi.NewSilaAccountAPI(apiBackend.AccountManager())},
	}
}
