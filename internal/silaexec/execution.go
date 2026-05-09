// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	bparams "github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/eth/catalyst"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/eth/filters"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/node"
)

// RegisterExecutionService registers the Sila execution service.
func RegisterExecutionService(stack *node.Node, cfg *ethconfig.Config) (*eth.EthAPIBackend, *eth.Ethereum) {
	return utils.RegisterEthService(stack, cfg)
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
		simBeacon, err := NewSimulatedBeacon(devPeriod, pendingFeeRecipient, ethBackend)
		if err != nil {
			return err
		}
		RegisterSimulatedBeaconAPIs(stack, simBeacon)
		stack.RegisterLifecycle(simBeacon)
		return nil
	}

	if beaconMode {
		srv := NewRPCServer()
		srv.RegisterName("engine", NewConsensusAPI(ethBackend))

		blsyncer := NewBeaconLightClient(beaconConfig)
		blsyncer.SetEngineRPC(DialInProc(srv))

		stack.RegisterLifecycle(blsyncer)
		return nil
	}

	return RegisterEngineAPI(stack, ethBackend)
}
