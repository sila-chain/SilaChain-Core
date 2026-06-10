// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// This file is part of the SilaChain library.

package main

import (
	"time"

	"fmt"
	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/beacon/blsync"
	bparams "github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/common"
	sila "github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/eth/catalyst"
	"github.com/sila-org/sila/eth/downloader"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/eth/filters"
	"github.com/sila-org/sila/ethclient"
	silabackend "github.com/sila-org/sila/internal/silaapi/backend"
	"github.com/sila-org/sila/internal/telemetry/tracesetup"
	"github.com/sila-org/sila/internal/version"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/rpc"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/console/prompt"
	"github.com/sila-org/sila/internal/debug"
	"github.com/sila-org/sila/internal/flags"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
	"go.uber.org/automaxprocs/maxprocs"
)

var (
	nodeFlags = slices.Concat([]cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag,
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideOsaka,
		utils.OverrideBPO1,
		utils.OverrideBPO2,
		utils.OverrideUBT,
		utils.OverrideGenesisFlag,
		utils.EnablePersonal,
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.BlobPoolDataDirFlag,
		utils.BlobPoolDataCapFlag,
		utils.BlobPoolPriceBumpFlag,
		utils.SyncModeFlag,
		utils.SyncTargetFlag,
		utils.ExitWhenSyncedFlag,
		utils.GCModeFlag,
		utils.SnapshotFlag,
		utils.TxLookupLimitFlag,
		utils.TransactionHistoryFlag,
		utils.ChainHistoryFlag,
		utils.LogHistoryFlag,
		utils.LogNoHistoryFlag,
		utils.LogExportCheckpointsFlag,
		utils.StateHistoryFlag,
		utils.TrienodeHistoryFlag,
		utils.TrienodeHistoryFullValueCheckpointFlag,
		utils.BinTrieGroupDepthFlag,
		utils.LightKDFFlag,
		utils.SilaRequiredBlocksFlag,
		utils.LegacyWhitelistFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheTrieJournalFlag,
		utils.CacheTrieRejournalFlag,
		utils.CacheGCFlag,
		utils.CacheSnapshotFlag,
		utils.CacheNoPrefetchFlag,
		utils.CachePreimagesFlag,
		utils.CacheLogSizeFlag,
		utils.FDLimitFlag,
		utils.CryptoKZGFlag,
		utils.ListenPortFlag,
		utils.DiscoveryPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.MiningEnabledFlag,
		utils.MinerGasLimitFlag,
		utils.MinerGasPriceFlag,
		utils.MinerPendingFeeRecipientFlag,
		utils.MinerExtraDataFlag,
		utils.MinerMaxBlobsFlag,
		utils.MinerRecommitIntervalFlag,
		utils.MinerEtherbaseFlag,
		utils.MinerNewPayloadTimeoutFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV4Flag,
		utils.DiscoveryV5Flag,
		utils.LegacyDiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.DeveloperFlag,
		utils.DeveloperGasLimitFlag,
		utils.DeveloperPeriodFlag,
		utils.SilaDevBeaconFlag,
		utils.VMEnableDebugFlag,
		utils.VMTraceFlag,
		utils.VMTraceJsonConfigFlag,
		utils.VMWitnessStatsFlag,
		utils.VMStatelessSelfValidationFlag,
		utils.NetworkIdFlag,
		utils.SilaStatsURLFlag,
		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,
		utils.GpoMaxGasPriceFlag,
		utils.GpoIgnoreGasPriceFlag,
		configFileFlag,
		utils.LogDebugFlag,
		utils.LogBacktraceAtFlag,
		utils.BeaconApiFlag,
		utils.BeaconApiHeaderFlag,
		utils.BeaconThresholdFlag,
		utils.BeaconNoFilterFlag,
		utils.BeaconConfigFlag,
		utils.BeaconGenesisRootFlag,
		utils.BeaconGenesisTimeFlag,
		utils.BeaconCheckpointFlag,
		utils.BeaconCheckpointFileFlag,
		utils.LogSlowBlockFlag,
	}, utils.NetworkFlags, utils.DatabaseFlags)

	rpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPCORSDomainFlag,
		utils.AuthListenFlag,
		utils.AuthPortFlag,
		utils.AuthVirtualHostsFlag,
		utils.JWTSecretFlag,
		utils.HTTPVirtualHostsFlag,
		utils.GraphQLEnabledFlag,
		utils.GraphQLCORSDomainFlag,
		utils.GraphQLVirtualHostsFlag,
		utils.HTTPApiFlag,
		utils.HTTPPathPrefixFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.WSPathPrefixFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.InsecureUnlockAllowedFlag,
		utils.RPCGlobalGasCapFlag,
		utils.RPCGlobalEVMTimeoutFlag,
		utils.RPCGlobalTxFeeCapFlag,
		utils.RPCGlobalLogQueryLimit,
		utils.AllowUnprotectedTxs,
		utils.BatchRequestLimit,
		utils.BatchResponseMaxSize,
		utils.RPCTxSyncDefaultTimeoutFlag,
		utils.RPCTxSyncMaxTimeoutFlag,
		utils.RPCGlobalRangeLimitFlag,
		utils.RPCTelemetryFlag,
		utils.RPCTelemetryEndpointFlag,
		utils.RPCTelemetryUserFlag,
		utils.RPCTelemetryPasswordFlag,
		utils.RPCTelemetryInstanceIDFlag,
		utils.RPCTelemetryTagsFlag,
		utils.RPCTelemetrySampleRatioFlag,
	}
)
var metricsFlags = []cli.Flag{
	utils.MetricsEnabledFlag,
	utils.MetricsEnabledExpensiveFlag,
	utils.MetricsHTTPFlag,
	utils.MetricsPortFlag,
	utils.MetricsEnableInfluxDBFlag,
	utils.MetricsInfluxDBEndpointFlag,
	utils.MetricsInfluxDBDatabaseFlag,
	utils.MetricsInfluxDBUsernameFlag,
	utils.MetricsInfluxDBPasswordFlag,
	utils.MetricsInfluxDBTagsFlag,
	utils.MetricsInfluxDBIntervalFlag,
	utils.MetricsEnableInfluxDBV2Flag,
	utils.MetricsInfluxDBTokenFlag,
	utils.MetricsInfluxDBBucketFlag,
	utils.MetricsInfluxDBOrganizationFlag,
	utils.StateSizeTrackingFlag,
}

