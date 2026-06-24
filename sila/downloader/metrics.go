// Copyright 2015 The sila Authors
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/sila-org/sila/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("sila/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("sila/downloader/headers/req", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("sila/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("sila/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("sila/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("sila/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("sila/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("sila/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("sila/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("sila/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("sila/downloader/receipts/timeout", nil)

	throttleCounter = metrics.NewRegisteredCounter("sila/downloader/throttle", nil)

	// snapPeerSkipMeter tracks snap peers skipped by the state syncer because
	// they negotiated a version below the one the syncer requires.
	snapPeerSkipMeter = metrics.NewRegisteredMeter("sila/downloader/snap/peerskip", nil)
)
