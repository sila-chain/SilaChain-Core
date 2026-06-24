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
	"context"
	"strings"
	"sync"
	"time"

	"github.com/sila-org/sila/beacon/silaEngine"
	"github.com/sila-org/sila/beacon/params"
	"github.com/sila-org/sila/beacon/types"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	ctypes "github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/rpc"
)

type silaEngineClient struct {
	config     *params.ClientConfig
	rpc        *rpc.Client
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
}

func startSilaEngineClient(config *params.ClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent) *silaEngineClient {
	ctx, cancel := context.WithCancel(context.Background())
	ec := &silaEngineClient{
		config:     config,
		rpc:        rpc,
		rootCtx:    ctx,
		cancelRoot: cancel,
	}
	ec.wg.Add(1)
	go ec.updateLoop(headCh)
	return ec
}

func (ec *silaEngineClient) stop() {
	ec.cancelRoot()
	ec.wg.Wait()
}

func (ec *silaEngineClient) updateLoop(headCh <-chan types.ChainHeadEvent) {
	defer ec.wg.Done()

	for {
		select {
		case <-ec.rootCtx.Done():
			log.Debug("Stopping silaEngine API update loop")
			return

		case event := <-headCh:
			if ec.rpc == nil { // dry run, no silaEngine API specified
				log.Info("New execution block retrieved", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "finalized", event.Finalized)
				continue
			}

			fork := ec.config.ForkAtEpoch(event.BeaconHead.Epoch())
			forkName := strings.ToLower(fork.Name)

			log.Debug("Calling NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash())
			if status, err := ec.callNewPayload(forkName, event); err == nil {
				log.Info("Successful NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "status", status)
			} else {
				log.Error("Failed NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "error", err)
			}

			log.Debug("Calling ForkchoiceUpdated", "head", event.Block.Hash())
			if status, err := ec.callForkchoiceUpdated(forkName, event); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", event.Block.Hash(), "status", status)
			} else {
				if err.Error() == "beacon syncer reorging" {
					log.Debug("Failed ForkchoiceUpdated", "head", event.Block.Hash(), "error", err)
					continue // ignore beacon syncer reorging errors, this error can occur if the blsync is skipping a block
				}
				log.Error("Failed ForkchoiceUpdated", "head", event.Block.Hash(), "error", err)
			}
		}
	}
}

func (ec *silaEngineClient) callNewPayload(fork string, event types.ChainHeadEvent) (string, error) {
	execData := silaEngine.BlockToExecutableData(event.Block, nil, nil, nil).ExecutionPayload

	var (
		method string
		params = []any{execData}
	)
	switch fork {
	case "altair", "bellatrix":
		method = "silaEngine_newPayloadV1"
	case "capella":
		method = "silaEngine_newPayloadV2"
	case "deneb":
		method = "silaEngine_newPayloadV3"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		params = append(params, blobHashes, parentBeaconRoot)
	default: // electra, fulu and above
		method = "silaEngine_newPayloadV4"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		hexRequests := make([]hexutil.Bytes, len(event.ExecRequests))
		for i := range event.ExecRequests {
			hexRequests[i] = hexutil.Bytes(event.ExecRequests[i])
		}
		params = append(params, blobHashes, parentBeaconRoot, hexRequests)
	}

	ctx, cancel := context.WithTimeout(ec.rootCtx, time.Second*5)
	defer cancel()
	var resp silaEngine.PayloadStatusV1
	err := ec.rpc.CallContext(ctx, &resp, method, params...)
	return resp.Status, err
}

func collectBlobHashes(b *ctypes.Block) []common.Hash {
	list := make([]common.Hash, 0)
	for _, tx := range b.Transactions() {
		list = append(list, tx.BlobHashes()...)
	}
	return list
}

func (ec *silaEngineClient) callForkchoiceUpdated(fork string, event types.ChainHeadEvent) (string, error) {
	update := silaEngine.ForkchoiceStateV1{
		HeadBlockHash:      event.Block.Hash(),
		SafeBlockHash:      event.Finalized,
		FinalizedBlockHash: event.Finalized,
	}

	var method string
	switch fork {
	case "altair", "bellatrix":
		method = "silaEngine_forkchoiceUpdatedV1"
	case "capella":
		method = "silaEngine_forkchoiceUpdatedV2"
	default: // deneb, electra, fulu and above
		method = "silaEngine_forkchoiceUpdatedV3"
	}

	ctx, cancel := context.WithTimeout(ec.rootCtx, time.Second*5)
	defer cancel()
	var resp silaEngine.ForkChoiceResponse
	err := ec.rpc.CallContext(ctx, &resp, method, update, nil)
	return resp.PayloadStatus.Status, err
}
