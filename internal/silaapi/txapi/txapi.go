package txapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/sila-org/sila/crypto/kzg4844"
	"github.com/sila-org/sila/internal/silaapi"
	"github.com/sila-org/sila/internal/silaapi/addrlock"
	"github.com/sila-org/sila/internal/silaapi/blockapi"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	"github.com/sila-org/sila/internal/silaapi/txargs"
	"github.com/sila-org/sila/internal/silaapi/txfee"
	"github.com/sila-org/sila/log"
	"math/big"
	"time"

	"github.com/sila-org/sila/accounts"
	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/consensus"
	"github.com/sila-org/sila/core"
	"github.com/sila-org/sila/core/state"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/core/vm"
	"github.com/sila-org/sila/event"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/internal/silaapi/rpctx"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

type TransactionAPI struct {
	b         Backend
	nonceLock *addrlock.AddrLocker
	signer    types.Signer
}

// NewTransactionAPI creates a new RPC service with methods for interacting with transactions.
func NewTransactionAPI(b Backend, nonceLock *addrlock.AddrLocker) *TransactionAPI {
	// The signer used by the API should always be the 'latest' known one because we expect
	// signers to be backwards-compatible with old transactions.
	signer := types.LatestSigner(b.ChainConfig())
	return &TransactionAPI{b, nonceLock, signer}
}

func (api *TransactionAPI) Backend() Backend {
	return api.b
}

func (api *TransactionAPI) Signer() types.Signer {
	return api.signer
}

func (api *TransactionAPI) NonceLock() *addrlock.AddrLocker {
	return api.nonceLock
}

type Backend interface {
	blockapi.BlockChainBackend
	GetCanonicalTransaction(common.Hash) (bool, *types.Transaction, common.Hash, uint64, uint64)
	GetPoolTransaction(common.Hash) *types.Transaction
	TxIndexDone() bool
	HeaderByNumber(context.Context, rpc.BlockNumber) (*types.Header, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	Engine() consensus.Engine
	CurrentHeader() *types.Header
	CurrentBlock() *types.Header
	ChainConfig() *params.ChainConfig
	RPCTxFeeCap() float64
	UnprotectedAllowed() bool
	SendTx(context.Context, *types.Transaction) error
	GetCanonicalReceipt(*types.Transaction, common.Hash, uint64, uint64) (*types.Receipt, error)
	GetPoolTransactions() (types.Transactions, error)
	AccountManager() *accounts.Manager
	GetPoolNonce(context.Context, common.Address) (uint64, error)
	StateAndHeaderByNumberOrHash(context.Context, rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	GetEVM(context.Context, *state.StateDB, *types.Header, *vm.Config, *vm.BlockContext) *vm.EVM
	RPCGasCap() uint64
	SuggestGasTipCap(context.Context) (*big.Int, error)
	BlobBaseFee(context.Context) *big.Int
	SubscribeChainEvent(chan<- core.ChainEvent) event.Subscription
	RPCTxSyncDefaultTimeout() time.Duration
	RPCTxSyncMaxTimeout() time.Duration
}

func (api *TransactionAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*hexutil.Uint, error) {
	return blockapi.GetBlockTransactionCountByNumber(ctx, api.b, blockNr)
}

func (api *TransactionAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint, error) {
	return blockapi.GetBlockTransactionCountByHash(ctx, api.b, blockHash)
}

func (api *TransactionAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) (*rpctx.RPCTransaction, error) {
	return blockapi.GetTransactionByBlockNumberAndIndex(ctx, api.b, blockNr, index)
}

func (api *TransactionAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) (*rpctx.RPCTransaction, error) {
	return blockapi.GetTransactionByBlockHashAndIndex(ctx, api.b, blockHash, index)
}

func (api *TransactionAPI) GetRawTransactionByBlockNumberAndIndex(ctx context.Context, blockNr rpc.BlockNumber, index hexutil.Uint) hexutil.Bytes {
	return blockapi.GetRawTransactionByBlockNumberAndIndex(ctx, api.b, blockNr, index)
}

func (api *TransactionAPI) GetRawTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint) hexutil.Bytes {
	return blockapi.GetRawTransactionByBlockHashAndIndex(ctx, api.b, blockHash, index)
}

func (api *TransactionAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	return GetTransactionCount(ctx, api.b, address, blockNrOrHash)
}

func GetTransactionCount(ctx context.Context, b Backend, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Uint64, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok && blockNr == rpc.PendingBlockNumber {
		nonce, err := b.GetPoolNonce(ctx, address)
		if err != nil {
			return nil, err
		}
		return (*hexutil.Uint64)(&nonce), nil
	}
	state, _, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	nonce := state.GetNonce(address)
	return (*hexutil.Uint64)(&nonce), state.Error()
}

func (api *TransactionAPI) GetTransactionByHash(ctx context.Context, hash common.Hash) (*rpctx.RPCTransaction, error) {
	return GetTransactionByHash(ctx, api.b, hash)
}

// GetRawTransactionByHash returns the bytes of the transaction for the given hash.
func (api *TransactionAPI) GetRawTransactionByHash(ctx context.Context, hash common.Hash) (hexutil.Bytes, error) {
	return GetRawTransactionByHash(api.b, hash)
}

// GetTransactionReceipt returns the transaction receipt for the given transaction hash.
func (api *TransactionAPI) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	return GetTransactionReceipt(api.b, api.signer, hash)
}

