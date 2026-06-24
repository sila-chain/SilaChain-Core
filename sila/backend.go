// Copyright 2014 The sila Authors
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

// Package sila implements the Sila protocol.
package sila

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/filtermaps"
	"github.com/sila-org/sila/core/history"
	"github.com/sila-org/sila/core/rawdb"
	"github.com/sila-org/sila/core/state/pruner"
	"github.com/sila-org/sila/core/txpool"
	"github.com/sila-org/sila/core/txpool/blobpool"
	"github.com/sila-org/sila/core/txpool/legacypool"
	"github.com/sila-org/sila/core/txpool/locals"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/sila/downloader"
	"github.com/sila-org/sila/sila/silaconfig"
	"github.com/sila-org/sila/sila/gasprice"
	"github.com/sila-org/sila/sila/protocols/sila"
	"github.com/sila-org/sila/sila/protocols/snap"
	"github.com/sila-org/sila/sila/tracers"
	"github.com/sila-org/sila/siladb"
	"github.com/sila-org/sila/event"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/shutdowncheck"
	"github.com/sila-org/sila/internal/version"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/miner"
	"github.com/sila-org/sila/node"
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/p2p/dnsdisc"
	"github.com/sila-org/sila/p2p/enode"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rlp"
	"github.com/sila-org/sila/rpc"
	silaversion "github.com/sila-org/sila/version"
)

const (
	// This is the fairness knob for the discovery mixer. When looking for peers, we'll
	// wait this long for a single source of candidates before moving on and trying other
	// sources. If this timeout expires, the source will be skipped in this round, but it
	// will continue to fetch in the background and will have a chance with a new timeout
	// in the next rounds, giving it overall more time but a proportionally smaller share.
	// We expect a normal source to produce ~10 candidates per second.
	discmixTimeout = 100 * time.Millisecond

	// discoveryPrefetchBuffer is the number of peers to pre-fetch from a discovery
	// source. It is useful to avoid the negative effects of potential longer timeouts
	// in the discovery, keeping dial progress while waiting for the next batch of
	// candidates.
	discoveryPrefetchBuffer = 32

	// maxParallelENRRequests is the maximum number of parallel ENR requests that can be
	// performed by a disc/v4 source.
	maxParallelENRRequests = 16
)

// Config contains the configuration options of the SILA protocol.
// Deprecated: use silaconfig.Config instead.
type Config = silaconfig.Config

// Sila implements the Sila full node service.
type Sila struct {
	// core protocol objects
	config         *silaconfig.Config
	txPool         *txpool.TxPool
	blobTxPool     *blobpool.BlobPool
	blobCache      *blobpool.Cache
	localTxTracker *locals.TxTracker
	blockchain     *core.BlockChain

	handler *handler
	discmix *enode.FairMix
	dropper *dropper

	// DB interfaces
	chainDb siladb.Database // Block chain database

	silaEngine         consensus.SilaEngine
	accountManager *accounts.Manager

	filterMaps      *filtermaps.FilterMaps
	closeFilterMaps chan chan struct{}

	// Chain event subscriptions driving updateFilterMapsHeads. The
	// subscriptions are registered and consumed in Start.
	fmHeadEventCh  chan core.ChainEvent
	fmHeadSub      event.Subscription
	fmBlockProcCh  chan bool
	fmBlockProcSub event.Subscription

	APIBackend *SilaAPIBackend

	miner    *miner.Miner
	gasPrice *big.Int

	networkID     uint64
	netRPCService *silaapi.NetAPI

	p2pServer *p2p.Server

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)

	shutdownTracker *shutdowncheck.ShutdownTracker // Tracks if and when the node has shutdown ungracefully
}

