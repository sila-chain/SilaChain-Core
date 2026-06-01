// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package txfee

import (
	"fmt"
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
