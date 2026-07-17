# GoOrbit — Project Overview Notes
# Current State: Complete Distributed File Storage System

---

# WHAT IS GOORBIT?

GoOrbit is a **Distributed File Storage System** built in Go.

Instead of storing files on one central server, multiple computers (called **nodes**) work together to store and retrieve files over a peer-to-peer network.

Think of it like Google Drive, except there is **no central server**.

```text
              GoOrbit Network

        +------------------------+
        |      Node A            |
        | Stores some files      |
        +-----------+------------+
                    |
                    |
          TCP Connection
                    |
        +-----------+------------+
        |      Node B            |
        | Stores some files      |
        +-----------+------------+
                    |
                    |
          TCP Connection
                    |
        +-----------+------------+
        |      Node C            |
        | Stores some files      |
        +------------------------+
```

Every node can:

- Store files
- Retrieve files
- Send files
- Receive files
- Replicate files
- Communicate with other peers

There is **no master node**.

---

# PROJECT GOAL

The goal of GoOrbit is to learn how real distributed storage systems work by implementing the major building blocks from scratch.

The project demonstrates concepts such as:

- Peer-to-peer networking
- TCP communication
- Distributed storage
- Content-addressable storage
- Encryption
- Streaming large files
- RPC messaging
- Bootstrapping peers

---

# WHAT HAS BEEN BUILT?

```text
GoOrbit

├── ✅ P2P Networking
│      ├── TCP Transport
│      ├── Peer Connections
│      ├── Bootstrapping
│      └── RPC Messaging
│
├── ✅ Distributed File Server
│      ├── Store Files
│      ├── Retrieve Files
│      ├── Broadcast Messages
│      └── Event Loop
│
├── ✅ Storage Layer
│      ├── Content-addressable paths
│      ├── Local file management
│      └── File deletion
│
├── ✅ Encryption Layer
│      ├── AES-CTR Encryption
│      ├── AES-CTR Decryption
│      └── Secure Streaming
│
├── ✅ Streaming Layer
│      ├── io.Copy
│      ├── Stream Encryption
│      └── Stream Decryption
│
└── ✅ Distributed File Replication
       ├── Broadcast Metadata
       ├── Send File Streams
       ├── Retrieve Missing Files
       └── Synchronize Peers
```

---

# PROJECT FOLDER STRUCTURE

```text
GoOrbit/

├── main.go
│      Entry point of the application.
│      Creates nodes and starts the distributed network.
│
├── server.go
│      Distributed FileServer implementation.
│      Coordinates networking and storage.
│
├── store.go
│      Local storage engine.
│      Handles reading, writing and deleting files.
│
├── crypto.go
│      Encryption and decryption utilities.
│
├── store_test.go
│      Tests for storage layer.
│
├── crypto_test.go
│      Tests encryption and decryption.
│
├── go.mod
│
├── go.sum
│
├── Makefile
│
├── p2p/
│   │
│   ├── transport.go
│   │      Transport and Peer interfaces.
│   │
│   ├── message.go
│   │      RPC message structure.
│   │
│   ├── handshake.go
│   │      Handshake function type.
│   │
│   ├── encoding.go
│   │      Message decoders.
│   │
│   ├── tcp_transport.go
│   │      Complete TCP implementation.
│   │
│   └── tcp_transport_test.go
│          TCP transport tests.
│
└── notes/
    │
    ├── 00_project_overview_notes.md
    ├── 01_transport_interfaces_notes.md
    ├── 02_message_notes.md
    ├── 03_handshake_notes.md
    ├── 04_encoding_notes.md
    ├── 05_tcp_transport_notes.md
    ├── 06_main_notes.md
    ├── 07_go_concepts_notes.md
    ├── 08_store_notes.md
    ├── 09_store_test_notes.md
    ├── 10_server_notes.md
    ├── 11_crypto_notes.md
    ├── 12_crypto_test_notes.md
    ├── 13_message_flow_notes.md
    ├── 14_project_architecture.md
    └── 15_end_to_end_execution.md
```

---

# LEARNING ORDER

The best order to understand the project is:

```text
NETWORKING

1. transport.go
2. message.go
3. handshake.go
4. encoding.go
5. tcp_transport.go

↓

STORAGE

6. store.go
7. store_test.go

↓

SECURITY

8. crypto.go
9. crypto_test.go

↓

COORDINATION

10. server.go

↓

APPLICATION

11. main.go

↓

SYSTEM FLOW

12. Message Flow Notes
13. Architecture Notes
14. End-to-End Execution Notes
```

---

# WHAT EACH FILE DOES

| File | Responsibility |
|------|----------------|
| transport.go | Defines Peer and Transport interfaces |
| message.go | Defines RPC message format |
| handshake.go | Defines handshake function |
| encoding.go | Converts TCP bytes into RPC messages |
| tcp_transport.go | TCP listener, peers, networking |
| store.go | Stores and retrieves files |
| store_test.go | Tests storage functionality |
| crypto.go | Encrypts and decrypts file streams |
| crypto_test.go | Tests encryption correctness |
| server.go | Coordinates storage and networking |
| main.go | Starts the distributed system |

