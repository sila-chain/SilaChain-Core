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

package rawdb

import (
	"encoding/binary"
	"errors"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/siladb"
	"github.com/sila-org/sila/log"
)

// ReadPreimage retrieves a single preimage of the provided hash.
func ReadPreimage(db siladb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(preimageKey(hash))
	if len(data) == 0 {
		preimageMissCounter.Inc(1)
	} else {
		preimageHitsCounter.Inc(1)
	}
	return data
}

// WritePreimages writes the provided set of preimages to the database.
func WritePreimages(db siladb.KeyValueWriter, preimages map[common.Hash][]byte) {
	for hash, preimage := range preimages {
		if err := db.Put(preimageKey(hash), preimage); err != nil {
			log.Crit("Failed to store trie preimage", "err", err)
		}
	}
	preimageCounter.Inc(int64(len(preimages)))
}

// ReadCode retrieves the contract code of the provided code hash.
func ReadCode(db siladb.KeyValueReader, hash common.Hash) []byte {
	// Try with the prefixed code scheme first, if not then try with legacy
	// scheme.
	data := ReadCodeWithPrefix(db, hash)
	if len(data) != 0 {
		return data
	}
	data, _ = db.Get(hash.Bytes())
	return data
}

// ReadCodeWithPrefix retrieves the contract code of the provided code hash.
// The main difference between this function and ReadCode is this function
// will only check the existence with latest scheme(with prefix).
func ReadCodeWithPrefix(db siladb.KeyValueReader, hash common.Hash) []byte {
	data, _ := db.Get(codeKey(hash))
	return data
}

// HasCode checks if the contract code corresponding to the
// provided code hash is present in the db.
func HasCode(db siladb.KeyValueReader, hash common.Hash) bool {
	// Try with the prefixed code scheme first, if not then try with legacy
	// scheme.
	if ok := HasCodeWithPrefix(db, hash); ok {
		return true
	}
	ok, _ := db.Has(hash.Bytes())
	return ok
}

// HasCodeWithPrefix checks if the contract code corresponding to the
// provided code hash is present in the db. This function will only check
// presence using the prefix-scheme.
func HasCodeWithPrefix(db siladb.KeyValueReader, hash common.Hash) bool {
	ok, _ := db.Has(codeKey(hash))
	return ok
}

// WriteCode writes the provided contract code database.
func WriteCode(db siladb.KeyValueWriter, hash common.Hash, code []byte) {
	if err := db.Put(codeKey(hash), code); err != nil {
		log.Crit("Failed to store contract code", "err", err)
	}
}

// DeleteCode deletes the specified contract code from the database.
func DeleteCode(db siladb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(codeKey(hash)); err != nil {
		log.Crit("Failed to delete contract code", "err", err)
	}
}