// New creates a new Sila object (including the initialisation of the common Sila object),
// whose lifecycle will be managed by the provided node.
func New(stack *node.Node, config *silaconfig.Config) (*Sila, error) {
	// Ensure configuration values are compatible and sane
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if !config.HistoryMode.IsValid() {
		return nil, fmt.Errorf("invalid history mode %d", config.HistoryMode)
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Sign() <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", silaconfig.Defaults.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(silaconfig.Defaults.Miner.GasPrice)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 && config.StateScheme == rawdb.HashScheme {
		if config.SnapshotCache > 0 {
			config.TrieCleanCache += config.TrieDirtyCache * 3 / 5
			config.SnapshotCache += config.TrieDirtyCache * 2 / 5
		} else {
			config.TrieCleanCache += config.TrieDirtyCache
		}
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	dbOptions := node.DatabaseOptions{
		Cache:             config.DatabaseCache,
		Handles:           config.DatabaseHandles,
		AncientsDirectory: config.DatabaseFreezer,
		EraDirectory:      config.DatabaseEra,
		MetricsNamespace:  "sila/db/chaindata/",
	}
	chainDb, err := stack.OpenDatabaseWithOptions("chaindata", dbOptions)
	if err != nil {
		return nil, err
	}
	scheme, err := rawdb.ParseStateScheme(config.StateScheme, chainDb)
	if err != nil {
		return nil, err
	}
	// Try to recover offline state pruning only in hash-based.
	if scheme == rawdb.HashScheme {
		if err := pruner.RecoverPruning(stack.ResolvePath(""), chainDb); err != nil {
			log.Error("Failed to recover state", "error", err)
		}
	}

	// Here we determine genesis hash and active ChainConfig.
	// We need these to figure out the consensus parameters and to set up history pruning.
	chainConfig, genesisHash, err := core.LoadChainConfig(chainDb, config.Genesis)
	if err != nil {
		return nil, err
	}
	silaEngine, err := silaconfig.CreateConsensusEngine(chainConfig, chainDb)
	if err != nil {
		return nil, err
	}
	// Set networkID to chainID by default.
	networkID := config.NetworkId
	if networkID == 0 {
		networkID = chainConfig.ChainID.Uint64()
	}

	// Assemble the Sila object.
	sila := &Sila{
		config:          config,
		chainDb:         chainDb,
		accountManager:  stack.AccountManager(),
		silaEngine:          silaEngine,
		networkID:       networkID,
		gasPrice:        config.Miner.GasPrice,
		p2pServer:       stack.Server(),
		discmix:         enode.NewFairMix(discmixTimeout),
		shutdownTracker: shutdowncheck.NewShutdownTracker(chainDb),
		fmHeadEventCh:   make(chan core.ChainEvent, 10),
		fmBlockProcCh:   make(chan bool, 10),
	}
	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising Sila protocol", "network", networkID, "dbversion", dbVer)

	// Create BlockChain object.
	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Sila %s only supports v%d", *bcVersion, version.WithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}
	histPolicy, err := history.NewPolicy(config.HistoryMode, genesisHash)
	if err != nil {
		return nil, err
	}
	var (
		options = &core.BlockChainConfig{
			TrieCleanLimit:          config.TrieCleanCache,
			NoPrefetch:              config.NoPrefetch,
			TrieDirtyLimit:          config.TrieDirtyCache,
			ArchiveMode:             config.NoPruning,
			TrieTimeLimit:           config.TrieTimeout,
			SnapshotLimit:           config.SnapshotCache,
			Preimages:               config.Preimages,
			StateHistory:            config.StateHistory,
			TrienodeHistory:         config.TrienodeHistory,
			NodeFullValueCheckpoint: config.NodeFullValueCheckpoint,
			BinTrieGroupDepth:       config.BinTrieGroupDepth,
			StateScheme:             scheme,
			HistoryPolicy:           histPolicy,
			TxLookupLimit:           int64(min(config.TransactionHistory, math.MaxInt64)),
			VmConfig: vm.Config{
				EnablePreimageRecording: config.EnablePreimageRecording,
			},
			// Enables file journaling for the trie database. The journal files will be stored
			// within the data directory. The corresponding paths will be either:
			// - DATADIR/triedb/merkle.journal
			// - DATADIR/triedb/verkle.journal
			TrieJournalDirectory: stack.ResolvePath("triedb"),
			StateSizeTracking:    config.EnableStateSizeTracking,
			SlowBlockThreshold:   config.SlowBlockThreshold,

			StatelessSelfValidation: config.StatelessSelfValidation,
			EnableWitnessStats:      config.EnableWitnessStats,
		}
	)
	if config.VMTrace != "" {
		traceConfig := json.RawMessage("{}")
		if config.VMTraceJsonConfig != "" {
			traceConfig = json.RawMessage(config.VMTraceJsonConfig)
		}
		t, err := tracers.LiveDirectory.New(config.VMTrace, traceConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create tracer %s: %v", config.VMTrace, err)
		}
		options.VmConfig.Tracer = t
	}
	// Override the chain config with provided settings.
	var overrides core.ChainOverrides
	if config.OverrideOsaka != nil {
		overrides.OverrideOsaka = config.OverrideOsaka
	}
	if config.OverrideBPO1 != nil {
		overrides.OverrideBPO1 = config.OverrideBPO1
	}
	if config.OverrideBPO2 != nil {
		overrides.OverrideBPO2 = config.OverrideBPO2
	}
	if config.OverrideUBT != nil {
		overrides.OverrideUBT = config.OverrideUBT
	}
	options.Overrides = &overrides

	sila.blockchain, err = core.NewBlockChain(chainDb, config.Genesis, sila.silaEngine, options)
	if err != nil {
		return nil, err
	}

	// Initialize filtermaps log index.
	fmConfig := filtermaps.Config{
		History:        config.LogHistory,
		Disabled:       config.LogNoHistory,
		ExportFileName: config.LogExportCheckpoints,
		HashScheme:     scheme == rawdb.HashScheme,
	}
	chainView := sila.newChainView(sila.blockchain.CurrentBlock())
	historyCutoff, _ := sila.blockchain.HistoryPruningCutoff()
	var finalBlock uint64
	if fb := sila.blockchain.CurrentFinalBlock(); fb != nil {
		finalBlock = fb.Number.Uint64()
	}
	filterMaps, err := filtermaps.NewFilterMaps(chainDb, chainView, historyCutoff, finalBlock, filtermaps.DefaultParams, fmConfig)
	if err != nil {
		return nil, err
	}
	sila.filterMaps = filterMaps
	sila.closeFilterMaps = make(chan chan struct{})

	// TxPool
	if config.TxPool.Journal != "" {
		config.TxPool.Journal = stack.ResolvePath(config.TxPool.Journal)
	}
	legacyPool := legacypool.New(config.TxPool, sila.blockchain)

	if config.BlobPool.Datadir != "" {
		config.BlobPool.Datadir = stack.ResolvePath(config.BlobPool.Datadir)
	}
	sila.blobTxPool = blobpool.New(config.BlobPool, sila.blockchain, legacyPool.HasPendingAuth)
	sila.blobCache = blobpool.NewCache(sila.blobTxPool)

	sila.txPool, err = txpool.New(config.TxPool.PriceLimit, sila.blockchain, []txpool.SubPool{legacyPool, sila.blobTxPool})
	if err != nil {
		return nil, err
	}

	if !config.TxPool.NoLocals {
		rejournal := config.TxPool.Rejournal
		if rejournal < time.Second {
			log.Warn("Sanitizing invalid txpool journal time", "provided", rejournal, "updated", time.Second)
			rejournal = time.Second
		}
		sila.localTxTracker = locals.New(config.TxPool.Journal, rejournal, sila.blockchain.Config(), sila.txPool)
		stack.RegisterLifecycle(sila.localTxTracker)
	}

	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := options.TrieCleanLimit + options.TrieDirtyLimit + options.SnapshotLimit
	if sila.handler, err = newHandler(&handlerConfig{
		NodeID:         sila.p2pServer.Self().ID(),
		Database:       chainDb,
		Chain:          sila.blockchain,
		TxPool:         sila.txPool,
		Network:        networkID,
		Sync:           config.SyncMode,
		BloomCache:     uint64(cacheLimit),
		RequiredBlocks: config.RequiredBlocks,
		SnapV2:         config.SnapV2,
	}); err != nil {
		return nil, err
	}

	sila.dropper = newDropper(sila.p2pServer.MaxDialedConns(), sila.p2pServer.MaxInboundConns())

	sila.miner = miner.New(sila, config.Miner, sila.silaEngine)
	sila.miner.SetExtra(makeExtraData(config.Miner.ExtraData))
	sila.miner.SetPrioAddresses(config.TxPool.Locals)

	sila.APIBackend = &SilaAPIBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, sila, nil}
	if sila.APIBackend.allowUnprotectedTxs {
		log.Info("Unprotected transactions allowed")
	}
	sila.APIBackend.gpo = gasprice.NewOracle(sila.APIBackend, config.GPO, config.Miner.GasPrice)

	// Start the RPC service
	sila.netRPCService = silaapi.NewNetAPI(sila.p2pServer, networkID)

	// Register the backend on the node
	stack.RegisterAPIs(sila.APIs())
	stack.RegisterProtocols(sila.Protocols())
	stack.RegisterLifecycle(sila)

	// Successful startup; push a marker and check previous unclean shutdowns.
	sila.shutdownTracker.MarkStartup()

	return sila, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(silaversion.Major<<16 | silaversion.Minor<<8 | silaversion.Patch),
			"sila",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// APIs return the collection of RPC services the sila package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Sila) APIs() []rpc.API {
	apis := silaapi.GetAPIs(s.APIBackend)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "miner",
			Service:   NewMinerAPI(s),
		}, {
			Namespace: "sila",
			Service:   downloader.NewDownloaderAPI(s.handler.downloader, s.blockchain),
		}, {
			Namespace: "admin",
			Service:   NewAdminAPI(s),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(s),
		}, {
			Namespace: "net",
			Service:   s.netRPCService,
		},
	}...)
}

