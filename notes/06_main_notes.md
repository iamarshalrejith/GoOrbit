# main.go — Entry Point (Updated)
# Current State: FileServer + Store + TCP Transport

---

## 1. WHAT IS THIS FILE DOING?

`main.go` wires ALL the layers together and launches the node.

It now does MORE than before — it creates a full FileServer (not just a raw transport).

---

## 2. THE CODE

```go
package main

import (
    "log"
    "time"

    "github.com/iamarshalrejith/GoOrbit/p2p"
)

func main() {
    tcpTransportOpts := p2p.TCPTransportOpts{
        ListenAddr:    ":3000",
        HandshakeFunc: p2p.NOPHandshakeFunc,
        Decoder:       p2p.DefaultDecoder{},
    }
    tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

    fileServerOpts := FileServerOpts{
        StorageRoot:       "3000_network",
        PathTransformFunc: CASPATHTransformFunc,
        Transport:         tcpTransport,
    }

    s := NewFileServer(fileServerOpts)

    go func() {
        time.Sleep(time.Second * 3)
        s.Stop()
    }()

    if err := s.Start(); err != nil {
        log.Fatal(err)
    }
}
```

---

## 3. WHAT CHANGED FROM THE OLD main.go

### Old main.go

```go
// Created transport directly
tr := p2p.NewTCPTransport(tcpOpts)

// Manually consumed messages
go func() {
    for {
        msg := <-tr.Consume()
        fmt.Printf("%+v\n", msg)
    }
}()

tr.ListenAndAccept()
select {}
```

### New main.go

```go
// Still creates transport
tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

// But NOW wraps it in a FileServer
s := NewFileServer(fileServerOpts)

// FileServer handles message consuming internally (in loop())
s.Start()
```

```text
Before:  main.go manually reads from the transport channel
After:   FileServer.loop() handles that — main.go just says "Start"
```

The message reading logic moved INTO the server. main.go is cleaner.

---

## 4. STEP BY STEP EXECUTION

### Step 1 — TCP Transport Setup

```go
tcpTransportOpts := p2p.TCPTransportOpts{
    ListenAddr:    ":3000",
    HandshakeFunc: p2p.NOPHandshakeFunc,
    Decoder:       p2p.DefaultDecoder{},
}
tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)
```

```text
Port: 3000
Handshake: skip (NOP)
Decoder: raw bytes into Payload
```

Network layer created. Not listening yet.

---

### Step 2 — FileServer Setup

```go
fileServerOpts := FileServerOpts{
    StorageRoot:       "3000_network",
    PathTransformFunc: CASPATHTransformFunc,
    Transport:         tcpTransport,
}
s := NewFileServer(fileServerOpts)
```

```text
StorageRoot: "3000_network"
    → all files stored in ./3000_network/ folder on disk

PathTransformFunc: CASPATHTransformFunc
    → SHA1 hash-based file paths

Transport: tcpTransport
    → the network layer we just built
```

FileServer is created. Not started yet.

---

### Step 3 — Auto-Stop Goroutine

```go
go func() {
    time.Sleep(time.Second * 3)
    s.Stop()
}()
```

A background goroutine that waits 3 seconds then stops the server.

```text
This is temporary — just for development/testing.
In production, Stop() would be triggered by:
    - OS signal (Ctrl+C)
    - Admin command
    - Error condition
```

`time.Sleep(time.Second * 3)` = pause for 3 seconds.

---

### Step 4 — Start

```go
if err := s.Start(); err != nil {
    log.Fatal(err)
}
```

```text
Start() → ListenAndAccept() → opens port :3000
        → loop() → blocks here

After 3 seconds → Stop() closes quitch
               → loop() returns
               → Start() returns nil
               → main() exits
```

---

## 5. FULL EXECUTION TIMELINE

```text
t=0.0s  main() starts
t=0.0s  TCPTransport created (port not open yet)
t=0.0s  FileServer created (store ready, quitch channel ready)
t=0.0s  Background goroutine starts (will stop server at t=3s)
t=0.0s  s.Start() called
t=0.0s    → ListenAndAccept() → port :3000 open
t=0.0s    → loop() starts → blocking

--- server is running, ready for connections ---

t=3.0s  Background goroutine: s.Stop() → close(quitch)
t=3.0s  loop() select: <-s.quitch fires
t=3.0s  loop() returns
t=3.0s  defer in loop: log "stopped" + Transport.Close()
t=3.0s  Start() returns nil
t=3.0s  main() exits
```

---

## 6. FOLDER STRUCTURE CREATED AT RUNTIME

When the server runs and receives files, disk looks like:

```text
GoOrbit/
├── main.go
├── server.go
├── store.go
└── 3000_network/              ← created by FileServer (StorageRoot)
    └── a3f9c/
        └── 1b2d4/
            └── ...
                └── a3f9c1b2d4...  ← actual file (CAS path)
```

Different nodes use different root folders:
```text
Node on port 3000 → "3000_network/"
Node on port 4000 → "4000_network/"
Node on port 5000 → "5000_network/"
```

This lets you run multiple nodes on the same machine for testing.

---

## 7. MENTAL MODEL — How All Layers Connect

```text
main.go
│
├── TCPTransport (:3000)
│       │
│       ├── accepts connections
│       ├── runs handshake
│       └── pushes RPC messages → rpcch channel
│
└── FileServer
        │
        ├── Transport = TCPTransport (network in)
        │
        ├── Store (disk layer)
        │       └── 3000_network/ (CAS folder structure)
        │
        └── loop()
                │
                ├── reads from Transport.Consume() (messages from peers)
                └── reads from quitch (stop signal)
```

---

# End Notes
