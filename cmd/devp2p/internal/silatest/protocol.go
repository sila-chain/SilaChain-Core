// Copyright 2023 The sila Authors
// This file is part of sila.
//
// sila is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sila is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with sila. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"github.com/sila-org/sila/p2p"
	"github.com/sila-org/sila/rlp"
)

// Unexported devp2p message codes from p2p/peer.go.
const (
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

// Unexported devp2p protocol lengths from p2p package.
const (
	baseProtoLen = 16
	silaProtoLen  = 18
	// snapProtoLen accommodates snap/2 (EIP-8189) which extends snap/1 with two
	// additional message codes (GetBlockAccessLists=0x08, BlockAccessLists=0x09).
	// Using 10 is safe for snap/1 connections because the extra codes are simply
	// never used on that protocol version.
	snapProtoLen = 10
)

// Unexported handshake structure from p2p/peer.go.
type protoHandshake struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte
	Rest       []rlp.RawValue `rlp:"tail"`
}

type Hello = protoHandshake

// Proto is an enum representing devp2p protocol types.
type Proto int

const (
	baseProto Proto = iota
	silaProto
	snapProto
)

// getProto returns the protocol a certain message code is associated with
// (assuming the negotiated capabilities are exactly {sila,snap})
func getProto(code uint64) Proto {
	switch {
	case code < baseProtoLen:
		return baseProto
	case code < baseProtoLen+silaProtoLen:
		return silaProto
	case code < baseProtoLen+silaProtoLen+snapProtoLen:
		return snapProto
	default:
		panic("unhandled msg code beyond last protocol")
	}
}

// protoOffset will return the offset at which the specified protocol's messages
// begin.
func protoOffset(proto Proto) uint64 {
	switch proto {
	case baseProto:
		return 0
	case silaProto:
		return baseProtoLen
	case snapProto:
		return baseProtoLen + silaProtoLen
	default:
		panic("unhandled protocol")
	}
}
