// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package netapi

import (
	"fmt"

	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/p2p"
)

// NetAPI offers Sila network related RPC methods.
type NetAPI struct {
	net            *p2p.Server
	networkVersion uint64
}

// NewNetAPI creates a new Sila net API instance.
func NewNetAPI(net *p2p.Server, networkVersion uint64) *NetAPI {
	return &NetAPI{net, networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (api *NetAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers.
func (api *NetAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(api.net.PeerCount())
}

// Version returns the current legacy-compatible network protocol version.
func (api *NetAPI) Version() string {
	return fmt.Sprintf("%d", api.networkVersion)
}
