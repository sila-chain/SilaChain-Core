# SilaChain Ethereum Identity Classification Map

## 1. Runtime compatibility
Purpose: Keep temporarily. Required for protocol/runtime compatibility.
Examples:
- eth/*
- ethclient/*
- eth_* RPC method names

## 2. RPC compatibility
Purpose: Wrap behind Sila APIs before removing.
Examples:
- eth_call
- eth_chainId
- eth_getLogs
- eth_sendRawTransaction
- engine_* legacy compatibility

## 3. ABI / contract binding layer
Purpose: Needs Sila-native naming plan before changes.
Examples:
- accounts/abi/*
- accounts/abi/abigen/*
- accounts/abi/bind/*

## 4. Accounts / signing layer
Purpose: High risk. Standards must be isolated, not blindly renamed.
Examples:
- Ethereum Signed Message
- keystore
- HD path coin type 60
- external signer

## 5. Hardware wallet compatibility
Purpose: Keep as external protocol compatibility.
Examples:
- Ledger Ethereum app
- Trezor Ethereum messages
- messages-ethereum.proto

## 6. User-facing docs and wording
Purpose: Safe cleanup area.
Examples:
- README.md
- cmd/* README/tutorial/help text
- CLI Usage strings

## 7. Tests and generated fixtures
Purpose: Rename only after runtime/API replacement.
Examples:
- testdata
- generated bindings
- expected RPC JSON files

## 8. External standards that must remain or be isolated
Purpose: Do not delete until Sila alternative exists.
Examples:
- Ethereum Node Record
- Web3 Secret Storage
- Ethereum Signed Message
- eth_* JSON-RPC compatibility
