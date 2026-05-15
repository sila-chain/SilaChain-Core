// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/eth"
	ethconfig "github.com/sila-org/sila/eth/ethconfig"
	"github.com/sila-org/sila/eth/filters"
	ethapi "github.com/sila-org/sila/internal/ethapi"
	"github.com/sila-org/sila/node"
)

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
