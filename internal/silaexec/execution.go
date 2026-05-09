// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/beacon/blsync"
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/eth/catalyst"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/eth/filters"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/node"
	"github.com/sila-org/sila/rpc"
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

// NewSimulatedBeacon creates the dev-mode simulated beacon.
var NewSimulatedBeacon = catalyst.NewSimulatedBeacon

// RegisterSimulatedBeaconAPIs registers dev-mode simulated beacon APIs.
var RegisterSimulatedBeaconAPIs = catalyst.RegisterSimulatedBeaconAPIs

// NewConsensusAPI creates the engine consensus API.
var NewConsensusAPI = catalyst.NewConsensusAPI

// NewBeaconLightClient creates the beacon light sync client.
var NewBeaconLightClient = blsync.NewClient

// DialInProc creates an in-process RPC client.
var DialInProc = rpc.DialInProc

// NewRPCServer creates a new RPC server.
var NewRPCServer = rpc.NewServer
