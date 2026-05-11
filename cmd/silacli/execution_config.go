// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/metrics"
	"github.com/sila-org/sila/node"
)

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
