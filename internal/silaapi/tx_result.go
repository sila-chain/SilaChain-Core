// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/sila-org/sila/common/math"
	"github.com/sila-org/sila/params"
	"math/big"
	"time"

	"github.com/davecgh/go-spew/spew"
	ethereum "github.com/sila-org/sila"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/ethdb"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/rlp"
	"github.com/sila-org/sila/rpc"
)

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}

// SilaAccountAPI provides an API to access accounts managed by this node.
type SilaAccountAPI struct {
	am *accounts.Manager
}

// NewSilaAccountAPI creates a new SilaChain account API.
func NewSilaAccountAPI(am *accounts.Manager) *SilaAccountAPI {
	return &SilaAccountAPI{am: am}
}

// Accounts returns the collection of accounts this node manages.
func (api *SilaAccountAPI) Accounts() []common.Address {
	return api.am.Accounts()
}

// FeeHistoryResult is the Sila fee market history response.
type FeeHistoryResult struct {
	OldestBlock      *hexutil.Big     `json:"oldestBlock"`
	Reward           [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee          []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio     []float64        `json:"gasUsedRatio"`
	BlobBaseFee      []*hexutil.Big   `json:"baseFeePerBlobGas,omitempty"`
	BlobGasUsedRatio []float64        `json:"blobGasUsedRatio,omitempty"`
}

// SilaAPIBackend is the minimal backend required by SilaAPI helpers.
type SilaAPIBackend interface {
	SyncProgress(ctx context.Context) ethereum.SyncProgress
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, []*big.Int, []float64, error)
	BlobBaseFee(ctx context.Context) *big.Int
	CurrentHeader() *types.Header
}

type SilaAPI struct {
	b SilaAPIBackend
}

