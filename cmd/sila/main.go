// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/sila-org/sila/cmd/silacli"
	"github.com/sila-org/sila/console/prompt"
	"github.com/sila-org/sila/internal/debug"
	"github.com/urfave/cli/v2"
)

type silaAppConfig = silacli.AppConfig

var defaultSilaAppConfig = silacli.SilaAppConfig
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
		consoleCommand,
		attachCommand,
		javascriptCommand,
		snapshotCommand,
		bintrieCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = slices.Concat(
		nodeFlags,
		rpcFlags,
		consoleFlags,
		debug.Flags,
		metricsFlags,
	)

	silacli.ConfigureEnv(app, cfg)

	app.Before = func(ctx *cli.Context) error {
		return silacli.Before(ctx, app, cfg)
	}

	app.After = func(ctx *cli.Context) error {
		return silacli.After(ctx, prompt.Stdin.Close)
	}
}

func runSilaCommand(ctx *cli.Context) error {
	return runSilaRuntime(ctx)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
