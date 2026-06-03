// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package registry

import (
	"github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/rpc"
)

func GetAPIs(apiBackend ethapi.Backend) []rpc.API {
	nonceLock := new(addrlock.AddrLocker)
	return []rpc.API{
		{Namespace: "sila", Service: silaapi.NewSilaAPI(apiBackend)},
		{Namespace: "eth", Service: silaapi.NewSilaAPI(apiBackend)},
		{Namespace: "sila", Service: ethapi.NewBlockChainAPI(apiBackend)},
		{Namespace: "eth", Service: ethapi.NewBlockChainAPI(apiBackend)},
		{Namespace: "sila", Service: ethapi.NewTransactionAPI(apiBackend, nonceLock)},
		{Namespace: "eth", Service: ethapi.NewTransactionAPI(apiBackend, nonceLock)},
		{Namespace: "txpool", Service: ethapi.NewTxPoolAPI(apiBackend)},
		{Namespace: "debug", Service: ethapi.NewDebugAPI(apiBackend)},
		{Namespace: "sila", Service: silaapi.NewSilaAccountAPI(apiBackend.AccountManager())},
		{Namespace: "eth", Service: silaapi.NewSilaAccountAPI(apiBackend.AccountManager())},
	}
}
