// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/cmd/silacli"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// EthstatsConfig represents ethstats connectivity configuration.
type EthstatsConfig struct {
	URL string `toml:",omitempty"`
}

// ExecutionConfig represents the shared execution runtime configuration.
type ExecutionConfig struct {
	Eth      ethconfig.Config
	Node     node.Config
	Ethstats EthstatsConfig
	Metrics  metrics.Config
}

// DefaultExecutionConfig returns the shared execution defaults.
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		Eth:     ethconfig.Defaults,
		Node:    silacli.DefaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}
}

// LoadBaseConfig loads the shared execution configuration.
func LoadBaseConfig(
	ctx *cli.Context,
	configFile string,
	applyNode func(*cli.Context, *node.Config),
) ExecutionConfig {
	cfg := DefaultExecutionConfig()

	LoadConfigOrFatal(configFile, &cfg)

	applyNode(ctx, &cfg.Node)

	return cfg
}

// ApplyNodeConfig applies node configuration defaults.
var ApplyNodeConfig = utils.SetNodeConfig

// NewNodeOrFatal creates a node or exits on failure.
func NewNodeOrFatal(cfg *node.Config) *node.Node {
	stack, err := node.New(cfg)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	return stack
}

// Prepare prepares the shared runtime context.
func Prepare(ctx *cli.Context) {
	switch {
	case ctx.IsSet(utils.SepoliaFlag.Name):
		log.Info("Starting Sila on Sepolia testnet...")

	case ctx.IsSet(utils.HoleskyFlag.Name):
		log.Info("Starting Sila on Holesky testnet...")

	case ctx.IsSet(utils.HoodiFlag.Name):
		log.Info("Starting Sila on Hoodi testnet...")

	case !ctx.IsSet(utils.NetworkIdFlag.Name):
		log.Info("Starting Sila on Sila mainnet...")
	}
}
