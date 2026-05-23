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

package catalyst

import (
	"errors"

	"github.com/sila-org/sila/beacon/engine"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/miner"
	"github.com/sila-org/sila/rpc"
)

// testingAPI implements the testing_ namespace.
// It's an engine-API adjacent namespace for testing purposes.
type testingAPI struct {
	eth *eth.SilaChain
}

func newTestingAPI(backend *eth.SilaChain) rpc.API {
	return rpc.API{
		Namespace:     "testing",
		Service:       &testingAPI{backend},
		Version:       "1.0",
		Authenticated: false,
	}
}

func (api *testingAPI) BuildBlockV1(parentHash common.Hash, payloadAttributes engine.PayloadAttributes, transactions *[]hexutil.Bytes, extraData *hexutil.Bytes) (*engine.ExecutionPayloadEnvelope, error) {
	if api.eth.BlockChain().CurrentBlock().Hash() != parentHash {
		return nil, errors.New("parentHash is not current head")
	}
	// If transactions is empty but not nil, build an empty block
	// If the transactions is nil, build a block with the current transactions from the txpool
	// If the transactions is not nil and not empty, build a block with the transactions
	buildEmpty := transactions != nil && len(*transactions) == 0
	var txs []*types.Transaction
	if transactions != nil {
		dec := make([][]byte, 0, len(*transactions))
		for _, tx := range *transactions {
			dec = append(dec, tx)
		}
		var err error
		txs, err = engine.DecodeTransactions(dec)
		if err != nil {
			return nil, err
		}
	}
	extra := make([]byte, 0)
	if extraData != nil {
		extra = *extraData
	}
	args := &miner.BuildPayloadArgs{
		Parent:       parentHash,
		Timestamp:    payloadAttributes.Timestamp,
		FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
		Random:       payloadAttributes.Random,
		Withdrawals:  payloadAttributes.Withdrawals,
		BeaconRoot:   payloadAttributes.BeaconRoot,
		SlotNum:      payloadAttributes.SlotNumber,
	}
	return api.eth.Miner().BuildTestingPayload(args, txs, buildEmpty, extra)
}
