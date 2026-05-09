// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/beacon/blsync"
	"github.com/sila-org/sila/eth/catalyst"
	"github.com/sila-org/sila/rpc"
)

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
