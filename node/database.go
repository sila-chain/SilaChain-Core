// Copyright 2024 The sila Authors
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

package node

import (
	"fmt"

	"github.com/sila-org/sila/core/rawdb"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/siladb"
	"github.com/sila-org/sila/siladb/leveldb"
	"github.com/sila-org/sila/siladb/pebble"
)

// DatabaseOptions contains the options to apply when opening a database.
type DatabaseOptions struct {
	// Directory for storing chain history ("freezer").
	AncientsDirectory string

	// The optional Era folder, which can be either a subfolder under
	// ancient/chain or a directory specified via an absolute path.
	EraDirectory string

	MetricsNamespace string // the namespace for database relevant metrics
	Cache            int    // the capacity(in megabytes) of the data caching
	Handles          int    // number of files to be open simultaneously
	ReadOnly         bool   // if true, no writes can be performed
}

type internalOpenOptions struct {
	directory string
	dbEngine  string // "leveldb" | "pebble"
	DatabaseOptions
}

// openDatabase opens both a disk-based key-value database such as leveldb or pebble, but also
// integrates it with a freezer database -- if the AncientDir option has been
// set on the provided OpenOptions.
// The passed o.AncientDir indicates the path of root ancient directory where
// the chain freezer can be opened.
func openDatabase(o internalOpenOptions) (siladb.Database, error) {
	kvdb, err := openKeyValueDatabase(o)
	if err != nil {
		return nil, err
	}
	opts := rawdb.OpenOptions{
		Ancient:          o.AncientsDirectory,
		Era:              o.EraDirectory,
		MetricsNamespace: o.MetricsNamespace,
		ReadOnly:         o.ReadOnly,
	}
	frdb, err := rawdb.Open(kvdb, opts)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}

// openKeyValueDatabase opens a disk-based key-value database, e.g. leveldb or pebble.
//
//						  type == null          type != null
//					   +----------------------------------------
//	db is non-existent |  pebble default  |  specified type
//	db is existent     |  from db         |  specified type (if compatible)
func openKeyValueDatabase(o internalOpenOptions) (siladb.KeyValueStore, error) {
	// Reject any unsupported database type
	if len(o.dbEngine) != 0 && o.dbEngine != rawdb.DBLeveldb && o.dbEngine != rawdb.DBPebble {
		return nil, fmt.Errorf("unknown db.silaEngine %v", o.dbEngine)
	}
	// Retrieve any pre-existing database's type and use that or the requested one
	// as long as there's no conflict between the two types
	existingDb := rawdb.PreexistingDatabase(o.directory)
	if len(existingDb) != 0 && len(o.dbEngine) != 0 && o.dbEngine != existingDb {
		return nil, fmt.Errorf("db.silaEngine choice was %v but found pre-existing %v database in specified data directory", o.dbEngine, existingDb)
	}
	if o.dbEngine == rawdb.DBPebble || existingDb == rawdb.DBPebble {
		log.Info("Using pebble as the backing database")
		return newPebbleDBDatabase(o.directory, o.Cache, o.Handles, o.MetricsNamespace, o.ReadOnly)
	}
	if o.dbEngine == rawdb.DBLeveldb || existingDb == rawdb.DBLeveldb {
		log.Info("Using leveldb as the backing database")
		return newLevelDBDatabase(o.directory, o.Cache, o.Handles, o.MetricsNamespace, o.ReadOnly)
	}
	// No pre-existing database, no user-requested one either. Default to Pebble.
	log.Info("Defaulting to pebble as the backing database")
	return newPebbleDBDatabase(o.directory, o.Cache, o.Handles, o.MetricsNamespace, o.ReadOnly)
}

// newLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func newLevelDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (siladb.KeyValueStore, error) {
	db, err := leveldb.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	log.Info("Using LevelDB as the backing database")
	return db, nil
}

// newPebbleDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
//
// If the database already exists with a legacy pebble v1 format, it is opened
// using pebble v1 for backward compatibility and a warning is logged directing
// the user to upgrade offline. New databases use pebble v2.
func newPebbleDBDatabase(file string, cache int, handles int, namespace string, readonly bool) (siladb.KeyValueStore, error) {
	if pebble.NeedsV1(file) {
		log.Warn("Pebble database uses legacy v1 format; upgrade offline with 'sila db pebble-upgrade'")
		db, err := pebble.NewV1(file, cache, handles, namespace, readonly)
		if err != nil {
			return nil, err
		}
		return db, nil
	}
	db, err := pebble.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	return db, nil
}
