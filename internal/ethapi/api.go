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
	ethereum "github.com/sila-org/sila"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/chainctx"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	gomath "math"
	"math/big"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/common/math"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/forkid"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/eth/gasestimator"
	"github.com/sila-org/sila/eth/tracers/logger"
	"github.com/sila-org/sila/internal/silaapi/override"
	"github.com/sila-org/sila/internal/silaapi/proofapi"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/internal/silaapi/txapi"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rlp"
	"github.com/sila-org/sila/rpc"
)

// estimateGasErrorRatio is the amount of overestimation eth_estimateGas is
// allowed to produce in order to speed up calculations.
const estimateGasErrorRatio = 0.015

// maxGetStorageSlots is the maximum total number of storage slots that can
// be requested in a single eth_getStorageValues call.
const maxGetStorageSlots = 1024

var errBlobTxNotSupported = errors.New("signing blob transactions not supported")
var errSubClosed = errors.New("chain subscription closed")

// SilaAPI provides an API to access SilaChain related information.
type SilaAPIBackend interface {
	SyncProgress(ctx context.Context) ethereum.SyncProgress
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, []*big.Int, []float64, error)
	BlobBaseFee(ctx context.Context) *big.Int
	CurrentHeader() *types.Header
}

type SilaAPI struct {
	b SilaAPIBackend
}

// NewSilaAPI creates a new SilaChain protocol API.
func NewSilaAPI(b SilaAPIBackend) *SilaAPI {
	return &SilaAPI{b}
}

// GasPrice returns a suggestion for a gas price for legacy transactions.
func (api *SilaAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	return silaapi.GasPrice(ctx, api.b)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func (api *SilaAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	return silaapi.MaxPriorityFeePerGas(ctx, api.b)
}

// FeeHistory returns the fee market history.
func (api *SilaAPI) FeeHistory(ctx context.Context, blockCount math.HexOrDecimal64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*silaapi.FeeHistoryResult, error) {
	return silaapi.FeeHistory(ctx, api.b, blockCount, lastBlock, rewardPercentiles)
}

// BlobBaseFee returns the base fee for blob gas at the current head.
func (api *SilaAPI) BlobBaseFee(ctx context.Context) *hexutil.Big {
	return silaapi.BlobBaseFee(ctx, api.b)
}

// Syncing returns false in case the node is currently not syncing with the network.
func (api *SilaAPI) Syncing(ctx context.Context) (interface{}, error) {
	return silaapi.Syncing(ctx, api.b)
}

// TxPoolAPI offers and API for the transaction pool. It only operates on data that is non-confidential.
type TxPoolAPI struct {
	b Backend
}

// NewTxPoolAPI creates a new tx pool service that gives information about the transaction pool.
func NewTxPoolAPI(b Backend) *TxPoolAPI {
	return &TxPoolAPI{b}
}

// flattenTxs builds the RPC transaction map keyed by nonce for a set of pool txs.
func flattenTxs(txs types.Transactions, header *types.Header, cfg *params.ChainConfig) map[string]*RPCTransaction {
	dump := make(map[string]*RPCTransaction, len(txs))
	for _, tx := range txs {
		dump[fmt.Sprintf("%d", tx.Nonce())] = rpctx.NewRPCPendingTransaction(tx, header, cfg)
	}
	return dump
}

// Content returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) Content() map[string]map[string]map[string]*RPCTransaction {
	pending, queue := api.b.TxPoolContent()
	content := map[string]map[string]map[string]*RPCTransaction{
		"pending": make(map[string]map[string]*RPCTransaction, len(pending)),
		"queued":  make(map[string]map[string]*RPCTransaction, len(queue)),
	}
	curHeader := api.b.CurrentHeader()
	// Flatten the pending transactions
	for account, txs := range pending {
		content["pending"][account.Hex()] = flattenTxs(txs, curHeader, api.b.ChainConfig())
	}
	// Flatten the queued transactions
	for account, txs := range queue {
		content["queued"][account.Hex()] = flattenTxs(txs, curHeader, api.b.ChainConfig())
	}
	return content
}

// ContentFrom returns the transactions contained within the transaction pool.
func (api *TxPoolAPI) ContentFrom(addr common.Address) map[string]map[string]*RPCTransaction {
	content := make(map[string]map[string]*RPCTransaction, 2)
	pending, queue := api.b.TxPoolContentFrom(addr)
	curHeader := api.b.CurrentHeader()

	// Build the pending transactions
	content["pending"] = flattenTxs(pending, curHeader, api.b.ChainConfig())

	// Build the queued transactions
	content["queued"] = flattenTxs(queue, curHeader, api.b.ChainConfig())

	return content
}

