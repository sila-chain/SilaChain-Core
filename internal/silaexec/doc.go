// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

// Package silaexec owns the shared Sila execution runtime wiring.
//
// This package provides the execution/runtime boundary used by cmd/sila.
// The CLI/bootstrap layer remains in cmd/silacli, while protocol assembly,
// backend registration, engine API wiring, metrics, telemetry, dev mode
// and node startup execution belong here.
package silaexec
