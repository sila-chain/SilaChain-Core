// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package errors

import (
	"errors"
	"fmt"
	"github.com/sila-org/sila/core"

	"github.com/sila-org/sila/accounts/abi"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/vm"
)

// RevertError is an API error that encompasses an EVM revert with JSON error
// code and a binary data blob.
type RevertError struct {
	error
	reason string
}

// ErrorCode returns the JSON error code for a revert.
func (e *RevertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *RevertError) ErrorData() interface{} {
	return e.reason
}

// NewRevertError creates a RevertError instance with the provided revert data.
func NewRevertError(revert []byte) *RevertError {
	err := vm.ErrExecutionReverted

	reason, errUnpack := abi.UnpackRevert(revert)
	if errUnpack == nil {
		err = fmt.Errorf("%w: %v", vm.ErrExecutionReverted, reason)
	}

	return &RevertError{
		error:  err,
		reason: hexutil.Encode(revert),
	}
}

type InvalidTxError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *InvalidTxError) Error() string {
	return e.Message
}

func (e *InvalidTxError) ErrorCode() int {
	return e.Code
}

type InvalidParamsError struct{ Message string }

func (e *InvalidParamsError) Error() string  { return e.Message }
func (e *InvalidParamsError) ErrorCode() int { return -32602 }

type ClientLimitExceededError struct{ Message string }

func (e *ClientLimitExceededError) Error() string  { return e.Message }
func (e *ClientLimitExceededError) ErrorCode() int { return -38026 }

type InvalidBlockNumberError struct{ Message string }

func (e *InvalidBlockNumberError) Error() string  { return e.Message }
func (e *InvalidBlockNumberError) ErrorCode() int { return -38020 }

type InvalidBlockTimestampError struct{ Message string }

func (e *InvalidBlockTimestampError) Error() string  { return e.Message }
func (e *InvalidBlockTimestampError) ErrorCode() int { return -38021 }

type BlockGasLimitReachedError struct{ Message string }

func (e *BlockGasLimitReachedError) Error() string  { return e.Message }
func (e *BlockGasLimitReachedError) ErrorCode() int { return -38015 }

// TxIndexingError is an API error that indicates the transaction indexing is not
// fully finished yet with JSON error code and a binary data blob.
type TxIndexingError struct{}

// NewTxIndexingError creates a TxIndexingError instance.
func NewTxIndexingError() *TxIndexingError { return &TxIndexingError{} }

// Error implement error interface, returning the error message.
func (e *TxIndexingError) Error() string {
	return "transaction indexing is in progress"
}

// ErrorCode returns the JSON error code for a revert.
func (e *TxIndexingError) ErrorCode() int {
	return -32000
}

// ErrorData returns the hex encoded revert reason.
func (e *TxIndexingError) ErrorData() interface{} {
	return "transaction indexing is in progress"
}

func TxValidationError(err error) *InvalidTxError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &InvalidTxError{Message: err.Error(), Code: -38011}
	case errors.Is(err, core.ErrNonceTooLow):
		return &InvalidTxError{Message: err.Error(), Code: -38010}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &InvalidTxError{Message: err.Error(), Code: -38024}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &InvalidTxError{Message: err.Error(), Code: -32602}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &InvalidTxError{Message: err.Error(), Code: -32602}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &InvalidTxError{Message: err.Error(), Code: -32602}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &InvalidTxError{Message: err.Error(), Code: -32602}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &InvalidTxError{Message: err.Error(), Code: -38014}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &InvalidTxError{Message: err.Error(), Code: -38013}
	case errors.Is(err, core.ErrInsufficientFundsForTransfer):
		return &InvalidTxError{Message: err.Error(), Code: -38014}
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		return &InvalidTxError{Message: err.Error(), Code: -38025}
	}
	return &InvalidTxError{
		Message: err.Error(),
		Code:    -32603,
	}
}

type TxSyncTimeoutError struct {
	Message string
	Hash    common.Hash
}

func (e *TxSyncTimeoutError) Error() string {
	return e.Message
}

func (e *TxSyncTimeoutError) ErrorCode() int {
	return 4
}

func (e *TxSyncTimeoutError) ErrorData() interface{} {
	return e.Hash.Hex()
}
