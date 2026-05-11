// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"github.com/sila-org/sila/internal/silaexec"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

func runSilaRuntime(ctx *cli.Context) error {
	return silaexec.RunRuntime(ctx, silaRuntimeHooks(), false)
}

func silaRuntimeHooks() silaexec.RuntimeHooks {
	return silaexec.RuntimeHooks{
		Prepare: silaexec.Prepare,
		MakeFullNode: func(ctx *cli.Context) silaexec.NodeLifecycle {
			return makeSilaFullNode(ctx)
		},
		StartNode: func(ctx *cli.Context, stack silaexec.NodeLifecycle, isConsole bool) {
			silaexec.StartExecutionNode(ctx, stack.(*node.Node), isConsole)
		},
	}
}

func makeSilaFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := silaexec.BuildExecutionStack(
		ctx,
		"",
	)

	return silaexec.BuildExecutionNode(
		ctx,
		stack,
		&cfg,
		nil,
	)
}
func prepare(ctx *cli.Context) {
	silaexec.Prepare(ctx)
}

func startNode(ctx *cli.Context, stack *node.Node, isConsole bool) {
	silaexec.StartExecutionNode(ctx, stack, isConsole)
}
