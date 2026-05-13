// Copyright 2018 The SilaChain Authors
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

package metrics

// CPUStats is the system and process CPU stats.
// All values are in seconds.
type CPUStats struct {
	GlobalTime float64 // Time spent by the CPU working on all processes
	GlobalWait float64 // Time spent by waiting on disk for all processes
	LocalTime  float64 // Time spent by the CPU working on this process
}
