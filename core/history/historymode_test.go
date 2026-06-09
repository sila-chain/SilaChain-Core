// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SilaChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the SilaChain library. If not, see <http://www.gnu.org/licenses/>.

package history

import (
	"testing"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/params"
)

func TestEthereumLegacyNewPolicy(t *testing.T) {
	// Ethereum legacy KeepAll remains valid because it does not require a static prune point.
	p, err := NewPolicy(KeepAll, params.MainnetGenesisHash)
	if err != nil {
		t.Fatalf("Ethereum legacy KeepAll: %v", err)
	}
	if p.Mode != KeepAll || p.Target != nil {
		t.Errorf("Ethereum legacy KeepAll: unexpected policy %+v", p)
	}

	if _, err = NewPolicy(KeepPostMerge, params.MainnetGenesisHash); err == nil {
		t.Fatal("Ethereum legacy PostMerge: expected unavailable prune point")
	}
	if _, err = NewPolicy(KeepPostPrague, params.MainnetGenesisHash); err == nil {
		t.Fatal("Ethereum legacy PostPrague: expected unavailable prune point")
	}

	if _, err = NewPolicy(KeepPostMerge, common.HexToHash("0xdeadbeef")); err == nil {
		t.Fatal("PostMerge unknown network: expected error")
	}
}

func TestSilaNewPolicy(t *testing.T) {
	p, err := NewPolicy(KeepAll, params.SilaMainnetGenesisHash)
	if err != nil {
		t.Fatalf("Sila KeepAll: %v", err)
	}
	if p.Mode != KeepAll || p.Target != nil {
		t.Fatalf("Sila KeepAll: unexpected policy: %#v", p)
	}

	if _, err := NewPolicy(KeepPostMerge, params.SilaMainnetGenesisHash); err == nil {
		t.Fatalf("Sila KeepPostMerge: expected unavailable prune point")
	}
	if _, err := NewPolicy(KeepPostPrague, params.SilaMainnetGenesisHash); err == nil {
		t.Fatalf("Sila KeepPostPrague: expected unavailable prune point")
	}
}
