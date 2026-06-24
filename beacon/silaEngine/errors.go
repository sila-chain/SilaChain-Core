// Copyright 2022 The sila Authors
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

package silaEngine

import (
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/rpc"
)

// SilaEngineAPIError is a standardized error message between consensus and execution
// clients, also containing any custom error message Sila might include.
type SilaEngineAPIError struct {
	code int
	msg  string
	err  error
}

func (e *SilaEngineAPIError) ErrorCode() int { return e.code }
func (e *SilaEngineAPIError) Error() string  { return e.msg }
func (e *SilaEngineAPIError) ErrorData() interface{} {
	if e.err == nil {
		return nil
	}
	return struct {
		Error string `json:"err"`
	}{e.err.Error()}
}

// With returns a copy of the error with a new embedded custom data field.
func (e *SilaEngineAPIError) With(err error) *SilaEngineAPIError {
	return &SilaEngineAPIError{
		code: e.code,
		msg:  e.msg,
		err:  err,
	}
}

var (
	_ rpc.Error     = new(SilaEngineAPIError)
	_ rpc.DataError = new(SilaEngineAPIError)
)

var (
	// VALID is returned by the silaEngine API in the following calls:
	//   - newPayloadV1:       if the payload was already known or was just validated and executed
	//   - forkchoiceUpdateV1: if the chain accepted the reorg (might ignore if it's stale)
	VALID = "VALID"

	// INVALID is returned by the silaEngine API in the following calls:
	//   - newPayloadV1:       if the payload failed to execute on top of the local chain
	//   - forkchoiceUpdateV1: if the new head is unknown, pre-merge, or reorg to it fails
	INVALID = "INVALID"

	// SYNCING is returned by the silaEngine API in the following calls:
	//   - newPayloadV1:       if the payload was accepted on top of an active sync
	//   - forkchoiceUpdateV1: if the new head was seen before, but not part of the chain
	SYNCING = "SYNCING"

	// ACCEPTED is returned by the silaEngine API in the following calls:
	//   - newPayloadV1: if the payload was accepted, but not processed (side chain)
	ACCEPTED = "ACCEPTED"

	GenericServerError       = &SilaEngineAPIError{code: -32000, msg: "Server error"}
	UnknownPayload           = &SilaEngineAPIError{code: -38001, msg: "Unknown payload"}
	InvalidForkChoiceState   = &SilaEngineAPIError{code: -38002, msg: "Invalid forkchoice state"}
	InvalidPayloadAttributes = &SilaEngineAPIError{code: -38003, msg: "Invalid payload attributes"}
	TooLargeRequest          = &SilaEngineAPIError{code: -38004, msg: "Too large request"}
	InvalidParams            = &SilaEngineAPIError{code: -32602, msg: "Invalid parameters"}
	UnsupportedFork          = &SilaEngineAPIError{code: -38005, msg: "Unsupported fork"}
	TooDeepReorg             = &SilaEngineAPIError{code: -38006, msg: "Too deep reorg"}

	STATUS_INVALID         = ForkChoiceResponse{PayloadStatus: PayloadStatusV1{Status: INVALID}, PayloadID: nil}
	STATUS_SYNCING         = ForkChoiceResponse{PayloadStatus: PayloadStatusV1{Status: SYNCING}, PayloadID: nil}
	INVALID_TERMINAL_BLOCK = PayloadStatusV1{Status: INVALID, LatestValidHash: &common.Hash{}}
)
