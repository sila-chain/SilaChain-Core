// Copyright 2025 The sila Authors
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

package catalyst

import "github.com/sila-org/sila/metrics"

var (
	// Number of blobs requested via getBlobsV2
	getBlobsRequestedCounter = metrics.NewRegisteredCounter("silaEngine/getblobs/requested", nil)

	// Number of blobs requested via getBlobsV2 that are present in the blobpool
	getBlobsAvailableCounter = metrics.NewRegisteredCounter("silaEngine/getblobs/available", nil)

	// Number of times getBlobsV2/V3 responded with all blobs
	getBlobsRequestCompleteHit = metrics.NewRegisteredCounter("silaEngine/getblobs/hit", nil)

	// Number of times getBlobsV2/V3 responded with no blobs. V2 will return no
	// blobs if it doesn't have all the blobs (all or nothing).
	getBlobsRequestMiss = metrics.NewRegisteredCounter("silaEngine/getblobs/miss", nil)

	// Number of times getBlobsV3 responded with some, but not all, blobs
	getBlobsRequestPartialHit = metrics.NewRegisteredCounter("silaEngine/getblobs/partial", nil)
)
