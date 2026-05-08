// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package main

type silaAppConfig struct {
	Usage            string
	EnvPrefix        string
	ClientIdentifier string
}

var defaultSilaAppConfig = silaAppConfig{
	Usage:            "the SilaChain command line interface",
	EnvPrefix:        "GETH",
	ClientIdentifier: "sila",
}
