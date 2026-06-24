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

package blsync

import (
	"github.com/sila-org/sila/beacon/light"
	"github.com/sila-org/sila/beacon/light/api"
	"github.com/sila-org/sila/beacon/light/request"
	"github.com/sila-org/sila/beacon/light/sync"
	"github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/beacon/types"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/mclock"
	"github.com/sila-org/sila/siladb/memorydb"
	"github.com/sila-org/sila/event"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/rpc"
)

type Client struct {
	urls         []string
	customHeader map[string]string
	config       *params.ClientConfig
	scheduler    *request.Scheduler
	blockSync    *beaconBlockSync
	silaEngineRPC    *rpc.Client

	chainHeadSub event.Subscription
	silaEngineClient *silaEngineClient
}

func NewClient(config params.ClientConfig) *Client {
	// create data structures
	var (
		db             = memorydb.New()
		committeeChain = light.NewCommitteeChain(db, &config.ChainConfig, config.Threshold, !config.NoFilter)
		headTracker    = light.NewHeadTracker(committeeChain, config.Threshold, func(checkpoint common.Hash) {
			if saved, err := config.SaveCheckpointToFile(checkpoint); saved {
				log.Debug("Saved beacon checkpoint", "file", config.CheckpointFile, "checkpoint", checkpoint)
			} else if err != nil {
				log.Error("Failed to save beacon checkpoint", "file", config.CheckpointFile, "checkpoint", checkpoint, "error", err)
			}
		})
	)
	headSync := sync.NewHeadSync(headTracker, committeeChain)

	// set up scheduler and sync modules
	scheduler := request.NewScheduler()
	checkpointInit := sync.NewCheckpointInit(committeeChain, config.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	beaconBlockSync := newBeaconBlockSync(headTracker)
	scheduler.RegisterTarget(headTracker)
	scheduler.RegisterTarget(committeeChain)
	scheduler.RegisterModule(checkpointInit, "checkpointInit")
	scheduler.RegisterModule(forwardSync, "forwardSync")
	scheduler.RegisterModule(headSync, "headSync")
	scheduler.RegisterModule(beaconBlockSync, "beaconBlockSync")

	return &Client{
		scheduler:    scheduler,
		urls:         config.Apis,
		customHeader: config.CustomHeader,
		config:       &config,
		blockSync:    beaconBlockSync,
	}
}

func (c *Client) SetEngineRPC(silaEngine *rpc.Client) {
	c.silaEngineRPC = silaEngine
}

func (c *Client) Start() error {
	headCh := make(chan types.ChainHeadEvent, 16)
	c.chainHeadSub = c.blockSync.SubscribeChainHead(headCh)
	c.silaEngineClient = startSilaEngineClient(c.config, c.silaEngineRPC, headCh)

	c.scheduler.Start()
	for _, url := range c.urls {
		beaconApi := api.NewBeaconLightApi(url, c.customHeader)
		c.scheduler.RegisterServer(request.NewServer(api.NewApiServer(beaconApi), &mclock.System{}))
	}
	return nil
}

func (c *Client) Stop() error {
	c.silaEngineClient.stop()
	c.chainHeadSub.Unsubscribe()
	c.scheduler.Stop()
	return nil
}
