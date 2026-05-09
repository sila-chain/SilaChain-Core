// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import "github.com/sila-org/sila/cmd/silacli"

// ConfigTOMLSettings exposes the shared TOML config settings boundary.
var ConfigTOMLSettings = silacli.ConfigTOMLSettings

// ApplyNodeConfig applies node configuration defaults to the execution config.
var ApplyNodeConfig = silacli.ApplyNodeConfig

// LoadBaseConfig loads the shared base execution configuration.
var LoadBaseConfig = silacli.LoadBaseConfig

// NewNodeOrFatal creates a node from config or exits on failure.
var NewNodeOrFatal = silacli.NewNodeOrFatal

// ExecutionConfig represents the shared execution runtime configuration
// boundary used by Sila execution clients.
type ExecutionConfig = silacli.ExecutionConfig
