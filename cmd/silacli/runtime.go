// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.
package silacli

import (
	"fmt"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/log"
	"github.com/urfave/cli/v2"
)

type RuntimeHooks struct {
	Prepare      func(*cli.Context)
	MakeFullNode func(*cli.Context) NodeLifecycle
	StartNode    func(*cli.Context, NodeLifecycle, bool)
}

type NodeLifecycle interface {
	Close() error
	Wait()
}

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

func RunRuntime(ctx *cli.Context, hooks RuntimeHooks, isConsole bool) error {
	if args := ctx.Args().Slice(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	hooks.Prepare(ctx)
	stack := hooks.MakeFullNode(ctx)
	defer stack.Close()

	hooks.StartNode(ctx, stack, isConsole)
	stack.Wait()
	return nil
}
