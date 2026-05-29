// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package rpctx

import (
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/types"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/params"
)

type RPCTransaction = ethapi.RPCTransaction

func NewRPCPendingTransaction(tx *types.Transaction, current *types.Header, config *params.ChainConfig) *RPCTransaction {
	return ethapi.NewRPCPendingTransaction(tx, current, config)
}

func MarshalReceipt(receipt *types.Receipt, blockHash common.Hash, blockNumber uint64, signer types.Signer, tx *types.Transaction, txIndex int) map[string]interface{} {
	return ethapi.MarshalReceipt(receipt, blockHash, blockNumber, signer, tx, txIndex)
}
