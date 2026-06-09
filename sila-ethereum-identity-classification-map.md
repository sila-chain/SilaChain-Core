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

## Sila Mainnet Bootnodes and DNS Status

Status: intentionally empty until Sila-owned production bootnodes and DNS discovery are provisioned.

`params.SilaMainnetBootnodes` is the only default bootnode list used by Sila mainnet paths and devp2p discovery tooling. It remains empty until real Sila-owned bootnodes are available.

`params.KnownDNSNetwork(params.SilaMainnetGenesisHash, protocol)` intentionally returns an empty string to prevent inherited Ethereum DNS discovery from being used by Sila mainnet.

Do not add Ethereum bootnodes or Ethereum DNS discovery back to Sila mainnet defaults. Do not add placeholder bootnodes.

## Sila Mainnet Build Verification

Status: verified on HEAD `ccca54708`.

Confirmed build command:
- `go build -o .\build\bin\sila.exe .\cmd\sila`

Confirmed binary identity:
- Name: `Sila`
- Version: `1.17.3-unstable`
- Git Commit: `ccca547082cd7d6447f15c75a33b8106a0307671`
- Go Version: `go1.26.1`
- OS/Arch: `windows/amd64`

Confirmed `sila dumpgenesis --mainnet` values:
- ChainID: `2026`
- ExtraData: `SilaChain Mainnet`
- DepositContractAddress: `0x0000000000000000000000000000000000000000`
- GasLimit: `30000000`
- BaseFeePerGas: `1000000000`
- Alloc: empty `{}`

This confirms the current Sila binary builds successfully and exposes the Sila mainnet genesis baseline.

## Sila Production Bootnodes Provisioning Plan

Status: pending real Sila-owned bootnode infrastructure.

Production bootnodes must be added only after real Sila nodes are provisioned, reachable, and verified. The bootnode list must not contain Ethereum nodes, temporary nodes, private local nodes, or placeholder ENRs/enodes.

Required bootnode acceptance checks:
1. Generate Sila-owned node keys.
2. Start persistent Sila bootnodes on stable public infrastructure.
3. Verify each node advertises the correct public IP and UDP/TCP discovery ports.
4. Verify each enode/ENR is reachable from an external network.
5. Verify the node participates in Sila discovery without using inherited Ethereum DNS.
6. Add only verified Sila enode URLs to `params.SilaMainnetBootnodes`.
7. Run `go test ./params ./cmd/utils ./cmd/devp2p ./cmd/sila ./core/... ./cmd/...`.
8. Rebuild `sila.exe` and verify `sila version` and `sila dumpgenesis --mainnet`.
9. Commit, push, and tag only after all checks pass.

Do not add any bootnode value until it has passed the acceptance checks above.

## Sila Production DNS Discovery Provisioning Plan

Status: pending Sila-owned DNS discovery infrastructure.

Sila mainnet DNS discovery must remain disabled until a Sila-owned ENR tree is created, signed, published, and verified. The Sila mainnet path must not use inherited Ethereum DNS discovery.

Required DNS discovery acceptance checks:
1. Provision a Sila-owned DNS zone for discovery.
2. Generate and protect the ENR tree signing key.
3. Build the ENR tree only from verified Sila production bootnodes.
4. Publish the ENR tree under a Sila-owned DNS name.
5. Verify DNS resolution from an external network.
6. Verify `params.KnownDNSNetwork(params.SilaMainnetGenesisHash, protocol)` returns only the Sila-owned discovery URL.
7. Verify Sila startup uses the Sila DNS URL only when configured for Sila mainnet.
8. Run `go test ./params ./cmd/utils ./cmd/sila ./p2p/... ./core/... ./cmd/...`.
9. Rebuild `sila.exe` and verify `sila version` and `sila dumpgenesis --mainnet`.
10. Commit, push, and tag only after all checks pass.

Do not enable DNS discovery for Sila mainnet until the DNS zone, ENR tree, and signing key are all Sila-owned and verified.

## Sila Chain-Derived Checkpoints Generation Plan

Status: pending canonical Sila chain data.

Sila filter/log checkpoints must be generated only from finalized canonical Sila chain data. The inherited Ethereum-family checkpoint files remain compatibility data until Sila checkpoints are generated and verified.

Required checkpoint acceptance checks:
1. Run a canonical Sila chain with finalized blocks.
2. Generate log/filter checkpoints from Sila chain data only.
3. Verify checkpoint block hashes against the canonical Sila chain.
4. Verify checkpoint first-index values against Sila log history.
5. Add Sila checkpoint data only after it is generated from real Sila chain history.
6. Do not add empty, placeholder, copied, or Ethereum-derived checkpoint data.
7. Run `go test ./core/filtermaps ./core/... ./cmd/...`.
8. Commit, push, and tag only after all checks pass.

Do not replace `core/filtermaps` checkpoint data until Sila-generated checkpoints exist and pass verification.

## Sila Bootnode Command Template

Status: command template only. Do not run for production until a real server and public IP are available.

Production key generation must happen outside the repository on the target bootnode host:

- `devp2p.exe key generate <secure-keyfile>`
- `devp2p.exe key to-enode --ip <PUBLIC_IP> --tcp 30303 --udp 30303 <secure-keyfile>`
- `devp2p.exe key to-enr --ip <PUBLIC_IP> --tcp 30303 --udp 30303 <secure-keyfile>`
- `devp2p.exe discv4 listen --nodekey <hex-nodekey> --addr :30303 --extaddr <PUBLIC_IP>:30303`

Do not commit node keys. Do not store production node keys inside the repository.
