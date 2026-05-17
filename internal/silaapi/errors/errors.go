// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package errors

import (
	"fmt"

	"github.com/sila-org/sila/accounts/abi"
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