func (s *Sila) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Sila) Miner() *miner.Miner { return s.miner }

func (s *Sila) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Sila) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Sila) TxPool() *txpool.TxPool             { return s.txPool }
func (s *Sila) BlobTxPool() *blobpool.BlobPool     { return s.blobTxPool }
func (s *Sila) BlobCache() *blobpool.Cache         { return s.blobCache }
func (s *Sila) SilaEngine() consensus.SilaEngine           { return s.silaEngine }
func (s *Sila) ChainDb() siladb.Database            { return s.chainDb }
func (s *Sila) IsListening() bool                  { return true } // Always listening
func (s *Sila) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *Sila) Synced() bool                       { return s.handler.synced.Load() }
func (s *Sila) SetSynced()                         { s.handler.enableSyncedFeatures() }
func (s *Sila) ArchiveMode() bool                  { return s.config.NoPruning }

// Protocols returns all the currently configured
// network protocols to start.
func (s *Sila) Protocols() []p2p.Protocol {
	protos := sila.MakeProtocols((*silaHandler)(s.handler), s.networkID, s.discmix)
	if s.config.SnapshotCache > 0 {
		protos = append(protos, snap.MakeProtocols((*snapHandler)(s.handler), s.config.SnapV2)...)
	}
	return protos
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// Sila protocol implementation.
func (s *Sila) Start() error {
	if err := s.setupDiscovery(); err != nil {
		return err
	}

	// Regularly update shutdown marker
	s.shutdownTracker.Start()

	// Start the networking layer
	s.handler.Start(s.p2pServer.MaxPeers)

	// Start the connection manager
	s.dropper.Start(s.p2pServer, func() bool { return !s.Synced() })

	// Subscribe to chain events for the filterMaps head updater.
	s.fmHeadSub = s.blockchain.SubscribeChainEvent(s.fmHeadEventCh)
	s.fmBlockProcSub = s.blockchain.SubscribeBlockProcessingEvent(s.fmBlockProcCh)

	// start log indexer
	s.filterMaps.Start()
	go s.updateFilterMapsHeads()
	return nil
}

func (s *Sila) newChainView(head *types.Header) *filtermaps.ChainView {
	if head == nil {
		return nil
	}
	return filtermaps.NewChainView(s.blockchain, head.Number.Uint64(), head.Hash())
}

func (s *Sila) updateFilterMapsHeads() {
	headEventCh := s.fmHeadEventCh
	blockProcCh := s.fmBlockProcCh
	defer func() {
		s.fmHeadSub.Unsubscribe()
		s.fmBlockProcSub.Unsubscribe()
		for {
			select {
			case <-headEventCh:
			case <-blockProcCh:
			default:
				return
			}
		}
	}()

	var head *types.Header
	setHead := func(newHead *types.Header) {
		if newHead == nil {
			return
		}
		if head == nil || newHead.Hash() != head.Hash() {
			head = newHead
			chainView := s.newChainView(head)
			if chainView == nil {
				return
			}
			historyCutoff, _ := s.blockchain.HistoryPruningCutoff()
			var finalBlock uint64
			if fb := s.blockchain.CurrentFinalBlock(); fb != nil {
				finalBlock = fb.Number.Uint64()
			}
			s.filterMaps.SetTarget(chainView, historyCutoff, finalBlock)
		}
	}
	setHead(s.blockchain.CurrentBlock())

	for {
		select {
		case ev := <-headEventCh:
			setHead(ev.Header)
		case blockProc := <-blockProcCh:
			s.filterMaps.SetBlockProcessing(blockProc)
		case <-time.After(time.Second * 10):
			setHead(s.blockchain.CurrentBlock())
		case ch := <-s.closeFilterMaps:
			close(ch)
			return
		}
	}
}

func (s *Sila) setupDiscovery() error {
	sila.StartENRUpdater(s.blockchain, s.p2pServer.LocalNode())

	// Add sila nodes from DNS.
	dnsclient := dnsdisc.NewClient(dnsdisc.Config{})
	if len(s.config.SilaDiscoveryURLs) > 0 {
		iter, err := dnsclient.NewIterator(s.config.SilaDiscoveryURLs...)
		if err != nil {
			return err
		}
		s.discmix.AddSource(iter)
	}

	// Add snap nodes from DNS.
	if len(s.config.SnapDiscoveryURLs) > 0 {
		iter, err := dnsclient.NewIterator(s.config.SnapDiscoveryURLs...)
		if err != nil {
			return err
		}
		s.discmix.AddSource(iter)
	}

	// Add DHT nodes from discv4.
	if s.p2pServer.DiscoveryV4() != nil {
		iter := s.p2pServer.DiscoveryV4().RandomNodes()
		resolverFunc := func(ctx context.Context, enr *enode.Node) *enode.Node {
			// RequestENR does not yet support context. It will simply time out.
			// If the ENR can't be resolved, RequestENR will return nil. We don't
			// care about the specific error here, so we ignore it.
			nn, _ := s.p2pServer.DiscoveryV4().RequestENR(enr)
			return nn
		}
		iter = enode.AsyncFilter(iter, resolverFunc, maxParallelENRRequests)
		iter = enode.Filter(iter, sila.NewNodeFilter(s.blockchain))
		iter = enode.NewBufferIter(iter, discoveryPrefetchBuffer)
		s.discmix.AddSource(iter)
	}

	// Add DHT nodes from discv5.
	if s.p2pServer.DiscoveryV5() != nil {
		filter := sila.NewNodeFilter(s.blockchain)
		iter := enode.Filter(s.p2pServer.DiscoveryV5().RandomNodes(), filter)
		iter = enode.NewBufferIter(iter, discoveryPrefetchBuffer)
		s.discmix.AddSource(iter)
	}

	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Sila protocol.
func (s *Sila) Stop() error {
	// Stop all the peer-related stuff first.
	s.discmix.Close()
	s.dropper.Stop()
	s.handler.Stop()

	// Then stop everything else.
	ch := make(chan struct{})
	s.closeFilterMaps <- ch
	<-ch
	s.filterMaps.Stop()
	s.blobCache.Stop()
	s.txPool.Close()
	s.blockchain.Stop()
	s.silaEngine.Close()

	// Clean shutdown marker as the last thing before closing db
	s.shutdownTracker.Stop()

	s.chainDb.Close()

	return nil
}
