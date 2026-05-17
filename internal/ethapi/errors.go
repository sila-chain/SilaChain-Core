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
	"errors"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/vm"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
)

type txSyncTimeoutError struct {
	msg  string
	hash common.Hash
}

// NewTxIndexingError creates a TxIndexingError instance.
func NewTxIndexingError() *ethapierrors.TxIndexingError {
	return ethapierrors.NewTxIndexingError()
}

type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
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
	errCodeVMError                 = -32015
	errCodeTxSyncTimeout           = 4
)

func txValidationError(err error) *ethapierrors.InvalidTxError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeSenderIsNotEOA}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeIntrinsicGas}
	case errors.Is(err, core.ErrInsufficientFundsForTransfer):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		return &ethapierrors.InvalidTxError{Message: err.Error(), Code: errCodeMaxInitCodeSizeExceeded}
	}
	return &ethapierrors.InvalidTxError{
		Message: err.Error(),
		Code:    errCodeInternalError,
	}
}

func (e *txSyncTimeoutError) Error() string {
	return e.msg
}

func (e *txSyncTimeoutError) ErrorCode() int {
	return errCodeTxSyncTimeout
}

func (e *txSyncTimeoutError) ErrorData() interface{} {
	return e.hash.Hex()
}