type silaAppConfig struct {
	Usage            string
	EnvPrefix        string
	ClientIdentifier string
}

var defaultSilaAppConfig = silaAppConfig{
	Usage:            "the SilaChain command line interface",
	EnvPrefix:        "SILA",
	ClientIdentifier: "sila",
}

var app = newConfiguredSilaApp(defaultSilaAppConfig)

func newSilaApp(cfg silaAppConfig) *cli.App {
	return flags.NewApp(cfg.Usage)
}

func newConfiguredSilaApp(cfg silaAppConfig) *cli.App {
	app := newSilaApp(cfg)
	initSilaApp(app, cfg)
	return app
}

func initSilaApp(app *cli.App, cfg silaAppConfig) {
	SetClientIdentifier(cfg.ClientIdentifier)

	app.Action = runSilaCommand
	app.Commands = []*cli.Command{
		initCommand,
		importCommand,
		exportCommand,
		importHistoryCommand,
		exportHistoryCommand,
		importPreimagesCommand,
		removedbCommand,
		dumpCommand,
		dumpGenesisCommand,
		pruneHistoryCommand,
		downloadEraCommand,
		accountCommand,
		walletCommand,
		versionCommand,
		licenseCommand,
		dumpConfigCommand,
		dbCommand,
		consoleCommand,
		attachCommand,
		javascriptCommand,
		snapshotCommand,
		bintrieCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = slices.Concat(
		nodeFlags,
		rpcFlags,
		consoleFlags,
		debug.Flags,
		metricsFlags,
	)

	flags.AutoEnvVars(app.Flags, cfg.EnvPrefix)

	app.Before = func(ctx *cli.Context) error {
		maxprocs.Set()
		flags.MigrateGlobalFlags(ctx)
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		flags.CheckEnvVars(ctx, app.Flags, cfg.EnvPrefix)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return prompt.Stdin.Close()
	}
}

func runSilaCommand(ctx *cli.Context) error {
	if args := ctx.Args().Slice(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	prepare(ctx)
	stack := makeFullNode(ctx)
	startNode(ctx, stack, false)
	stack.Wait()
	return nil
}

func prepare(ctx *cli.Context) {
	Prepare(ctx)
}

func startNode(ctx *cli.Context, stack *node.Node, isConsole bool) {
	StartExecutionNode(ctx, stack, isConsole)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

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

var clientIdentifier = "sila"

func SetClientIdentifier(name string) {
	clientIdentifier = name
}

func ClientIdentifier() string {
	return clientIdentifier
}

func DefaultNodeConfig() node.Config {
	git, _ := version.VCS()

	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = version.WithCommit(git.Commit, git.Date)
	cfg.IPCPath = clientIdentifier + ".ipc"

	return cfg
}

// SilaAPIBackend is the Sila execution API backend exposed by cmd/sila.
type SilaAPIBackend = sila.SilaAPIBackend

// RegisterExecutionService registers the Sila execution service.
func RegisterExecutionService(stack *node.Node, cfg *ethconfig.Config) (*SilaAPIBackend, *sila.SilaChain) {
	return utils.RegisterSilaService(stack, cfg)
}

// RegisterSyncOverrideService configures synchronization override service.
func RegisterSyncOverrideService(stack *node.Node, silaBackend *sila.SilaChain, target common.Hash, exitWhenSynced bool) {
	utils.RegisterSyncOverrideService(stack, silaBackend, target, exitWhenSynced)
}

// RegisterEngineAPI launches the Sila Engine API for interacting with an external consensus client.
func RegisterEngineAPI(stack *node.Node, silaBackend *sila.SilaChain) error {
	return catalyst.Register(stack, silaBackend)
}

// ConfigureConsensusRuntime configures the execution consensus runtime.
func ConfigureConsensusRuntime(
	stack *node.Node,
	silaBackend *sila.SilaChain,
	devMode bool,
	devPeriod uint64,
	pendingFeeRecipient common.Address,
	beaconMode bool,
	beaconConfig bparams.ClientConfig,
) error {
	if devMode {
		simBeacon, err := catalyst.NewSimulatedBeacon(devPeriod, pendingFeeRecipient, silaBackend)
		if err != nil {
			return err
		}
		catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)
		return nil
	}

	if beaconMode {
		srv := rpc.NewServer()
		srv.RegisterName("silaEngine", catalyst.NewConsensusAPI(silaBackend))

		blsyncer := blsync.NewClient(beaconConfig)
		blsyncer.SetEngineRPC(rpc.DialInProc(srv))

		stack.RegisterLifecycle(blsyncer)
		return nil
	}

	return RegisterEngineAPI(stack, silaBackend)
}

// RegisterBuildInfoGauge creates gauge with SilaChain system and build information.
func RegisterBuildInfoGauge(silaBackend *sila.SilaChain, version string) {
	if silaBackend == nil {
		return
	}
	var protos []string
	for _, p := range silaBackend.Protocols() {
		protos = append(protos, fmt.Sprintf("%v/%d", p.Name, p.Version))
	}
	metrics.NewRegisteredGaugeInfo("sila/info", nil).Update(metrics.GaugeInfoValue{
		"arch":      runtime.GOARCH,
		"os":        runtime.GOOS,
		"version":   version,
		"protocols": strings.Join(protos, ","),
	})
}

// RegisterFilterAPI configures the log filter RPC API.
func RegisterFilterAPI(stack *node.Node, backend silabackend.Backend, cfg *ethconfig.Config) *filters.FilterSystem {
	return utils.RegisterFilterAPI(stack, backend, cfg)
}

// RegisterGraphQLService configures GraphQL if requested.
func RegisterGraphQLService(stack *node.Node, backend silabackend.Backend, filterSystem *filters.FilterSystem, cfg *node.Config) {
	utils.RegisterGraphQLService(stack, backend, filterSystem, cfg)
}

// RegisterSilaStatsService adds the Sila stats daemon if requested.
func RegisterSilaStatsService(stack *node.Node, backend *SilaAPIBackend, url string) {
	utils.RegisterSilaStatsService(stack, backend, url)
}

// StartExecutionNode starts the node and shared execution runtime services.
func StartExecutionNode(ctx *cli.Context, stack *node.Node, isConsole bool) {
	utils.StartNode(ctx, stack, isConsole)

	if ctx.IsSet(utils.UnlockedAccountFlag.Name) {
		log.Warn(`The "unlock" flag has been deprecated and has no effect`)
	}

	startWalletLifecycle(stack)

	if ctx.Bool(utils.ExitWhenSyncedFlag.Name) {
		startSyncExitLifecycle(stack)
	}
}

func startWalletLifecycle(stack *node.Node) {
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	rpcClient := stack.Attach()
	ethClient := ethclient.NewSilaClient(rpcClient)

	go func() {
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}

		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}

			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()

				log.Info(
					"New wallet appeared",
					"url", event.Wallet.URL(),
					"status", status,
				)

				var derivationPaths []accounts.DerivationPath

				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(
						derivationPaths,
						accounts.SilaLegacyLedgerBaseDerivationPath,
					)
				}

				derivationPaths = append(
					derivationPaths,
					accounts.SilaBaseDerivationPath,
				)

				event.Wallet.SelfDerive(
					derivationPaths,
					ethClient,
				)

			case accounts.WalletDropped:
				log.Info(
					"Old wallet dropped",
					"url", event.Wallet.URL(),
				)

				event.Wallet.Close()
			}
		}
	}()
}

