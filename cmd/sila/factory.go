// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"github.com/sila-org/sila/internal/silaexec"
	"github.com/urfave/cli/v2"
)

// MakeExecutionNode exposes the shared execution node factory boundary.
func MakeExecutionNode(ctx *cli.Context) silaexec.NodeLifecycle {
	return makeFullNode(ctx)
}
