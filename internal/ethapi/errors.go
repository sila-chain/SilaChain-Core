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
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
)

// NewTxIndexingError creates a TxIndexingError instance.
func NewTxIndexingError() *ethapierrors.TxIndexingError {
	return ethapierrors.NewTxIndexingError()
}

const (
	errCodeNonceTooHigh            = -38011
	errCodeNonceTooLow             = -38010
	errCodeIntrinsicGas            = -38013
	errCodeInsufficientFunds       = -38014
	errCodeBlockGasLimitReached    = -38015
	errCodeBlockNumberInvalid      = -38020
	errCodeBlockTimestampInvalid   = -38021
	errCodeSenderIsNotEOA          = -38024
	errCodeMaxInitCodeSizeExceeded = -38025
	errCodeClientLimitExceeded     = -38026
	errCodeInternalError           = -32603
	errCodeInvalidParams           = -32602
	errCodeTxSyncTimeout           = 4
)
