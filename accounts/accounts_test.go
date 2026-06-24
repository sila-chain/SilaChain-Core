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

package accounts

import (
	"bytes"
	"testing"

	"github.com/sila-org/sila/common/hexutil"
)

func TestTextHash(t *testing.T) {
	t.Parallel()
	hash := TextHash([]byte("Hello Joe"))
	want := hexutil.MustDecode("0x528ff247b5d9e0848333f7e848f747fc456a17ba0f42b10878aaa092ea8a1d1c")
	if !bytes.Equal(hash, want) {
		t.Fatalf("wrong hash: %x", hash)
	}
}
