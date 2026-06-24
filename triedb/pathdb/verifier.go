// Copyright 2020 The sila Authors
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

package pathdb

import (
	"errors"
	"fmt"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/rawdb"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/siladb"
	"github.com/sila-org/sila/trie"
	"github.com/sila-org/sila/triedb/internal"
)

// VerifyState traverses the flat states specified by the given state root and
// ensures they are matched with each other.
func (db *Database) VerifyState(root common.Hash) error {
	acctIt, err := db.AccountIterator(root, common.Hash{})
	if err != nil {
		return err // The required snapshot might not exist.
	}
	defer acctIt.Release()

	got, err := internal.GenerateTrieRoot(nil, "", acctIt, common.Hash{}, stackTrieHasher, func(_ siladb.KeyValueWriter, accountHash, codeHash common.Hash, stat *internal.GenerateStats) (common.Hash, error) {
		// Migrate the code first, commit the contract code into the tmp db.
		if codeHash != types.EmptyCodeHash {
			code := rawdb.ReadCode(db.diskdb, codeHash)
			if len(code) == 0 {
				return common.Hash{}, errors.New("failed to read contract code")
			}
		}
		// Then migrate all storage trie nodes into the tmp db.
		storageIt, err := db.StorageIterator(root, accountHash, common.Hash{})
		if err != nil {
			return common.Hash{}, err
		}
		defer storageIt.Release()

		hash, err := internal.GenerateTrieRoot(nil, "", storageIt, accountHash, stackTrieHasher, nil, stat, false, nil)
		if err != nil {
			return common.Hash{}, err
		}
		return hash, nil
	}, internal.NewGenerateStats(), true, nil)

	if err != nil {
		return err
	}
	if got != root {
		return fmt.Errorf("state root hash mismatch: got %x, want %x", got, root)
	}
	return nil
}

func stackTrieHasher(_ siladb.KeyValueWriter, _ string, _ common.Hash, in chan internal.TrieKV, out chan common.Hash) {
	t := trie.NewStackTrie(nil)
	for leaf := range in {
		t.Update(leaf.Key[:], leaf.Value)
	}
	out <- t.Hash()
}
