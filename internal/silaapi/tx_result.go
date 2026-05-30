// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import (
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
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