// Status returns the number of pending and queued transaction in the pool.
func (api *TxPoolAPI) Status() map[string]hexutil.Uint {
	return silaapi.TxPoolStatus(api.b)
}

// Inspect retrieves the content of the transaction pool and flattens it into an
// easily inspectable list.
func (api *TxPoolAPI) Inspect() map[string]map[string]map[string]string {
	return silaapi.TxPoolInspect(api.b)
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
	return (*hexutil.Big)(api.b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func (api *BlockChainAPI) BlockNumber() hexutil.Uint64 {
	header, _ := api.b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber) // latest header should always be available
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *BlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	b := state.GetBalance(address).ToBig()
	return (*hexutil.Big)(b), state.Error()
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
	header, err := api.b.HeaderByNumber(ctx, number)
	if header != nil && err == nil {
		response := RPCMarshalHeader(header)
		if number == rpc.PendingBlockNumber {
			// Pending header need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetHeaderByHash returns the requested header by hash.
func (api *BlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header != nil {
		return RPCMarshalHeader(header)
	}
	return nil
}

// GetBlockByNumber returns the requested canonical block.
//   - When number is -1 the chain pending block is returned.
//   - When number is -2 the chain latest block is returned.
//   - When number is -3 the chain finalized block is returned.
//   - When number is -4 the chain safe block is returned.
//   - When fullTx is true all transactions in the block are returned, otherwise
//     only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.BlockByNumber(ctx, number)
	if block != nil && err == nil {
		response := blockapi.RPCMarshalBlock(block, true, fullTx, api.b.ChainConfig())
		if number == rpc.PendingBlockNumber {
			// Pending blocks need to nil out a few fields
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, nil
	}
	return nil, err
}

// GetBlockByHash returns the requested block. When fullTx is true all transactions in the block are returned in full
// detail, otherwise only the transaction hash is returned.
func (api *BlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := api.b.BlockByHash(ctx, hash)
	if block != nil {
		return blockapi.RPCMarshalBlock(block, true, fullTx, api.b.ChainConfig()), nil
	}
	return nil, err
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
	block, err := api.b.BlockByNumber(ctx, blockNr)
	if block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n, nil
	}
	return nil, err
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash
func (api *BlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	block, err := api.b.BlockByHash(ctx, blockHash)
	if block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n, nil
	}
	return nil, err
}

// GetCode returns the code stored at the given address in the state for the given block number.
func (api *BlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// GetStorageAt returns the storage from the state at the given address, key and
// block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta block
// numbers are also allowed.
func (api *BlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	key, _, err := decodeStorageKey(hexKey)
	if err != nil {
		return nil, &ethapierrors.InvalidParamsError{fmt.Sprintf("%v: %q", err, hexKey)}
	}
	res := state.GetState(address, key)
	return res[:], state.Error()
}

// GetStorageValues returns multiple storage slot values for multiple accounts
// at the given block.
func (api *BlockChainAPI) GetStorageValues(ctx context.Context, requests map[common.Address][]common.Hash, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address][]hexutil.Bytes, error) {
	// Count total slots requested.
	var totalSlots int
	for _, keys := range requests {
		totalSlots += len(keys)
		if totalSlots > maxGetStorageSlots {
			return nil, &ethapierrors.ClientLimitExceededError{Message: fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots)}
		}
	}
	if totalSlots == 0 {
		return nil, &ethapierrors.InvalidParamsError{Message: "empty request"}
	}

	state, _, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	result := make(map[common.Address][]hexutil.Bytes, len(requests))
	for addr, keys := range requests {
		vals := make([]hexutil.Bytes, len(keys))
		for i, key := range keys {
			v := state.GetState(addr, key)
			vals[i] = v[:]
		}
		if err := state.Error(); err != nil {
			return nil, err
		}
		result[addr] = vals
	}
	return result, nil
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

type ChainContextBackend = chainctx.Backend
type ChainContext = chainctx.ChainContext

func NewChainContext(ctx context.Context, backend ChainContextBackend) *ChainContext {
	return chainctx.NewChainContext(ctx, backend)
}
func doCall(ctx context.Context, b Backend, args TransactionArgs, state *state.StateDB, header *types.Header, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if blockOverrides != nil {
		if err := blockOverrides.Apply(&blockCtx); err != nil {
			return nil, err
		}
		// Override the header so callers that compute gas price from 1559 fee
		// fields see the overridden basefee. Otherwise GASPRICE/effectiveTip
		// would be derived from the pre-override basefee.
		header = blockOverrides.MakeHeader(header)
	}
	rules := b.ChainConfig().Rules(blockCtx.BlockNumber, blockCtx.Random != nil, blockCtx.Time)
	precompiles := vm.ActivePrecompiledContracts(rules)
	if err := overrides.Apply(state, precompiles); err != nil {
		return nil, err
	}

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	gp := core.NewGasPool(globalGasCap)
	if globalGasCap == 0 {
		gp = core.NewGasPool(gomath.MaxUint64)
	}
	return applyMessage(ctx, b, args, state, header, timeout, gp, &blockCtx, &vm.Config{NoBaseFee: true}, precompiles)
}

func applyMessage(ctx context.Context, b Backend, args TransactionArgs, state *state.StateDB, header *types.Header, timeout time.Duration, gp *core.GasPool, blockContext *vm.BlockContext, vmConfig *vm.Config, precompiles vm.PrecompiledContracts) (*core.ExecutionResult, error) {
	// Get a new instance of the EVM.
	if err := args.CallDefaults(gp.Gas(), blockContext.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	msg := args.ToMessage(header.BaseFee, true)
	// Lower the basefee to 0 to avoid breaking EVM
	// invariants (basefee < feecap).
	if msg.GasPrice.Sign() == 0 {
		blockContext.BaseFee = new(big.Int)
	}
	if msg.BlobGasFeeCap != nil && msg.BlobGasFeeCap.BitLen() == 0 {
		blockContext.BlobBaseFee = new(big.Int)
	}
	evm := b.GetEVM(ctx, state, header, vmConfig, blockContext)
	defer evm.Release()
	if precompiles != nil {
		evm.SetPrecompiles(precompiles)
	}
	res, err := applyMessageWithEVM(ctx, evm, msg, timeout, gp)
	// If an internal state error occurred, let that have precedence. Otherwise,
	// a "trie root missing" type of error will masquerade as e.g. "insufficient gas"
	if err := state.Error(); err != nil {
		return nil, err
	}
	return res, err
}

func applyMessageWithEVM(ctx context.Context, evm *vm.EVM, msg *core.Message, timeout time.Duration, gp *core.GasPool) (*core.ExecutionResult, error) {
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Execute the message.
	result, err := core.ApplyMessage(evm, msg, gp)

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}
	return result, nil
}

func DoCall(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	return doCall(ctx, b, args, state, header, overrides, blockOverrides, timeout, globalGasCap)
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
	result, err := DoCall(ctx, api.b, args, *blockNrOrHash, overrides, blockOverrides, api.b.RPCEVMTimeout(), api.b.RPCGasCap())
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

// DoEstimateGas returns the lowest possible gas limit that allows the transaction to run
// successfully at block `blockNrOrHash`. It returns error if the transaction would revert, or if
// there are unexpected failures. The gas limit is capped by both `args.Gas` (if non-nil &
// non-zero) and `gasCap` (if non-zero).
func DoEstimateGas(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, gasCap uint64) (hexutil.Uint64, error) {
	// Retrieve the base state and mutate it with any overrides
	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return 0, err
	}
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if blockOverrides != nil {
		if err := blockOverrides.Apply(&blockCtx); err != nil {
			return 0, err
		}
		header = blockOverrides.MakeHeader(header)
	}
	rules := b.ChainConfig().Rules(blockCtx.BlockNumber, blockCtx.Random != nil, blockCtx.Time)
	precompiles := vm.ActivePrecompiledContracts(rules)
	if err := overrides.Apply(state, precompiles); err != nil {
		return 0, err
	}
	// Construct the gas estimator option from the user input
	var blobBaseFee *big.Int
	if blockOverrides != nil && blockOverrides.BlobBaseFee != nil {
		blobBaseFee = blockOverrides.BlobBaseFee.ToInt()
	}
	opts := &gasestimator.Options{
		Config:      b.ChainConfig(),
		Chain:       NewChainContext(ctx, b),
		Header:      header,
		State:       state,
		BlobBaseFee: blobBaseFee,
		ErrorRatio:  estimateGasErrorRatio,
	}
	// Set any required transaction default, but make sure the gas cap itself is not messed with
	// if it was not specified in the original argument list.
	if args.Gas == nil {
		args.Gas = new(hexutil.Uint64)
	}
	if err := args.CallDefaults(gasCap, header.BaseFee, b.ChainConfig().ChainID); err != nil {
		return 0, err
	}
	call := args.ToMessage(header.BaseFee, true)

	// Run the gas estimation and wrap any revertals into a custom return
	estimate, revert, err := gasestimator.Estimate(ctx, call, opts, gasCap)
	if err != nil {
		if errors.Is(err, vm.ErrExecutionReverted) {
			return 0, ethapierrors.NewRevertError(revert)
		}
		return 0, err
	}
	return hexutil.Uint64(estimate), nil
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
	return DoEstimateGas(ctx, api.b, args, bNrOrHash, overrides, blockOverrides, api.b.RPCGasCap())
}

// RPCMarshalHeader converts the given header to the RPC output .
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}
	if head.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(head.BaseFee)
	}
	if head.WithdrawalsHash != nil {
		result["withdrawalsRoot"] = head.WithdrawalsHash
	}
	if head.BlobGasUsed != nil {
		result["blobGasUsed"] = hexutil.Uint64(*head.BlobGasUsed)
	}
	if head.ExcessBlobGas != nil {
		result["excessBlobGas"] = hexutil.Uint64(*head.ExcessBlobGas)
	}
	if head.ParentBeaconRoot != nil {
		result["parentBeaconBlockRoot"] = head.ParentBeaconRoot
	}
	if head.RequestsHash != nil {
		result["requestsHash"] = head.RequestsHash
	}
	if head.SlotNumber != nil {
		result["slotNumber"] = hexutil.Uint64(*head.SlotNumber)
	}
	return result
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
	if err = args.setFeeDefaults(ctx, b, header); err != nil {
		return nil, 0, nil, err
	}
	if args.Nonce == nil {
		nonce := hexutil.Uint64(db.GetNonce(args.from()))
		args.Nonce = &nonce
	}
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if err = args.CallDefaults(b.RPCGasCap(), blockCtx.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, 0, nil, err
	}

	var to common.Address
	if args.To != nil {
		to = *args.To
	} else {
		to = crypto.CreateAddress(args.from(), uint64(*args.Nonce))
	}
	isPostMerge := header.Difficulty.Sign() == 0
	// Retrieve the precompiles since they don't need to be added to the access list
	precompiles := vm.ActivePrecompiles(b.ChainConfig().Rules(header.Number, isPostMerge, header.Time))

	// addressesToExclude contains sender, receiver, precompiles and valid authorizations
	addressesToExclude := map[common.Address]struct{}{args.from(): {}, to: {}}
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
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
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
	account := accounts.Account{Address: args.from()}

	wallet, err := api.b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}

	if args.Nonce == nil {
		// Hold the mutex around signing to prevent concurrent assignment of
		// the same nonce to multiple accounts.
		api.nonceLock.LockAddr(args.from())
		defer api.nonceLock.UnlockAddr(args.from())
	}
	if args.IsEIP4844() {
		return common.Hash{}, errBlobTxNotSupported
	}

	// Set some sanity defaults and terminate on failure
	if err := args.setDefaults(ctx, api.b, sidecarConfig{}); err != nil {
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
	if err := args.setDefaults(ctx, api.b, config); err != nil {
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
	if err := args.setDefaults(ctx, api.b, config); err != nil {
		return nil, err
	}
	// Before actually sign the transaction, ensure the transaction fee is reasonable.
	tx := args.ToTransaction(types.DynamicFeeTxType)
	if err := checkTxFee(tx.GasPrice(), tx.Gas(), api.b.RPCTxFeeCap()); err != nil {
		return nil, err
	}
	signed, err := api.sign(args.from(), tx)
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
	if err := sendArgs.setDefaults(ctx, api.b, sidecarConfig{}); err != nil {
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
	if err := checkTxFee(price, gas, api.b.RPCTxFeeCap()); err != nil {
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
		if err == nil && pFrom == sendArgs.from() && api.signer.Hash(p) == wantSigHash {
			// Match. Re-sign and send the transaction.
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			signedTx, err := api.sign(sendArgs.from(), sendArgs.ToTransaction(types.DynamicFeeTxType))
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

// DebugAPI is the collection of SilaChain APIs exposed over the debugging
// namespace.
type DebugAPI struct {
	b Backend
}

// NewDebugAPI creates a new instance of DebugAPI.
func NewDebugAPI(b Backend) *DebugAPI {
	return &DebugAPI{b: b}
}

// GetRawHeader retrieves the RLP encoding for a single header.
func (api *DebugAPI) GetRawHeader(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	header, _ := api.b.HeaderByHash(ctx, hash)
	if header == nil {
		return nil, fmt.Errorf("header #%d not found", hash)
	}
	return rlp.EncodeToBytes(header)
}

// GetRawBlock retrieves the RLP encoded for a single block.
func (api *DebugAPI) GetRawBlock(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	block, _ := api.b.BlockByHash(ctx, hash)
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", hash)
	}
	return rlp.EncodeToBytes(block)
}

// GetRawReceipts retrieves the binary-encoded receipts of a single block.
func (api *DebugAPI) GetRawReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]hexutil.Bytes, error) {
	var hash common.Hash
	if h, ok := blockNrOrHash.Hash(); ok {
		hash = h
	} else {
		block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, err
		}
		hash = block.Hash()
	}
	receipts, err := api.b.GetReceipts(ctx, hash)
	if err != nil {
		return nil, err
	}
	result := make([]hexutil.Bytes, len(receipts))
	for i, receipt := range receipts {
		b, err := receipt.MarshalBinary()
		if err != nil {
			return nil, err
		}
		result[i] = b
	}
	return result, nil
}

// GetRawTransaction returns the bytes of the transaction for the given hash.
func (api *DebugAPI) GetRawTransaction(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	// Retrieve a finalized transaction, or a pooled otherwise
	found, tx, _, _, _ := api.b.GetCanonicalTransaction(hash)
	if !found {
		if tx = api.b.GetPoolTransaction(hash); tx != nil {
			return tx.MarshalBinary()
		}
		// If also not in the pool there is a chance the tx indexer is still in progress.
		if !api.b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		// Transaction is not found in the pool and the indexer is done.
		return nil, nil
	}
	return tx.MarshalBinary()
}

// PrintBlock retrieves a block and returns its pretty printed form.
func (api *DebugAPI) PrintBlock(ctx context.Context, number uint64) (string, error) {
	block, _ := api.b.BlockByNumber(ctx, rpc.BlockNumber(number))
	if block == nil {
		return "", fmt.Errorf("block #%d not found", number)
	}
	return spew.Sdump(block), nil
}

// ChaindbProperty returns leveldb properties of the key-value database.
func (api *DebugAPI) ChaindbProperty() (string, error) {
	return api.b.ChainDb().Stat()
}

// ChaindbCompact flattens the entire key-value database into a single level,
// removing all unused slots and merging all keys.
func (api *DebugAPI) ChaindbCompact() error {
	cstart := time.Now()
	for b := 0; b <= 255; b++ {
		var (
			start = []byte{byte(b)}
			end   = []byte{byte(b + 1)}
		)
		if b == 255 {
			end = nil
		}
		log.Info("Compacting database", "range", fmt.Sprintf("%#X-%#X", start, end), "elapsed", common.PrettyDuration(time.Since(cstart)))
		if err := api.b.ChainDb().Compact(start, end); err != nil {
			log.Error("Database compaction failed", "err", err)
			return err
		}
	}
	return nil
}

// SetHead rewinds the head of the blockchain to a previous block.
func (api *DebugAPI) SetHead(number hexutil.Uint64) error {
	header := api.b.CurrentHeader()
	if header == nil {
		return errors.New("current header is not available")
	}
	if header.Number.Uint64() <= uint64(number) {
		return errors.New("not allowed to rewind to a future block")
	}
	api.b.SetHead(uint64(number))
	return nil
}

// NetAPI offers network related RPC methods
type NetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewNetAPI creates a new net API instance.
func NewNetAPI(net *p2p.Server, networkVersion uint64) *NetAPI {
	return &NetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (api *NetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (api *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(api.net.PeerCount())
}

// Version returns the current legacy-compatible network protocol version.
func (api *NetAPI) Version() string {
	return fmt.Sprintf("%d", api.networkVersion)
}

// checkTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the cap).
func checkTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if cap == 0 {
		return nil
	}
	feeEth := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas))), new(big.Float).SetInt(big.NewInt(params.Ether)))
	feeFloat, _ := feeEth.Float64()
	if feeFloat > cap {
		return fmt.Errorf("tx fee (%.2f Sila) exceeds the configured cap (%.2f Sila)", feeFloat, cap)
	}
	return nil
}