func GetTransactionByHash(ctx context.Context, b Backend, hash common.Hash) (*rpctx.RPCTransaction, error) {
	found, tx, blockHash, blockNumber, index := b.GetCanonicalTransaction(hash)
	if !found {
		if tx := b.GetPoolTransaction(hash); tx != nil {
			return rpctx.NewRPCPendingTransaction(tx, b.CurrentHeader(), b.ChainConfig()), nil
		}
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	header, err := b.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	return rpctx.NewRPCTransaction(tx, blockHash, blockNumber, header.Time, index, header.BaseFee, b.ChainConfig()), nil
}

func GetRawTransactionByHash(b Backend, hash common.Hash) (hexutil.Bytes, error) {
	found, tx, _, _, _ := b.GetCanonicalTransaction(hash)
	if !found {
		if tx = b.GetPoolTransaction(hash); tx != nil {
			return tx.MarshalBinary()
		}
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	return tx.MarshalBinary()
}

func GetTransactionReceipt(b Backend, signer types.Signer, hash common.Hash) (map[string]interface{}, error) {
	found, tx, blockHash, blockNumber, index := b.GetCanonicalTransaction(hash)
	if !found {
		if !b.TxIndexDone() {
			return nil, ethapierrors.NewTxIndexingError()
		}
		return nil, nil
	}
	receipt, err := b.GetCanonicalReceipt(tx, blockHash, blockNumber, index)
	if err != nil {
		return nil, err
	}
	return rpctx.MarshalReceipt(receipt, blockHash, blockNumber, signer, tx, int(index)), nil
}

func PendingTransactions(b Backend, signer types.Signer) ([]*rpctx.RPCTransaction, error) {
	pending, err := b.GetPoolTransactions()
	if err != nil {
		return nil, err
	}
	accounts := make(map[common.Address]struct{})
	for _, wallet := range b.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			accounts[account.Address] = struct{}{}
		}
	}
	curHeader := b.CurrentHeader()
	transactions := make([]*rpctx.RPCTransaction, 0, len(pending))
	for _, tx := range pending {
		from, _ := types.Sender(signer, tx)
		if _, exists := accounts[from]; exists {
			transactions = append(transactions, rpctx.NewRPCPendingTransaction(tx, curHeader, b.ChainConfig()))
		}
	}
	return transactions, nil
}

func CurrentBlobSidecarVersion(b Backend) byte {
	h := b.CurrentHeader()
	if b.ChainConfig().IsOsaka(h.Number, h.Time) {
		return types.BlobSidecarVersion1
	}
	return types.BlobSidecarVersion0
}

func (api *TransactionAPI) SendTransaction(ctx context.Context, args txargs.TransactionArgs) (common.Hash, error) {
	return api.SendTransactionWithBlobError(ctx, args, errors.New("signing blob transactions not supported"))
}

func (api *TransactionAPI) SendTransactionWithBlobError(ctx context.Context, args txargs.TransactionArgs, blobTxErr error) (common.Hash, error) {
	if args.Nonce == nil {
		api.nonceLock.LockAddr(args.FromAddr())
		defer api.nonceLock.UnlockAddr(args.FromAddr())
	}
	if args.IsEIP4844() {
		return common.Hash{}, blobTxErr
	}
	return SendTransaction(ctx, api.b, args)
}

