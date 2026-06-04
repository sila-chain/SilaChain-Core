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
	"github.com/sila-org/sila/internal/silaapi/txapi"
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
	blobSidecarAllowed bool
	blobSidecarVersion byte
}

// setDefaults fills in default values for unspecified tx fields.
func setDefaults(args *TransactionArgs, ctx context.Context, b Backend, config sidecarConfig) error {
	return txapi.SetDefaults(args, ctx, b, txapi.SidecarConfig{
		BlobSidecarAllowed: config.blobSidecarAllowed,
		BlobSidecarVersion: config.blobSidecarVersion,
	})
}

// setFeeDefaults fills in default fee values for unspecified tx fields.
func setFeeDefaults(args *TransactionArgs, ctx context.Context, b Backend, head *types.Header) error {
	return txfee.SetFeeDefaults(args, ctx, b, head)
}

// setBlobTxSidecar adds the blob tx
func setBlobTxSidecar(args *TransactionArgs, ctx context.Context, config sidecarConfig) error {
	return txapi.SetBlobTxSidecar(args, ctx, txapi.SidecarConfig{
		BlobSidecarAllowed: config.blobSidecarAllowed,
		BlobSidecarVersion: config.blobSidecarVersion,
	})
}
