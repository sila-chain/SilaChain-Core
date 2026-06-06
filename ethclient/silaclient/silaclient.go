// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaclient

import (
	extensions "github.com/sila-org/sila/ethclient/gethclient"
	"github.com/sila-org/sila/rpc"
)

// Client is the primary SilaChain RPC extension client.
type Client = extensions.SilaClient

// New creates a SilaChain RPC extension client.
func New(c *rpc.Client) *Client {
	return extensions.NewSila(c)
}
