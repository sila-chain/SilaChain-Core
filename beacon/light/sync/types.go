// Copyright 2023 The SilaChain Authors
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

package sync

import (
	"github.com/sila-org/sila/beacon/light/request"
	"github.com/sila-org/sila/beacon/types"
	"github.com/sila-org/sila/common"
)

var (
	EvNewHead             = &request.EventType{Name: "newHead"}             // data: types.HeadInfo
	EvNewOptimisticUpdate = &request.EventType{Name: "newOptimisticUpdate"} // data: types.OptimisticUpdate
	EvNewFinalityUpdate   = &request.EventType{Name: "newFinalityUpdate"}   // data: types.FinalityUpdate
)

type (
	ReqUpdates struct {
		FirstPeriod, Count uint64
	}
	RespUpdates struct {
		Updates    []*types.LightClientUpdate
		Committees []*types.SerializedSyncCommittee
	}
	ReqHeader  common.Hash
	RespHeader struct {
		Header               types.Header
		Canonical, Finalized bool
	}
	ReqCheckpointData common.Hash
	ReqBeaconBlock    common.Hash
	ReqFinality       struct{}
)
