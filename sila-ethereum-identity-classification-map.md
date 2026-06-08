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

## Sila Mainnet Specification Draft

Status: design draft only. No runtime behavior is changed by this section.

SilaChain mainnet must not use Ethereum mainnet identity as its final public network identity.

Current inherited compatibility state:
- `params.MainnetChainConfig` still uses `ChainID: big.NewInt(1)`.
- `params.MainnetGenesisHash` still points to the inherited Ethereum mainnet genesis hash.
- `core.DefaultGenesisBlock()` still builds from `params.MainnetChainConfig`.
- `params.MainnetBootnodes` and DNS discovery are still inherited public Ethereum network discovery values.

Approved target direction:
- Sila mainnet should use an independent chain identity.
- Proposed final ChainID: `2026`.
- Proposed final NetworkID: `2026`.
- Sila mainnet must receive its own genesis block, genesis hash, bootnodes, and DNS discovery configuration.
- Existing Ethereum/Trezor/Ledger compatibility protocols must not be renamed or broken.
- Ethereum EVM logic must remain unchanged; only Sila network identity should be introduced.

Required implementation order:
1. Define Sila mainnet chain configuration.
2. Define Sila mainnet genesis.
3. Compute and record Sila mainnet genesis hash.
4. Replace inherited mainnet bootnodes with Sila-owned bootnodes.
5. Disable or replace inherited Ethereum DNS discovery for Sila mainnet.
6. Test `sila init`, `sila dumpgenesis`, chain ID reporting, and selected RPC paths.
7. Commit and push only after tests pass.

Do not directly change `ChainID` alone. ChainID, genesis, genesis hash, bootnodes, and discovery must move together.