---

# HIGH LEVEL ARCHITECTURE

```text
                    main.go
                        │
                        │
               Create FileServer
                        │
        ┌───────────────┴───────────────┐
        │                               │
        │                               │
   TCP Transport                    Storage
        │                               │
        │                               │
    Peer Network                 Local Files
        │                               │
        └───────────────┬───────────────┘
                        │
                  FileServer
                        │
          Store / Retrieve / Broadcast
```

---

# COMPLETE FILE FLOW

When a file is stored:

```text
User

↓

Store("picture.png")

↓

Generate storage path

↓

Encrypt file stream

↓

Save locally

↓

Broadcast metadata

↓

Peers receive metadata

↓

Peers prepare to receive stream

↓

Encrypted bytes streamed

↓

Peers decrypt

↓

Peers store locally

↓

Replication complete
```

---

# FILE RETRIEVAL FLOW

When a file is requested:

```text
User

↓

Get("picture.png")

↓

Check local storage

↓

Found?

├── YES
│
│   Return file
│
└── NO

↓

Broadcast GetFile message

↓

Peers check storage

↓

Peer found file

↓

Peer streams encrypted file

↓

Receive stream

↓

Decrypt stream

↓

Store locally

↓

Return file
```

---

# NODE STARTUP FLOW

```text
main()

↓

Create TCP Transport

↓

Create FileServer

↓

Start()

↓

ListenAndAccept()

↓

Open TCP port

↓

Accept incoming peers

↓

Bootstrap to known peers

↓

Receive RPC messages

↓

Store / Retrieve files

↓

Shutdown gracefully
```

---

# CONTENT-ADDRESSABLE STORAGE (CAS)

The storage layer creates deterministic storage paths using a SHA-1 hash of the file key (typically the filename).

For example:

```text
photo.png

↓

SHA1

↓

c565996f77...

↓

storage/

↓

c5659/

96f77/

...

↓

c565996f77...
```

This approach is inspired by Content-Addressable Storage (CAS).

> **Note:** In this project, the file key is hashed to generate the storage path. A true CAS system would hash the file contents instead of the filename.

Benefits include:

- Deterministic storage paths
- Even directory distribution
- Fast lookups
- Reduced directory size
- Easy future migration to true CAS

---

# KEY GO CONCEPTS USED

| Concept | Usage |
|---------|------|
| Interface | Peer, Transport, Decoder |
| Struct Embedding | FileServer, TCPTransport |
| Goroutines | Accept loop, connection handlers |
| Channels | RPC messages, shutdown signals |
| select | Wait for multiple events |
| Mutex | Peer synchronization |
| defer | Cleanup resources |
| io.Reader | Universal input stream |
| io.Writer | Universal output stream |
| io.Copy | Efficient streaming |
| bytes.Buffer | Temporary memory buffer |
| AES Cipher | File encryption |
| CTR Mode | Streaming encryption |
| SHA-1 | Storage path generation |
| binary.Read / Write | Stream metadata |
| Pointer Receivers | Shared object modification |

---

# CURRENT PROJECT STATUS

```text
Networking                 ✅

Peer Connections           ✅

Bootstrapping              ✅

RPC Messaging              ✅

Distributed Storage        ✅

Content-addressable Paths  ✅

Encryption                 ✅

Streaming                  ✅

File Replication           ✅

Distributed Retrieval      ✅

Unit Tests                 ✅
```

---

# POSSIBLE FUTURE IMPROVEMENTS

```text
⬜ True Content Hashing

⬜ File Chunking

⬜ Replication Factor

⬜ Compression

⬜ Metadata Persistence

⬜ Automatic Peer Discovery

⬜ Distributed Hash Table (DHT)

⬜ Fault Tolerance

⬜ Versioning

⬜ Load Balancing
```

---

# HOW TO RUN

Run the project:

```bash
go run .
```

Run all tests:

```bash
go test ./...
```

Using Makefile:

```bash
make run

make test
```

---

# TYPICAL DEMONSTRATION

A complete demonstration looks like:

```text
Start Node A

↓

Start Node B

↓

Start Node C

↓

Bootstrap connections established

↓

Store file on Node C

↓

File encrypted

↓

Stored locally

↓

Metadata broadcast

↓

File replicated to peers

↓

Delete local copy

↓

Request file again

↓

Peer streams encrypted file

↓

File decrypted while receiving

↓

Stored locally

↓

File returned successfully
```

---

# PROJECT SUMMARY

GoOrbit combines several important distributed systems concepts into one project.

It demonstrates how independent nodes communicate over TCP, exchange RPC messages, securely stream encrypted files, and coordinate storage without relying on a central server.

By completing this project, you gain hands-on experience with networking, concurrency, storage systems, cryptography, streaming, and distributed system design—the same foundational ideas used in systems like Git, BitTorrent, IPFS, Dropbox, and distributed object storage platforms.