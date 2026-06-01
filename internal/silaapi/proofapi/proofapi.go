package proofapi

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
	"github.com/sila-org/sila/crypto"
	ethapierrors "github.com/sila-org/sila/internal/silaapi/errors"
	"github.com/sila-org/sila/rpc"
)

// AccountResult structs for GetProof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// ProofList implements ethdb.KeyValueWriter and collects the proofs as
// hex-strings for delivery to rpc-caller.
type ProofList []string

func (n *ProofList) Put(key []byte, value []byte) error {
	*n = append(*n, hexutil.Encode(value))
	return nil
}

func (n *ProofList) Delete(key []byte) error {
	panic("not supported")
}

// DecodeStorageKey parses a hex-encoded 32-byte hash.
// For legacy compatibility reasons, we parse these keys leniently,
// with the 0x prefix being optional.
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

const maxGetProofKeys = 1024

type Backend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
}

// GetProof returns the Merkle-proof for a given account and optionally some storage keys.
func GetProof(ctx context.Context, b Backend, address common.Address, storageKeys []string, blockNrOrHash rpc.BlockNumberOrHash) (*AccountResult, error) {
	if len(storageKeys) > maxGetProofKeys {
		return nil, &ethapierrors.InvalidParamsError{Message: fmt.Sprintf("too many storage keys requested (max %d, got %d)", maxGetProofKeys, len(storageKeys))}
	}
	var (
		keys         = make([]common.Hash, len(storageKeys))
		keyLengths   = make([]int, len(storageKeys))
		storageProof = make([]StorageResult, len(storageKeys))
	)
	// Deserialize all keys. This prevents state access on invalid input.
	for i, hexKey := range storageKeys {
		var err error
		keys[i], keyLengths[i], err = DecodeStorageKey(hexKey)
		if err != nil {
			return nil, &ethapierrors.InvalidParamsError{Message: fmt.Sprintf("%v: %q", err, hexKey)}
		}
	}
	statedb, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if statedb == nil || err != nil {
		return nil, err
	}
	codeHash := statedb.GetCodeHash(address)
	storageRoot := statedb.GetStorageRoot(address)

	if len(keys) > 0 {
		var storageTrie state.Trie
		if storageRoot != types.EmptyRootHash && storageRoot != (common.Hash{}) {
			st, err := statedb.Database().OpenStorageTrie(header.Root, address, storageRoot, nil)
			if err != nil {
				return nil, err
			}
			storageTrie = st
		}
		// Create the proofs for the storageKeys.
		for i, key := range keys {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			// Output key encoding is a bit special: if the input was a 32-byte hash, it is
			// returned as such. Otherwise, we apply the QUANTITY encoding mandated by the
			// JSON-RPC spec for getProof. This behavior exists to preserve backwards
			// compatibility with older client versions.
			var outputKey string
			if keyLengths[i] != 32 {
				outputKey = hexutil.EncodeBig(key.Big())
			} else {
				outputKey = hexutil.Encode(key[:])
			}
			if storageTrie == nil {
				storageProof[i] = StorageResult{outputKey, &hexutil.Big{}, []string{}}
				continue
			}
			var proof ProofList
			if err := storageTrie.Prove(crypto.Keccak256(key.Bytes()), &proof); err != nil {
				return nil, err
			}
			value := (*hexutil.Big)(statedb.GetState(address, key).Big())
			storageProof[i] = StorageResult{outputKey, value, proof}
		}
	}
	// Create the accountProof.
	tr, err := statedb.Database().OpenTrie(header.Root)
	if err != nil {
		return nil, err
	}
	var accountProof ProofList
	if err := tr.Prove(crypto.Keccak256(address.Bytes()), &accountProof); err != nil {
		return nil, err
	}
	balance := statedb.GetBalance(address).ToBig()
	return &AccountResult{
		Address:      address,
		AccountProof: accountProof,
		Balance:      (*hexutil.Big)(balance),
		CodeHash:     codeHash,
		Nonce:        hexutil.Uint64(statedb.GetNonce(address)),
		StorageHash:  storageRoot,
		StorageProof: storageProof,
	}, statedb.Error()
}
