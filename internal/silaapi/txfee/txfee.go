// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package txfee

import (
	"context"
	"errors"
	"fmt"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/consensus/misc/eip4844"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/internal/silaapi/txargs"
	"math/big"

	"github.com/sila-org/sila/params"
)

// CheckTxFee checks whether the fee of the given transaction is reasonable under the cap.
func CheckTxFee(gasPrice *big.Int, gas uint64, cap float64) error {
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

type FeeBackend interface {
	ChainConfig() *params.ChainConfig
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
}

// SetFeeDefaults fills in default fee values for unspecified tx fields.
func SetFeeDefaults(args *txargs.TransactionArgs, ctx context.Context, b FeeBackend, head *types.Header) error {
	if args.BlobFeeCap != nil && args.BlobFeeCap.ToInt().Sign() == 0 {
		return errors.New("maxFeePerBlobGas, if specified, must be non-zero")
	}
	if b.ChainConfig().IsCancun(head.Number, head.Time) {
		setCancunFeeDefaults(args, b.ChainConfig(), head)
	}
	if args.GasPrice != nil && (args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil) {
		return errors.New("both gasPrice and (maxFeePerGas or maxPriorityFeePerGas) specified")
	}

	eip1559ParamsSet := args.MaxFeePerGas != nil && args.MaxPriorityFeePerGas != nil
	if args.GasPrice == nil && eip1559ParamsSet {
		if args.MaxFeePerGas.ToInt().Sign() == 0 {
			return errors.New("maxFeePerGas must be non-zero")
		}
		if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
			return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
		}
		return nil
	}

	isLondon := b.ChainConfig().IsLondon(head.Number)
	if args.GasPrice != nil && !eip1559ParamsSet {
		if args.GasPrice.ToInt().Sign() == 0 && isLondon {
			return errors.New("gasPrice must be non-zero after london fork")
		}
		return nil
	}

	if isLondon {
		if err := setLondonFeeDefaults(args, ctx, head, b); err != nil {
			return err
		}
	} else {
		if args.MaxFeePerGas != nil || args.MaxPriorityFeePerGas != nil {
			return errors.New("maxFeePerGas and maxPriorityFeePerGas are not valid before London is active")
		}
		price, err := b.SuggestGasTipCap(ctx)
		if err != nil {
			return err
		}
		args.GasPrice = (*hexutil.Big)(price)
	}
	return nil
}

func setCancunFeeDefaults(args *txargs.TransactionArgs, config *params.ChainConfig, head *types.Header) {
	if args.BlobHashes != nil && args.BlobFeeCap == nil {
		blobBaseFee := eip4844.CalcBlobFee(config, head)
		val := new(big.Int).Mul(blobBaseFee, big.NewInt(2))
		args.BlobFeeCap = (*hexutil.Big)(val)
	}
}

func setLondonFeeDefaults(args *txargs.TransactionArgs, ctx context.Context, head *types.Header, b FeeBackend) error {
	if args.MaxPriorityFeePerGas == nil {
		tip, err := b.SuggestGasTipCap(ctx)
		if err != nil {
			return err
		}
		args.MaxPriorityFeePerGas = (*hexutil.Big)(tip)
	}
	if args.MaxFeePerGas == nil {
		val := new(big.Int).Add(
			args.MaxPriorityFeePerGas.ToInt(),
			new(big.Int).Mul(head.BaseFee, big.NewInt(2)),
		)
		args.MaxFeePerGas = (*hexutil.Big)(val)
	}
	if args.MaxFeePerGas.ToInt().Cmp(args.MaxPriorityFeePerGas.ToInt()) < 0 {
		return fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", args.MaxFeePerGas, args.MaxPriorityFeePerGas)
	}
	return nil
}