func (api *TransactionAPI) FillTransaction(ctx context.Context, args txargs.TransactionArgs) (*silaapi.SignTransactionResult, error) {
	return FillTransaction(ctx, api.b, args)
}

func (api *TransactionAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	return SendRawTransaction(ctx, api.b, input)
}

func (api *TransactionAPI) Sign(addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	return Sign(api.b, addr, data)
}

func (api *TransactionAPI) SignTransaction(ctx context.Context, args txargs.TransactionArgs) (*silaapi.SignTransactionResult, error) {
	return SignTransaction(ctx, api.b, args)
}

// SendRawTransaction will add the signed transaction to the transaction pool.
// The sender is responsible for signing the transaction and using the correct nonce.
func SendRawTransaction(ctx context.Context, b Backend, input hexutil.Bytes) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return common.Hash{}, err
	}

	if sc := tx.BlobTxSidecar(); sc != nil {
		exp := CurrentBlobSidecarVersion(b)
		if sc.Version == types.BlobSidecarVersion0 && exp == types.BlobSidecarVersion1 {
			if err := sc.ToV1(); err != nil {
				return common.Hash{}, fmt.Errorf("blob sidecar conversion failed: %v", err)
			}
			tx = tx.WithBlobTxSidecar(sc)
		}
	}

	return callapi.SubmitTransaction(ctx, b, tx)
}

type SidecarConfig struct {
	BlobSidecarAllowed bool
	BlobSidecarVersion byte
}

// SetDefaults fills in default values for unspecified tx fields.
func SetDefaults(args *txargs.TransactionArgs, ctx context.Context, b Backend, config SidecarConfig) error {
	if err := SetBlobTxSidecar(args, ctx, config); err != nil {
		return err
	}
	if err := txfee.SetFeeDefaults(args, ctx, b, b.CurrentHeader()); err != nil {
		return err
	}

	if args.Value == nil {
		args.Value = new(hexutil.Big)
	}
	if args.Nonce == nil {
		nonce, err := b.GetPoolNonce(ctx, args.FromAddr())
		if err != nil {
			return err
		}
		args.Nonce = (*hexutil.Uint64)(&nonce)
	}
	if args.Data != nil && args.Input != nil && !bytes.Equal(*args.Data, *args.Input) {
		return errors.New(`both "data" and "input" are set and not equal. Please use "input" to pass transaction call data`)
	}

	if args.BlobHashes != nil && len(args.BlobHashes) == 0 {
		return errors.New("need at least 1 blob for a blob transaction")
	}
	if args.BlobHashes != nil && len(args.BlobHashes) > params.BlobTxMaxBlobs {
		return fmt.Errorf("too many blobs in transaction (have=%d, max=%d)", len(args.BlobHashes), params.BlobTxMaxBlobs)
	}

	if args.To == nil {
		if args.BlobHashes != nil {
			return errors.New(`missing "to" in blob transaction`)
		}
		if len(args.DataBytes()) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
	}

	if args.Gas == nil {
		data := args.DataBytes()
		callArgs := txargs.TransactionArgs{
			From:                 args.From,
			To:                   args.To,
			GasPrice:             args.GasPrice,
			MaxFeePerGas:         args.MaxFeePerGas,
			MaxPriorityFeePerGas: args.MaxPriorityFeePerGas,
			Value:                args.Value,
			Data:                 (*hexutil.Bytes)(&data),
			AccessList:           args.AccessList,
			BlobFeeCap:           args.BlobFeeCap,
			BlobHashes:           args.BlobHashes,
			AuthorizationList:    args.AuthorizationList,
		}
		latestBlockNr := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		estimated, err := callapi.DoEstimateGas(ctx, b, callArgs, latestBlockNr, nil, nil, b.RPCGasCap())
		if err != nil {
			return err
		}
		args.Gas = &estimated
		log.Trace("Estimated gas usage automatically", "gas", args.Gas)
	}

	want := b.ChainConfig().ChainID
	if args.ChainID != nil {
		if have := (*big.Int)(args.ChainID); have.Cmp(want) != 0 {
			return fmt.Errorf("chainId does not match node's (have=%v, want=%v)", have, want)
		}
	} else {
		args.ChainID = (*hexutil.Big)(want)
	}
	return nil
}