func startSyncExitLifecycle(stack *node.Node) {
	go func() {
		sub := stack.EventMux().Subscribe(downloader.DoneEvent{})
		defer sub.Unsubscribe()

		for {
			event := <-sub.Chan()

			if event == nil {
				continue
			}

			done, ok := event.Data.(downloader.DoneEvent)
			if !ok {
				continue
			}

			timestamp := time.Unix(int64(done.Latest.Time), 0)

			if time.Since(timestamp) < 10*time.Minute {
				log.Info(
					"Synchronisation completed",
					"latestnum", done.Latest.Number,
					"latesthash", done.Latest.Hash(),
					"age", common.PrettyAge(timestamp),
				)

				stack.Close()
			}
		}
	}()
}

// SetupTelemetry sets up OpenTelemetry reporting if enabled.
func SetupTelemetry(cfg node.OpenTelemetryConfig, stack *node.Node) error {
	return tracesetup.SetupTelemetry(cfg, stack)
}

// SyncTargetFromContext parses the optional sync target hash from CLI context.
func SyncTargetFromContext(ctx *cli.Context) (common.Hash, error) {
	var synctarget common.Hash
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		target := ctx.String(utils.SyncTargetFlag.Name)
		if !common.IsHexHash(target) {
			return common.Hash{}, fmt.Errorf("sync target hash is not a valid hex hash: %s", target)
		}
		synctarget = common.HexToHash(target)
	}
	return synctarget, nil
}
