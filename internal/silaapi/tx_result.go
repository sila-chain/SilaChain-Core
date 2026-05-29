// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import (
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
)

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes      `json:"raw"`
	Tx  *types.Transaction `json:"tx"`
}
