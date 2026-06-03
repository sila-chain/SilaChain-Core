// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package registry

import (
	"github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/backend"
	"github.com/sila-org/sila/rpc"
)

func NewSilaBlockChainAPI(apiBackend backend.Backend) interface{} {
	return ethapi.NewSilaBlockChainAPI(apiBackend)
}

func NewSilaTransactionAPI(apiBackend backend.Backend, nonceLock *addrlock.AddrLocker) interface{} {
	return ethapi.NewSilaTransactionAPI(apiBackend, nonceLock)
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
