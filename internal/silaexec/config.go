// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/cmd/silacli"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// ExecutionConfig represents the shared execution runtime configuration.
type ExecutionConfig = silacli.ExecutionConfig

// LoadBaseConfig loads the shared execution configuration.
var LoadBaseConfig = silacli.LoadBaseConfig

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
