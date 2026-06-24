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

package sila

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/sila-org/sila"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/consensus/misc/sip1559"
	"github.com/sila-org/sila/consensus/misc/sip4844"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/filtermaps"
	"github.com/sila-org/sila/core/history"
	"github.com/sila-org/sila/core/rawdb"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/txpool"
	"github.com/sila-org/sila/core/txpool/locals"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/event"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
	"github.com/sila-org/sila/sila/gasprice"
	"github.com/sila-org/sila/sila/tracers"
	"github.com/sila-org/sila/siladb"
)

// SilaAPIBackend implements silaapi.Backend and tracers.Backend for full nodes
type SilaAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	sila                *Sila
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *SilaAPIBackend) ChainConfig() *params.ChainConfig {
	return b.sila.blockchain.Config()
}

func (b *SilaAPIBackend) CurrentBlock() *types.Header {
	return b.sila.blockchain.CurrentBlock()
}

func (b *SilaAPIBackend) SetHead(number uint64) error {
	b.sila.handler.downloader.Cancel()
	return b.sila.blockchain.SetHead(number)
}

func (b *SilaAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, _ := b.sila.miner.Pending()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.sila.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		block := b.sila.blockchain.CurrentFinalBlock()
		if block == nil {
			return nil, errors.New("finalized block not found")
		}
		return block, nil
	}
	if number == rpc.SafeBlockNumber {
		block := b.sila.blockchain.CurrentSafeBlock()
		if block == nil {
			return nil, errors.New("safe block not found")
		}
		return block, nil
	}
	var bn uint64
	if number == rpc.EarliestBlockNumber {
		bn = b.HistoryPruningCutoff()
	} else {
		bn = uint64(number)
	}
	return b.sila.blockchain.GetHeaderByNumber(bn), nil
}

func (b *SilaAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.sila.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.sila.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *SilaAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.sila.blockchain.GetHeaderByHash(hash), nil
}

