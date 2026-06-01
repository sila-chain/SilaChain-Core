package chainctx

import (
	"context"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

// Backend provides methods required to implement ChainContext.
type Backend interface {
	Engine() consensus.Engine
	HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	CurrentHeader() *types.Header
	ChainConfig() *params.ChainConfig
}

// ChainContext is an implementation of core.ChainContext. It's main use-case
// is instantiating a vm.BlockContext without having access to the BlockChain object.
type ChainContext struct {
	b   Backend
	ctx context.Context
}

// NewChainContext creates a new ChainContext object.
func NewChainContext(ctx context.Context, backend Backend) *ChainContext {
	return &ChainContext{ctx: ctx, b: backend}
}

func (context *ChainContext) Engine() consensus.Engine {
	return context.b.Engine()
}

func (context *ChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	// This method is called to get the hash for a block number when executing the BLOCKHASH
	// opcode. Hence no need to search for non-canonical blocks.
	header, err := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	if err != nil || header.Hash() != hash {
		return nil
	}
	return header
}

func (context *ChainContext) Config() *params.ChainConfig {
	return context.b.ChainConfig()
}

func (context *ChainContext) CurrentHeader() *types.Header {
	return context.b.CurrentHeader()
}

func (context *ChainContext) GetHeaderByNumber(number uint64) *types.Header {
	header, _ := context.b.HeaderByNumber(context.ctx, rpc.BlockNumber(number))
	return header
}

func (context *ChainContext) GetHeaderByHash(hash common.Hash) *types.Header {
	header, _ := context.b.HeaderByHash(context.ctx, hash)
	return header
}
