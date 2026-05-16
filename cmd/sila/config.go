// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SilaChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the SilaChain library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"reflect"
	"unicode"

	"github.com/naoina/toml"

	"fmt"
	bparams "github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/internal/flags"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/metrics"
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
var tomlSettings = ConfigTOMLSettings

type gethConfig = ExecutionConfig

// makeConfigNode loads the real execution/node wiring layer.
//
// Shared bootstrap/runtime/config helpers belong in the Sila CLI layer.
// Real protocol wiring, account backends and Ethereum-compatible
// execution assembly remain inside the Sila execution runtime boundary.

func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	cfg := LoadBaseConfig(
		ctx,
		ctx.String(configFileFlag.Name),
		utils.SetNodeConfig,
	)
	stack := NewNodeOrFatal(&cfg.Node)
	// Node doesn't by default populate account manager backends
	if err := SetAccountManagerBackends(stack.Config(), stack.AccountManager(), stack.KeyStoreDir()); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.IsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.EthStatsURLFlag.Name)
	}
	ApplyMetricConfig(ctx, &cfg)

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

	return BuildExecutionNode(ctx, stack, &cfg, func() {
		banner := constructDevModeBanner(ctx, cfg)
		for _, line := range strings.Split(banner, "\n") {
			log.Warn(line)
		}
	})
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

// ApplyMetricConfig applies metrics CLI flags to the execution configuration.
func ApplyMetricConfig(ctx *cli.Context, cfg *ExecutionConfig) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		log.Warn("Expensive metrics are collected by default, please remove this flag", "flag", utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.IsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.String(utils.MetricsHTTPFlag.Name)
	}
	if ctx.IsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.Int(utils.MetricsPortFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.String(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.String(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.String(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.String(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.String(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBIntervalFlag.Name) {
		cfg.Metrics.InfluxDBInterval = ctx.Duration(utils.MetricsInfluxDBIntervalFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.String(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.String(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.String(utils.MetricsInfluxDBOrganizationFlag.Name)
	}
	var (
		enableExport   = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
		enableExportV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	)
	if enableExport || enableExportV2 {
		v1FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name)

		v2FlagIsSet := ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) ||
			ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name)

		if enableExport && v2FlagIsSet {
			utils.Fatalf("Flags --%s, --%s, --%s are only available for influxdb-v2", utils.MetricsInfluxDBOrganizationFlag.Name, utils.MetricsInfluxDBTokenFlag.Name, utils.MetricsInfluxDBBucketFlag.Name)
		} else if enableExportV2 && v1FlagIsSet {
			utils.Fatalf("Flags --%s, --%s are only available for influxdb-v1", utils.MetricsInfluxDBUsernameFlag.Name, utils.MetricsInfluxDBPasswordFlag.Name)
		}
	}
}

var deprecatedConfigFields = map[string]bool{
	"ethconfig.Config.EVMInterpreter":          true,
	"ethconfig.Config.EWASMInterpreter":        true,
	"ethconfig.Config.TrieCleanCacheJournal":   true,
	"ethconfig.Config.TrieCleanCacheRejournal": true,
	"ethconfig.Config.LightServ":               true,
	"ethconfig.Config.LightIngress":            true,
	"ethconfig.Config.LightEgress":             true,
	"ethconfig.Config.LightPeers":              true,
	"ethconfig.Config.LightNoPrune":            true,
	"ethconfig.Config.LightNoSyncServe":        true,
}

var ConfigTOMLSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		id := fmt.Sprintf("%s.%s", rt.String(), field)
		if deprecatedConfigFields[id] {
			log.Warn(fmt.Sprintf("Config field '%s' is deprecated and won't have any effect.", id))
			return nil
		}
		var link string
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

func LoadConfig(file string, cfg any) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = ConfigTOMLSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func LoadConfigOrFatal(file string, cfg any) {
	if file == "" {
		return
	}
	if err := LoadConfig(file, cfg); err != nil {
		utils.Fatalf("%v", err)
	}
}

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
		Node:    DefaultNodeConfig(),
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
