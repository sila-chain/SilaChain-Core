// Copyright 2026 The SilaChain Authors
// This file is part of the SilaChain library.

// Package silaapi is the Sila execution API boundary.
//
// Code outside this package should depend on Sila API boundary packages instead
// of importing internal/ethapi directly. The aliases and wrappers here preserve
// the upstream execution behavior while allowing Sila-facing packages to avoid
// direct ethapi coupling.
package silaapi
