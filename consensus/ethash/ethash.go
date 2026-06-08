// Copyright 2017 The SilaChain Authors
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

// Package ethash implements the legacy proof-of-work consensus engine for SilaPoW compatibility.
package ethash

import (
	"time"

	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core/types"
)

// Ethash is the legacy proof-of-work consensus engine retained for SilaPoW compatibility, implementing the legacy
// algorithm.
type Ethash struct {
	fakeFail  *uint64        // Block number which fails PoW check even in fake mode
	fakeDelay *time.Duration // Time delay to sleep for before returning from verify
	fakeFull  bool           // Accepts everything as valid
}

// SilaPoW is the public Sila compatibility name for the legacy proof-of-work engine.
type SilaPoW = Ethash

// NewFaker creates a SilaPoW compatibility consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the SilaChain
// consensus rules.
func NewFaker() *SilaPoW {
	return new(SilaPoW)
}

// NewFakeFailer creates a SilaPoW compatibility consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the SilaChain consensus rules.
func NewFakeFailer(fail uint64) *SilaPoW {
	return &SilaPoW{
		fakeFail: &fail,
	}
}

// NewFakeDelayer creates a SilaPoW compatibility consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the SilaChain consensus rules.
func NewFakeDelayer(delay time.Duration) *SilaPoW {
	return &SilaPoW{
		fakeDelay: &delay,
	}
}

// NewFullFaker creates a SilaPoW compatibility consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *SilaPoW {
	return &SilaPoW{
		fakeFull: true,
	}
}

// Close closes the exit channel to notify all backend threads exiting.
func (silapow *Ethash) Close() error {
	return nil
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel. For the SilaPoW compatibility engine, this method will
// just panic as sealing is not supported anymore.
func (silapow *Ethash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	panic("SilaPoW compatibility sealing not supported any more")
}
