// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/internal/debug"
	"github.com/sila-org/sila/internal/flags"
	"github.com/urfave/cli/v2"
	"go.uber.org/automaxprocs/maxprocs"
)

func Before(ctx *cli.Context, app *cli.App, cfg AppConfig) error {
	maxprocs.Set()
	flags.MigrateGlobalFlags(ctx)
	if err := debug.Setup(ctx); err != nil {
		return err
	}
	flags.CheckEnvVars(ctx, app.Flags, cfg.EnvPrefix)
	return nil
}
