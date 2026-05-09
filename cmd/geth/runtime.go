// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.
package main

import (
	"github.com/sila-org/sila/cmd/silacli"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

func prepare(ctx *cli.Context) {
	silacli.Prepare(ctx)
}

func runSilaNode(ctx *cli.Context, isConsole bool) error {
	return silacli.RunRuntime(ctx, silacli.RuntimeHooks{
		Prepare:      prepare,
		MakeFullNode: func(ctx *cli.Context) silacli.NodeLifecycle { return makeFullNode(ctx) },
		StartNode: func(ctx *cli.Context, stack silacli.NodeLifecycle, isConsole bool) {
			startNode(ctx, stack.(*node.Node), isConsole)
		},
	}, isConsole)
}
