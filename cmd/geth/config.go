// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/internal/flags"
	"github.com/sila-org/sila/internal/silaexec"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
	"os"
	"slices"
	"strings"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Export configuration values in a TOML format",
		ArgsUsage:   "<dumpfile (optional)>",
		Flags:       slices.Concat(nodeFlags, rpcFlags),
		Description: `Export configuration values in TOML format (to stdout by default).`,
	}

	configFileFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "TOML configuration file",
		Category: flags.EthCategory,
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = silaexec.ConfigTOMLSettings

type gethConfig = silaexec.ExecutionConfig

// makeConfigNode loads the real execution/node wiring layer.
//
// Shared bootstrap/runtime/config helpers belong in cmd/silacli.
// Real protocol wiring, account backends and Ethereum-compatible
// execution assembly remain inside cmd/geth.

func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	cfg := silaexec.LoadBaseConfig(
		ctx,
		ctx.String(configFileFlag.Name),
		silaexec.ApplyNodeConfig,
	)
	stack := silaexec.NewNodeOrFatal(&cfg.Node)
	// Node doesn't by default populate account manager backends
	if err := silaexec.SetAccountManagerBackends(stack.Config(), stack.AccountManager(), stack.KeyStoreDir()); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}
	silaexec.ApplyMetricConfig(ctx, &cfg)

	return stack, cfg
}

// constructs the disclaimer text block which will be printed in the logs upon
// startup when Sila is running in dev mode.
func constructDevModeBanner(ctx *cli.Context, cfg gethConfig) string {
	devModeBanner := `You are running Sila in --dev mode. Please note the following:

  1. This mode is only intended for fast, iterative development without assumptions on
     security or persistence.
  2. The database is created in memory unless specified otherwise. Therefore, shutting down
     your computer or losing power will wipe your entire block data and chain state for
     your dev environment.
  3. A random, pre-allocated developer account will be available and unlocked as
     eth.coinbase, which can be used for testing. The random dev account is temporary,
     stored on a ramdisk, and will be lost if your machine is restarted.
  4. Mining is enabled by default. However, the client will only seal blocks if transactions
     are pending in the mempool. The miner's minimum accepted gas price is 1.
  5. Networking is disabled; there is no listen-address, the maximum number of peers is set
     to 0, and discovery is disabled.
`
	if !ctx.IsSet(utils.DataDirFlag.Name) {
		devModeBanner += fmt.Sprintf(`

 Running in ephemeral mode.  The following account has been prefunded in the genesis:

       Account
       ------------------
       0x%x (10^49 ETH)
`, cfg.Eth.Miner.PendingFeeRecipient)
		if cfg.Eth.Miner.PendingFeeRecipient == utils.DeveloperAddr {
			devModeBanner += fmt.Sprintf(`
       Private Key
       ------------------
       0x%x
`, crypto.FromECDSA(utils.DeveloperKey))
		}
	}

	return devModeBanner
}

// makeFullNode loads SilaChain configuration and creates the execution backend.
func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)
	silaexec.ApplyProtocolOverrides(ctx, &cfg.Eth)

	// Start metrics export if enabled.
	silaexec.SetupMetrics(&cfg.Metrics)

	// Setup OpenTelemetry reporting if enabled.
	if err := silaexec.SetupTelemetry(cfg.Node.OpenTelemetry, stack); err != nil {
		utils.Fatalf("failed to setup OpenTelemetry: %v", err)
	}

	// Add Sila execution service.
	backend, eth := silaexec.RegisterExecutionService(stack, &cfg.Eth)

	// Create gauge with SilaChain system and build information
	silaexec.RegisterBuildInfoGauge(eth, cfg.Node.Version)

	// Configure log filter RPC API.
	filterSystem := silaexec.RegisterFilterAPI(stack, backend, &cfg.Eth)

	// Configure GraphQL if requested.
	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		silaexec.RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}
	// Add the Sila stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		silaexec.RegisterEthStatsService(stack, backend, cfg.Ethstats.URL)
	}
	// Configure synchronization override service
	synctarget, err := silaexec.SyncTargetFromContext(ctx)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	silaexec.RegisterSyncOverrideService(stack, eth, synctarget, ctx.Bool(utils.ExitWhenSyncedFlag.Name))

	if ctx.IsSet(utils.DeveloperFlag.Name) {
		// Start dev mode.
		simBeacon, err := silaexec.NewSimulatedBeacon(ctx.Uint64(utils.DeveloperPeriodFlag.Name), cfg.Eth.Miner.PendingFeeRecipient, eth)
		if err != nil {
			utils.Fatalf("failed to register dev mode catalyst service: %v", err)
		}
		silaexec.RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)

		banner := constructDevModeBanner(ctx, cfg)
		for _, line := range strings.Split(banner, "\n") {
			log.Warn(line)
		}
	} else if ctx.IsSet(utils.BeaconApiFlag.Name) {
		// Start blsync mode.
		srv := silaexec.NewRPCServer()
		srv.RegisterName("engine", silaexec.NewConsensusAPI(eth))
		blsyncer := silaexec.NewBeaconLightClient(utils.MakeBeaconLightConfig(ctx))
		blsyncer.SetEngineRPC(silaexec.DialInProc(srv))
		stack.RegisterLifecycle(blsyncer)
	} else {
		// Launch the engine API for interacting with external consensus client.
		err := silaexec.RegisterEngineAPI(stack, eth)
		if err != nil {
			utils.Fatalf("failed to register catalyst service: %v", err)
		}
	}
	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Eth.Genesis != nil {
		cfg.Eth.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}
