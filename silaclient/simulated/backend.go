// Copyright 2023 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

package simulated

import (
	"errors"
	"time"

	silaapi "github.com/sila-org/sila"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/node"
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
	silabackend "github.com/sila-org/sila/sila"
	"github.com/sila-org/sila/sila/catalyst"
	"github.com/sila-org/sila/sila/filters"
	"github.com/sila-org/sila/sila/silaconfig"
	"github.com/sila-org/sila/silaclient"
)

// Client exposes the methods provided by the Sila RPC client.
type Client interface {
	silaapi.BlockNumberReader
	silaapi.ChainReader
	silaapi.ChainStateReader
	silaapi.ContractCaller
	silaapi.GasEstimator
	silaapi.GasPricer
	silaapi.GasPricer1559
	silaapi.FeeHistoryReader
	silaapi.LogFilterer
	silaapi.PendingStateReader
	silaapi.PendingContractCaller
	silaapi.TransactionReader
	silaapi.TransactionSender
	silaapi.ChainIDReader
}

// simClient wraps silaclient. This exists to prevent extracting silaclient.Client
// from the Client interface returned by Backend.
type simClient struct {
	*silaclient.Client
}

// Backend is a simulated blockchain. You can use it to test your contracts or
// other code that interacts with the Sila chain.
type Backend struct {
	node   *node.Node
	beacon *catalyst.SimulatedBeacon
	client simClient
}

// NewBackend creates a new simulated blockchain that can be used as a backend for
// contract bindings in unit tests.
//
// A simulated backend always uses chainID 1337.
func NewBackend(alloc types.GenesisAlloc, options ...func(nodeConf *node.Config, silaConf *silaconfig.Config)) *Backend {
	// Create the default configurations for the outer node shell and the Sila
	// service to mutate with the options afterwards
	nodeConf := node.DefaultConfig
	nodeConf.DataDir = ""
	nodeConf.P2P = p2p.Config{NoDiscovery: true}

	silaConf := silaconfig.Defaults
	silaConf.Genesis = &core.Genesis{
		Config:   params.AllDevChainProtocolChanges,
		GasLimit: silaconfig.Defaults.Miner.GasCeil,
		Alloc:    alloc,
	}
	silaConf.SyncMode = silaconfig.FullSync
	silaConf.TxPool.NoLocals = true
	// Disable log indexing to force unindexed log search
	silaConf.LogNoHistory = true

	for _, option := range options {
		option(&nodeConf, &silaConf)
	}
	// Assemble the Sila stack to run the chain with
	stack, err := node.New(&nodeConf)
	if err != nil {
		panic(err) // this should never happen
	}
	sim, err := newWithNode(stack, &silaConf, 0)
	if err != nil {
		panic(err) // this should never happen
	}
	return sim
}

// newWithNode sets up a simulated backend on an existing node. The provided node
// must not be started and will be started by this method.
func newWithNode(stack *node.Node, conf *silabackend.Config, blockPeriod uint64) (*Backend, error) {
	backend, err := silabackend.New(stack, conf)
	if err != nil {
		return nil, err
	}
	// Register the filter system
	filterSystem := filters.NewFilterSystem(backend.APIBackend, filters.Config{})
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "sila",
		Service:   filters.NewFilterAPI(filterSystem),
	}})
	// Start the node
	if err := stack.Start(); err != nil {
		return nil, err
	}
	// Set up the simulated beacon
	beacon, err := catalyst.NewSimulatedBeacon(blockPeriod, common.Address{}, backend)
	if err != nil {
		return nil, err
	}
	// Reorg our chain back to genesis
	if err := beacon.Fork(backend.BlockChain().GetCanonicalHash(0)); err != nil {
		return nil, err
	}
	return &Backend{
		node:   stack,
		beacon: beacon,
		client: simClient{silaclient.NewClient(stack.Attach())},
	}, nil
}

// Close shuts down the simBackend.
// The simulated backend can't be used afterwards.
func (n *Backend) Close() error {
	if n.client.Client != nil {
		n.client.Close()
		n.client = simClient{}
	}
	var err error
	if n.beacon != nil {
		err = n.beacon.Stop()
		n.beacon = nil
	}
	if n.node != nil {
		err = errors.Join(err, n.node.Close())
		n.node = nil
	}
	return err
}

// Commit seals a block and moves the chain forward to a new empty block.
func (n *Backend) Commit() common.Hash {
	return n.beacon.Commit()
}

// Rollback removes all pending transactions, reverting to the last committed state.
func (n *Backend) Rollback() {
	n.beacon.Rollback()
}

// Fork creates a side-chain that can be used to simulate reorgs.
//
// This function should be called with the ancestor block where the new side
// chain should be started. Transactions (old and new) can then be applied on
// top and Commit-ed.
//
// Note, the side-chain will only become canonical (and trigger the events) when
// it becomes longer. Until then CallContract will still operate on the current
// canonical chain.
//
// There is a % chance that the side chain becomes canonical at the same length
// to simulate live network behavior.
func (n *Backend) Fork(parentHash common.Hash) error {
	return n.beacon.Fork(parentHash)
}

// AdjustTime changes the block timestamp and creates a new block.
// It can only be called on empty blocks.
func (n *Backend) AdjustTime(adjustment time.Duration) error {
	return n.beacon.AdjustTime(adjustment)
}

// Client returns a client that accesses the simulated chain.
func (n *Backend) Client() Client {
	return n.client
}
