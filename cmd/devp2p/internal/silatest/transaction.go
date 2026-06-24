// Copyright 2020 The sila Authors
// This file is part of sila.
//
// sila is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sila is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with sila. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/internal/utesting"
	"github.com/sila-org/sila/rlp"
	"github.com/sila-org/sila/sila/protocols/sila"
)

// sendTxs sends the given transactions to the node and
// expects the node to accept and propagate them.
func (s *Suite) sendTxs(t *utesting.T, txs []*types.Transaction) error {
	// Open sending conn.
	sendConn, err := s.dial()
	if err != nil {
		return err
	}
	defer sendConn.Close()
	if err = sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}

	// Open receiving conn.
	recvConn, err := s.dial()
	if err != nil {
		return err
	}
	defer recvConn.Close()
	if err = recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}

	encTxs, _ := rlp.EncodeToRawList(txs)
	if err = sendConn.Write(silaProto, sila.TransactionsMsg, sila.TransactionsPacket{RawList: encTxs}); err != nil {
		return fmt.Errorf("failed to write message to connection: %v", err)
	}

	var (
		got = make(map[common.Hash]bool)
		end = time.Now().Add(timeout)
	)

	// Wait for the transaction announcements, make sure all txs ar propagated.
	for time.Now().Before(end) {
		msg, err := recvConn.ReadEth()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}
		switch msg := msg.(type) {
		case *sila.TransactionsPacket:
			txs, _ := msg.Items()
			for _, tx := range txs {
				got[tx.Hash()] = true
			}
		case *sila.NewPooledTransactionHashesPacket:
			for _, hash := range msg.Hashes {
				got[hash] = true
			}
		case *sila.GetBlockHeadersPacket:
			headers, err := s.chain.GetHeaders(msg)
			if err != nil {
				t.Logf("invalid GetBlockHeaders request: %v", err)
			}
			encHeaders, _ := rlp.EncodeToRawList(headers)
			recvConn.Write(silaProto, sila.BlockHeadersMsg, &sila.BlockHeadersPacket{
				RequestId: msg.RequestId,
				List:      encHeaders,
			})
		default:
			return fmt.Errorf("unexpected sila wire msg: %s", pretty.Sdump(msg))
		}

		// Check if all txs received.
		allReceived := func() bool {
			for _, tx := range txs {
				if !got[tx.Hash()] {
					return false
				}
			}
			return true
		}
		if allReceived() {
			return nil
		}
	}

	return errors.New("timed out waiting for txs")
}

func (s *Suite) sendInvalidTxs(t *utesting.T, txs []*types.Transaction) error {
	// Open sending conn.
	sendConn, err := s.dial()
	if err != nil {
		return err
	}
	defer sendConn.Close()
	if err = sendConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	sendConn.SetDeadline(time.Now().Add(timeout))

	// Open receiving conn.
	recvConn, err := s.dial()
	if err != nil {
		return err
	}
	defer recvConn.Close()
	if err = recvConn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}
	recvConn.SetDeadline(time.Now().Add(timeout))

	if err = sendConn.Write(silaProto, sila.TransactionsMsg, txs); err != nil {
		return fmt.Errorf("failed to write message to connection: %w", err)
	}

	// Make map of invalid txs.
	invalids := make(map[common.Hash]struct{})
	for _, tx := range txs {
		invalids[tx.Hash()] = struct{}{}
	}

	// Get responses.
	recvConn.SetReadDeadline(time.Now().Add(timeout))
	for {
		msg, err := recvConn.ReadEth()
		if errors.Is(err, os.ErrDeadlineExceeded) {
			// Successful if no invalid txs are propagated before timeout.
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}

		switch msg := msg.(type) {
		case *sila.TransactionsPacket:
			received, err := msg.Items()
			if err != nil {
				return fmt.Errorf("failed to decode received transactions: %w", err)
			}
			for _, tx := range received {
				if _, ok := invalids[tx.Hash()]; ok {
					return fmt.Errorf("received bad tx: %s", tx.Hash())
				}
			}
		case *sila.NewPooledTransactionHashesPacket:
			for _, hash := range msg.Hashes {
				if _, ok := invalids[hash]; ok {
					return fmt.Errorf("received bad tx: %s", hash)
				}
			}
		case *sila.GetBlockHeadersPacket:
			headers, err := s.chain.GetHeaders(msg)
			if err != nil {
				t.Logf("invalid GetBlockHeaders request: %v", err)
			}
			encHeaders, _ := rlp.EncodeToRawList(headers)
			recvConn.Write(silaProto, sila.BlockHeadersMsg, &sila.BlockHeadersPacket{
				RequestId: msg.RequestId,
				List:      encHeaders,
			})
		default:
			return fmt.Errorf("unexpected sila message: %v", pretty.Sdump(msg))
		}
	}
}
