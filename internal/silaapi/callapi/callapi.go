// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package callapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/crypto"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/params"
)

type SubmitBackend interface {
	RPCTxFeeCap() float64
	UnprotectedAllowed() bool
	SendTx(ctx context.Context, signedTx *types.Transaction) error
	CurrentBlock() *types.Header
	ChainConfig() *params.ChainConfig
}

var DoCall = ethapi.DoCall
var DoEstimateGas = ethapi.DoEstimateGas

// SubmitTransaction is a helper function that submits tx to txPool and logs a message.
func SubmitTransaction(ctx context.Context, b SubmitBackend, tx *types.Transaction) (common.Hash, error) {
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