// NewSilaAPI creates a new SilaChain protocol API.
func NewSilaAPI(b SilaAPIBackend) *SilaAPI {
	return &SilaAPI{b}
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func (api *SilaAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	return GasPrice(ctx, api.b)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func (api *SilaAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	return MaxPriorityFeePerGas(ctx, api.b)
}

// FeeHistory returns the fee market history.
func (api *SilaAPI) FeeHistory(ctx context.Context, blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*FeeHistoryResult, error) {
	return FeeHistory(ctx, api.b, blockCount, lastBlock, rewardPercentiles)
}

// BlobBaseFee returns the base fee for blob gas at the current head.
func (api *SilaAPI) BlobBaseFee(ctx context.Context) *hexutil.Big {
	return BlobBaseFee(ctx, api.b)
}

// Syncing returns false in case the node is currently not syncing with the network.
func (api *SilaAPI) Syncing(ctx context.Context) (interface{}, error) {
	return Syncing(ctx, api.b)
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func GasPrice(ctx context.Context, b SilaAPIBackend) (*hexutil.Big, error) {
	tipcap, err := b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	if head := b.CurrentHeader(); head.BaseFee != nil {
		tipcap.Add(tipcap, head.BaseFee)
	}
	return (*hexutil.Big)(tipcap), err
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func MaxPriorityFeePerGas(ctx context.Context, b SilaAPIBackend) (*hexutil.Big, error) {
	tipcap, err := b.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tipcap), err
}

// FeeHistory returns the fee market history.
func FeeHistory(ctx context.Context, b SilaAPIBackend, blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*FeeHistoryResult, error) {
	oldest, reward, baseFee, gasUsed, blobBaseFee, blobGasUsed, err := b.FeeHistory(ctx, uint64(blockCount), lastBlock, rewardPercentiles)
	if err != nil {
		return nil, err
	}
	results := &FeeHistoryResult{
		OldestBlock:  (*hexutil.Big)(oldest),
		GasUsedRatio: gasUsed,
	}
	if reward != nil {
		results.Reward = make([][]*hexutil.Big, len(reward))
		for i, w := range reward {
			results.Reward[i] = make([]*hexutil.Big, len(w))
			for j, v := range w {
				results.Reward[i][j] = (*hexutil.Big)(v)
			}
		}
	}
	if baseFee != nil {
		results.BaseFee = make([]*hexutil.Big, len(baseFee))
		for i, v := range baseFee {
			results.BaseFee[i] = (*hexutil.Big)(v)
		}
	}
	if blobBaseFee != nil {
		results.BlobBaseFee = make([]*hexutil.Big, len(blobBaseFee))
		for i, v := range blobBaseFee {
			results.BlobBaseFee[i] = (*hexutil.Big)(v)
		}
	}
	if blobGasUsed != nil {
		results.BlobGasUsedRatio = blobGasUsed
	}
	return results, nil
}

// BlobBaseFee returns the base fee for blob gas at the current head.
func BlobBaseFee(ctx context.Context, b SilaAPIBackend) *hexutil.Big {
	return (*hexutil.Big)(b.BlobBaseFee(ctx))
}

// Syncing returns false in case the node is currently not syncing with the network.
func Syncing(ctx context.Context, b SilaAPIBackend) (interface{}, error) {
	progress := b.SyncProgress(ctx)
	if progress.Done() {
		return false, nil
	}
	return map[string]interface{}{
		"startingBlock":          hexutil.Uint64(progress.StartingBlock),
		"currentBlock":           hexutil.Uint64(progress.CurrentBlock),
		"highestBlock":           hexutil.Uint64(progress.HighestBlock),
		"syncedAccounts":         hexutil.Uint64(progress.SyncedAccounts),
		"syncedAccountBytes":     hexutil.Uint64(progress.SyncedAccountBytes),
		"syncedBytecodes":        hexutil.Uint64(progress.SyncedBytecodes),
		"syncedBytecodeBytes":    hexutil.Uint64(progress.SyncedBytecodeBytes),
		"syncedStorage":          hexutil.Uint64(progress.SyncedStorage),
		"syncedStorageBytes":     hexutil.Uint64(progress.SyncedStorageBytes),
		"healedTrienodes":        hexutil.Uint64(progress.HealedTrienodes),
		"healedTrienodeBytes":    hexutil.Uint64(progress.HealedTrienodeBytes),
		"healedBytecodes":        hexutil.Uint64(progress.HealedBytecodes),
		"healedBytecodeBytes":    hexutil.Uint64(progress.HealedBytecodeBytes),
		"healingTrienodes":       hexutil.Uint64(progress.HealingTrienodes),
		"healingBytecode":        hexutil.Uint64(progress.HealingBytecode),
		"txIndexFinishedBlocks":  hexutil.Uint64(progress.TxIndexFinishedBlocks),
		"txIndexRemainingBlocks": hexutil.Uint64(progress.TxIndexRemainingBlocks),
		"stateIndexRemaining":    hexutil.Uint64(progress.StateIndexRemaining),
		"trienodeIndexRemaining": hexutil.Uint64(progress.TrienodeIndexRemaining),
	}, nil
}

type RPCTransaction = rpctx.RPCTransaction

type TxPoolAPI struct {
	b TxPoolBackend
}

// NewTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolAPI(b TxPoolBackend) *TxPoolAPI {
	return &TxPoolAPI{b}
}

// flattenTxs builds the RPC transaction map keyed by nonce for a set of pool txs.
func flattenTxs(txs types.Transactions, header *types.Header, cfg *params.ChainConfig) map[string]*RPCTransaction {
	dump := make(map[string]*RPCTransaction, len(txs))
	for _, tx := range txs {
		dump[fmt.Sprintf("%d", tx.Nonce())] = rpctx.NewRPCPendingTransaction(tx, header, cfg)
	}
	return dump
}

// Content returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	pending, queue := api.b.TxPoolContent()
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction, len(pending)),
		"queued":  make(map[string]map[string]*RPCTransaction, len(queue)),
	}
	curHeader := api.b.CurrentHeader()
	for account, txs := range pending {
		content["pending"][account.Hex()] = flattenTxs(txs, curHeader, api.b.ChainConfig())
	}
	for account, txs := range queue {
		content["queued"][account.Hex()] = flattenTxs(txs, curHeader, api.b.ChainConfig())
	}
	return content
}