// ReadStateID retrieves the state id with the provided state root.
func ReadStateID(db siladb.KeyValueReader, root common.Hash) *uint64 {
	data, err := db.Get(stateIDKey(root))
	if err != nil || len(data) == 0 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteStateID writes the provided state lookup to database.
func WriteStateID(db siladb.KeyValueWriter, root common.Hash, id uint64) {
	var buff [8]byte
	binary.BigEndian.PutUint64(buff[:], id)
	if err := db.Put(stateIDKey(root), buff[:]); err != nil {
		log.Crit("Failed to store state ID", "err", err)
	}
}

// ReadPersistentStateID retrieves the id of the persistent state from the database.
func ReadPersistentStateID(db siladb.KeyValueReader) uint64 {
	data, _ := db.Get(persistentStateIDKey)
	if len(data) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(data)
}

// WritePersistentStateID stores the id of the persistent state into database.
func WritePersistentStateID(db siladb.KeyValueWriter, number uint64) {
	if err := db.Put(persistentStateIDKey, encodeBlockNumber(number)); err != nil {
		log.Crit("Failed to store the persistent state ID", "err", err)
	}
}

// ReadTrieJournal retrieves the serialized in-memory trie nodes of layers saved at
// the last shutdown.
func ReadTrieJournal(db siladb.KeyValueReader) []byte {
	data, _ := db.Get(trieJournalKey)
	return data
}

// WriteTrieJournal stores the serialized in-memory trie nodes of layers to save at
// shutdown.
func WriteTrieJournal(db siladb.KeyValueWriter, journal []byte) {
	if err := db.Put(trieJournalKey, journal); err != nil {
		log.Crit("Failed to store tries journal", "err", err)
	}
}

// ReadStateHistoryMeta retrieves the metadata corresponding to the specified
// state history. Compute the position of state history in freezer by minus
// one since the id of first state history starts from one(zero for initial
// state).
func ReadStateHistoryMeta(db siladb.AncientReaderOp, id uint64) []byte {
	blob, err := db.Ancient(stateHistoryMeta, id-1)
	if err != nil {
		return nil
	}
	return blob
}

// ReadStateHistoryMetaList retrieves a batch of meta objects with the specified
// start position and count. Compute the position of state history in freezer by
// minus one since the id of first state history starts from one(zero for initial
// state).
func ReadStateHistoryMetaList(db siladb.AncientReaderOp, start uint64, count uint64) ([][]byte, error) {
	return db.AncientRange(stateHistoryMeta, start-1, count, 0)
}

// ReadStateAccountIndex retrieves the account index blob for the specified
// state history. The index contains fixed-size entries with offsets and lengths
// into the concatenated account data table. Compute the position of state
// history in freezer by minus one since the id of first state history starts
// from one (zero for initial state).
func ReadStateAccountIndex(db siladb.AncientReaderOp, id uint64) []byte {
	blob, err := db.Ancient(stateHistoryAccountIndex, id-1)
	if err != nil {
		return nil
	}
	return blob
}

// ReadStateStorageIndex retrieves the storage index blob for the specified
// state history. The index contains fixed-size entries that locate storage slot
// data in the concatenated storage data table. Compute the position of state
// history in freezer by minus one since the id of first state history starts
// from one (zero for initial state).
func ReadStateStorageIndex(db siladb.AncientReaderOp, id uint64, offset, length int) ([]byte, error) {
	return db.AncientBytes(stateHistoryStorageIndex, id-1, uint64(offset), uint64(length))
}

// ReadStateAccountHistory retrieves the concatenated account data blob for the
// specified state history. Offsets and lengths are resolved via the account
// index. Compute the position of state history in freezer by minus one since
// the id of first state history starts from one (zero for initial state).
func ReadStateAccountHistory(db siladb.AncientReaderOp, id uint64, offset, length int) ([]byte, error) {
	return db.AncientBytes(stateHistoryAccountData, id-1, uint64(offset), uint64(length))
}

// ReadStateStorageHistory retrieves the concatenated storage slot data blob for
// the specified state history. Locations are resolved via the account and
// storage indexes. Compute the position of state history in freezer by minus
// one since the id of first state history starts from one (zero for initial
// state).
func ReadStateStorageHistory(db siladb.AncientReaderOp, id uint64, offset, length int) ([]byte, error) {
	return db.AncientBytes(stateHistoryStorageData, id-1, uint64(offset), uint64(length))
}

// ReadStateHistory retrieves the state history from database with provided id.
// Compute the position of state history in freezer by minus one since the id
// of first state history starts from one(zero for initial state).
func ReadStateHistory(db siladb.AncientReaderOp, id uint64) ([]byte, []byte, []byte, []byte, []byte, error) {
	meta, err := db.Ancient(stateHistoryMeta, id-1)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	accountIndex, err := db.Ancient(stateHistoryAccountIndex, id-1)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	storageIndex, err := db.Ancient(stateHistoryStorageIndex, id-1)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	accountData, err := db.Ancient(stateHistoryAccountData, id-1)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	storageData, err := db.Ancient(stateHistoryStorageData, id-1)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return meta, accountIndex, storageIndex, accountData, storageData, nil
}

// ReadStateHistoryList retrieves a list of state histories from database with
// specific range. Compute the position of state history in freezer by minus one
// since the id of first state history starts from one(zero for initial state).
func ReadStateHistoryList(db siladb.AncientReaderOp, start uint64, count uint64) ([][]byte, [][]byte, [][]byte, [][]byte, [][]byte, error) {
	metaList, err := db.AncientRange(stateHistoryMeta, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	aIndexList, err := db.AncientRange(stateHistoryAccountIndex, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	sIndexList, err := db.AncientRange(stateHistoryStorageIndex, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	aDataList, err := db.AncientRange(stateHistoryAccountData, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	sDataList, err := db.AncientRange(stateHistoryStorageData, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if len(metaList) != len(aIndexList) || len(metaList) != len(sIndexList) || len(metaList) != len(aDataList) || len(metaList) != len(sDataList) {
		return nil, nil, nil, nil, nil, errors.New("state history is corrupted")
	}
	return metaList, aIndexList, sIndexList, aDataList, sDataList, nil
}

// WriteStateHistory writes the provided state history to database. Compute the
// position of state history in freezer by minus one since the id of first state
// history starts from one(zero for initial state).
func WriteStateHistory(db siladb.AncientWriter, id uint64, meta []byte, accountIndex []byte, storageIndex []byte, accounts []byte, storages []byte) error {
	_, err := db.ModifyAncients(func(op siladb.AncientWriteOp) error {
		if err := op.AppendRaw(stateHistoryMeta, id-1, meta); err != nil {
			return err
		}
		if err := op.AppendRaw(stateHistoryAccountIndex, id-1, accountIndex); err != nil {
			return err
		}
		if err := op.AppendRaw(stateHistoryStorageIndex, id-1, storageIndex); err != nil {
			return err
		}
		if err := op.AppendRaw(stateHistoryAccountData, id-1, accounts); err != nil {
			return err
		}
		return op.AppendRaw(stateHistoryStorageData, id-1, storages)
	})
	return err
}

// ReadTrienodeHistory retrieves the trienode history corresponding to the specified id.
// Compute the position of trienode history in freezer by minus one since the id of first
// trienode history starts from one(zero for initial state).
func ReadTrienodeHistory(db siladb.AncientReaderOp, id uint64) ([]byte, []byte, []byte, error) {
	header, err := db.Ancient(trienodeHistoryHeaderTable, id-1)
	if err != nil {
		return nil, nil, nil, err
	}
	keySection, err := db.Ancient(trienodeHistoryKeySectionTable, id-1)
	if err != nil {
		return nil, nil, nil, err
	}
	valueSection, err := db.Ancient(trienodeHistoryValueSectionTable, id-1)
	if err != nil {
		return nil, nil, nil, err
	}
	return header, keySection, valueSection, nil
}

// ReadTrienodeHistoryHeader retrieves the header section of trienode history.
func ReadTrienodeHistoryHeader(db siladb.AncientReaderOp, id uint64) ([]byte, error) {
	return db.Ancient(trienodeHistoryHeaderTable, id-1)
}

// ReadTrienodeHistoryKeySection retrieves the key section of trienode history.
func ReadTrienodeHistoryKeySection(db siladb.AncientReaderOp, id uint64, offset uint64, length uint64) ([]byte, error) {
	return db.AncientBytes(trienodeHistoryKeySectionTable, id-1, offset, length)
}

// ReadTrienodeHistoryValueSection retrieves the value section of trienode history.
func ReadTrienodeHistoryValueSection(db siladb.AncientReaderOp, id uint64, offset uint64, length uint64) ([]byte, error) {
	return db.AncientBytes(trienodeHistoryValueSectionTable, id-1, offset, length)
}

// ReadTrienodeHistoryList retrieves the a list of trienode history corresponding
// to the specified range.
// Compute the position of trienode history in freezer by minus one since the id
// of first trienode history starts from one(zero for initial state).
func ReadTrienodeHistoryList(db siladb.AncientReaderOp, start uint64, count uint64) ([][]byte, [][]byte, [][]byte, error) {
	header, err := db.AncientRange(trienodeHistoryHeaderTable, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	keySection, err := db.AncientRange(trienodeHistoryKeySectionTable, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	valueSection, err := db.AncientRange(trienodeHistoryValueSectionTable, start-1, count, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(header) != len(keySection) || len(header) != len(valueSection) {
		return nil, nil, nil, errors.New("trienode history is corrupted")
	}
	return header, keySection, valueSection, nil
}

// WriteTrienodeHistory writes the provided trienode history to database.
// Compute the position of trienode history in freezer by minus one since
// the id of first state history starts from one(zero for initial state).
func WriteTrienodeHistory(db siladb.AncientWriter, id uint64, header []byte, keySection []byte, valueSection []byte) error {
	_, err := db.ModifyAncients(func(op siladb.AncientWriteOp) error {
		if err := op.AppendRaw(trienodeHistoryHeaderTable, id-1, header); err != nil {
			return err
		}
		if err := op.AppendRaw(trienodeHistoryKeySectionTable, id-1, keySection); err != nil {
			return err
		}
		return op.AppendRaw(trienodeHistoryValueSectionTable, id-1, valueSection)
	})
	return err
}