// SetBlobTxSidecar adds the blob tx sidecar.
func SetBlobTxSidecar(args *txargs.TransactionArgs, ctx context.Context, config SidecarConfig) error {
	if args.Blobs == nil {
		return nil
	}
	if !config.BlobSidecarAllowed {
		return errors.New(`"blobs" is not supported for this RPC method`)
	}
	if args.Commitments == nil && args.Proofs != nil {
		return errors.New(`blob proofs provided while commitments were not`)
	} else if args.Commitments != nil && args.Proofs == nil {
		return errors.New(`blob commitments provided while proofs were not`)
	}

	n := len(args.Blobs)
	if args.BlobHashes != nil && len(args.BlobHashes) != n {
		return fmt.Errorf("number of blobs and hashes mismatch (have=%d, want=%d)", len(args.BlobHashes), n)
	}
	if args.Commitments != nil && len(args.Commitments) != n {
		return fmt.Errorf("number of blobs and commitments mismatch (have=%d, want=%d)", len(args.Commitments), n)
	}

	proofLen := n
	if config.BlobSidecarVersion == types.BlobSidecarVersion1 {
		proofLen = n * kzg4844.CellProofsPerBlob
	}
	if args.Proofs != nil && len(args.Proofs) != proofLen {
		if len(args.Proofs) != n {
			return fmt.Errorf("number of blobs and proofs mismatch (have=%d, want=%d)", len(args.Proofs), proofLen)
		}
		log.Debug("Unset legacy commitments and proofs", "blobs", n, "proofs", len(args.Proofs))
		args.Commitments, args.Proofs = nil, nil
	}

	if args.Commitments == nil {
		var (
			commitments = make([]kzg4844.Commitment, n)
			proofs      = make([]kzg4844.Proof, 0, proofLen)
		)
		for i := range args.Blobs {
			c, err := kzg4844.BlobToCommitment(&args.Blobs[i])
			if err != nil {
				return fmt.Errorf("blobs[%d]: error computing commitment: %v", i, err)
			}
			commitments[i] = c

			switch config.BlobSidecarVersion {
			case types.BlobSidecarVersion0:
				p, err := kzg4844.ComputeBlobProof(&args.Blobs[i], c)
				if err != nil {
					return fmt.Errorf("blobs[%d]: error computing proof: %v", i, err)
				}
				proofs = append(proofs, p)
			case types.BlobSidecarVersion1:
				ps, err := kzg4844.ComputeCellProofs(&args.Blobs[i])
				if err != nil {
					return fmt.Errorf("blobs[%d]: error computing proof: %v", i, err)
				}
				proofs = append(proofs, ps...)
			}
		}
		args.Commitments = commitments
		args.Proofs = proofs
	} else {
		switch config.BlobSidecarVersion {
		case types.BlobSidecarVersion0:
			for i := range args.Blobs {
				if err := kzg4844.VerifyBlobProof(&args.Blobs[i], args.Commitments[i], args.Proofs[i]); err != nil {
					return fmt.Errorf("failed to verify blob proof: %v", err)
				}
			}
		case types.BlobSidecarVersion1:
			if err := kzg4844.VerifyCellProofs(args.Blobs, args.Commitments, args.Proofs); err != nil {
				return fmt.Errorf("failed to verify blob proof: %v", err)
			}
		}
	}

	hashes := make([]common.Hash, n)
	hasher := sha256.New()
	for i, c := range args.Commitments {
		hashes[i] = kzg4844.CalcBlobHashV1(hasher, &c)
	}
	if args.BlobHashes != nil {
		for i, h := range hashes {
			if h != args.BlobHashes[i] {
				return fmt.Errorf("blob hash verification failed (have=%s, want=%s)", args.BlobHashes[i], h)
			}
		}
	} else {
		args.BlobHashes = hashes
	}
	return nil
}

