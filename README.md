# SilaChain

Golang execution layer implementation of the SilaChain protocol, derived from Ethereum architecture.

[![API Reference](https://pkg.go.dev/badge/github.com/sila-org/sila.svg)](https://pkg.go.dev/github.com/sila-org/sila?tab=doc)
[![Go Report Card](https://goreportcard.com/badge/github.com/sila-org/sila)](https://goreportcard.com/report/github.com/sila-org/sila)

Automated builds are available for stable releases and unstable main-branch builds.
Archives are published at https://sila-blockchain.org/downloads/.

## Building the source

Building `sila` requires Go version 1.23 or later and a C compiler.

```shell
make sila
```

To build the full suite of utilities:

```shell
make all
```

## Executables

| Command | Description |
| :--: | --- |
| **`sila`** | Main SilaChain CLI client. It is the entry point into the SilaChain network and can run as a full node, archive node, or light node. It exposes JSON-RPC endpoints over HTTP, WebSocket, and IPC transports. Use `sila --help` for command line options. |
| `clef` | Stand-alone signing tool, usable as a backend signer for `sila`. |
| `devp2p` | Utilities for interacting with nodes on the networking layer without running a full blockchain. |
| `abigen` | Source code generator for Ethereum/Sila contract ABIs into type-safe Go packages. |
| `evm` | Developer utility for running and debugging EVM bytecode snippets. |
| `rlpdump` | Developer utility for converting binary RLP dumps into a readable hierarchical representation. |

## Running SilaChain

```shell
sila console
```

Attach to an already running instance:

```shell
sila attach
```

## Configuration

```shell
sila --config /path/to/your_config.toml
sila --your-favourite-flags dumpconfig
```

## Docker quick start

```shell
docker run -d --name sila-node -v /Users/alice/sila:/root \
           -p 8545:8545 -p 30303:30303 \
           sila-org/sila
```

## Programmatically interfacing Sila nodes

`sila` supports JSON-RPC APIs over HTTP, WebSockets, and IPC.

## Private networks

Maintaining a private network requires careful configuration of genesis state, peer discovery, networking, and consensus-layer integration when applicable.

## Contribution

Thank you for considering contributing to SilaChain.

## License

The SilaChain library, meaning code outside the `cmd` directory, is licensed under the GNU Lesser General Public License v3.0.

The SilaChain binaries, meaning code inside the `cmd` directory, are licensed under the GNU General Public License v3.0.
