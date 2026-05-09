// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"
	"os"

	"github.com/sila-org/sila/cmd/silacli"
	"github.com/urfave/cli/v2"
)

func main() {
	app := silacli.NewApp(silacli.SilaAppConfig)
	silacli.ConfigureEnv(app, silacli.SilaAppConfig)
	app.Before = func(ctx *cli.Context) error {
		return silacli.Before(ctx, app, silacli.SilaAppConfig)
	}
	app.After = func(ctx *cli.Context) error {
		return silacli.After(ctx, func() error { return nil })
	}
	app.Action = runSilaCommand
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
