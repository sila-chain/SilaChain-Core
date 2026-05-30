// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import (
	"context"
	"github.com/sila-org/sila/common/math"
	"github.com/sila-org/sila/params"
	"math/big"

	ethereum "github.com/sila-org/sila"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
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

// TxPoolBackend is the minimal backend required by TxPoolAPI.
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
