// Copyright 2014 The sila Authors
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

package core

import (
	"time"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/types"
)

// NewTxsEvent is posted when a batch of transactions enter the transaction pool.
type NewTxsEvent struct{ Txs []*types.Transaction }

// RemovedLogsEvent is posted when a reorg happens
type RemovedLogsEvent struct{ Logs []*types.Log }

type ChainEvent struct {
	Header       *types.Header
	Receipts     []*types.Receipt
	Transactions []*types.Transaction
}

type ChainHeadEvent struct {
	Header *types.Header
}

// NewPayloadEvent is posted when silaEngine_newPayloadVX processes a block.
type NewPayloadEvent struct {
	Hash           common.Hash
	Number         uint64
	ProcessingTime time.Duration
}
