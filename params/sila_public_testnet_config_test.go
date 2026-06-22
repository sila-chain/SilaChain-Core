package params

import (
	"math/big"
	"testing"
)

func TestSilaPublicTestnetChainConfig(t *testing.T) {
	if SilaPublicTestnetChainConfig == nil {
		t.Fatal("SilaPublicTestnetChainConfig is nil")
	}
	if got, want := SilaPublicTestnetChainConfig.ChainID, big.NewInt(20263001); got.Cmp(want) != 0 {
		t.Fatalf("unexpected Sila public testnet ChainID: got %s want %s", got, want)
	}
	if SilaPublicTestnetChainConfig.ChainID.Cmp(big.NewInt(20262026)) == 0 {
		t.Fatal("Sila public testnet ChainID must not use reserved Sila mainnet ChainID")
	}
	if SilaPublicTestnetChainConfig.ChainID.Cmp(SilaMainnetChainConfig.ChainID) == 0 {
		t.Fatal("Sila public testnet ChainID must not equal local/dev Sila ChainID")
	}
	if got, want := NetworkNames[SilaPublicTestnetChainConfig.ChainID.String()], "sila-public-testnet"; got != want {
		t.Fatalf("unexpected network name: got %q want %q", got, want)
	}
}

func TestSilaMainnetChainConfigRemainsLocalOnly2026(t *testing.T) {
	if got, want := SilaMainnetChainConfig.ChainID, big.NewInt(2026); got.Cmp(want) != 0 {
		t.Fatalf("unexpected Sila local/mainnet ChainID: got %s want %s", got, want)
	}
}
