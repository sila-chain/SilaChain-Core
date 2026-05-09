// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silacli

import (
	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/node"
	"github.com/urfave/cli/v2"
)

func ApplyNodeConfig(ctx *cli.Context, cfg *node.Config) {
	utils.SetNodeConfig(ctx, cfg)
}
