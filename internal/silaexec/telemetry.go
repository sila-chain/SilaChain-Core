// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is derived from the go-ethereum library.

package silaexec

import (
	"github.com/sila-org/sila/internal/telemetry/tracesetup"
	"github.com/sila-org/sila/node"
)

// SetupTelemetry sets up OpenTelemetry reporting if enabled.
func SetupTelemetry(cfg node.OpenTelemetryConfig, stack *node.Node) error {
	return tracesetup.SetupTelemetry(cfg, stack)
}
