# GoOrbit — Project Overview Notes
# Current State: Network + Storage + Server 

---

## WHAT IS GOORBIT?

GoOrbit is a **Distributed File Storage System**.

Think of it like Google Drive — but no company owns it.

Many computers (nodes) talk to each other and together store files across the network.

```text
[Your Computer]  <--->  [Friend's Computer]  <--->  [Server in Germany]
      │                        │                           │
      +------------- Shared File Storage -----------------+
```

No central server. Files survive even if one node goes down.

---

## WHAT HAVE WE BUILT SO FAR?

```text
GoOrbit — Full Vision
│
├── ❌ Encryption Layer      (not built yet)
├── ❌ Streaming Layer       (not built yet)
├── ❌ File Chunking         (not built yet)
├── ✅ File Server           ← NEW — coordinates network + storage
├── ✅ Storage Layer         ← NEW — CAS disk storage
└── ✅ Network Layer         ← done previously
       └── TCP Transport
```

---

## PROJECT FOLDER STRUCTURE

```text
GoOrbit/
│
├── main.go                      ← Entry point. Wires everything together.
├── server.go                    ← FileServer — coordinator of network + storage
├── store.go                     ← Storage engine — saves/reads/deletes files on disk
├── store_test.go                ← Tests for the storage engine
├── go.mod                       ← Module name + dependencies
├── go.sum                       ← Dependency checksums (auto-generated)
├── Makefile                     ← Shortcuts: make run, make test
│
├── p2p/                         ← All networking code
│   ├── transport.go             ← Interfaces (Peer, Transport)
│   ├── message.go               ← RPC struct (what a message looks like)
│   ├── handshake.go             ← Handshake function type + NOP placeholder
│   ├── encoding.go              ← Decoders (DefaultDecoder, GOBDecoder)
│   ├── tcp_transport.go         ← Full TCP implementation
│   └── tcp_transport_test.go    ← Tests for TCP transport
│
└── notes/                       ← notes
    ├── 00_project_overview_notes.md      ← This file
    ├── 01_transport_interfaces_notes.md  ← Peer & Transport interfaces
    ├── 02_message_notes.md               ← RPC struct
    ├── 03_handshake_notes.md             ← HandshakeFunc
    ├── 04_encoding_notes.md              ← Decoders
    ├── 05_tcp_transport_notes.md         ← TCPPeer, TCPTransport
    ├── 06_main_notes.md                  ← main.go (updated)
    ├── 07_go_concepts_notes.md           ← All Go concepts
    ├── 08_store_notes.md                 ← store.go (CAS storage)
    ├── 09_store_test_notes.md            ← store_test.go
    └── 10_server_notes.md               ← server.go (FileServer)
```

---

## READ ORDER (Correct Learning Sequence)

```text
--- NETWORKING LAYER ---
1. p2p/transport.go          → Peer & Transport interfaces
2. p2p/message.go            → What an RPC message is
3. p2p/handshake.go          → Verification concept
4. p2p/encoding.go           → How bytes become messages
5. p2p/tcp_transport.go      → Full TCP implementation

--- STORAGE LAYER ---
6. store.go                  → CAS disk storage engine
7. store_test.go             → How it's tested

--- COORDINATION LAYER ---
8. server.go                 → FileServer wires network + storage
9. main.go                   → Wires everything, starts the node
```

---

## WHAT EACH FILE DOES

| File | Role |
|------|------|
| `p2p/transport.go` | Contracts — Peer and Transport interfaces |
| `p2p/message.go` | RPC struct — shape of every network message |
| `p2p/handshake.go` | Handshake function type + NOP placeholder |
| `p2p/encoding.go` | Decode raw TCP bytes into RPC structs |
| `p2p/tcp_transport.go` | Actual TCP connections, peers, message reading |
| `store.go` | Save, read, delete files on disk using CAS paths |
| `store_test.go` | Tests for all store operations |
| `server.go` | FileServer — coordinates network and storage |
| `main.go` | Wires everything together and starts the node |

---

## THE FULL FLOW — When a Node Starts

```text
main.go
│
├── Create TCPTransport (:3000)
│       └── not listening yet
│
├── Create FileServer
│       ├── Transport = TCPTransport
│       ├── Store → root = "3000_network/"
│       └── quitch channel ready
│
├── Background goroutine (will Stop() after 3 seconds)
│
└── s.Start()
        │
        ├── Transport.ListenAndAccept()
        │       → port :3000 open
        │       → accept loop running (goroutine)
        │
        └── loop() ← BLOCKS HERE
                │
                select:
                ├── peer message arrives → print it
                └── quitch closed (Stop called) → return

        defer: log + Transport.Close()
```

---

## WHAT IS CONTENT-ADDRESSABLE STORAGE (CAS)?

Instead of storing files by their name, GoOrbit stores them by their **content hash**.

```text
Normal:
    "vacation.jpg" → stored at /files/vacation.jpg

CAS (GoOrbit):
    "vacation.jpg" content → SHA1 → hash
    stored at /3000_network/c5659/96f77/.../c565996f77...
```

Benefits:
```text
Same content → same location → no duplicates
Content determines location → rename doesn't affect storage
Like how Git stores files
```

---

## KEY GO CONCEPTS USED SO FAR

| Concept | Where Used |
|---------|------------|
| Interface | Peer, Transport, Decoder |
| Struct + Embedding | TCPTransport, FileServer, Store |
| Goroutine | acceptLoop, handleConn, auto-stop in main |
| Channel | rpcch (messages), quitch (stop signal) |
| `chan struct{}` | quitch — zero-cost signal |
| `close(ch)` | Stop() — wakes all waiters |
| `select` | loop() — wait on multiple channels |
| Pointer | NewTCPPeer, NewStore, etc. |
| defer | conn cleanup, Transport.Close() |
| Function type | HandshakeFunc, PathTransformFunc |
| `io.Reader` | writeStream, Decode — universal data source |
| `io.Copy` | efficient data transfer |
| `os.MkdirAll` | create nested CAS folders |
| SHA1 + hex | CAS path generation |

---

## WHAT'S COMING NEXT

```text
✅ Phase 1: P2P Networking
   └── TCP connections, listening, reading messages

✅ Phase 2: Content-Addressable Storage
   └── Store files by SHA1 hash on disk

✅ Phase 3: File Server
   └── Coordinator between network and storage

⬜ Phase 4: Dialing (Connecting TO other nodes)
   └── Currently we only accept — we don't initiate connections

⬜ Phase 5: Sending Files Between Nodes
   └── Upload to one node → replicate to others

⬜ Phase 6: Encryption
   └── Encrypt files before storing/sending

⬜ Phase 7: Streaming
   └── Handle large files without loading into memory
```

---

## HOW TO RUN

```bash
go run main.go        # run the server (stops after 3 seconds)
go test ./...         # run all tests
make run              # via Makefile
make test             # via Makefile
```

To manually test networking:
```bash
# Terminal 1
go run main.go

# Terminal 2 (connect with raw TCP)
nc localhost 3000
# type anything → server prints the bytes
```

---
