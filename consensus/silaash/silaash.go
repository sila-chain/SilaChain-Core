// Copyright 2017 The sila Authors
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

// Package silaash implements the silaash proof-of-work consensus silaEngine.
package silaash

import (
	"time"

	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core/types"
)

// Silaash is a consensus silaEngine based on proof-of-work implementing the silaash
// algorithm.
type Silaash struct {
	fakeFail  *uint64        // Block number which fails PoW check even in fake mode
	fakeDelay *time.Duration // Time delay to sleep for before returning from verify
	fakeFull  bool           // Accepts everything as valid
}

// NewFaker creates an silaash consensus silaEngine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Sila
// consensus rules.
func NewFaker() *Silaash {
	return new(Silaash)
}

// NewFakeFailer creates a silaash consensus silaEngine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the Sila consensus rules.
func NewFakeFailer(fail uint64) *Silaash {
	return &Silaash{
		fakeFail: &fail,
	}
}

// NewFakeDelayer creates a silaash consensus silaEngine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the Sila consensus rules.
func NewFakeDelayer(delay time.Duration) *Silaash {
	return &Silaash{
		fakeDelay: &delay,
	}
}

// NewFullFaker creates an silaash consensus silaEngine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *Silaash {
	return &Silaash{
		fakeFull: true,
	}
}

// Close closes the exit channel to notify all backend threads exiting.
func (silaash *Silaash) Close() error {
	return nil
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel. For the silaash silaEngine, this method will
// just panic as sealing is not supported anymore.
func (silaash *Silaash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	panic("silaash (pow) sealing not supported any more")
}
