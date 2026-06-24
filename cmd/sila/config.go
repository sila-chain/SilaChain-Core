// Copyright 2017 The sila Authors
// This file is part of sila.
//
// sila is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sila is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with sila. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/accounts/external"
	"github.com/sila-org/sila/accounts/keystore"
	"github.com/sila-org/sila/accounts/scwallet"
	"github.com/sila-org/sila/accounts/usbwallet"
	"github.com/sila-org/sila/beacon/blsync"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/crypto"
	"github.com/sila-org/sila/sila/catalyst"
	"github.com/sila-org/sila/sila/silaconfig"
	"github.com/sila-org/sila/sila/syncer"
	"github.com/sila-org/sila/internal/flags"
	"github.com/sila-org/sila/internal/telemetry/tracesetup"
	"github.com/sila-org/sila/internal/version"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
	"github.com/sila-org/sila/rpc"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
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
		Category: flags.SilaCategory,
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
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

var deprecatedConfigFields = map[string]bool{
	"silaconfig.Config.EVMInterpreter":          true,
	"silaconfig.Config.EWASMInterpreter":        true,
	"silaconfig.Config.TrieCleanCacheJournal":   true,
	"silaconfig.Config.TrieCleanCacheRejournal": true,
	"silaconfig.Config.LightServ":               true,
	"silaconfig.Config.LightIngress":            true,
	"silaconfig.Config.LightEgress":             true,
	"silaconfig.Config.LightPeers":              true,
	"silaconfig.Config.LightNoPrune":            true,
	"silaconfig.Config.LightNoSyncServe":        true,
}

type silastatsConfig struct {
	URL string `toml:",omitempty"`
}

type silaConfig struct {
	Sila      silaconfig.Config
	Node     node.Config
	Ethstats silastatsConfig
	Metrics  metrics.Config
}

func loadConfig(file string, cfg *silaConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	git, _ := version.VCS()
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = version.WithCommit(git.Commit, git.Date)
	cfg.HTTPModules = append(cfg.HTTPModules, "sila")
	cfg.WSModules = append(cfg.WSModules, "sila")
	cfg.IPCPath = clientIdentifier + ".ipc"
	return cfg
}

// loadBaseConfig loads the silaConfig based on the given command line
// parameters and config file.
func loadBaseConfig(ctx *cli.Context) silaConfig {
	// Load defaults.
	cfg := silaConfig{
		Sila:     silaconfig.Defaults,
		Node:    defaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}

	// Load config file.
	if file := ctx.String(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	return cfg
}

// makeConfigNode loads sila configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, silaConfig) {
	cfg := loadBaseConfig(ctx)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	// Node doesn't by default populate account manager backends
	if err := setAccountManagerBackends(stack.Config(), stack.AccountManager(), stack.KeyStoreDir()); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetSilaConfig(ctx, stack, &cfg.Sila)
	if ctx.IsSet(utils.SilaStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.String(utils.SilaStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, &cfg)

	return stack, cfg
}

// constructs the disclaimer text block which will be printed in the logs upon
// startup when Sila is running in dev mode.
func constructDevModeBanner(ctx *cli.Context, cfg silaConfig) string {
	devModeBanner := `You are running Sila in --dev mode. Please note the following:

  1. This mode is only intended for fast, iterative development without assumptions on
     security or persistence.
  2. The database is created in memory unless specified otherwise. Therefore, shutting down
     your computer or losing power will wipe your entire block data and chain state for
     your dev environment.
  3. A random, pre-allocated developer account will be available and unlocked as
     sila.coinbase, which can be used for testing. The random dev account is temporary,
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
       0x%x (10^49 SILA)
`, cfg.Sila.Miner.PendingFeeRecipient)
		if cfg.Sila.Miner.PendingFeeRecipient == utils.DeveloperAddr {
			devModeBanner += fmt.Sprintf(`
       Private Key
       ------------------
       0x%x
`, crypto.FromECDSA(utils.DeveloperKey))
		}
	}

	return devModeBanner
}

