// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import (
	"context"
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
