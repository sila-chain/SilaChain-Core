// Copyright 2026 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

package history

import (
	"testing"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/params"
)

func TestNewPolicy(t *testing.T) {
	// KeepAll: no target.
	p, err := NewPolicy(KeepAll, params.SilaMainnetGenesisHash)
	if err != nil {
		t.Fatalf("KeepAll: %v", err)
	}
	if p.Mode != KeepAll || p.Target != nil {
		t.Errorf("KeepAll: unexpected policy %+v", p)
	}

	// PostMerge: resolves known mainnet prune point.
	p, err = NewPolicy(KeepPostMerge, params.SilaMainnetGenesisHash)
	if err != nil {
		t.Fatalf("PostMerge: %v", err)
	}
	if p.Target == nil || p.Target.BlockNumber != 15537393 {
		t.Errorf("PostMerge: unexpected target %+v", p.Target)
	}

	// PostPrague: resolves known mainnet prune point.
	p, err = NewPolicy(KeepPostPrague, params.SilaMainnetGenesisHash)
	if err != nil {
		t.Fatalf("PostPrague: %v", err)
	}
	if p.Target == nil || p.Target.BlockNumber != 22431084 {
		t.Errorf("PostPrague: unexpected target %+v", p.Target)
	}

	// PostMerge on unknown network: error.
	if _, err = NewPolicy(KeepPostMerge, common.HexToHash("0xdeadbeef")); err == nil {
		t.Fatal("PostMerge unknown network: expected error")
	}
}
