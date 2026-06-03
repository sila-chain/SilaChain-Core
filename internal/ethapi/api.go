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
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/forkid"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/eth/tracers/logger"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	"github.com/sila-org/sila/internal/silaapi/chainctx"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/netapi"
	"github.com/sila-org/sila/internal/silaapi/override"
	"github.com/sila-org/sila/internal/silaapi/proofapi"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/internal/silaapi/txapi"
	"github.com/sila-org/sila/internal/silaapi/txfee"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
	"math/big"
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

// NewBlockChainAPI creates a new SilaChain blockchain API.
func NewBlockChainAPI(b Backend) *BlockChainAPI {
	return &BlockChainAPI{b}
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

func decodeStorageKey(s string) (common.Hash, int, error) {
	return proofapi.DecodeStorageKey(s)
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
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", blockNr, "hash", block.Hash(), "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return blockapi.RPCMarshalBlock(block, false, false, api.b.ChainConfig()), nil
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index.
func (api *BlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := api.b.BlockByHash(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			log.Debug("Requested uncle not found", "number", block.Number(), "hash", blockHash, "index", index)
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return blockapi.RPCMarshalBlock(block, false, false, api.b.ChainConfig()), nil
	}
	return nil, err
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
	var (
		err      error
		block    *types.Block
		receipts types.Receipts
	)
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		block, receipts, _ = api.b.Pending()
		if block == nil {
			return nil, errors.New("pending receipts is not available")
		}
	} else {
		block, err = api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		receipts, err = api.b.GetReceipts(ctx, block.Hash())
		if err != nil {
			return nil, err
		}
	}
	txs := block.Transactions()
	if len(txs) != len(receipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(receipts))
	}
	// Derive the sender.
	signer := types.MakeSigner(api.b.ChainConfig(), block.Number(), block.Time())

	result := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		result[i] = rpctx.MarshalReceipt(receipt, block.Hash(), block.NumberU64(), signer, txs[i], i)
	}
	return result, nil
}

