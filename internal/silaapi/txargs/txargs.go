// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package txargs

import (
	"math/big"

	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/types"
	ethapi "github.com/sila-org/sila/internal/ethapi"
)

type TransactionArgs = ethapi.TransactionArgs

func CallDefaults(args *TransactionArgs, globalGasCap uint64, baseFee *big.Int, chainID *big.Int) error {
	return args.CallDefaults(globalGasCap, baseFee, chainID)
}

func ToMessage(args *TransactionArgs, baseFee *big.Int, skipNonceCheck bool) *core.Message {
	return args.ToMessage(baseFee, skipNonceCheck)
}

func ToTransaction(args *TransactionArgs, defaultType int) *types.Transaction {
	return args.ToTransaction(defaultType)
}
