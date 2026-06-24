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

package main

import (
	"fmt"

	"github.com/sila-org/sila/params"
)

// getChainConfig returns the appropriate chain configuration based on the chainID.
// Returns an error for unsupported chain IDs.
func getChainConfig(chainID uint64) (*params.ChainConfig, error) {
	switch chainID {
	case 0, params.SilaMainnetChainConfig.ChainID.Uint64():
		return params.SilaMainnetChainConfig, nil
	case params.SilaPublicTestnetChainConfig.ChainID.Uint64():
		return params.SilaPublicTestnetChainConfig, nil
	case params.SilaDevTestnetChainConfig.ChainID.Uint64():
		return params.SilaDevTestnetChainConfig, nil
	default:
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}
