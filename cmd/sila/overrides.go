// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"github.com/sila-org/sila/cmd/utils"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/urfave/cli/v2"
)

// ApplyProtocolOverrides applies execution protocol override flags.
func ApplyProtocolOverrides(ctx *cli.Context, cfg *ethconfig.Config) {
	if ctx.IsSet(utils.OverrideOsaka.Name) {
		v := ctx.Uint64(utils.OverrideOsaka.Name)
		cfg.OverrideOsaka = &v
	}
	if ctx.IsSet(utils.OverrideBPO1.Name) {
		v := ctx.Uint64(utils.OverrideBPO1.Name)
		cfg.OverrideBPO1 = &v
	}
	if ctx.IsSet(utils.OverrideBPO2.Name) {
		v := ctx.Uint64(utils.OverrideBPO2.Name)
		cfg.OverrideBPO2 = &v
	}
	if ctx.IsSet(utils.OverrideUBT.Name) {
		v := ctx.Uint64(utils.OverrideUBT.Name)
		cfg.OverrideUBT = &v
	}
}