// Call executes the given transaction on the state for the given block number.
//
// Additionally, the caller can specify a batch of contract for fields overriding.
//
// Note, this function doesn't make and changes in the state/blockchain and is
// useful to execute and retrieve values.
func (api *BlockChainAPI) Call(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Bytes, error) {
	if blockNrOrHash == nil {
		latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &latest
	}
	result, err := callapi.DoCall(ctx, api.b, args, *blockNrOrHash, overrides, blockOverrides, api.b.RPCEVMTimeout(), api.b.RPCGasCap())
	if err != nil {
		return nil, err
	}
	if errors.Is(result.Err, vm.ErrExecutionReverted) {
		return nil, ethapierrors.NewRevertError(result.Revert())
	}
	return result.Return(), result.Err
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
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return callapi.DoEstimateGas(ctx, api.b, args, bNrOrHash, overrides, blockOverrides, api.b.RPCGasCap())
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	return blockapi.RPCMarshalHeader(head)
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction.
type RPCTransaction = rpctx.RPCTransaction

// accessListResult returns an optional accesslist
// It's the result of the `debug_createAccessList` RPC call.
// It contains an error if the transaction itself failed.
type accessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

// CreateAccessList creates an EIP-2930 type AccessList for the given transaction.
// Reexec and BlockNrOrHash can be specified to create the accessList on top of a certain state.
// StateOverrides can be used to create the accessList while taking into account state changes from previous transactions.
func (api *BlockChainAPI) CreateAccessList(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride) (*accessListResult, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	acl, gasUsed, vmerr, err := AccessList(ctx, api.b, bNrOrHash, args, stateOverrides)
	if err != nil {
		return nil, err
	}
	result := &accessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmerr != nil {
		result.Error = vmerr.Error()
	}
	return result, nil
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

// AccessList creates an access list for the given transaction.
// If the accesslist creation fails an error is returned.
// If the transaction itself fails, an vmErr is returned.
func AccessList(ctx context.Context, b Backend, blockNrOrHash rpc.BlockNumberOrHash, args TransactionArgs, stateOverrides *override.StateOverride) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	// Retrieve the execution context
	db, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if db == nil || err != nil {
		return nil, 0, nil, err
	}

	// Apply state overrides immediately after StateAndHeaderByNumberOrHash.
	// If not applied here, there could be cases where user-specified overrides (e.g., nonce)
	// may conflict with default values from the database, leading to inconsistencies.
	if stateOverrides != nil {
		if err := stateOverrides.Apply(db, nil); err != nil {
			return nil, 0, nil, err
		}
	}

	// Ensure any missing fields are filled, extract the recipient and input data
	if err = setFeeDefaults(&args, ctx, b, header); err != nil {
		return nil, 0, nil, err
	}
	if args.Nonce == nil {
		nonce := hexutil.Uint64(db.GetNonce(args.FromAddr()))
		args.Nonce = &nonce
	}
	blockCtx := core.NewEVMBlockContext(header, chainctx.NewChainContext(ctx, b), nil)
	if err = args.CallDefaults(b.RPCGasCap(), blockCtx.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, 0, nil, err
	}

	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.FromAddr(), uint64(*args.Nonce))
	}
	isPostMerge := header.Difficulty.Sign() == 0
	// Retrieve the precompiles since they don't need to be added to the access list
	precompiles := vm.ActivePrecompiles(b.ChainConfig().Rules(header.Number, isPostMerge, header.Time))

	// addressesToExclude contains sender, receiver, precompiles and valid authorizations
	addressesToExclude := map[common.Address]struct{}{args.FromAddr(): {}, to: {}}
	for _, addr := range precompiles {
		addressesToExclude[addr] = struct{}{}
	}

	// Prevent redundant operations if args contain more authorizations than EVM may handle
	maxAuthorizations := uint64(*args.Gas) / params.CallNewAccountGas
	if uint64(len(args.AuthorizationList)) > maxAuthorizations {
		return nil, 0, nil, errors.New("insufficient gas to process all authorizations")
	}

	for _, auth := range args.AuthorizationList {
		// Duplicating stateTransition.validateAuthorization() logic
		if (!auth.ChainID.IsZero() && auth.ChainID.CmpBig(b.ChainConfig().ChainID) != 0) || auth.Nonce+1 < auth.Nonce {
			continue
		}

		if authority, err := auth.Authority(); err == nil {
			addressesToExclude[authority] = struct{}{}
		}
	}

	// Create an initial tracer
	prevTracer := logger.NewAccessListTracer(nil, addressesToExclude)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, addressesToExclude)
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, nil, err
		}
		// Retrieve the current access list to expand
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)

		// Copy the original db so we don't modify it
		statedb := db.Copy()
		// Set the accesslist to the last al
		args.AccessList = &accessList
		msg := args.ToMessage(header.BaseFee, true)

		// Apply the transaction with the access list tracer
		tracer := logger.NewAccessListTracer(accessList, addressesToExclude)
		config := vm.Config{Tracer: tracer.Hooks(), NoBaseFee: true}
		evm := b.GetEVM(ctx, statedb, header, &config, nil)

		// Lower the basefee to 0 to avoid breaking EVM
		// invariants (basefee < feecap).
		if msg.GasPrice.Sign() == 0 {
			evm.Context.BaseFee = new(big.Int)
		}
		if msg.BlobGasFeeCap != nil && msg.BlobGasFeeCap.BitLen() == 0 {
			evm.Context.BlobBaseFee = new(big.Int)
		}
		res, err := core.ApplyMessage(evm, msg, nil)
		evm.Release()
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to apply transaction: %v err: %v", args.ToTransaction(types.LegacyTxType).Hash(), err)
		}
		if tracer.Equal(prevTracer) {
			return accessList, res.UsedGas, res.Err, nil
		}
		prevTracer = tracer
	}
}

// TransactionAPI exposes methods for reading and creating transaction data.
type TransactionAPI struct {
	b         Backend
	nonceLock *addrlock.AddrLocker
	signer    types.Signer
}