func (b *SilaAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, _ := b.sila.miner.Pending()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		header := b.sila.blockchain.CurrentBlock()
		return b.sila.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.FinalizedBlockNumber {
		header := b.sila.blockchain.CurrentFinalBlock()
		if header == nil {
			return nil, errors.New("finalized block not found")
		}
		return b.sila.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.SafeBlockNumber {
		header := b.sila.blockchain.CurrentSafeBlock()
		if header == nil {
			return nil, errors.New("safe block not found")
		}
		return b.sila.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	bn := uint64(number) // the resolved number
	if number == rpc.EarliestBlockNumber {
		bn = b.HistoryPruningCutoff()
	}
	block := b.sila.blockchain.GetBlockByNumber(bn)
	if block == nil && bn < b.HistoryPruningCutoff() {
		return nil, &history.PrunedHistoryError{}
	}
	return block, nil
}

func (b *SilaAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	number := b.sila.blockchain.GetBlockNumber(hash)
	if number == nil {
		return nil, nil
	}
	block := b.sila.blockchain.GetBlock(hash, *number)
	if block == nil && *number < b.HistoryPruningCutoff() {
		return nil, &history.PrunedHistoryError{}
	}
	return block, nil
}

// GetBody returns body of a block. It does not resolve special block numbers.
func (b *SilaAPIBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	if number < 0 || hash == (common.Hash{}) {
		return nil, errors.New("invalid arguments; expect hash and no special block numbers")
	}
	body := b.sila.blockchain.GetBody(hash)
	if body == nil {
		if uint64(number) < b.HistoryPruningCutoff() {
			return nil, &history.PrunedHistoryError{}
		}
		return nil, errors.New("block body not found")
	}
	return body, nil
}

func (b *SilaAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.sila.blockchain.GetHeaderByHash(hash)
		if header == nil {
			// Return 'null' and no error if block is not found.
			// This behavior is required by RPC spec.
			return nil, nil
		}
		if blockNrOrHash.RequireCanonical && b.sila.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.sila.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			if header.Number.Uint64() < b.HistoryPruningCutoff() {
				return nil, &history.PrunedHistoryError{}
			}
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *SilaAPIBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	return b.sila.miner.Pending()
}

func (b *SilaAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, state := b.sila.miner.Pending()
		if block == nil || state == nil {
			return nil, nil, errors.New("pending state is not available")
		}
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.sila.BlockChain().StateAt(header)
	if err != nil {
		stateDb, err = b.sila.BlockChain().HistoricState(header)
		if err != nil {
			return nil, nil, err
		}
	}
	return stateDb, header, nil
}

func (b *SilaAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.sila.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.sila.BlockChain().StateAt(header)
		if err != nil {
			stateDb, err = b.sila.BlockChain().HistoricState(header)
			if err != nil {
				return nil, nil, err
			}
		}
		return stateDb, header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *SilaAPIBackend) HistoryPruningCutoff() uint64 {
	bn, _ := b.sila.blockchain.HistoryPruningCutoff()
	return bn
}

func (b *SilaAPIBackend) HistoryRetention() silaapi.HistoryRetention {
	cfg := b.sila.config
	return silaapi.HistoryRetention{
		TxIndexHistory:   cfg.TransactionHistory,
		LogIndexHistory:  cfg.LogHistory,
		LogIndexDisabled: cfg.LogNoHistory,
		StateHistory:     cfg.StateHistory,
		TrienodeHistory:  cfg.TrienodeHistory,
		StateArchive:     cfg.NoPruning,
		StateScheme:      b.sila.blockchain.TrieDB().Scheme(),
	}
}

func (b *SilaAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.sila.blockchain.GetReceiptsByHash(hash), nil
}

func (b *SilaAPIBackend) GetCanonicalReceipt(tx *types.Transaction, blockHash common.Hash, blockNumber, blockIndex uint64) (*types.Receipt, error) {
	return b.sila.blockchain.GetCanonicalReceipt(tx, blockHash, blockNumber, blockIndex)
}

func (b *SilaAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.sila.chainDb, hash, number), nil
}

func (b *SilaAPIBackend) GetEVM(ctx context.Context, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM {
	if vmConfig == nil {
		vmConfig = b.sila.blockchain.GetVMConfig()
	}
	var context vm.BlockContext
	if blockCtx != nil {
		context = *blockCtx
	} else {
		context = core.NewEVMBlockContext(header, b.sila.BlockChain(), nil)
	}
	return vm.NewEVM(context, state, b.ChainConfig(), *vmConfig)
}

func (b *SilaAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.sila.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *SilaAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.sila.BlockChain().SubscribeChainEvent(ch)
}

func (b *SilaAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.sila.BlockChain().SubscribeChainHeadEvent(ch)
}

// SubscribeNewPayloadEvent registers a subscription for NewPayloadEvent.
func (b *SilaAPIBackend) SubscribeNewPayloadEvent(ch chan<- core.NewPayloadEvent) event.Subscription {
	return b.sila.BlockChain().SubscribeNewPayloadEvent(ch)
}

func (b *SilaAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.sila.BlockChain().SubscribeLogsEvent(ch)
}

func (b *SilaAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	err := b.sila.txPool.Add([]*types.Transaction{signedTx}, false)[0]

	// If the local transaction tracker is not configured, returns whatever
	// returned from the txpool.
	if b.sila.localTxTracker == nil {
		return err
	}
	// If the transaction fails with an error indicating it is invalid, or if there is
	// very little chance it will be accepted later (e.g., the gas price is below the
	// configured minimum, or the sender has insufficient funds to cover the cost),
	// propagate the error to the user.
	if err != nil && !locals.IsTemporaryReject(err) {
		return err
	}
	// No error will be returned to user if the transaction fails with a temporary
	// error and might be accepted later (e.g., the transaction pool is full).
	// Locally submitted transactions will be resubmitted later via the local tracker.
	b.sila.localTxTracker.Track(signedTx)
	return nil
}

func (b *SilaAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, _ := b.sila.txPool.Pending(txpool.PendingFilter{})
	var txs types.Transactions
	for _, batch := range pending {
		for _, lazy := range batch {
			if tx := lazy.Resolve(); tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	return txs, nil
}

func (b *SilaAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.sila.txPool.Get(hash)
}

// GetCanonicalTransaction retrieves the lookup along with the transaction itself
// associate with the given transaction hash.
//
// A null will be returned if the transaction is not found. The transaction is not
// existent from the node's perspective. This can be due to the transaction indexer
// not being finished. The caller must explicitly check the indexer progress.
//
// Notably, only the transaction in the canonical chain is visible.
func (b *SilaAPIBackend) GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64) {
	lookup, tx := b.sila.blockchain.GetCanonicalTransaction(txHash)
	if lookup == nil || tx == nil {
		return false, nil, common.Hash{}, 0, 0
	}
	return true, tx, lookup.BlockHash, lookup.BlockIndex, lookup.Index
}

// TxIndexDone returns true if the transaction indexer has finished indexing.
func (b *SilaAPIBackend) TxIndexDone() bool {
	return b.sila.blockchain.TxIndexDone()
}

func (b *SilaAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.sila.txPool.PoolNonce(addr), nil
}

func (b *SilaAPIBackend) Stats() (runnable int, blocked int) {
	return b.sila.txPool.Stats()
}

func (b *SilaAPIBackend) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return b.sila.txPool.Content()
}

func (b *SilaAPIBackend) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return b.sila.txPool.ContentFrom(addr)
}

func (b *SilaAPIBackend) TxPool() *txpool.TxPool {
	return b.sila.txPool
}

func (b *SilaAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.sila.txPool.SubscribeTransactions(ch, true)
}

func (b *SilaAPIBackend) SyncProgress(ctx context.Context) sila.SyncProgress {
	prog := b.sila.Downloader().Progress()
	if txProg, err := b.sila.blockchain.TxIndexProgress(); err == nil {
		prog.TxIndexFinishedBlocks = txProg.Indexed
		prog.TxIndexRemainingBlocks = txProg.Remaining
	}
	stateRemain, trienodeRemain, err := b.sila.blockchain.StateIndexProgress()
	if err == nil {
		prog.StateIndexRemaining = stateRemain
		prog.TrienodeIndexRemaining = trienodeRemain
	}
	return prog
}

func (b *SilaAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *SilaAPIBackend) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, baseFeePerBlobGas []*big.Int, blobGasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *SilaAPIBackend) BaseFee(ctx context.Context) *big.Int {
	header := b.CurrentHeader()
	next := new(big.Int).Add(header.Number, common.Big1)
	if b.ChainConfig().IsLondon(next) {
		return sip1559.CalcBaseFee(b.ChainConfig(), header)
	}
	return nil
}

