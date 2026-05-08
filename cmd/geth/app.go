// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"slices"
	"sort"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/console/prompt"
	"github.com/sila-org/sila/internal/debug"
	"github.com/sila-org/sila/internal/flags"
	"github.com/urfave/cli/v2"
	"go.uber.org/automaxprocs/maxprocs"
)

var app = newConfiguredSilaApp(defaultSilaAppConfig)

func newSilaApp(cfg silaAppConfig) *cli.App {
	return flags.NewApp(cfg.Usage)
}

func newConfiguredSilaApp(cfg silaAppConfig) *cli.App {
	app := newSilaApp(cfg)
	initSilaApp(app, cfg)
	return app
}

func initSilaApp(app *cli.App, cfg silaAppConfig) {
	setClientIdentifier(cfg.ClientIdentifier)

	// Initialize the CLI app and start SilaChain
	app.Action = geth
	app.Commands = []*cli.Command{
		// See chaincmd.go:
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
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		consoleCommand,
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		versionCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_legacy.go
		utils.ShowDeprecated,
		// See snapshot.go
		snapshotCommand,
		// See bintrie_convert.go
		bintrieCommand,
	}
	if logTestCommand != nil {
		app.Commands = append(app.Commands, logTestCommand)
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = slices.Concat(
		nodeFlags,
		rpcFlags,
		consoleFlags,
		debug.Flags,
		metricsFlags,
	)
	flags.AutoEnvVars(app.Flags, cfg.EnvPrefix)

	app.Before = func(ctx *cli.Context) error {
		maxprocs.Set() // Automatically set GOMAXPROCS to match Linux container CPU quota.
		flags.MigrateGlobalFlags(ctx)
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		flags.CheckEnvVars(ctx, app.Flags, cfg.EnvPrefix)
		return nil
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.
		return nil
	}
}
