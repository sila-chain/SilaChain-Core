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

package eth

import (
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/txpool"
	"github.com/sila-org/sila/eth/protocols/eth"
)

// syncTransactions starts sending all currently pending transactions to the given peer.
func (h *handler) syncTransactions(p *eth.Peer) {
	var hashes []common.Hash
	pending, _ := h.txpool.Pending(txpool.PendingFilter{BlobTxs: false})
	for _, batch := range pending {
		for _, tx := range batch {
			hashes = append(hashes, tx.Hash)
		}
	}
	if len(hashes) == 0 {
		return
	}
	p.AsyncSendPooledTransactionHashes(hashes)
}