// NewTransactionAPI creates a new RPC service with methods for interacting with transactions.
func NewTransactionAPI(b Backend, nonceLock *addrlock.AddrLocker) *TransactionAPI {
	// The signer used by the API should always be the 'latest' known one because we expect
	// signers to be backwards-compatible with old transactions.
	signer := types.LatestSigner(b.ChainConfig())
	return &TransactionAPI{b, nonceLock, signer}
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func (api *TransactionAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n, nil
	}
	return nil, err
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func (api *TransactionAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	block, err := api.b.BlockByHash(ctx, blockHash)
	if block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n, nil
	}
	return nil, err
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func (api *TransactionAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (*RPCTransaction, error) {
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		return blockapi.NewRPCTransactionFromBlockIndex(block, uint64(index), api.b.ChainConfig()), nil
	}
	return nil, err
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func (api *TransactionAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (*RPCTransaction, error) {
	block, err := api.b.BlockByHash(ctx, blockHash)
	if block != nil {
		return blockapi.NewRPCTransactionFromBlockIndex(block, uint64(index), api.b.ChainConfig()), nil
	}
	return nil, err
}

// GetRawTransactionByBlockNumberAndIndex returns the bytes of the transaction for the given block number and index.
func (api *TransactionAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	if block, _ := api.b.BlockByNumber(ctx, blockNr); block != nil {
		return blockapi.NewRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the bytes of the transaction for the given block hash and index.
func (api *TransactionAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	if block, _ := api.b.BlockByHash(ctx, blockHash); block != nil {
		return blockapi.NewRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetTransactionCount returns the number of transactions the given address has sent for the given block number
func (api *TransactionAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	// Ask transaction pool for the nonce which includes pending transactions
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		nonce, err := api.b.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	// Resolve block number and use its state to ask for the nonce
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

// GetTransactionByHash returns the transaction for the given hash
func (api *TransactionAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*RPCTransaction, error) {
	return txapi.GetTransactionByHash(ctx, api.b, hash)
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (api *TransactionAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	return txapi.GetRawTransactionByHash(api.b, hash)
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (api *TransactionAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	return txapi.GetTransactionReceipt(api.b, api.signer, hash)
}

// sign is a helper function that signs a transaction with the private key of the given address.
func (api *TransactionAPI) sign(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := api.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	return wallet.SignTx(account, tx, api.b.ChainConfig().ChainID)
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b Backend, tx *types.Transaction) (common.Hash, error) {
	// If the transaction fee cap is already specified, ensure the
	// fee of the given transaction is _reasonable_.
	if err := txfee.CheckTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	if !b.UnprotectedAllowed() && !tx.Protected() {
		// Ensure only eip155 signed transactions are submitted if EIP155Required is set.
		return common.Hash{}, errors.New("only replay-protected (EIP-155) transactions allowed over RPC")
	}
	if err := b.SendTx(ctx, tx); err != nil {
		return common.Hash{}, err
	}
	// Print a log with full tx details for manual investigations and interventions
	head := b.CurrentBlock()
	signer := types.MakeSigner(b.ChainConfig(), head.Number, head.Time)
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, err
	}

	if tx.To() == nil {
		addr := crypto.CreateAddress(from, tx.Nonce())
		log.Info("Submitted contract creation", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "contract", addr.Hex(), "value", tx.Value())
	} else {
		log.Info("Submitted transaction", "hash", tx.Hash().Hex(), "from", from, "nonce", tx.Nonce(), "recipient", tx.To(), "value", tx.Value())
	}
	return tx.Hash(), nil
}

// SendTransaction creates a transaction for the given argument, sign it and submit it to the
// transaction pool.
//
// This API is not capable for submitting blob transaction with sidecar.
func (api *TransactionAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: args.FromAddr()}

	wallet, err := api.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		api.nonceLock.LockAddr(args.FromAddr())
		defer api.nonceLock.UnlockAddr(args.FromAddr())
	}
	if args.IsEIP4844() {
		return common.Hash{}, errBlobTxNotSupported
	}

	// Set some sanity defaults and terminate on failure
	if err := setDefaults(&args, ctx, api.b, sidecarConfig{}); err != nil {
		return common.Hash{}, err
	}
	// Assemble the transaction and sign with the wallet
	tx := args.ToTransaction(types.DynamicFeeTxType)

	signed, err := wallet.SignTx(account, tx, api.b.ChainConfig().ChainID)
	if err != nil {
		return common.Hash{}, err
	}
	return SubmitTransaction(ctx, api.b, signed)
}

// FillTransaction fills the defaults (nonce, gas, gasPrice or 1559 fields)
// on a given unsigned transaction, and returns it to the caller for further
// processing (signing + broadcast).
func (api *TransactionAPI) FillTransaction(ctx context.Context, args TransactionArgs) (*silaapi.SignTransactionResult, error) {
	// Set some sanity defaults and terminate on failure
	config := sidecarConfig{
		blobSidecarAllowed: true,
		blobSidecarVersion: api.currentBlobSidecarVersion(),
	}
	if err := setDefaults(&args, ctx, api.b, config); err != nil {
		return nil, err
	}
	// Assemble the transaction and obtain rlp
	tx := args.ToTransaction(types.DynamicFeeTxType)
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &silaapi.SignTransactionResult{Raw: data, Tx: tx}, nil
}

func (api *TransactionAPI) currentBlobSidecarVersion() byte {
	h := api.b.CurrentHeader()
	if api.b.ChainConfig().IsOsaka(h.Number, h.Time) {
		return types.BlobSidecarVersion1
	}
	return types.BlobSidecarVersion0
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func (api *TransactionAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}

	// Convert legacy blob transaction proofs.
	// TODO: remove in a future SilaChain release
	if sc := tx.BlobTxSidecar(); sc != nil {
		exp := api.currentBlobSidecarVersion()
		if sc.Version == types.BlobSidecarVersion0 && exp == types.BlobSidecarVersion1 {
			if err := sc.ToV1(); err != nil {
				return common.Hash{}, fmt.Errorf("blob sidecar conversion failed: %v", err)
			}
			tx = tx.WithBlobTxSidecar(sc)
		}
	}

	return SubmitTransaction(ctx, api.b, tx)
}

// SendRawTransactionSync will add the signed transaction to the transaction pool
// and wait until the transaction has been included in a block and return the receipt, or the timeout.
func (api *TransactionAPI) SendRawTransactionSync(ctx context.Context, input hexutil.Bytes, timeoutMs *uint64) (map[string]interface{}, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return nil, err
	}

	// Convert legacy blob transaction proofs.
	// TODO: remove in a future SilaChain release
	if sc := tx.BlobTxSidecar(); sc != nil {
		exp := api.currentBlobSidecarVersion()
		if sc.Version == types.BlobSidecarVersion0 && exp == types.BlobSidecarVersion1 {
			if err := sc.ToV1(); err != nil {
				return nil, fmt.Errorf("blob sidecar conversion failed: %v", err)
			}
			tx = tx.WithBlobTxSidecar(sc)
		}
	}

	ch := make(chan core.ChainEvent, 128)
	sub := api.b.SubscribeChainEvent(ch)
	defer sub.Unsubscribe()

	hash, err := SubmitTransaction(ctx, api.b, tx)
	if err != nil {
		return nil, err
	}

	var (
		maxTimeout     = api.b.RPCTxSyncMaxTimeout()
		defaultTimeout = api.b.RPCTxSyncDefaultTimeout()
		timeout        = defaultTimeout
	)
	if timeoutMs != nil && *timeoutMs > 0 {
		req := time.Duration(*timeoutMs) * time.Millisecond
		if req > maxTimeout {
			timeout = maxTimeout
		} else {
			timeout = req
		}
	}
	receiptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Fast path.
	if r, err := api.GetTransactionReceipt(receiptCtx, hash); err == nil && r != nil {
		return r, nil
	}

	// Monitor the receipts
	for {
		select {
		case <-receiptCtx.Done():
			// If server-side wait window elapsed, return the structured timeout.
			if errors.Is(receiptCtx.Err(), context.DeadlineExceeded) {
				return nil, &txSyncTimeoutError{
					msg:  fmt.Sprintf("The transaction was added to the transaction pool but wasn't processed in %v", timeout),
					hash: hash,
				}
			}
			return nil, receiptCtx.Err()

		case err, ok := <-sub.Err():
			if !ok {
				return nil, errSubClosed
			}
			return nil, err

		case ev, ok := <-ch:
			if !ok {
				return nil, errSubClosed
			}
			rs, txs := ev.Receipts, ev.Transactions
			if len(rs) == 0 || len(rs) != len(txs) {
				continue
			}
			for i := range rs {
				if rs[i].TxHash == hash {
					if rs[i].BlockNumber != nil && rs[i].BlockHash != (common.Hash{}) {
						signer := types.LatestSigner(api.b.ChainConfig())
						return rpctx.MarshalReceipt(
							rs[i],
							rs[i].BlockHash,
							rs[i].BlockNumber.Uint64(),
							signer,
							txs[i],
							int(rs[i].TransactionIndex),
						), nil
					}
					return api.GetTransactionReceipt(receiptCtx, hash)
				}
			}
		}
	}
}

// Sign calculates an ECDSA signature using the legacy Ethereum signed-message prefix for compatibility:
// keccak256("\x19Ethereum Signed Message:\n" + len(message) + message).
//
// Note, the produced signature conforms to the secp256k1 curve R, S and V values,
// where the V value will be 27 or 28 for legacy reasons.
//
// The account associated with addr must be unlocked.
//
// JSON-RPC eth_sign
func (api *TransactionAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	wallet, err := api.b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Sign the requested hash with the wallet
	signature, err := wallet.SignText(account, data)
	if err == nil {
		signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	}
	return signature, err
}

// SignTransaction will sign the given transaction with the from account.
// The node needs to have the private key of the account corresponding with
// the given from address and it needs to be unlocked.
func (api *TransactionAPI) SignTransaction(ctx context.Context, args TransactionArgs) (*silaapi.SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, errors.New("gas not specified")
	}
	if args.GasPrice == nil && (args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil) {
		return nil, errors.New("missing gasPrice or maxFeePerGas/maxPriorityFeePerGas")
	}
	if args.Nonce == nil {
		return nil, errors.New("nonce not specified")
	}
	sidecarVersion := types.BlobSidecarVersion0
	if len(args.Blobs) > 0 {
		h := api.b.CurrentHeader()
		if api.b.ChainConfig().IsOsaka(h.Number, h.Time) {
			sidecarVersion = types.BlobSidecarVersion1
		}
	}

	config := sidecarConfig{
		blobSidecarAllowed: true,
		blobSidecarVersion: sidecarVersion,
	}
	if err := setDefaults(&args, ctx, api.b, config); err != nil {
		return nil, err
	}
	// Before actually sign the transaction, ensure the transaction fee is reasonable.
	tx := args.ToTransaction(types.DynamicFeeTxType)
	if err := txfee.CheckTxFee(tx.GasPrice(), tx.Gas(), api.b.RPCTxFeeCap()); err != nil {
		return nil, err
	}
	signed, err := api.sign(args.FromAddr(), tx)
	if err != nil {
		return nil, err
	}
	// If the transaction-to-sign was a blob transaction, then the signed one
	// no longer retains the blobs, only the blob hashes. In this step, we need
	// to put back the blob(s).
	if args.IsEIP4844() {
		signed = signed.WithBlobTxSidecar(types.NewBlobTxSidecar(sidecarVersion, args.Blobs, args.Commitments, args.Proofs))
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &silaapi.SignTransactionResult{Raw: data, Tx: signed}, nil
}

// PendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (api *TransactionAPI) PendingTransactions() ([]*RPCTransaction, error) {
	return txapi.PendingTransactions(api.b, api.signer)
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
//
// This API is not capable for submitting blob transaction with sidecar.
func (api *TransactionAPI) Resend(ctx context.Context, sendArgs TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, errors.New("missing transaction nonce in transaction spec")
	}
	if err := setDefaults(&sendArgs, ctx, api.b, sidecarConfig{}); err != nil {
		return common.Hash{}, err
	}
	matchTx := sendArgs.ToTransaction(types.DynamicFeeTxType)

	// Before replacing the old transaction, ensure the _new_ transaction fee is reasonable.
	price := matchTx.GasPrice()
	if gasPrice != nil {
		price = gasPrice.ToInt()
	}
	gas := matchTx.Gas()
	if gasLimit != nil {
		gas = uint64(*gasLimit)
	}
	if err := txfee.CheckTxFee(price, gas, api.b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	// Iterate the pending list for replacement
	pending, err := api.b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}
	for _, p := range pending {
		wantSigHash := api.signer.Hash(matchTx)
		pFrom, err := types.Sender(api.signer, p)
		if err == nil && pFrom == sendArgs.FromAddr() && api.signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := api.sign(sendArgs.FromAddr(), sendArgs.ToTransaction(types.DynamicFeeTxType))
			if err != nil {
				return common.Hash{}, err
			}
			if err = api.b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}
	return common.Hash{}, fmt.Errorf("transaction %#x not found", matchTx.Hash())
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
