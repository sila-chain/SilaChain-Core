// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

import (
	"fmt"
	"os"

	"github.com/sila-org/sila/cmd/silacli"
)

func main() {
	cfg := silacli.SilaAppConfig
	fmt.Fprintf(os.Stdout, "%s [%s]\n", cfg.Usage, cfg.EnvPrefix)
}
