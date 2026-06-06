// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package callapi

import (
	"context"
	"errors"
	"fmt"
	gomath "math"
	"math/big"
	"time"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/eth/gasestimator"
	"github.com/sila-org/sila/eth/tracers/logger"
	"github.com/sila-org/sila/internal/silaapi/chainctx"
	silaapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/evmexec"
	"github.com/sila-org/sila/internal/silaapi/override"
	"github.com/sila-org/sila/internal/silaapi/txargs"
	"github.com/sila-org/sila/internal/silaapi/txfee"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

type SubmitBackend interface {
	RPCTxFeeCap() float64
	UnprotectedAllowed() bool
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	CurrentBlock() *types.Header
	ChainConfig() *params.ChainConfig
}

// estimateGasErrorRatio is the amount of overestimation eth_estimateGas is
// allowed to produce in order to speed up calculations.
const estimateGasErrorRatio = 0.015

type TransactionArgs = txargs.TransactionArgs
type ChainContextBackend = chainctx.Backend
type ChainContext = chainctx.ChainContext

type CallBackend interface {
	chainctx.Backend
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	ChainConfig() *params.ChainConfig
	GetEVM(ctx context.Context, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM
}

type API struct {
	b AccessListBackend
}

func NewAPI(b AccessListBackend) *API {
	return &API{b: b}
}

func (api *API) Call(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Bytes, error) {
	return Call(ctx, api.b, args, blockNrOrHash, overrides, blockOverrides, api.b.RPCEVMTimeout(), api.b.RPCGasCap())
}

func (api *API) EstimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Uint64, error) {
	return EstimateGas(ctx, api.b, args, blockNrOrHash, overrides, blockOverrides, api.b.RPCGasCap())
}

func (api *API) CreateAccessList(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride) (*AccessListResult, error) {
	return CreateAccessList(ctx, api.b, args, blockNrOrHash, stateOverrides)
}
func NewChainContext(ctx context.Context, backend ChainContextBackend) *ChainContext {
	return chainctx.NewChainContext(ctx, backend)
}

func doCall(ctx context.Context, b CallBackend, args TransactionArgs, state *state.StateDB, header *types.Header, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if blockOverrides != nil {
		if err := blockOverrides.Apply(&blockCtx); err != nil {
			return nil, err
		}
		header = blockOverrides.MakeHeader(header)
	}
	rules := b.ChainConfig().Rules(blockCtx.BlockNumber, blockCtx.Random != nil, blockCtx.Time)
	precompiles := vm.ActivePrecompiledContracts(rules)
	if err := overrides.Apply(state, precompiles); err != nil {
		return nil, err
	}

	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	gp := core.NewGasPool(globalGasCap)
	if globalGasCap == 0 {
		gp = core.NewGasPool(gomath.MaxUint64)
	}
	return applyMessage(ctx, b, args, state, header, timeout, gp, &blockCtx, &vm.Config{NoBaseFee: true}, precompiles)
}

func applyMessage(ctx context.Context, b CallBackend, args TransactionArgs, state *state.StateDB, header *types.Header, timeout time.Duration, gp *core.GasPool, blockContext *vm.BlockContext, vmConfig *vm.Config, precompiles vm.PrecompiledContracts) (*core.ExecutionResult, error) {
	if err := args.CallDefaults(gp.Gas(), blockContext.BaseFee, b.ChainConfig().ChainID); err != nil {
		return nil, err
	}
	msg := args.ToMessage(header.BaseFee, true)
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
	res, err := evmexec.ApplyMessageWithEVM(ctx, evm, msg, timeout, gp)
	if err := state.Error(); err != nil {
		return nil, err
	}
	return res, err
}

func Call(ctx context.Context, b CallBackend, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (hexutil.Bytes, error) {
	if blockNrOrHash == nil {
		latest := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		blockNrOrHash = &latest
	}
	result, err := DoCall(ctx, b, args, *blockNrOrHash, overrides, blockOverrides, timeout, globalGasCap)
	if err != nil {
		return nil, err
	}
	if errors.Is(result.Err, vm.ErrExecutionReverted) {
		return nil, silaapierrors.NewRevertError(result.Revert())
	}
	return result.Return(), result.Err
}
func DoCall(ctx context.Context, b CallBackend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	return doCall(ctx, b, args, state, header, overrides, blockOverrides, timeout, globalGasCap)
}

func EstimateGas(ctx context.Context, b CallBackend, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, gasCap uint64) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return DoEstimateGas(ctx, b, args, bNrOrHash, overrides, blockOverrides, gasCap)
}
func DoEstimateGas(ctx context.Context, b CallBackend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, gasCap uint64) (hexutil.Uint64, error) {
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
	if args.Gas == nil {
		args.Gas = new(hexutil.Uint64)
	}
	if err := args.CallDefaults(gasCap, header.BaseFee, b.ChainConfig().ChainID); err != nil {
		return 0, err
	}
	call := args.ToMessage(header.BaseFee, true)

	estimate, revert, err := gasestimator.Estimate(ctx, call, opts, gasCap)
	if err != nil {
		if errors.Is(err, vm.ErrExecutionReverted) {
			return 0, silaapierrors.NewRevertError(revert)
		}
		return 0, err
	}
	return hexutil.Uint64(estimate), nil
}

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b SubmitBackend, tx *types.Transaction) (common.Hash, error) {
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

type AccessListBackend interface {
	CallBackend
	txfee.FeeBackend
	RPCGasCap() uint64
	RPCEVMTimeout() time.Duration
}

// AccessList creates an access list for the given transaction.
// If the accesslist creation fails an error is returned.
// If the transaction itself fails, an vmErr is returned.
type AccessListResult struct {
	Accesslist *types.AccessList `json:"accessList"`
	Error      string            `json:"error,omitempty"`
	GasUsed    hexutil.Uint64    `json:"gasUsed"`
}

func CreateAccessList(ctx context.Context, b AccessListBackend, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, stateOverrides *override.StateOverride) (*AccessListResult, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	acl, gasUsed, vmerr, err := AccessList(ctx, b, bNrOrHash, args, stateOverrides)
	if err != nil {
		return nil, err
	}
	result := &AccessListResult{Accesslist: &acl, GasUsed: hexutil.Uint64(gasUsed)}
	if vmerr != nil {
		result.Error = vmerr.Error()
	}
	return result, nil
}
func AccessList(ctx context.Context, b AccessListBackend, blockNrOrHash rpc.BlockNumberOrHash, args TransactionArgs, stateOverrides *override.StateOverride) (acl types.AccessList, gasUsed uint64, vmErr error, err error) {
	db, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if db == nil || err != nil {
		return nil, 0, nil, err
	}
	if stateOverrides != nil {
		if err := stateOverrides.Apply(db, nil); err != nil {
			return nil, 0, nil, err
		}
	}
	if err = txfee.SetFeeDefaults(&args, ctx, b, header); err != nil {
		return nil, 0, nil, err
	}
	if args.Nonce == nil {
		nonce := hexutil.Uint64(db.GetNonce(args.FromAddr()))
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
		to = crypto.CreateAddress(args.FromAddr(), uint64(*args.Nonce))
	}
	isPostMerge := header.Difficulty.Sign() == 0
	precompiles := vm.ActivePrecompiles(b.ChainConfig().Rules(header.Number, isPostMerge, header.Time))

	addressesToExclude := map[common.Address]struct{}{args.FromAddr(): {}, to: {}}
	for _, addr := range precompiles {
		addressesToExclude[addr] = struct{}{}
	}

	maxAuthorizations := uint64(*args.Gas) / params.CallNewAccountGas
	if uint64(len(args.AuthorizationList)) > maxAuthorizations {
		return nil, 0, nil, errors.New("insufficient gas to process all authorizations")
	}

	for _, auth := range args.AuthorizationList {
		if (!auth.ChainID.IsZero() && auth.ChainID.CmpBig(b.ChainConfig().ChainID) != 0) || auth.Nonce+1 < auth.Nonce {
			continue
		}
		if authority, err := auth.Authority(); err == nil {
			addressesToExclude[authority] = struct{}{}
		}
	}

	prevTracer := logger.NewAccessListTracer(nil, addressesToExclude)
	if args.AccessList != nil {
		prevTracer = logger.NewAccessListTracer(*args.AccessList, addressesToExclude)
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, nil, err
		}
		accessList := prevTracer.AccessList()
		log.Trace("Creating access list", "input", accessList)

		statedb := db.Copy()
		args.AccessList = &accessList
		msg := args.ToMessage(header.BaseFee, true)

		tracer := logger.NewAccessListTracer(accessList, addressesToExclude)
		config := vm.Config{Tracer: tracer.Hooks(), NoBaseFee: true}
		evm := b.GetEVM(ctx, statedb, header, &config, nil)

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
