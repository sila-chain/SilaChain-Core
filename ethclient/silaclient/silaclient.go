// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaclient

import (
	gethclient "github.com/sila-org/sila/ethclient/gethclient"
	"github.com/sila-org/sila/rpc"
)

// Client is the primary SilaChain RPC extension client.
type Client = gethclient.SilaClient

// New creates a SilaChain RPC extension client.
func New(c *rpc.Client) *Client {
	return gethclient.NewSila(c)
}
