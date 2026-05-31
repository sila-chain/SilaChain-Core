package proofapi

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
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
