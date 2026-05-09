// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/internal/flags"
	"github.com/urfave/cli/v2"
)

func ConfigureEnv(app *cli.App, cfg AppConfig) {
	flags.AutoEnvVars(app.Flags, cfg.EnvPrefix)
}
