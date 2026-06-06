// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package blockapi

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/forkid"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

type BlockChainAPI struct {
	b BlockChainBackend
}

func NewBlockChainAPI(b BlockChainBackend) *BlockChainAPI {
	return &BlockChainAPI{b: b}
}

func (api *BlockChainAPI) ChainId() *hexutil.Big {
	return ChainId(api.b)
}

func (api *BlockChainAPI) BlockNumber() hexutil.Uint64 {
	return BlockNumber(api.b)
}

func (api *BlockChainAPI) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	return GetBalance(ctx, api.b, address, blockNrOrHash)
}

func (api *BlockChainAPI) GetCode(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return GetCode(ctx, api.b, address, blockNrOrHash)
}

func (api *BlockChainAPI) GetStorageAt(ctx context.Context, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	return GetStorageAt(ctx, api.b, address, hexKey, blockNrOrHash)
}

func (api *BlockChainAPI) GetStorageValues(ctx context.Context, requests map[common.Address][]common.Hash, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address][]hexutil.Bytes, error) {
	return GetStorageValues(ctx, api.b, requests, blockNrOrHash)
}

func (api *BlockChainAPI) GetBlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	return GetBlockReceipts(ctx, api.b, blockNrOrHash)
}

func (api *BlockChainAPI) GetHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (map[string]interface{}, error) {
	return GetHeaderByNumber(ctx, api.b, number)
}

func (api *BlockChainAPI) GetHeaderByHash(ctx context.Context, hash common.Hash) map[string]interface{} {
	return GetHeaderByHash(ctx, api.b, hash)
}

func (api *BlockChainAPI) GetBlockByNumber(ctx context.Context, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	return GetBlockByNumber(ctx, api.b, number, fullTx)
}

func (api *BlockChainAPI) GetBlockByHash(ctx context.Context, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	return GetBlockByHash(ctx, api.b, hash, fullTx)
}

func (api *BlockChainAPI) GetUncleByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	return GetUncleByBlockNumberAndIndex(ctx, api.b, blockNr, index)
}

func (api *BlockChainAPI) GetUncleByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	return GetUncleByBlockHashAndIndex(ctx, api.b, blockHash, index)
}

func (api *BlockChainAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	return GetUncleCountByBlockNumber(ctx, api.b, blockNr)
}

func (api *BlockChainAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	return GetUncleCountByBlockHash(ctx, api.b, blockHash)
}

func (api *BlockChainAPI) Config(ctx context.Context) (*ChainConfigResponse, error) {
	return GetConfig(ctx, api.b)
}

// RPCMarshalBlockWithTransactions converts the given block to the RPC output using the supplied transaction formatter.
func RPCMarshalBlockWithTransactions(block *types.Block, inclTx bool, formatTx func(int, *types.Transaction) interface{}) map[string]interface{} {
	fields := RPCMarshalHeader(block.Header())
	fields["size"] = hexutil.Uint64(block.Size())

	if inclTx {
		txs := block.Transactions()
		transactions := make([]interface{}, len(txs))
		for i, tx := range txs {
			transactions[i] = formatTx(i, tx)
		}
		fields["transactions"] = transactions
	}
	uncles := block.Uncles()
	uncleHashes := make([]common.Hash, len(uncles))
	for i, uncle := range uncles {
		uncleHashes[i] = uncle.Hash()
	}
	fields["uncles"] = uncleHashes
	if block.Withdrawals() != nil {
		fields["withdrawals"] = block.Withdrawals()
	}
	return fields
}

// RPCMarshalBlock converts the given block to the RPC output.
func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool, config *params.ChainConfig) map[string]interface{} {
	formatTx := func(idx int, tx *types.Transaction) interface{} {
		return tx.Hash()
	}
	if fullTx {
		formatTx = func(idx int, tx *types.Transaction) interface{} {
			return NewRPCTransactionFromBlockIndex(block, uint64(idx), config)
		}
	}
	return RPCMarshalBlockWithTransactions(block, inclTx, formatTx)
}

