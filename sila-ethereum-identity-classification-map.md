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

Status: implemented for Sila mainnet identity, with inherited Ethereum compatibility boundaries retained only where explicitly required.

SilaChain mainnet must not use Ethereum mainnet identity as its final public network identity.

Current Sila mainnet identity state:
- `params.SilaMainnetChainConfig` is the production Sila mainnet config.
- `params.SilaMainnetGenesisHash` is the production Sila mainnet genesis hash.
- `core.SilaDefaultGenesisBlock()` is the production Sila mainnet genesis block.
- `params.MainnetChainConfig`, `params.MainnetGenesisHash`, `core.DefaultGenesisBlock()`, and Ethereum bootnodes remain only as inherited compatibility boundaries where explicitly required.

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

## Sila Mainnet Genesis Verification

Status: verified by `sila dumpgenesis --mainnet`.

Confirmed Sila mainnet genesis values:
- ChainID: `2026`
- Network name: `sila-mainnet`
- DepositContractAddress: `0x0000000000000000000000000000000000000000`
- ExtraData: `SilaChain Mainnet`
- GasLimit: `30000000`
- BaseFeePerGas: `1000000000`
- Alloc: empty `{}`

This confirms Sila mainnet does not inherit the Ethereum mainnet deposit contract or Ethereum prealloc state.

Do not add a non-zero deposit contract address until a Sila-owned deposit contract and consensus genesis are finalized.
Do not add initial allocations unless a final Sila mainnet allocation policy is approved.

## Sila Filter/Log Checkpoints Status

Status: deferred until Sila chain-derived checkpoints exist.

`core/filtermaps/checkpoints.go` intentionally keeps the inherited Ethereum-family checkpoint JSON files for compatibility. No Sila checkpoint file is added until real Sila finalized-chain log checkpoints are generated from canonical Sila chain data.

Do not add mock checkpoint JSON files. Do not replace inherited checkpoint data without a generated Sila equivalent.
