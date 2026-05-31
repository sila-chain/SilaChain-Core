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
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

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

// RPCMarshalBlock converts the given block to the RPC output with transaction hashes only.
func RPCMarshalBlock(block *types.Block, inclTx bool, fullTx bool, config *params.ChainConfig) map[string]interface{} {
	return RPCMarshalBlockWithTransactions(block, inclTx, func(idx int, tx *types.Transaction) interface{} {
		return tx.Hash()
	})
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