// NewRPCTransactionFromBlockIndex returns a transaction that will serialize to the RPC representation.
func NewRPCTransactionFromBlockIndex(block *types.Block, index uint64, config *params.ChainConfig) *rpctx.RPCTransaction {
	txs := block.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	return rpctx.NewRPCTransaction(txs[index], block.Hash(), block.NumberU64(), block.Time(), index, block.BaseFee(), config)
}

// NewRPCRawTransactionFromBlockIndex returns the bytes of a transaction given a block and a transaction index.
func NewRPCRawTransactionFromBlockIndex(block *types.Block, index uint64) hexutil.Bytes {
	txs := block.Transactions()
	if index >= uint64(len(txs)) {
		return nil
	}
	blob, _ := txs[index].MarshalBinary()
	return blob
}

const maxGetStorageSlots = 1024

// BlockChainBackend is the minimal backend required by Sila blockchain helpers.
type BlockChainBackend interface {
	ChainConfig() *params.ChainConfig
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
	BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error)
	GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error)
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	Pending() (*types.Block, types.Receipts, *state.StateDB)
	CurrentHeader() *types.Header
}

type ChainConfigInfo struct {
	ActivationTime  uint64                    `json:"activationTime"`
	BlobSchedule    *params.BlobConfig        `json:"blobSchedule"`
	ChainId         *hexutil.Big              `json:"chainId"`
	ForkId          hexutil.Bytes             `json:"forkId"`
	Precompiles     map[string]common.Address `json:"precompiles"`
	SystemContracts map[string]common.Address `json:"systemContracts"`
}

type ChainConfigResponse struct {
	Current *ChainConfigInfo `json:"current"`
	Next    *ChainConfigInfo `json:"next"`
	Last    *ChainConfigInfo `json:"last"`
}

func GetConfig(ctx context.Context, b BlockChainBackend) (*ChainConfigResponse, error) {
	genesis, err := b.HeaderByNumber(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to load genesis: %w", err)
	}
	assemble := func(c *params.ChainConfig, ts *uint64) *ChainConfigInfo {
		if ts == nil {
			return nil
		}
		t := *ts

		rules := c.Rules(c.LondonBlock, true, t)
		precompiles := make(map[string]common.Address)
		for addr, c := range vm.ActivePrecompiledContracts(rules) {
			precompiles[c.Name()] = addr
		}
		activationTime := t
		if genesis.Time >= t {
			activationTime = 0
		}
		forkid := forkid.NewID(c, types.NewBlockWithHeader(genesis), ^uint64(0), t).Hash
		return &ChainConfigInfo{
			ActivationTime:  activationTime,
			BlobSchedule:    c.BlobConfig(c.LatestFork(t)),
			ChainId:         (*hexutil.Big)(c.ChainID),
			ForkId:          forkid[:],
			Precompiles:     precompiles,
			SystemContracts: c.ActiveSystemContracts(t),
		}
	}
	c := b.ChainConfig()
	t := b.CurrentHeader().Time
	resp := ChainConfigResponse{
		Next:    assemble(c, c.Timestamp(c.LatestFork(t)+1)),
		Current: assemble(c, c.Timestamp(c.LatestFork(t))),
		Last:    assemble(c, c.Timestamp(c.LatestFork(^uint64(0)))),
	}
	if resp.Next == nil {
		resp.Last = nil
	}
	return &resp, nil
}

// ChainId returns the replay-protection chain id for the current SilaChain config.
func ChainId(b BlockChainBackend) *hexutil.Big {
	return (*hexutil.Big)(b.ChainConfig().ChainID)
}

// BlockNumber returns the block number of the chain head.
func BlockNumber(b BlockChainBackend) hexutil.Uint64 {
	header, _ := b.HeaderByNumber(context.Background(), rpc.LatestBlockNumber)
	return hexutil.Uint64(header.Number.Uint64())
}

