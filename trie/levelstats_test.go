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

package trie

import "testing"

func TestLevelStatsAddLeafDepthBounds(t *testing.T) {
	stats := NewLevelStats()
	stats.AddLeaf(15)

	if got := stats.LeafDepths()[15]; got != 1 {
		t.Fatalf("leaf count at depth 15 = %d, want 1", got)
	}
}

func TestLevelStatsAddLeafPanicsOnDepth16(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for depth >= 16")
		}
	}()
	NewLevelStats().AddLeaf(16)
}
