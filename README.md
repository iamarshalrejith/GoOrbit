# GoOrbit

A peer-to-peer distributed file storage system written in Go — think of it as a tiny, self-hosted "Google Drive" with no central server. Nodes connect to each other over TCP, store files locally using content-addressable storage (CAS), and replicate files across the network on demand, with AES encryption protecting data in transit.

Built as a hands-on project to learn Go's networking, concurrency, and storage primitives from the ground up.

---

## Features

- **Peer-to-peer networking** — custom TCP transport layer with a pluggable handshake and message encoding/decoding system
- **Content-Addressable Storage (CAS)** — files are stored on disk by the SHA-1 hash of their content (similar to how Git stores objects), avoiding duplicate storage and decoupling file names from file location
- **Distributed replication** — storing a file on one node broadcasts it and streams the bytes to every connected peer
- **Network-wide retrieval** — if a file isn't found on the local disk, the node asks its peers over the network and streams it back
- **Encryption in transit** — file data is encrypted with AES (CTR mode) before being sent to peers, and decrypted on arrival
- **Bootstrap nodes** — new nodes can join an existing network by dialing one or more known peer addresses
- **Concurrency-first design** — goroutines and channels handle connection accepting, message consumption, and peer broadcasting

## Architecture

```
GoOrbit/
├── main.go              # Entry point — wires transport, storage, and server together
├── server.go             # FileServer — coordinates networking and storage
├── store.go               # CAS storage engine (write/read/delete files on disk)
├── crypto.go               # AES encryption helpers, ID/key generation, hashing
├── p2p/
│   ├── transport.go        # Peer & Transport interfaces
│   ├── message.go            # RPC message struct
│   ├── handshake.go            # Handshake function type
│   ├── encoding.go               # Message decoders (GOB / default)
│   └── tcp_transport.go            # TCP implementation of the Transport interface
└── notes/                          # Personal learning notes on each package/concept
```

### How it works

1. Each node starts a `TCPTransport` that listens for incoming connections and dials any configured bootstrap peers to join the network.
2. A `FileServer` sits on top of the transport and a local `Store` (the CAS disk engine), and coordinates everything.
3. **Storing a file**: the file is written to local disk, then a `MessageStoreFile` is broadcast to all connected peers, followed by the (encrypted) file bytes streamed over the same connections.
4. **Retrieving a file**: the node checks local disk first; if the file isn't there, it broadcasts a `MessageGetFile` request, and peers that have the file stream it back (or report they don't have it, so the request doesn't hang).

## Getting Started

### Prerequisites
- Go 1.25+

### Run

```bash
go run main.go
# or
make run
```

This spins up a small local network of nodes on ports `:3000`, `:4000`, and `:5000`, has one of them store and retrieve a series of files, and prints the results to the console.

### Test

```bash
go test ./... -v
# or
make test
```

## Roadmap

- [x] P2P TCP networking (listen, dial, accept, message passing)
- [x] Content-addressable storage on disk
- [x] File server coordinating network + storage
- [x] File replication across peers
- [x] AES encryption for file transfers


## Notes

The `notes/` folder contains my own learning notes written while building each piece of this project (transport interfaces, message encoding, handshakes, CAS storage, the file server, and general Go concepts used along the way) — kept in the repo as a record of the process, not as user-facing documentation.

## License

This project is for personal learning purposes. Feel free to explore, fork, or use it as reference for your own P2P projects.