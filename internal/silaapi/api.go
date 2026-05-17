// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

package silaapi

import ethapi "github.com/sila-org/sila/internal/ethapi"

type Backend = ethapi.Backend
type NetAPI = ethapi.NetAPI

var GetAPIs = ethapi.GetAPIs
var NewNetAPI = ethapi.NewNetAPI
var RPCMarshalBlock = ethapi.RPCMarshalBlock
var DoCall = ethapi.DoCall
var DoEstimateGas = ethapi.DoEstimateGas
var SubmitTransaction = ethapi.SubmitTransaction