func (b *SilaAPIBackend) BlobBaseFee(ctx context.Context) *big.Int {
	if excess := b.CurrentHeader().ExcessBlobGas; excess != nil {
		return sip4844.CalcBlobFee(b.ChainConfig(), b.CurrentHeader())
	}
	return nil
}

func (b *SilaAPIBackend) ChainDb() siladb.Database {
	return b.sila.ChainDb()
}

func (b *SilaAPIBackend) AccountManager() *accounts.Manager {
	return b.sila.AccountManager()
}

func (b *SilaAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *SilaAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *SilaAPIBackend) RPCGasCap() uint64 {
	return b.sila.config.RPCGasCap
}

func (b *SilaAPIBackend) RPCEVMTimeout() time.Duration {
	return b.sila.config.RPCEVMTimeout
}

func (b *SilaAPIBackend) RPCTxFeeCap() float64 {
	return b.sila.config.RPCTxFeeCap
}

func (b *SilaAPIBackend) CurrentView() *filtermaps.ChainView {
	head := b.sila.blockchain.CurrentBlock()
	if head == nil {
		return nil
	}
	return filtermaps.NewChainView(b.sila.blockchain, head.Number.Uint64(), head.Hash())
}

func (b *SilaAPIBackend) NewMatcherBackend() filtermaps.MatcherBackend {
	return b.sila.filterMaps.NewMatcherBackend()
}

func (b *SilaAPIBackend) SilaEngine() consensus.SilaEngine {
	return b.sila.silaEngine
}

func (b *SilaAPIBackend) CurrentHeader() *types.Header {
	return b.sila.blockchain.CurrentHeader()
}

func (b *SilaAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.sila.stateAtBlock(ctx, block, base, readOnly, preferDisk)
}

func (b *SilaAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int) (*types.Transaction, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.sila.stateAtTransaction(ctx, block, txIndex)
}

func (b *SilaAPIBackend) RPCTxSyncDefaultTimeout() time.Duration {
	return b.sila.config.TxSyncDefaultTimeout
}

func (b *SilaAPIBackend) RPCTxSyncMaxTimeout() time.Duration {
	return b.sila.config.TxSyncMaxTimeout
}
