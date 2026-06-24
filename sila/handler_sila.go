// Copyright 2020 The sila Authors
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
	"errors"
	"fmt"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/sila/protocols/sila"
	"github.com/sila-org/sila/p2p/enode"
)

// silaHandler implements the sila.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type silaHandler handler

func (h *silaHandler) Chain() *core.BlockChain { return h.chain }
func (h *silaHandler) TxPool() sila.TxPool      { return h.txpool }

// RunPeer is invoked when a peer joins on the `sila` protocol.
func (h *silaHandler) RunPeer(peer *sila.Peer, hand sila.Handler) error {
	return (*handler)(h).runEthPeer(peer, hand)
}

// PeerInfo retrieves all known `sila` information about a peer.
func (h *silaHandler) PeerInfo(id enode.ID) interface{} {
	if p := h.peers.peer(id.String()); p != nil {
		return p.info()
	}
	return nil
}

// AcceptTxs retrieves whether transaction processing is enabled on the node
// or if inbound transactions should simply be dropped.
func (h *silaHandler) AcceptTxs() bool {
	return h.synced.Load()
}

// Handle is invoked from a peer's message handler when it receives a new remote
// message that the handler couldn't consume and serve itself.
func (h *silaHandler) Handle(peer *sila.Peer, packet sila.Packet) error {
	// Consume any broadcasts and announces, forwarding the rest to the downloader
	switch packet := packet.(type) {
	case *sila.NewPooledTransactionHashesPacket:
		return h.txFetcher.Notify(peer.ID(), packet.Types, packet.Sizes, packet.Hashes)

	case *sila.TransactionsPacket:
		txs, err := packet.Items()
		if err != nil {
			return fmt.Errorf("Transactions: %v", err)
		}
		if err := handleTransactions(peer, txs, true); err != nil {
			return fmt.Errorf("Transactions: %v", err)
		}
		return h.txFetcher.Enqueue(peer.ID(), txs, false)

	case *sila.PooledTransactionsPacket:
		txs, err := packet.List.Items()
		if err != nil {
			return fmt.Errorf("PooledTransactions: %v", err)
		}
		if err := handleTransactions(peer, txs, false); err != nil {
			return fmt.Errorf("PooledTransactions: %v", err)
		}
		return h.txFetcher.Enqueue(peer.ID(), txs, true)

	default:
		return fmt.Errorf("unexpected sila packet type: %T", packet)
	}
}

// handleTransactions marks all given transactions as known to the peer
// and performs basic validations.
func handleTransactions(peer *sila.Peer, list []*types.Transaction, directBroadcast bool) error {
	seen := make(map[common.Hash]struct{})
	for _, tx := range list {
		if tx.Type() == types.BlobTxType {
			if directBroadcast {
				return errors.New("disallowed broadcast blob transaction")
			} else {
				// If we receive any blob transactions missing sidecars, or with
				// sidecars that don't correspond to the versioned hashes reported
				// in the header, disconnect from the sending peer.
				if tx.BlobTxSidecar() == nil {
					return errors.New("received sidecar-less blob transaction")
				}
				if err := tx.BlobTxSidecar().ValidateBlobCommitmentHashes(tx.BlobHashes()); err != nil {
					return err
				}
			}
		}

		// Check for duplicates.
		hash := tx.Hash()
		if _, exists := seen[hash]; exists {
			return fmt.Errorf("multiple copies of the same hash %v", hash)
		}
		seen[hash] = struct{}{}

		// Mark as known.
		peer.MarkTransaction(hash)
	}
	return nil
}
