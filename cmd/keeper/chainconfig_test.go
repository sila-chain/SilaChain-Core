package main

import (
	"testing"

	"github.com/sila-org/sila/params"
)

func TestGetChainConfigSupportsSilaPublicTestnet(t *testing.T) {
	cfg, err := getChainConfig(params.SilaPublicTestnetChainConfig.ChainID.Uint64())
	if err != nil {
		t.Fatalf("getChainConfig returned error for Sila public testnet: %v", err)
	}
	if cfg != params.SilaPublicTestnetChainConfig {
		t.Fatal("getChainConfig did not return SilaPublicTestnetChainConfig")
	}
}

func TestGetChainConfigKeepsSilaMainnetDefault(t *testing.T) {
	cfg, err := getChainConfig(0)
	if err != nil {
		t.Fatalf("getChainConfig returned error for default Sila mainnet: %v", err)
	}
	if cfg != params.SilaMainnetChainConfig {
		t.Fatal("getChainConfig default must remain SilaMainnetChainConfig")
	}
}