func SignTransaction(ctx context.Context, b Backend, args txargs.TransactionArgs) (*silaapi.SignTransactionResult, error) {
	if args.Gas == nil {
		return nil, errors.New("gas not specified")
	}
	if args.GasPrice == nil && (args.MaxPriorityFeePerGas == nil || args.MaxFeePerGas == nil) {
		return nil, errors.New("missing gasPrice or maxFeePerGas/maxPriorityFeePerGas")
	}
	if args.Nonce == nil {
		return nil, errors.New("nonce not specified")
	}
	sidecarVersion := CurrentBlobSidecarVersion(b)
	config := SidecarConfig{
		BlobSidecarAllowed: true,
		BlobSidecarVersion: sidecarVersion,
	}
	if err := SetDefaults(&args, ctx, b, config); err != nil {
		return nil, err
	}
	tx := args.ToTransaction(types.DynamicFeeTxType)
	if err := txfee.CheckTxFee(tx.GasPrice(), tx.Gas(), b.RPCTxFeeCap()); err != nil {
		return nil, err
	}

	account := accounts.Account{Address: args.FromAddr()}
	wallet, err := b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	signed, err := wallet.SignTx(account, tx, b.ChainConfig().ChainID)
	if err != nil {
		return nil, err
	}
	if args.IsEIP4844() {
		signed = signed.WithBlobTxSidecar(types.NewBlobTxSidecar(sidecarVersion, args.Blobs, args.Commitments, args.Proofs))
	}
	data, err := signed.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &silaapi.SignTransactionResult{Raw: data, Tx: signed}, nil
}

func Resend(ctx context.Context, b Backend, signer types.Signer, sendArgs txargs.TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	if sendArgs.Nonce == nil {
		return common.Hash{}, errors.New("missing transaction nonce in transaction spec")
	}
	if err := SetDefaults(&sendArgs, ctx, b, SidecarConfig{}); err != nil {
		return common.Hash{}, err
	}
	matchTx := sendArgs.ToTransaction(types.DynamicFeeTxType)

	price := matchTx.GasPrice()
	if gasPrice != nil {
		price = gasPrice.ToInt()
	}
	gas := matchTx.Gas()
	if gasLimit != nil {
		gas = uint64(*gasLimit)
	}
	if err := txfee.CheckTxFee(price, gas, b.RPCTxFeeCap()); err != nil {
		return common.Hash{}, err
	}
	pending, err := b.GetPoolTransactions()
	if err != nil {
		return common.Hash{}, err
	}
	for _, p := range pending {
		wantSigHash := signer.Hash(matchTx)
		pFrom, err := types.Sender(signer, p)
		if err == nil && pFrom == sendArgs.FromAddr() && signer.Hash(p) == wantSigHash {
			if gasPrice != nil && (*big.Int)(gasPrice).Sign() != 0 {
				sendArgs.GasPrice = gasPrice
			}
			if gasLimit != nil && *gasLimit != 0 {
				sendArgs.Gas = gasLimit
			}
			tx := sendArgs.ToTransaction(types.DynamicFeeTxType)
			account := accounts.Account{Address: sendArgs.FromAddr()}
			wallet, err := b.AccountManager().Find(account)
			if err != nil {
				return common.Hash{}, err
			}
			signedTx, err := wallet.SignTx(account, tx, b.ChainConfig().ChainID)
			if err != nil {
				return common.Hash{}, err
			}
			if err = b.SendTx(ctx, signedTx); err != nil {
				return common.Hash{}, err
			}
			return signedTx.Hash(), nil
		}
	}
	return common.Hash{}, fmt.Errorf("transaction %#x not found", matchTx.Hash())
}

func FillTransaction(ctx context.Context, b Backend, args txargs.TransactionArgs) (*silaapi.SignTransactionResult, error) {
	config := SidecarConfig{
		BlobSidecarAllowed: true,
		BlobSidecarVersion: CurrentBlobSidecarVersion(b),
	}
	if err := SetDefaults(&args, ctx, b, config); err != nil {
		return nil, err
	}
	tx := args.ToTransaction(types.DynamicFeeTxType)
	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return &silaapi.SignTransactionResult{Raw: data, Tx: tx}, nil
}

