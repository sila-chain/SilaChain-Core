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
	"fmt"

	"github.com/sila-org/sila/common"
)

// HistoryMode configures history pruning.
type HistoryMode uint32

const (
	// KeepAll (default) means that all chain history down to genesis block will be kept.
	KeepAll HistoryMode = iota

	// KeepPostMerge sets the history pruning point to the merge activation block.
	KeepPostMerge

	// KeepPostPrague sets the history pruning point to the Prague (Pectra) activation block.
	KeepPostPrague
)

func (m HistoryMode) IsValid() bool {
	return m <= KeepPostPrague
}

func (m HistoryMode) String() string {
	switch m {
	case KeepAll:
		return "all"
	case KeepPostMerge:
		return "postmerge"
	case KeepPostPrague:
		return "postprague"
	default:
		return fmt.Sprintf("invalid HistoryMode(%d)", m)
	}
}

// MarshalText implements encoding.TextMarshaler.
func (m HistoryMode) MarshalText() ([]byte, error) {
	if m.IsValid() {
		return []byte(m.String()), nil
	}
	return nil, fmt.Errorf("unknown history mode %d", m)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (m *HistoryMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "all":
		*m = KeepAll
	case "postmerge":
		*m = KeepPostMerge
	case "postprague":
		*m = KeepPostPrague
	default:
		return fmt.Errorf(`unknown history mode %q, want "all", "postmerge", or "postprague"`, text)
	}
	return nil
}

// PrunePoint identifies a specific block for history pruning.
type PrunePoint struct {
	BlockNumber uint64
	BlockHash   common.Hash
}

// staticPrunePoints contains the pre-defined history pruning cutoff blocks for
// known networks, keyed by history mode and genesis hash. They point to the first
// block after the respective fork. Any pruning should truncate *up to* but
// excluding the given block.
var staticPrunePoints = map[HistoryMode]map[common.Hash]*PrunePoint{
	KeepPostMerge:  {},
	KeepPostPrague: {},
}

// HistoryPolicy describes the configured history pruning strategy. It captures
// user intent as opposed to the actual DB state.
type HistoryPolicy struct {
	Mode HistoryMode
	// Static prune point for PostMerge/PostPrague, nil otherwise.
	Target *PrunePoint
}

// NewPolicy constructs a HistoryPolicy from the given mode and genesis hash.
func NewPolicy(mode HistoryMode, genesisHash common.Hash) (HistoryPolicy, error) {
	switch mode {
	case KeepAll:
		return HistoryPolicy{Mode: KeepAll}, nil

	case KeepPostMerge, KeepPostPrague:
		point := staticPrunePoints[mode][genesisHash]
		if point == nil {
			return HistoryPolicy{}, fmt.Errorf("%s history pruning not available for network %s", mode, genesisHash.Hex())
		}
		return HistoryPolicy{Mode: mode, Target: point}, nil

	default:
		return HistoryPolicy{}, fmt.Errorf("invalid history mode: %d", mode)
	}
}

// PrunedHistoryError is returned by APIs when the requested history is pruned.
type PrunedHistoryError struct{}

func (e *PrunedHistoryError) Error() string  { return "pruned history unavailable" }
func (e *PrunedHistoryError) ErrorCode() int { return 4444 }
