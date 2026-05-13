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
	"errors"
	"testing"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/forkid"
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/p2p/enode"
)

// Tests that handshake failures are detected and reported correctly.
func TestHandshake69(t *testing.T) { testHandshake(t, ETH69) }

func testHandshake(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a test backend only to have some valid genesis chain
	backend := newTestBackend(3)
	defer backend.close()

	var (
		genesis = backend.chain.Genesis()
		head    = backend.chain.CurrentBlock()
		forkID  = forkid.NewID(backend.chain.Config(), backend.chain.Genesis(), backend.chain.CurrentHeader().Number.Uint64(), backend.chain.CurrentHeader().Time)
	)
	tests := []struct {
		code uint64
		data interface{}
		want error
	}{
		{
			code: TransactionsMsg, data: []interface{}{},
			want: errNoStatusMsg,
		},
		{
			code: StatusMsg, data: StatusPacket{10, 1, genesis.Hash(), forkID, 0, head.Number.Uint64(), head.Hash()},
			want: errProtocolVersionMismatch,
		},
		{
			code: StatusMsg, data: StatusPacket{uint32(protocol), 999, genesis.Hash(), forkID, 0, head.Number.Uint64(), head.Hash()},
			want: errNetworkIDMismatch,
		},
		{
			code: StatusMsg, data: StatusPacket{uint32(protocol), 1, common.Hash{3}, forkID, 0, head.Number.Uint64(), head.Hash()},
			want: errGenesisMismatch,
		},
		{
			code: StatusMsg, data: StatusPacket{uint32(protocol), 1, genesis.Hash(), forkid.ID{Hash: [4]byte{0x00, 0x01, 0x02, 0x03}}, 0, head.Number.Uint64(), head.Hash()},
			want: errForkIDRejected,
		},
		{
			code: StatusMsg, data: StatusPacket{uint32(protocol), 1, genesis.Hash(), forkID, head.Number.Uint64() + 1, head.Number.Uint64(), head.Hash()},
			want: errInvalidBlockRange,
		},
	}
	for i, test := range tests {
		// Create the two peers to shake with each other
		app, net := p2p.MsgPipe()
		defer app.Close()
		defer net.Close()

		peer := NewPeer(protocol, p2p.NewPeer(enode.ID{}, "peer", nil), net, nil, nil)
		defer peer.Close()

		// Send the junk test with one peer, check the handshake failure
		go p2p.Send(app, test.code, test.data)

		err := peer.Handshake(1, backend.chain, BlockRangeUpdatePacket{})
		if err == nil {
			t.Errorf("test %d: protocol returned nil error, want %q", i, test.want)
		} else if !errors.Is(err, test.want) {
			t.Errorf("test %d: wrong error: got %q, want %q", i, err, test.want)
		}
	}
}
