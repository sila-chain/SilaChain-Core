// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package chainctx

import (
	"context"

	ethapi "github.com/sila-org/sila/internal/ethapi"
)

type ChainContext = ethapi.ChainContext
type Backend = ethapi.ChainContextBackend

func NewChainContext(ctx context.Context, backend Backend) *ChainContext {
	return ethapi.NewChainContext(ctx, backend)
}
