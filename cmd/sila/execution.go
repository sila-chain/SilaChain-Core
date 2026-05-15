// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sila-org/sila/beacon/blsync"
	bparams "github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/eth/catalyst"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/eth/filters"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
	"github.com/sila-org/sila/rpc"
)

// RegisterExecutionService registers the Sila execution service.
func RegisterExecutionService(stack *node.Node, cfg *ethconfig.Config) (*eth.EthAPIBackend, *eth.Ethereum) {
	return utils.RegisterEthService(stack, cfg)
}

// RegisterSyncOverrideService configures synchronization override service.
func RegisterSyncOverrideService(stack *node.Node, ethBackend *eth.Ethereum, target common.Hash, exitWhenSynced bool) {
	utils.RegisterSyncOverrideService(stack, ethBackend, target, exitWhenSynced)
}

// RegisterEngineAPI launches the engine API for interacting with an external consensus client.
func RegisterEngineAPI(stack *node.Node, ethBackend *eth.Ethereum) error {
	return catalyst.Register(stack, ethBackend)
}

// ConfigureConsensusRuntime configures the execution consensus runtime.
func ConfigureConsensusRuntime(
	stack *node.Node,
	ethBackend *eth.Ethereum,
	devMode bool,
	devPeriod uint64,
	pendingFeeRecipient common.Address,
	beaconMode bool,
	beaconConfig bparams.ClientConfig,
) error {
	if devMode {
		simBeacon, err := catalyst.NewSimulatedBeacon(devPeriod, pendingFeeRecipient, ethBackend)
		if err != nil {
			return err
		}
		catalyst.RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)
		return nil
	}

	if beaconMode {
		srv := rpc.NewServer()
		srv.RegisterName("engine", catalyst.NewConsensusAPI(ethBackend))

		blsyncer := blsync.NewClient(beaconConfig)
		blsyncer.SetEngineRPC(rpc.DialInProc(srv))

		stack.RegisterLifecycle(blsyncer)
		return nil
	}

	return RegisterEngineAPI(stack, ethBackend)
}

// RegisterBuildInfoGauge creates gauge with SilaChain system and build information.
func RegisterBuildInfoGauge(ethBackend *eth.Ethereum, version string) {
	if ethBackend == nil {
		return
	}
	var protos []string
	for _, p := range ethBackend.Protocols() {
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
func RegisterFilterAPI(stack *node.Node, backend ethapi.Backend, cfg *ethconfig.Config) *filters.FilterSystem {
	return utils.RegisterFilterAPI(stack, backend, cfg)
}

// RegisterGraphQLService configures GraphQL if requested.
func RegisterGraphQLService(stack *node.Node, backend ethapi.Backend, filterSystem *filters.FilterSystem, cfg *node.Config) {
	utils.RegisterGraphQLService(stack, backend, filterSystem, cfg)
}

// RegisterEthStatsService adds the Sila stats daemon if requested.
func RegisterEthStatsService(stack *node.Node, backend *eth.EthAPIBackend, url string) {
	utils.RegisterEthStatsService(stack, backend, url)
}