// GetBalance returns the amount of wei for the given address in the state of the given block number or hash.
func GetBalance(ctx context.Context, b BlockChainBackend, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	balance := state.GetBalance(address).ToBig()
	return (*hexutil.Big)(balance), state.Error()
}

// GetCode returns the code stored at the given address in the state for the given block number or hash.
func GetCode(ctx context.Context, b BlockChainBackend, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	code := state.GetCode(address)
	return code, state.Error()
}

// DecodeStorageKey parses a hex-encoded 32-byte storage key.
func DecodeStorageKey(s string) (h common.Hash, inputLength int, err error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if (len(s) & 1) > 0 {
		s = "0" + s
	}
	if len(s) > 64 {
		return common.Hash{}, len(s) / 2, errors.New("storage key too long (want at most 32 bytes)")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return common.Hash{}, 0, errors.New("invalid hex in storage key")
	}
	return common.BytesToHash(b), len(b), nil
}

// GetStorageAt returns the storage from the state at the given address, key and block number or hash.
func GetStorageAt(ctx context.Context, b BlockChainBackend, address common.Address, hexKey string, blockNrOrHash rpc.BlockNumberOrHash) (hexutil.Bytes, error) {
	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	key, _, err := DecodeStorageKey(hexKey)
	if err != nil {
		return nil, err
	}
	res := state.GetState(address, key)
	return res[:], state.Error()
}

// GetStorageValues returns multiple storage slot values for multiple accounts at the given block.
func GetStorageValues(ctx context.Context, b BlockChainBackend, requests map[common.Address][]common.Hash, blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address][]hexutil.Bytes, error) {
	var totalSlots int
	for _, keys := range requests {
		totalSlots += len(keys)
		if totalSlots > maxGetStorageSlots {
			return nil, &ethapierrors.ClientLimitExceededError{Message: fmt.Sprintf("too many slots (max %d)", maxGetStorageSlots)}
		}
	}
	if totalSlots == 0 {
		return nil, &ethapierrors.InvalidParamsError{Message: "empty request"}
	}

	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	result := make(map[common.Address][]hexutil.Bytes, len(requests))
	for addr, keys := range requests {
		vals := make([]hexutil.Bytes, len(keys))
		for i, key := range keys {
			v := state.GetState(addr, key)
			vals[i] = v[:]
		}
		if err := state.Error(); err != nil {
			return nil, err
		}
		result[addr] = vals
	}
	return result, nil
}

// GetUncleByBlockNumberAndIndex returns the uncle block for the given block number and index.
func GetUncleByBlockNumberAndIndex(ctx context.Context, b BlockChainBackend, blockNr rpc.BlockNumber, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := b.BlockByNumber(ctx, blockNr)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return RPCMarshalBlock(block, false, false, b.ChainConfig()), nil
	}
	return nil, err
}

// GetUncleByBlockHashAndIndex returns the uncle block for the given block hash and index.
func GetUncleByBlockHashAndIndex(ctx context.Context, b BlockChainBackend, blockHash common.Hash, index hexutil.Uint) (map[string]interface{}, error) {
	block, err := b.BlockByHash(ctx, blockHash)
	if block != nil {
		uncles := block.Uncles()
		if index >= hexutil.Uint(len(uncles)) {
			return nil, nil
		}
		block = types.NewBlockWithHeader(uncles[index])
		return RPCMarshalBlock(block, false, false, b.ChainConfig()), nil
	}
	return nil, err
}

// GetUncleCountByBlockNumber returns number of uncles in the block for the given block number.
func GetUncleCountByBlockNumber(ctx context.Context, b BlockChainBackend, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	block, err := b.BlockByNumber(ctx, blockNr)
	if block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n, nil
	}
	return nil, err
}

