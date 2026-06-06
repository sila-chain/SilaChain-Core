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

package ethapi

import (
	"context"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	"github.com/sila-org/sila/internal/silaapi/netapi"
	"github.com/sila-org/sila/internal/silaapi/override"
	"github.com/sila-org/sila/internal/silaapi/proofapi"
	"github.com/sila-org/sila/internal/silaapi/rpctx"

	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/rpc"
)

// estimateGasErrorRatio is the amount of overestimation eth_estimateGas is
// allowed to produce in order to speed up calculations.
const estimateGasErrorRatio = 0.015

// maxGetStorageSlots is the maximum total number of storage slots that can
// be requested in a single eth_getStorageValues call.
const maxGetStorageSlots = 1024

type SilaAPIBackend = silaapi.SilaAPIBackend
type SilaAPI = silaapi.SilaAPI

// NewSilaAPI creates a new SilaChain protocol API.
func NewSilaAPI(b SilaAPIBackend) *SilaAPI {
	return silaapi.NewSilaAPI(b)
}

type TxPoolAPI = silaapi.TxPoolAPI

// NewTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolAPI(b Backend) *TxPoolAPI {
	return silaapi.NewTxPoolAPI(b)
}

// BlockChainAPI provides an API to access SilaChain blockchain data.
type BlockChainAPI struct {
	b Backend
	*blockapi.BlockChainAPI
	*callapi.API
}

// NewSilaBlockChainAPI creates a new SilaChain blockchain API.
func NewSilaBlockChainAPI(b Backend) *BlockChainAPI {
	return &BlockChainAPI{
		b:             b,
		BlockChainAPI: blockapi.NewBlockChainAPI(b),
		API:           callapi.NewAPI(b),
	}
}

// NewBlockChainAPI creates a new SilaChain blockchain API.
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return NewSilaBlockChainAPI(b)
}

// AccountResult structs for GetProof
type AccountResult = proofapi.AccountResult
type StorageResult = proofapi.StorageResult
type proofList = proofapi.ProofList

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (api *BlockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	return proofapi.GetProof(ctx, api.b, address, storageKeys, blockNrOrHash)
}

// SimulateV1 executes series of transactions on top of a base state.
// The transactions are packed into blocks. For each block, block header
// fields can be overridden. The state can also be overridden prior to
// execution of each block.
//
// Note, this function doesn't make any changes in the state/blockchain and is
// useful to execute and retrieve values.
// EstimateGas returns the lowest possible gas limit that allows the transaction to run
// successfully at block `blockNrOrHash`, or the latest block if `blockNrOrHash` is unspecified. It
// returns error if the transaction would revert or if there are unexpected failures. The returned
// value is capped by both `args.Gas` (if non-nil & non-zero) and the backend's RPCGasCap
// configuration (if non-zero).
// Note: Required blob gas is not computed in this method.
func (api *BlockChainAPI) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Uint64, error) {
	return callapi.EstimateGas(ctx, api.b, args, blockNrOrHash, overrides, blockOverrides, api.b.RPCGasCap())
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	return blockapi.RPCMarshalHeader(head)
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction.
type RPCTransaction = rpctx.RPCTransaction

type accessListResult = callapi.AccessListResult

// CreateAccessList creates an EIP-2930 type AccessList for the given transaction.
// Reexec and BlockNrOrHash can be specified to create the accessList on top of a certain state.
// StateOverrides can be used to create the accessList while taking into account state changes from previous transactions.
func (api *BlockChainAPI) CreateAccessList(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride) (*accessListResult, error) {
	return callapi.CreateAccessList(ctx, api.b, args, blockNrOrHash, stateOverrides)
}

type DebugAPI struct {
	b Backend
	*silaapi.DebugAPI
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewDebugAPI(b Backend) *DebugAPI {
	return &DebugAPI{b: b, DebugAPI: silaapi.NewDebugAPI(b)}
}

type NetAPI = netapi.NetAPI

// NewNetAPI creates a new net API instance.
func NewNetAPI(net *p2p.Server, networkVersion uint64) *NetAPI {
	return netapi.NewNetAPI(net, networkVersion)
}
