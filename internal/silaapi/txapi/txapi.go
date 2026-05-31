package txapi

import (
	"context"

	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/params"
)

type Backend interface {
	GetCanonicalTransaction(common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64)
	GetPoolTransaction(common.Hash) *types.Transaction
	TxIndexDone() bool
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	CurrentHeader() *types.Header
	ChainConfig() *params.ChainConfig
	GetCanonicalReceipt(*types.Transaction, common.Hash, uint64, uint64) (*types.Receipt, error)
	GetPoolTransactions() (types.Transactions, error)
	AccountManager() *accounts.Manager
}

func GetTransactionByHash(ctx context.Context, b Backend, hash common.Hash) (*rpctx.RPCTransaction, error) {
	found, tx, blockHash, blockNumber, index := b.GetCanonicalTransaction(hash)
	if !found {
		if tx := b.GetPoolTransaction(hash); tx != nil {
			return rpctx.NewRPCPendingTransaction(tx, b.CurrentHeader(), b.ChainConfig()), nil
		}
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	header, err := b.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	return rpctx.NewRPCTransaction(tx, blockHash, blockNumber, header.Time, index, header.BaseFee, b.ChainConfig()), nil
}

func GetRawTransactionByHash(b Backend, hash common.Hash) (hexutil.Bytes, error) {
	found, tx, _, _, _ := b.GetCanonicalTransaction(hash)
	if !found {
		if tx = b.GetPoolTransaction(hash); tx != nil {
			return tx.MarshalBinary()
		}
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	return tx.MarshalBinary()
}

func GetTransactionReceipt(b Backend, signer types.Signer, hash common.Hash) (map[string]interface{}, error) {
	found, tx, blockHash, blockNumber, index := b.GetCanonicalTransaction(hash)
	if !found {
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	receipt, err := b.GetCanonicalReceipt(tx, blockHash, blockNumber, index)
	if err != nil {
		return nil, err
	}
	return rpctx.MarshalReceipt(receipt, blockHash, blockNumber, signer, tx, int(index)), nil
}

func PendingTransactions(b Backend, signer types.Signer) ([]*rpctx.RPCTransaction, error) {
	pending, err := b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	accounts := make(map[common.Address]struct{})
	for _, wallet := range b.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			accounts[account.Address] = struct{}{}
		}
	}
	curHeader := b.CurrentHeader()
	transactions := make([]*rpctx.RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		from, _ := types.Sender(signer, tx)
		if _, exists := accounts[from]; exists {
			transactions = append(transactions, rpctx.NewRPCPendingTransaction(tx, curHeader, b.ChainConfig()))
		}
	}
	return transactions, nil
}
