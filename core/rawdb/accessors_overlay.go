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

package rawdb

import (
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/siladb"
)

func ReadVerkleTransitionState(db siladb.KeyValueReader, hash common.Hash) ([]byte, error) {
	return db.Get(transitionStateKey(hash))
}

func WriteVerkleTransitionState(db siladb.KeyValueWriter, hash common.Hash, state []byte) error {
	return db.Put(transitionStateKey(hash), state)
}
