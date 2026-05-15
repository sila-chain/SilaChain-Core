// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"fmt"

	"github.com/sila-org/sila/cmd/utils"
	"github.com/sila-org/sila/common"
	"github.com/urfave/cli/v2"
)

// SyncTargetFromContext parses the optional sync target hash from CLI context.
func SyncTargetFromContext(ctx *cli.Context) (common.Hash, error) {
	var synctarget common.Hash
	if ctx.IsSet(utils.SyncTargetFlag.Name) {
		target := ctx.String(utils.SyncTargetFlag.Name)
		if !common.IsHexHash(target) {
			return common.Hash{}, fmt.Errorf("sync target hash is not a valid hex hash: %s", target)
		}
		synctarget = common.HexToHash(target)
	}
	return synctarget, nil
}
