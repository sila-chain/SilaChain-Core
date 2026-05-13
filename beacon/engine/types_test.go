// Copyright 2025 The SilaChain Authors
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

package engine

import (
	"testing"

	"github.com/sila-org/sila/common"
	"github.com/sila-org/sila/core/types"
	"github.com/sila-org/sila/crypto/kzg4844"
)

func TestBlobs(t *testing.T) {
	var (
		emptyBlob          = new(kzg4844.Blob)
		emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
		emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
		emptyCellProof, _  = kzg4844.ComputeCellProofs(emptyBlob)
	)
	header := types.Header{}
	block := types.NewBlock(&header, &types.Body{}, nil, nil)

	sidecarWithoutCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, []kzg4844.Proof{emptyBlobProof})
	env := BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithoutCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 1 {
		t.Fatalf("Expect 1 proof in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}

	sidecarWithCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, emptyCellProof)
	env = BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 128 {
		t.Fatalf("Expect 128 proofs in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}
}