// ContentFrom returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) ContentFrom(addr common.Address) map[string]map[string]*RPCTransaction {
	content := make(map[string]map[string]*RPCTransaction, 2)
	pending, queue := api.b.TxPoolContentFrom(addr)
	curHeader := api.b.CurrentHeader()

	content["pending"] = flattenTxs(pending, curHeader, api.b.ChainConfig())
	content["queued"] = flattenTxs(queue, curHeader, api.b.ChainConfig())

	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (api *TxPoolAPI) Status() map[string]hexutil.Uint {
	return TxPoolStatus(api.b)
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (api *TxPoolAPI) Inspect() map[string]map[string]map[string]string {
	return TxPoolInspect(api.b)
}

// TxPoolBackend is the minimal backend required by TxPoolAPI.
type DebugAPI struct {
	b DebugBackend
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewDebugAPI(b DebugBackend) *DebugAPI {
	return &DebugAPI{b: b}
}

// GetRawHeader retrieves the RLP encoding for a single header.
func (api *DebugAPI) GetRawHeader(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header == nil {
		return nil, fmt.Errorf("header #%d not found", hash)
	}
	return rlp.EncodeToBytes(header)
}

// GetRawBlock retrieves the RLP encoded for a single block.
func (api *DebugAPI) GetRawBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	block, _ := api.b.BlockByHash(ctx, hash)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", hash)
	}
	return rlp.EncodeToBytes(block)
}

// GetRawReceipts retrieves the binary-encoded receipts of a single block.
func (api *DebugAPI) GetRawReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	receipts, err := api.b.GetReceipts(ctx, hash)
	if err != nil {
		return nil, err
	}
	result := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		b, err := receipt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result[i] = b
	}
	return result, nil
}

// GetRawTransaction returns the bytes of the transaction for the given hash.
func (api *DebugAPI) GetRawTransaction(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	found, tx, _, _, _ := api.b.GetCanonicalTransaction(hash)
	if !found {
		if tx = api.b.GetPoolTransaction(hash); tx != nil {
			return tx.MarshalBinary()
		}
		if !api.b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	return tx.MarshalBinary()
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *DebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *DebugAPI) ChaindbProperty() (string, error) {
	return api.b.ChainDb().Stat()
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *DebugAPI) ChaindbCompact() error {
	cstart := time.Now()
	for b := 0; b <= 255; b++ {
		var (
			start = []byte{byte(b)}
			end   = []byte{byte(b + 1)}
		)
		if b == 255 {
			end = nil
		}
		log.Info("Compacting database", "range", fmt.Sprintf("%#X-%#X", start, end), "elapsed", common.PrettyDuration(time.Since(cstart)))
		if err := api.b.ChainDb().Compact(start, end); err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *DebugAPI) SetHead(number hexutil.Uint64) error {
	header := api.b.CurrentHeader()
	if header == nil {
		return errors.New("current header is not available")
	}
	if header.Number.Uint64() <= uint64(number) {
		return errors.New("not allowed to rewind to a future block")
	}
	api.b.SetHead(uint64(number))
	return nil
}

type DebugBackend interface {
	SetHead(number uint64)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	CurrentHeader() *types.Header
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error)
	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
	GetCanonicalTransaction(txHash common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64)
	GetPoolTransaction(txHash common.Hash) *types.Transaction
	TxIndexDone() bool
	ChainDb() ethdb.Database
}
type TxPoolBackend interface {
	CurrentHeader() *types.Header
	ChainConfig() *params.ChainConfig
	Stats() (pending int, queued int)
	TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction)
	TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction)
}

// TxPoolStatus returns the number of pending and queued transactions in the pool.
func TxPoolStatus(b TxPoolBackend) map[string]hexutil.Uint {
	pending, queue := b.Stats()
	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(pending),
		"queued":  hexutil.Uint(queue),
	}
}

// TxPoolInspect retrieves the content of the transaction pool and flattens it into an easily inspectable list.
func TxPoolInspect(b TxPoolBackend) map[string]map[string]map[string]string {
	pending, queue := b.TxPoolContent()
	content := map[string]map[string]map[string]string{
		"pending": make(map[string]map[string]string, len(pending)),
		"queued":  make(map[string]map[string]string, len(queue)),
	}
	format := func(tx *types.Transaction) string {
		if to := tx.To(); to != nil {
			return fmt.Sprintf("%s: %v wei + %v gas × %v wei", tx.To().Hex(), tx.Value(), tx.Gas(), tx.GasPrice())
		}
		return fmt.Sprintf("contract creation: %v wei + %v gas × %v wei", tx.Value(), tx.Gas(), tx.GasPrice())
	}
	for account, txs := range pending {
		dump := make(map[string]string, len(txs))
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["pending"][account.Hex()] = dump
	}
	for account, txs := range queue {
		dump := make(map[string]string, len(txs))
		for _, tx := range txs {
			dump[fmt.Sprintf("%d", tx.Nonce())] = format(tx)
		}
		content["queued"][account.Hex()] = dump
	}
	return content
}
