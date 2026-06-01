// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package evmexec

import (
	"context"
	"fmt"
	"time"

	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/vm"
)

// ApplyMessageWithEVM executes the message with the given EVM.
func ApplyMessageWithEVM(ctx context.Context, evm *vm.EVM, msg *core.Message, timeout time.Duration, gp *core.GasPool) (*core.ExecutionResult, error) {
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
