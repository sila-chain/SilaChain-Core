// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"

	"github.com/sila-org/sila/internal/silaexec"
	"github.com/urfave/cli/v2"
)

func runSilaRuntime(ctx *cli.Context) error {
	return silaexec.RunRuntime(ctx, silaRuntimeHooks(), false)
}

func silaRuntimeHooks() silaexec.RuntimeHooks {
	return silaexec.RuntimeHooks{
		Prepare:      silaexec.Prepare,
		MakeFullNode: makeSilaFullNode,
	}
}

func makeSilaFullNode(ctx *cli.Context) silaexec.NodeLifecycle {
	panic(fmt.Errorf("Sila runtime execution node wiring is not connected yet"))
}
