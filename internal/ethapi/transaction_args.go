// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.
//
// The SilaChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The SilaChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the SilaChain library. If not, see <http://www.gnu.org/licenses/>.

package ethapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/common/hexutil"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/crypto/kzg4844"
	"github.com/sila-org/sila/internal/silaapi/callapi"
	"github.com/sila-org/sila/internal/silaapi/txargs"
	"github.com/sila-org/sila/internal/silaapi/txfee"
	"github.com/sila-org/sila/log"
	"github.com/sila-org/sila/params"
	"github.com/sila-org/sila/rpc"
)

// TransactionArgs represents the arguments to construct a new transaction
// or a message call.
type TransactionArgs = txargs.TransactionArgs

// sidecarConfig defines the options for deriving missing fields of transactions.
type sidecarConfig struct {
	// This configures whether blobs are allowed to be passed and
	// the associated sidecar version should be attached.
	blobSidecarAllowed bool
	blobSidecarVersion byte
}

// setDefaults fills in default values for unspecified tx fields.
func setDefaults(args *TransactionArgs, ctx context.Context, b Backend, config sidecarConfig) error {
	if err := setBlobTxSidecar(args, ctx, config); err != nil {
		return err
	}
	if err := setFeeDefaults(args, ctx, b, b.CurrentHeader()); err != nil {
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

	// BlobTx fields
	if args.BlobHashes != nil && len(args.BlobHashes) == 0 {
		return errors.New("need at least 1 blob for a blob transaction")
	}
	if args.BlobHashes != nil && len(args.BlobHashes) > params.BlobTxMaxBlobs {
		return fmt.Errorf("too many blobs in transaction (have=%d, max=%d)", len(args.BlobHashes), params.BlobTxMaxBlobs)
	}

	// create check
	if args.To == nil {
		if args.BlobHashes != nil {
			return errors.New(`missing "to" in blob transaction`)
		}
		if len(args.DataBytes()) == 0 {
			return errors.New(`contract creation without any data provided`)
		}
	}

	if args.Gas == nil {
		// These fields are immutable during the estimation, safe to
		// pass the pointer directly.
		data := args.DataBytes()
		callArgs := TransactionArgs{
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

	// If chain id is provided, ensure it matches the local chain id. Otherwise, set the local
	// chain id as the default.
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

// setFeeDefaults fills in default fee values for unspecified tx fields.
func setFeeDefaults(args *TransactionArgs, ctx context.Context, b Backend, head *types.Header) error {
	return txfee.SetFeeDefaults(args, ctx, b, head)
}

// setBlobTxSidecar adds the blob tx
func setBlobTxSidecar(args *TransactionArgs, ctx context.Context, config sidecarConfig) error {
	// No blobs, we're done.
	if args.Blobs == nil {
		return nil
	}

	// Passing blobs is not allowed in all contexts, only in specific methods.
	if !config.blobSidecarAllowed {
		return errors.New(`"blobs" is not supported for this RPC method`)
	}

	// Assume user provides either only blobs (w/o hashes), or
	// blobs together with commitments and proofs.
	if args.Commitments == nil && args.Proofs != nil {
		return errors.New(`blob proofs provided while commitments were not`)
	} else if args.Commitments != nil && args.Proofs == nil {
		return errors.New(`blob commitments provided while proofs were not`)
	}

	// len(blobs) == len(commitments) == len(hashes)
	n := len(args.Blobs)
	if args.BlobHashes != nil && len(args.BlobHashes) != n {
		return fmt.Errorf("number of blobs and hashes mismatch (have=%d, want=%d)", len(args.BlobHashes), n)
	}
	if args.Commitments != nil && len(args.Commitments) != n {
		return fmt.Errorf("number of blobs and commitments mismatch (have=%d, want=%d)", len(args.Commitments), n)
	}

	// if V0: len(blobs) == len(proofs)
	// if V1: len(blobs) == len(proofs) * 128
	proofLen := n
	if config.blobSidecarVersion == types.BlobSidecarVersion1 {
		proofLen = n * kzg4844.CellProofsPerBlob
	}
	if args.Proofs != nil && len(args.Proofs) != proofLen {
		if len(args.Proofs) != n {
			return fmt.Errorf("number of blobs and proofs mismatch (have=%d, want=%d)", len(args.Proofs), proofLen)
		}
		// Unset the commitments and proofs, as they may be submitted in the legacy format
		log.Debug("Unset legacy commitments and proofs", "blobs", n, "proofs", len(args.Proofs))
		args.Commitments, args.Proofs = nil, nil
	}

	// Generate commitments and proofs if they are missing, or validate them if they
	// are provided.
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

			switch config.blobSidecarVersion {
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
		switch config.blobSidecarVersion {
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

	// Generate blob hashes if they are missing, or validate them if they are provided.
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
