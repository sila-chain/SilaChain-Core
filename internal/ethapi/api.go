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
	"errors"
	"fmt"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"

	"github.com/sila-org/sila/core/forkid"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/netapi"
	"github.com/sila-org/sila/internal/silaapi/override"
	"github.com/sila-org/sila/internal/silaapi/proofapi"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/internal/silaapi/txapi"

	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"

	"time"
)

// estimateGasErrorRatio is the amount of overestimation eth_estimateGas is
// allowed to produce in order to speed up calculations.
const estimateGasErrorRatio = 0.015

// maxGetStorageSlots is the maximum total number of storage slots that can
// be requested in a single eth_getStorageValues call.
const maxGetStorageSlots = 1024

var errBlobTxNotSupported = errors.New("signing blob transactions not supported")
var errSubClosed = errors.New("chain subscription closed")

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
}

// NewSilaBlockChainAPI creates a new SilaChain blockchain API.
func NewSilaBlockChainAPI(b Backend) *BlockChainAPI {
	return &BlockChainAPI{b}
}

// NewBlockChainAPI creates a new SilaChain blockchain API.
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return NewSilaBlockChainAPI(b)
}

// ChainId is the replay-protection chain id for the current SilaChain config.
//
// Note, this method does not conform to EIP-695 because the configured chain ID is always
// returned, regardless of the current head block. We used to return an error when the chain
// wasn't synced up to a block where EIP-155 is enabled, but this behavior caused issues
// in CL clients.
func (api *BlockChainAPI) ChainId() *hexutil.Big {
	return blockapi.ChainId(api.b)
}

// BlockNumber returns the block number of the chain head.
func (api *BlockChainAPI) BlockNumber() hexutil.Uint64 {
	return blockapi.BlockNumber(api.b)
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *BlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	return blockapi.GetBalance(ctx, api.b, address, blockNrOrHash)
}

// AccountResult structs for GetProof
type AccountResult = proofapi.AccountResult
type StorageResult = proofapi.StorageResult
type proofList = proofapi.ProofList

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func (api *BlockChainAPI) GetProof(ctx context.Context, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	return proofapi.GetProof(ctx, api.b, address, storageKeys, blockNrOrHash)
}

// GetHeaderByNumber returns the requested canonical block header.
//   - When number is -1 the chain pending header is returned.
//   - When number is -2 the chain latest header is returned.
//   - When number is -3 the chain finalized header is returned.
//   - When number is -4 the chain safe header is returned.
func (api *BlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	return blockapi.GetHeaderByNumber(ctx, api.b, number)
}

// GetHeaderByHash returns the requested header by hash.
func (api *BlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	return blockapi.GetHeaderByHash(ctx, api.b, hash)
}

