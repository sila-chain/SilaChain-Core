// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SilaChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the SilaChain library. If not, see <http://www.gnu.org/licenses/>.

// Package ethapi implements the general SilaChain API functions.
package ethapi

import (
	"context"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"math/big"
	"time"

	"github.com/sila-org/sila"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/filtermaps"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/ethdb"
	"github.com/sila-org/sila/event"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// General SilaChain API
	SyncProgress(ctx context.Context) ethereum.SyncProgress

	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, []*big.Int, []float64, error)
	BlobBaseFee(ctx context.Context) *big.Int
	ChainDb() ethdb.Database
	AccountManager() *accounts.Manager
	ExtRPCEnabled() bool
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // global tx fee cap for all transaction related APIs
	UnprotectedAllowed() bool     // allows only for EIP155 transactions.
	RPCTxSyncDefaultTimeout() time.Duration
	RPCTxSyncMaxTimeout() time.Duration

	// Blockchain API
	SetHead(number uint64)
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error)
	CurrentHeader() *types.Header
	CurrentBlock() *types.Header
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error)
	StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error)
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	Pending() (*types.Block, types.Receipts, *state.StateDB)
	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
	GetCanonicalReceipt(tx *types.Transaction, blockHash common.Hash, blockNumber, blockIndex uint64) (*types.Receipt, error)
	GetEVM(ctx context.Context, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription

	// Transaction pool API
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64)
	TxIndexDone() bool
	GetPoolTransactions() (types.Transactions, error)
	GetPoolTransaction(txHash common.Hash) *types.Transaction
	GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error)
	Stats() (pending int, queued int)
	TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction)
	TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction)
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription

	ChainConfig() *params.ChainConfig
	Engine() consensus.Engine
	HistoryPruningCutoff() uint64

	// This is copied from filters.Backend
	// eth/filters needs to be initialized from this backend type, so methods needed by
	// it must also be included here.
	GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error)
	GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error)
	SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription
	SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription

	CurrentView() *filtermaps.ChainView
	NewMatcherBackend() filtermaps.MatcherBackend
}

func GetAPIs(apiBackend Backend) []rpc.API {
	nonceLock := new(addrlock.AddrLocker)
	return []rpc.API{
		{
			Namespace: "eth",
			Service:   NewSilaAPI(apiBackend),
		},
		{
			Namespace: "sila",
			Service:   NewSilaAPI(apiBackend),
		}, {
			Namespace: "eth",
			Service:   NewBlockChainAPI(apiBackend),
		},
		{
			Namespace: "sila",
			Service:   NewBlockChainAPI(apiBackend),
		}, {
			Namespace: "eth",
			Service:   NewTransactionAPI(apiBackend, nonceLock),
		},
		{
			Namespace: "sila",
			Service:   NewTransactionAPI(apiBackend, nonceLock),
		}, {
			Namespace: "txpool",
			Service:   NewTxPoolAPI(apiBackend),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(apiBackend),
		}, {
			Namespace: "eth",
			Service:   NewSilaAccountAPI(apiBackend.AccountManager()),
		},
		{
			Namespace: "sila",
			Service:   NewSilaAccountAPI(apiBackend.AccountManager()),
		},
	}
}
