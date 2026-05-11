// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"github.com/sila-org/sila/cmd/silacli"
	"github.com/urfave/cli/v2"
)

var app = newConfiguredSilaApp(defaultSilaAppConfig)

func newSilaApp(cfg silaAppConfig) *cli.App {
	return silacli.NewApp(cfg)
}

func newConfiguredSilaApp(cfg silaAppConfig) *cli.App {
	app := newSilaApp(cfg)
	initSilaApp(app, cfg)
	return app
}

func initSilaApp(app *cli.App, cfg silaAppConfig) {
	silacli.SetClientIdentifier(cfg.ClientIdentifier)

	app.Action = runSilaCommand
	app.Commands = []*cli.Command{
		initCommand,
		importCommand,
		exportCommand,
		importHistoryCommand,
		exportHistoryCommand,
		importPreimagesCommand,
		removedbCommand,
		dumpCommand,
		dumpGenesisCommand,
		pruneHistoryCommand,
		downloadEraCommand,
		accountCommand,
		walletCommand,
		versionCommand,
		licenseCommand,
		dumpConfigCommand,
		dbCommand,
	}

	silacli.ConfigureEnv(app, cfg)

	app.Before = func(ctx *cli.Context) error {
		return silacli.Before(ctx, app, cfg)
	}

	app.After = func(ctx *cli.Context) error {
		return silacli.After(ctx, func() error { return nil })
	}
}
