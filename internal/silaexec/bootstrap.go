// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// BuildExecutionStack loads the shared execution/node bootstrap layer.
func BuildExecutionStack(
	ctx *cli.Context,
	configFile string,
) (*node.Node, ExecutionConfig) {
	cfg := LoadBaseConfig(
		ctx,
		configFile,
		ApplyNodeConfig,
	)

	stack := NewNodeOrFatal(&cfg.Node)

	if err := SetAccountManagerBackends(
		stack.Config(),
		stack.AccountManager(),
		stack.KeyStoreDir(),
	); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)

	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}

	ApplyMetricConfig(ctx, &cfg)

	return stack, cfg
}
