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

// Package ethconfig contains the configuration of the ETH and LES protocols.
package ethconfig

import (
	"errors"
	"time"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/consensus/beacon"
	"github.com/sila-org/sila/consensus/clique"
	"github.com/sila-org/sila/consensus/ethash"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/history"
	"github.com/sila-org/sila/core/txpool/blobpool"
	"github.com/sila-org/sila/core/txpool/legacypool"
	"github.com/sila-org/sila/eth/gasprice"
	"github.com/sila-org/sila/ethdb"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/miner"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/triedb"
	"github.com/sila-org/sila/triedb/pathdb"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// Defaults contains default settings for use on the SilaChain mainnet.
var Defaults = Config{
	HistoryMode:             history.KeepAll,
	SyncMode:                SnapSync,
	NetworkId:               0, // enable auto configuration of networkID == chainID
	TxLookupLimit:           2350000,
	TransactionHistory:      2350000,
	LogHistory:              2350000,
	StateHistory:            pathdb.Defaults.StateHistory,
	TrienodeHistory:         pathdb.Defaults.TrienodeHistory,
	NodeFullValueCheckpoint: pathdb.Defaults.FullValueCheckpoint,
	BinTrieGroupDepth:       triedb.DefaultBinTrieGroupDepth,
	DatabaseCache:           2048,
	TrieCleanCache:          614,
	TrieDirtyCache:          1024,
	SnapshotCache:           409,
	TrieTimeout:             60 * time.Minute,
	FilterLogCacheSize:      32,
	LogQueryLimit:           1000,
	Miner:                   miner.DefaultConfig,
	TxPool:                  legacypool.DefaultConfig,
	BlobPool:                blobpool.DefaultConfig,
	RPCGasCap:               50000000,
	RPCEVMTimeout:           5 * time.Second,
	GPO:                     FullNodeGPO,
	RPCTxFeeCap:             1, // 1 Sila
	TxSyncDefaultTimeout:    20 * time.Second,
	TxSyncMaxTimeout:        1 * time.Minute,
	SlowBlockThreshold:      -1, // Disabled by default; set via --debug.logslowblock flag
	RangeLimit:              0,
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for Sila execution protocols.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the SilaChain mainnet block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Network ID separates blockchains on the peer-to-peer networking level. When left
	// zero, the chain ID is used as network ID.
	NetworkId uint64
	SyncMode  SyncMode

	// HistoryMode configures chain history retention.
	HistoryMode history.HistoryMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// nodes to connect to.
	EthDiscoveryURLs  []string
	SnapDiscoveryURLs []string

	// State options.
	NoPruning  bool // Whether to disable pruning and flush everything to disk
	NoPrefetch bool // Whether to disable prefetching and only load state on demand

	// Deprecated: use 'TransactionHistory' instead.
	TxLookupLimit uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.

	TransactionHistory   uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.
	LogHistory           uint64 `toml:",omitempty"` // The maximum number of blocks from head where a log search index is maintained.
	LogNoHistory         bool   `toml:",omitempty"` // No log search index is maintained.
	LogExportCheckpoints string // export log index checkpoints to file
	StateHistory         uint64 `toml:",omitempty"` // The maximum number of blocks from head whose state histories are reserved.
	TrienodeHistory      int64  `toml:",omitempty"` // Number of blocks from the chain head for which trienode histories are retained

	// The frequency of full-value encoding. For example, a value of 16 means
	// that, on average, for a given trie node across its 16 consecutive historical
	// versions, only one version is stored in full format, while the others
	// are stored in diff mode for storage compression.
	NodeFullValueCheckpoint uint32 `toml:",omitempty"`

	// State scheme represents the scheme used to store Sila states and trie
	// nodes on top. It can be 'hash', 'path', or none which means use the scheme
	// consistent with persistent state.
	StateScheme string `toml:",omitempty"`

	// BinTrieGroupDepth is the number of levels per serialized group in binary trie.
	// Valid values are 1-8, with 8 being the default (byte-aligned groups).
	// Lower values create smaller groups with more nodes.
	BinTrieGroupDepth int `toml:",omitempty"`

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes sila verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// SlowBlockThreshold is the block execution time threshold beyond which
	// detailed statistics are logged. Negative means disabled (default), zero
	// logs all blocks, positive filters by execution time.
	SlowBlockThreshold time.Duration `toml:",omitempty"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string
	DatabaseEra        string

	TrieCleanCache int
	TrieDirtyCache int
	TrieTimeout    time.Duration
	SnapshotCache  int
	Preimages      bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// This is the maximum number of addresses or topics allowed in filter criteria
	// for eth_getLogs.
	LogQueryLimit int

	// Mining options
	Miner miner.Config

	// Transaction pool options
	TxPool   legacypool.Config
	BlobPool blobpool.Config

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Enables collection of witness trie access statistics
	EnableWitnessStats bool

	// Generate execution witnesses and self-check against them (testing purpose)
	StatelessSelfValidation bool

	// Enables tracking of state size
	EnableStateSizeTracking bool

	// Enables VM tracing
	VMTrace           string
	VMTraceJsonConfig string

	// RPCGasCap is the global gas cap for eth-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for eth-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee (price * gas limit) cap for
	// send-transaction variants. The unit is ether.
	RPCTxFeeCap float64

	// OverrideOsaka (TODO: remove after the fork)
	OverrideOsaka *uint64 `toml:",omitempty"`

	// OverrideBPO1 (TODO: remove after the fork)
	OverrideBPO1 *uint64 `toml:",omitempty"`

	// OverrideBPO2 (TODO: remove after the fork)
	OverrideBPO2 *uint64 `toml:",omitempty"`

	// OverrideUBT (TODO: remove after the fork)
	OverrideUBT *uint64 `toml:",omitempty"`

	// EIP-7966: eth_sendRawTransactionSync timeouts
	TxSyncDefaultTimeout time.Duration `toml:",omitempty"`
	TxSyncMaxTimeout     time.Duration `toml:",omitempty"`

	// RangeLimit restricts the maximum range (end - start) for range queries.
	RangeLimit uint64 `toml:",omitempty"`
}

// CreateConsensusEngine creates a consensus engine for the given chain config.
// Clique is allowed for now to live standalone, but ethash is forbidden and can
// only exist on already merged networks.
func CreateConsensusEngine(config *params.ChainConfig, db ethdb.Database) (consensus.Engine, error) {
	if config.TerminalTotalDifficulty == nil {
		log.Error("Sila only supports PoS networks. Please transition legacy networks using Sila v1.13.x.")
		return nil, errors.New("'terminalTotalDifficulty' is not set in genesis block")
	}
	// Wrap previously supported consensus engines into their post-merge counterpart
	if config.Clique != nil {
		return beacon.New(clique.New(config.Clique, db)), nil
	}
	return beacon.New(ethash.NewFaker()), nil
}