// GetBlockByNumber returns the requested canonical block.
//   - When number is -1 the chain pending block is returned.
//   - When number is -2 the chain latest block is returned.
//   - When number is -3 the chain finalized block is returned.
//   - When number is -4 the chain safe block is returned.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	return blockapi.GetBlockByNumber(ctx, api.b, number, fullTx)
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	return blockapi.GetBlockByHash(ctx, api.b, hash, fullTx)
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block hash and index.
func (api *BlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	return blockapi.GetUncleByBlockNumberAndIndex(ctx, api.b, blockNr, index)
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index.
func (api *BlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	return blockapi.GetUncleByBlockHashAndIndex(ctx, api.b, blockHash, index)
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number
func (api *BlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	return blockapi.GetUncleCountByBlockNumber(ctx, api.b, blockNr)
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (api *BlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	return blockapi.GetUncleCountByBlockHash(ctx, api.b, blockHash)
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (api *BlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return blockapi.GetCode(ctx, api.b, address, blockNrOrHash)
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (api *BlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return blockapi.GetStorageAt(ctx, api.b, address, hexKey, blockNrOrHash)
}

// GetStorageValues returns multiple storage slot values for multiple accounts
// at the given block.
func (api *BlockChainAPI) GetStorageValues(ctx context.Context, requests map[common.Address][]common.Hash, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address][]hexutil.Bytes, error) {
	return blockapi.GetStorageValues(ctx, api.b, requests, blockNrOrHash)
}

// GetBlockReceipts returns the block receipts for the given block hash or number or tag.
func (api *BlockChainAPI) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	return blockapi.GetBlockReceipts(ctx, api.b, blockNrOrHash)
}

// Call executes the given transaction on the state for the given block number.
//
// Additionally, the caller can specify a batch of contract for fields overriding.
//
// Note, this function doesn't make and changes in the state/blockchain and is
// useful to execute and retrieve values.
func (api *BlockChainAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Bytes, error) {
	return callapi.Call(ctx, api.b, args, blockNrOrHash, overrides, blockOverrides, api.b.RPCEVMTimeout(), api.b.RPCGasCap())
}

// SimulateV1 executes series of transactions on top of a base state.
// The transactions are packed into blocks. For each block, block header
// fields can be overridden. The state can also be overridden prior to
// execution of each block.
//
// Note, this function doesn't make any changes in the state/blockchain and is
// useful to execute and retrieve values.
func (api *BlockChainAPI) SimulateV1(ctx context.Context, opts simOpts, blockNrOrHash *rpc.BlockNumberOrHash) ([]*simBlockResult, error) {
	if len(opts.BlockStateCalls) == 0 {
		return nil, &ethapierrors.InvalidParamsError{Message: "empty input"}
	} else if len(opts.BlockStateCalls) > maxSimulateBlocks {
		return nil, &ethapierrors.ClientLimitExceededError{Message: "too many blocks"}
	}
	var totalCalls int
	for _, block := range opts.BlockStateCalls {
		if len(block.Calls) > maxSimulateCallsPerBlock {
			return nil, &ethapierrors.ClientLimitExceededError{Message: fmt.Sprintf("too many calls in block: %d > %d", len(block.Calls), maxSimulateCallsPerBlock)}
		}
		totalCalls += len(block.Calls)
		if totalCalls > maxSimulateTotalCalls {
			return nil, &ethapierrors.ClientLimitExceededError{Message: fmt.Sprintf("too many calls: %d > %d", totalCalls, maxSimulateTotalCalls)}
		}
	}
	if blockNrOrHash == nil {
		n := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &n
	}
	state, base, err := api.b.StateAndHeaderByNumberOrHash(ctx, *blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	sim := &simulator{
		b:              api.b,
		state:          state,
		base:           base,
		chainConfig:    api.b.ChainConfig(),
		budget:         newGasBudget(api.b.RPCGasCap()),
		traceTransfers: opts.TraceTransfers,
		validate:       opts.Validation,
		fullTx:         opts.ReturnFullTransactions,
	}
	return sim.execute(ctx, opts.BlockStateCalls)
}

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

type config struct {
	ActivationTime  uint64                    `json:"activationTime"`
	BlobSchedule    *params.BlobConfig        `json:"blobSchedule"`
	ChainId         *hexutil.Big              `json:"chainId"`
	ForkId          hexutil.Bytes             `json:"forkId"`
	Precompiles     map[string]common.Address `json:"precompiles"`
	SystemContracts map[string]common.Address `json:"systemContracts"`
}

type configResponse struct {
	Current *config `json:"current"`
	Next    *config `json:"next"`
	Last    *config `json:"last"`
}

// Config implements the EIP-7910 eth_config method.
func (api *BlockChainAPI) Config(ctx context.Context) (*configResponse, error) {
	genesis, err := api.b.HeaderByNumber(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to load genesis: %w", err)
	}
	assemble := func(c *params.ChainConfig, ts *uint64) *config {
		if ts == nil {
			return nil
		}
		t := *ts

		var (
			rules       = c.Rules(c.LondonBlock, true, t)
			precompiles = make(map[string]common.Address)
		)
		for addr, c := range vm.ActivePrecompiledContracts(rules) {
			precompiles[c.Name()] = addr
		}
		// Activation time is required. If a fork is activated at genesis the value 0 is used
		activationTime := t
		if genesis.Time >= t {
			activationTime = 0
		}
		forkid := forkid.NewID(c, types.NewBlockWithHeader(genesis), ^uint64(0), t).Hash
		return &config{
			ActivationTime:  activationTime,
			BlobSchedule:    c.BlobConfig(c.LatestFork(t)),
			ChainId:         (*hexutil.Big)(c.ChainID),
			ForkId:          forkid[:],
			Precompiles:     precompiles,
			SystemContracts: c.ActiveSystemContracts(t),
		}
	}
	var (
		c = api.b.ChainConfig()
		t = api.b.CurrentHeader().Time
	)
	resp := configResponse{
		Next:    assemble(c, c.Timestamp(c.LatestFork(t)+1)),
		Current: assemble(c, c.Timestamp(c.LatestFork(t))),
		Last:    assemble(c, c.Timestamp(c.LatestFork(^uint64(0)))),
	}
	// Nil out last if no future-fork is configured.
	if resp.Next == nil {
		resp.Last = nil
	}
	return &resp, nil
}

// TransactionAPI exposes methods for reading and creating transaction data.
type TransactionAPI struct {
	*txapi.TransactionAPI
}

// NewSilaTransactionAPI creates a new RPC service with methods for interacting with transactions.
func NewSilaTransactionAPI(b Backend, nonceLock *addrlock.AddrLocker) *TransactionAPI {
	return &TransactionAPI{
		TransactionAPI: txapi.NewTransactionAPI(b, nonceLock),
	}
}

// NewTransactionAPI creates a new RPC service with methods for interacting with transactions.
func NewTransactionAPI(b Backend, nonceLock *addrlock.AddrLocker) *TransactionAPI {
	return NewSilaTransactionAPI(b, nonceLock)
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
//
// This API is not capable for submitting blob transaction with sidecar.
func (api *TransactionAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	return api.SendTransactionWithBlobError(ctx, args, errBlobTxNotSupported)
}

// SendRawTransactionSync will add the signed transaction to the transaction pool
// and wait until the transaction has been included in a block and return the receipt, or the timeout.
func (api *TransactionAPI) SendRawTransactionSync(ctx context.Context, input hexutil.Bytes, timeoutMs *uint64) (map[string]interface{}, error) {
	return api.SendRawTransactionSyncWithErrors(ctx, input, timeoutMs, errSubClosed, func(hash common.Hash, timeout time.Duration) error {
		return &txSyncTimeoutError{
			msg:  fmt.Sprintf("The transaction was added to the transaction pool but wasn't processed in %v", timeout),
			hash: hash,
		}
	})
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
