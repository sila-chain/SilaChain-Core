// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/internal/version"
	"github.com/sila-org/sila/node"
)

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
	cfg.HTTPModules = append(cfg.HTTPModules, "eth")
	cfg.WSModules = append(cfg.WSModules, "eth")
	cfg.IPCPath = clientIdentifier + ".ipc"

	return cfg
}
