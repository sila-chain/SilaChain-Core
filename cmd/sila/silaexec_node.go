// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	bparams "github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

// BuildExecutionNode wires the Sila execution runtime into the node stack.
func BuildExecutionNode(ctx *cli.Context, stack *node.Node, cfg *ExecutionConfig, onDevMode func()) *node.Node {
	ApplyProtocolOverrides(ctx, &cfg.Eth)

	utils.SetupMetrics(&cfg.Metrics)

	if err := SetupTelemetry(cfg.Node.OpenTelemetry, stack); err != nil {
		utils.Fatalf("failed to setup OpenTelemetry: %v", err)
	}

	backend, eth := RegisterExecutionService(stack, &cfg.Eth)

	RegisterBuildInfoGauge(eth, cfg.Node.Version)

	filterSystem := RegisterFilterAPI(stack, backend, &cfg.Eth)

	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}
	if cfg.Ethstats.URL != "" {
		RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
	}

	synctarget, err := SyncTargetFromContext(ctx)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	RegisterSyncOverrideService(stack, eth, synctarget, ctx.Bool(utils.ExitWhenSyncedFlag.Name))

	beaconMode := ctx.IsSet(utils.BeaconApiFlag.Name)
	var beaconConfig bparams.ClientConfig
	if beaconMode {
		beaconConfig = utils.MakeBeaconLightConfig(ctx)
	}

	if err := ConfigureConsensusRuntime(
		stack,
		eth,
		ctx.IsSet(utils.DeveloperFlag.Name),
		ctx.Uint64(utils.DeveloperPeriodFlag.Name),
		cfg.Eth.Miner.PendingFeeRecipient,
		beaconMode,
		beaconConfig,
	); err != nil {
		if ctx.IsSet(utils.DeveloperFlag.Name) {
			utils.Fatalf("failed to register dev mode catalyst service: %v", err)
		}
		utils.Fatalf("failed to register catalyst service: %v", err)
	}

	if ctx.IsSet(utils.DeveloperFlag.Name) && onDevMode != nil {
		onDevMode()
	}

	return stack
}
