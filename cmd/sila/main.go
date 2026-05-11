// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func runSilaCommand(ctx *cli.Context) error {
	return runSilaRuntime(ctx)
}
func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