// makeFullNode loads sila configuration and creates the Sila backend.
func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)
	if ctx.IsSet(utils.OverrideOsaka.Name) {
		v := ctx.Uint64(utils.OverrideOsaka.Name)
		cfg.Sila.OverrideOsaka = &v
	}
	if ctx.IsSet(utils.OverrideBPO1.Name) {
		v := ctx.Uint64(utils.OverrideBPO1.Name)
		cfg.Sila.OverrideBPO1 = &v
	}
	if ctx.IsSet(utils.OverrideBPO2.Name) {
		v := ctx.Uint64(utils.OverrideBPO2.Name)
		cfg.Sila.OverrideBPO2 = &v
	}
	if ctx.IsSet(utils.OverrideUBT.Name) {
		v := ctx.Uint64(utils.OverrideUBT.Name)
		cfg.Sila.OverrideUBT = &v
	}

	// Start metrics export if enabled.
	utils.SetupMetrics(&cfg.Metrics)

	// Setup OpenTelemetry reporting if enabled.
	if err := tracesetup.SetupTelemetry(cfg.Node.OpenTelemetry, stack); err != nil {
		utils.Fatalf("failed to setup OpenTelemetry: %v", err)
	}

	// Add Sila service.
	backend, sila := utils.RegisterEthService(stack, &cfg.Sila)

	// Create gauge with sila system and build information
	if sila != nil { // The 'sila' backend may be nil in light mode
		var protos []string
		for _, p := range sila.Protocols() {
			protos = append(protos, fmt.Sprintf("%v/%d", p.Name, p.Version))
		}
		metrics.NewRegisteredGaugeInfo("sila/info", nil).Update(metrics.GaugeInfoValue{
			"arch":      runtime.GOARCH,
			"os":        runtime.GOOS,
			"version":   cfg.Node.Version,
			"protocols": strings.Join(protos, ","),
		})
	}

	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.Sila)

	// Configure GraphQL if requested.
	if ctx.Bool(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}
	// Add the Sila Stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		utils.RegisterSilaStatsService(stack, backend, cfg.Ethstats.URL)
	}

	// Configure synchronization override service
	syncConfig := syncer.Config{
		ExitWhenSynced: ctx.Bool(utils.ExitWhenSyncedFlag.Name),
	}
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		target := ctx.String(utils.SyncTargetFlag.Name)
		if !common.IsHexHash(target) {
			utils.Fatalf("sync target hash is not a valid hex hash: %s", target)
		}
		syncConfig.TargetBlock = common.HexToHash(target)
	}
	utils.RegisterSyncOverrideService(stack, sila, syncConfig)

	if ctx.Bool(utils.DeveloperFlag.Name) {
		// Start dev mode.
		simBeacon, err := catalyst.NewSimulatedBeacon(ctx.Uint64(utils.DeveloperPeriodFlag.Name), cfg.Sila.Miner.PendingFeeRecipient, sila)
		if err != nil {
			utils.Fatalf("failed to register dev mode catalyst service: %v", err)
		}
		catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)

		banner := constructDevModeBanner(ctx, cfg)
		for _, line := range strings.Split(banner, "\n") {
			log.Warn(line)
		}
	} else if ctx.IsSet(utils.BeaconApiFlag.Name) {
		// Start blsync mode.
		srv := rpc.NewServer()
		srv.RegisterName("silaEngine", catalyst.NewConsensusAPI(sila))
		blsyncer := blsync.NewClient(utils.MakeBeaconLightConfig(ctx))
		blsyncer.SetEngineRPC(rpc.DialInProc(srv))
		stack.RegisterLifecycle(blsyncer)
	} else {
		// Launch the silaEngine API for interacting with external consensus client.
		err := catalyst.Register(stack, sila)
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

	if cfg.Sila.Genesis != nil {
		cfg.Sila.Genesis = nil
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

func applyMetricConfig(ctx *cli.Context, cfg *silaConfig) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
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
	// Sanity-check the commandline flags. It is fine if some unused fields is part
	// of the toml-config, but we expect the commandline to only contain relevant
	// arguments, otherwise it indicates an error.
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

func setAccountManagerBackends(conf *node.Config, am *accounts.Manager, keydir string) error {
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extBackend, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extBackend)
			return nil
		} else {
			return fmt.Errorf("error connecting to external signer: %v", err)
		}
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}