func SendTransaction(ctx context.Context, b Backend, args txargs.TransactionArgs) (common.Hash, error) {
	account := accounts.Account{Address: args.FromAddr()}

	wallet, err := b.AccountManager().Find(account)
	if err != nil {
		return common.Hash{}, err
	}
	if args.IsEIP4844() {
		return common.Hash{}, errors.New("signing blob transactions not supported")
	}
	if err := SetDefaults(&args, ctx, b, SidecarConfig{}); err != nil {
		return common.Hash{}, err
	}
	tx := args.ToTransaction(types.DynamicFeeTxType)
	signed, err := wallet.SignTx(account, tx, b.ChainConfig().ChainID)
	if err != nil {
		return common.Hash{}, err
	}
	return callapi.SubmitTransaction(ctx, b, signed)
}

func (api *TransactionAPI) PendingTransactions() ([]*rpctx.RPCTransaction, error) {
	return PendingTransactions(api.b, api.signer)
}

func (api *TransactionAPI) Resend(ctx context.Context, sendArgs txargs.TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error) {
	return Resend(ctx, api.b, api.signer, sendArgs, gasPrice, gasLimit)
}

func (api *TransactionAPI) SendRawTransactionSync(ctx context.Context, input hexutil.Bytes, timeoutMs *uint64, subClosedErr error, timeoutErr func(common.Hash, time.Duration) error) (map[string]interface{}, error) {
	return SendRawTransactionSync(ctx, api.b, input, timeoutMs, subClosedErr, timeoutErr)
}

func SignWithAccount(b Backend, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	account := accounts.Account{Address: addr}
	wallet, err := b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	return wallet.SignTx(account, tx, b.ChainConfig().ChainID)
}

func SendRawTransactionSync(ctx context.Context, b Backend, input hexutil.Bytes, timeoutMs *uint64, subClosedErr error, timeoutErr func(common.Hash, time.Duration) error) (map[string]interface{}, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(input); err != nil {
		return nil, err
	}

	if sc := tx.BlobTxSidecar(); sc != nil {
		exp := CurrentBlobSidecarVersion(b)
		if sc.Version == types.BlobSidecarVersion0 && exp == types.BlobSidecarVersion1 {
			if err := sc.ToV1(); err != nil {
				return nil, fmt.Errorf("blob sidecar conversion failed: %v", err)
			}
			tx = tx.WithBlobTxSidecar(sc)
		}
	}

	ch := make(chan core.ChainEvent, 128)
	sub := b.SubscribeChainEvent(ch)
	defer sub.Unsubscribe()

	hash, err := callapi.SubmitTransaction(ctx, b, tx)
	if err != nil {
		return nil, err
	}

	maxTimeout := b.RPCTxSyncMaxTimeout()
	defaultTimeout := b.RPCTxSyncDefaultTimeout()
	timeout := defaultTimeout
	if timeoutMs != nil && *timeoutMs > 0 {
		req := time.Duration(*timeoutMs) * time.Millisecond
		if req > maxTimeout {
			timeout = maxTimeout
		} else {
			timeout = req
		}
	}
	receiptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if r, err := GetTransactionReceipt(b, types.LatestSigner(b.ChainConfig()), hash); err == nil && r != nil {
		return r, nil
	}

	for {
		select {
		case <-receiptCtx.Done():
			if errors.Is(receiptCtx.Err(), context.DeadlineExceeded) {
				return nil, timeoutErr(hash, timeout)
			}
			return nil, receiptCtx.Err()

		case err, ok := <-sub.Err():
			if !ok {
				return nil, subClosedErr
			}
			return nil, err

		case ev, ok := <-ch:
			if !ok {
				return nil, subClosedErr
			}
			rs, txs := ev.Receipts, ev.Transactions
			if len(rs) == 0 || len(rs) != len(txs) {
				continue
			}
			for i := range rs {
				if rs[i].TxHash == hash {
					if rs[i].BlockNumber != nil && rs[i].BlockHash != (common.Hash{}) {
						signer := types.LatestSigner(b.ChainConfig())
						return rpctx.MarshalReceipt(
							rs[i],
							rs[i].BlockHash,
							rs[i].BlockNumber.Uint64(),
							signer,
							txs[i],
							int(rs[i].TransactionIndex),
						), nil
					}
					return GetTransactionReceipt(b, types.LatestSigner(b.ChainConfig()), hash)
				}
			}
		}
	}
}

func Sign(b Backend, addr common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	account := accounts.Account{Address: addr}
	wallet, err := b.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	signature, err := wallet.SignText(account, data)
	if err == nil {
		signature[64] += 27
	}
	return signature, err
}