// GetUncleCountByBlockHash returns number of uncles in the block for the given block hash.
func GetUncleCountByBlockHash(ctx context.Context, b BlockChainBackend, blockHash common.Hash) (*hexutil.Uint, error) {
	block, err := b.BlockByHash(ctx, blockHash)
	if block != nil {
		n := hexutil.Uint(len(block.Uncles()))
		return &n, nil
	}
	return nil, err
}

// RPCMarshalHeader converts the given header to the RPC output.
func RPCMarshalHeader(head *types.Header) map[string]interface{} {
	result := map[string]interface{}{
		"number":           (*hexutil.Big)(head.Number),
		"hash":             head.Hash(),
		"parentHash":       head.ParentHash,
		"nonce":            head.Nonce,
		"mixHash":          head.MixDigest,
		"sha3Uncles":       head.UncleHash,
		"logsBloom":        head.Bloom,
		"stateRoot":        head.Root,
		"miner":            head.Coinbase,
		"difficulty":       (*hexutil.Big)(head.Difficulty),
		"extraData":        hexutil.Bytes(head.Extra),
		"gasLimit":         hexutil.Uint64(head.GasLimit),
		"gasUsed":          hexutil.Uint64(head.GasUsed),
		"timestamp":        hexutil.Uint64(head.Time),
		"transactionsRoot": head.TxHash,
		"receiptsRoot":     head.ReceiptHash,
	}
	if head.BaseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(head.BaseFee)
	}
	if head.WithdrawalsHash != nil {
		result["withdrawalsRoot"] = head.WithdrawalsHash
	}
	if head.BlobGasUsed != nil {
		result["blobGasUsed"] = hexutil.Uint64(*head.BlobGasUsed)
	}
	if head.ExcessBlobGas != nil {
		result["excessBlobGas"] = hexutil.Uint64(*head.ExcessBlobGas)
	}
	if head.ParentBeaconRoot != nil {
		result["parentBeaconBlockRoot"] = head.ParentBeaconRoot
	}
	if head.RequestsHash != nil {
		result["requestsHash"] = head.RequestsHash
	}
	if head.SlotNumber != nil {
		result["slotNumber"] = hexutil.Uint64(*head.SlotNumber)
	}
	return result
}

