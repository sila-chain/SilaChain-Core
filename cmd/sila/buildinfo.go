// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sila-org/sila/eth"
	"github.com/sila-org/sila/metrics"
)

// RegisterBuildInfoGauge creates gauge with SilaChain system and build information.
func RegisterBuildInfoGauge(ethBackend *eth.Ethereum, version string) {
	if ethBackend == nil {
		return
	}
	var protos []string
	for _, p := range ethBackend.Protocols() {
		protos = append(protos, fmt.Sprintf("%v/%d", p.Name, p.Version))
	}
	metrics.NewRegisteredGaugeInfo("sila/info", nil).Update(metrics.GaugeInfoValue{
		"arch":      runtime.GOARCH,
		"os":        runtime.GOOS,
		"version":   version,
		"protocols": strings.Join(protos, ","),
	})
}
