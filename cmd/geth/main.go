// Copyright 2014 The go-ethereum Authors
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

// SilaChain is a command-line execution client derived from Ethereum architecture.
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/internal/silaexec"
	"github.com/sila-org/sila/node"
	// Force-load the tracer engines to trigger registration
	_ "github.com/sila-org/sila/eth/tracers/js"
	_ "github.com/sila-org/sila/eth/tracers/live"
	_ "github.com/sila-org/sila/eth/tracers/native"

	"github.com/urfave/cli/v2"
)

var (
	clientIdentifier = "sila" // Client identifier to advertise over the network
)

func setClientIdentifier(name string) {
	clientIdentifier = name
}

var (
	// flags that configure the node
	nodeFlags = slices.Concat([]cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.MinFreeDiskSpaceFlag,
		utils.KeyStoreDirFlag,
		utils.ExternalSignerFlag,
		utils.NoUSBFlag, // deprecated
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.OverrideOsaka,
		utils.OverrideBPO1,
		utils.OverrideBPO2,
		utils.OverrideUBT,
		utils.OverrideGenesisFlag,
		utils.EnablePersonal, // deprecated
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
		utils.TxLookupLimitFlag, // deprecated
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
		utils.EthRequiredBlocksFlag,
		utils.LegacyWhitelistFlag, // deprecated
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheTrieJournalFlag,   // deprecated
		utils.CacheTrieRejournalFlag, // deprecated
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
		utils.MiningEnabledFlag, // deprecated
		utils.MinerGasLimitFlag,
		utils.MinerGasPriceFlag,
		utils.MinerEtherbaseFlag, // deprecated
		utils.MinerExtraDataFlag,
		utils.MinerMaxBlobsFlag,
		utils.MinerRecommitIntervalFlag,
		utils.MinerPendingFeeRecipientFlag,
		utils.MinerNewPayloadTimeoutFlag, // deprecated
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV4Flag,
		utils.DiscoveryV5Flag,
		utils.LegacyDiscoveryV5Flag, // deprecated
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.DeveloperFlag,
		utils.DeveloperGasLimitFlag,
		utils.DeveloperPeriodFlag,
		utils.VMEnableDebugFlag,
		utils.VMTraceFlag,
		utils.VMTraceJsonConfigFlag,
		utils.VMWitnessStatsFlag,
		utils.VMStatelessSelfValidationFlag,
		utils.NetworkIdFlag,
		utils.EthStatsURLFlag,
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

	metricsFlags = []cli.Flag{
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
)

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// geth is the main entry point into the system if no special subcommand is run.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func geth(ctx *cli.Context) error {
	return runSilaNode(ctx, false)
}

// startNode boots up the system node and all registered protocols, after which
// it starts the RPC/IPC interfaces and the miner.
func startNode(ctx *cli.Context, stack *node.Node, isConsole bool) {
	silaexec.StartExecutionNode(ctx, stack, isConsole)
}