// GetHeaderByNumber returns the requested canonical block header.
func GetHeaderByNumber(ctx context.Context, b BlockChainBackend, number rpc.BlockNumber) (map[string]interface{}, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if header != nil && err == nil {
		response := RPCMarshalHeader(header)
		if number == rpc.PendingBlockNumber {
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, err
	}
	return nil, err
}

// GetHeaderByHash returns the requested header by hash.
func GetHeaderByHash(ctx context.Context, b BlockChainBackend, hash common.Hash) map[string]interface{} {
	header, _ := b.HeaderByHash(ctx, hash)
	if header != nil {
		return RPCMarshalHeader(header)
	}
	return nil
}

// GetBlockByNumber returns the requested canonical block.
func GetBlockByNumber(ctx context.Context, b BlockChainBackend, number rpc.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	block, err := b.BlockByNumber(ctx, number)
	if block != nil && err == nil {
		response := RPCMarshalBlock(block, true, fullTx, b.ChainConfig())
		if number == rpc.PendingBlockNumber {
			for _, field := range []string{"hash", "nonce", "miner"} {
				response[field] = nil
			}
		}
		return response, nil
	}
	return nil, err
}

// GetBlockByHash returns the requested block.
func GetBlockByHash(ctx context.Context, b BlockChainBackend, hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	block, err := b.BlockByHash(ctx, hash)
	if block != nil {
		return RPCMarshalBlock(block, true, fullTx, b.ChainConfig()), nil
	}
	return nil, err
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block with the given block number.
func GetBlockTransactionCountByNumber(ctx context.Context, b BlockChainBackend, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	block, err := b.BlockByNumber(ctx, blockNr)
	if block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n, nil
	}
	return nil, err
}

// GetBlockTransactionCountByHash returns the number of transactions in the block with the given hash.
func GetBlockTransactionCountByHash(ctx context.Context, b BlockChainBackend, blockHash common.Hash) (*hexutil.Uint, error) {
	block, err := b.BlockByHash(ctx, blockHash)
	if block != nil {
		n := hexutil.Uint(len(block.Transactions()))
		return &n, nil
	}
	return nil, err
}

// GetTransactionByBlockNumberAndIndex returns the transaction for the given block number and index.
func GetTransactionByBlockNumberAndIndex(ctx context.Context, b BlockChainBackend, blockNr rpc.BlockNumber, index hexutil.Uint) (*rpctx.RPCTransaction, error) {
	block, err := b.BlockByNumber(ctx, blockNr)
	if block != nil {
		return NewRPCTransactionFromBlockIndex(block, uint64(index), b.ChainConfig()), nil
	}
	return nil, err
}

// GetTransactionByBlockHashAndIndex returns the transaction for the given block hash and index.
func GetTransactionByBlockHashAndIndex(ctx context.Context, b BlockChainBackend, blockHash common.Hash, index hexutil.Uint) (*rpctx.RPCTransaction, error) {
	block, err := b.BlockByHash(ctx, blockHash)
	if block != nil {
		return NewRPCTransactionFromBlockIndex(block, uint64(index), b.ChainConfig()), nil
	}
	return nil, err
}

// GetRawTransactionByBlockNumberAndIndex returns the raw transaction for the given block number and index.
func GetRawTransactionByBlockNumberAndIndex(ctx context.Context, b BlockChainBackend, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	block, _ := b.BlockByNumber(ctx, blockNr)
	if block != nil {
		return NewRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetRawTransactionByBlockHashAndIndex returns the raw transaction for the given block hash and index.
func GetRawTransactionByBlockHashAndIndex(ctx context.Context, b BlockChainBackend, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	block, _ := b.BlockByHash(ctx, blockHash)
	if block != nil {
		return NewRPCRawTransactionFromBlockIndex(block, uint64(index))
	}
	return nil
}

// GetBlockReceipts returns the block receipts for the given block hash, number, or tag.
func GetBlockReceipts(ctx context.Context, b BlockChainBackend, blockNrOrHash rpc.BlockNumberOrHash) ([]map[string]interface{}, error) {
	block, receipts, err := ReceiptsByBlockNumberOrHash(ctx, b, blockNrOrHash)
	if block == nil || err != nil {
		return nil, err
	}
	txs := block.Transactions()
	if len(txs) != len(receipts) {
		return nil, fmt.Errorf("receipts length mismatch: %d vs %d", len(txs), len(receipts))
	}
	signer := types.MakeSigner(b.ChainConfig(), block.Number(), block.Time())

	result := make([]map[string]interface{}, len(receipts))
	for i, receipt := range receipts {
		result[i] = rpctx.MarshalReceipt(receipt, block.Hash(), block.NumberU64(), signer, txs[i], i)
	}
	return result, nil
}

// ReceiptsByBlockNumberOrHash returns the block and receipts for the given block hash, number, or tag.
func ReceiptsByBlockNumberOrHash(ctx context.Context, b BlockChainBackend, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, types.Receipts, error) {
	var (
		err      error
		block    *types.Block
		receipts types.Receipts
	)
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		block, receipts, _ = b.Pending()
		if block == nil {
			return nil, nil, errors.New("pending receipts is not available")
		}
	} else {
		block, err = b.BlockByNumberOrHash(ctx, blockNrOrHash)
		if block == nil || err != nil {
			return nil, nil, err
		}
		receipts, err = b.GetReceipts(ctx, block.Hash())
		if err != nil {
			return nil, nil, err
		}
	}
	return block, receipts, nil
}
