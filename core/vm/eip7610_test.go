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

package vm

import (
	"fmt"
	"testing"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/params"
)

func Example_ethereumLegacyMainnetEIP7610Accounts() {
	list := eip7610Accounts[params.MainnetChainConfig.ChainID.Uint64()]
	fmt.Println(len(list))
	// Output:
	// 0
}

func Example_silaMainnetEIP7610Accounts() {
	list := eip7610Accounts[params.SilaMainnetChainConfig.ChainID.Uint64()]
	fmt.Println(len(list))
	// Output:
	// 0
}

func TestSilaEIP7610RejectedAccountListIsEmpty(t *testing.T) {
	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	if isEIP7610RejectedAccount(params.SilaMainnetChainConfig.ChainID, addr, true) {
		t.Fatal("Sila mainnet should not reject accounts through EIP-7610 historical list")
	}
	if isEIP7610RejectedAccount(params.SilaMainnetChainConfig.ChainID, addr, false) {
		t.Fatal("Sila mainnet should not reject accounts before EIP-158")
	}
}
