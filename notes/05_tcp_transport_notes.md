# tcp_transport.go — The Real Networking Engine

---

## 1. WHAT IS THIS FILE DOING?

This is the **most important file built so far**.

It implements everything defined in `transport.go` (the interface/contract).

It is the actual networking engine — handles TCP connections, peers, reading messages.

---

## 2. BIG PICTURE FIRST

```text
When a GoOrbit node starts:

1. Open TCP port 3000
2. Wait for other nodes to connect
3. When someone connects:
       a. Wrap them as a TCPPeer
       b. Run handshake
       c. Call OnPeer (notify the app)
       d. Keep reading their messages forever
4. Push each message into rpcch channel
5. main.go reads from channel and processes
```

---

## 3. PART A — TCPPeer

```go
type TCPPeer struct {
    conn     net.Conn
    outbound bool
}
```

Represents **one connected node** (one remote computer).

### Fields

```text
conn     → the actual TCP connection (the wire between two computers)
outbound → did WE connect to them? or did THEY connect to us?
```

### outbound explained

```text
outbound = true
    We ran:  net.Dial("tcp", "192.168.1.10:3000")
    We initiated the connection

outbound = false
    They connected to us
    Our listener.Accept() returned this connection
```

Why does this matter?

```text
Later in the project:
    Outbound peers and inbound peers
    may be treated differently
    (e.g., who sends first, who controls the stream)
```

### Constructor

```go
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
    return &TCPPeer{
        conn:     conn,
        outbound: outbound,
    }
}
```

`&TCPPeer{...}` → returns a **pointer** to the new struct.

### Close() — satisfies the Peer interface

```go
func (p *TCPPeer) Close() error {
    return p.conn.Close()
}
```

This is what makes `TCPPeer` qualify as a `Peer`.

The interface said: "Peer must have Close() error."
TCPPeer has it. Done. Go automatically treats it as a Peer.

---

## 4. PART B — TCPTransportOpts

```go
type TCPTransportOpts struct {
    ListenAddr    string
    HandshakeFunc HandshakeFunc
    Decoder       Decoder
    OnPeer        func(Peer) error
}
```

This is a **configuration/settings struct**.

Instead of passing 4 arguments to a function:

```go
NewTCPTransport(":3000", NOPHandshakeFunc, DefaultDecoder{}, onPeerFunc)  // messy
```

We group them cleanly:

```go
opts := TCPTransportOpts{
    ListenAddr:    ":3000",
    HandshakeFunc: NOPHandshakeFunc,
    Decoder:       DefaultDecoder{},
    OnPeer:        myFunc,
}
NewTCPTransport(opts)  // clean
```

### OnPeer — Callback Function

```go
OnPeer func(Peer) error
```

This is a **callback** — a function you provide that gets called automatically when an event happens.

```text
Like JavaScript:
    button.addEventListener("click", myFunction)

Here:
    When a new peer connects → call OnPeer(peer)
```

If `OnPeer` is not set (nil), it's skipped.

```go
if t.OnPeer != nil {
    if err = t.OnPeer(peer); err != nil {
        return
    }
}
```

---

## 5. PART C — TCPTransport

```go
type TCPTransport struct {
    TCPTransportOpts           // embedded struct
    listener net.Listener      // TCP listener
    rpcch    chan RPC           // message channel
}
```

This is the **main object**. The networking engine.

### Embedding (TCPTransportOpts inside TCPTransport)

Go doesn't have inheritance. But it has **embedding**.

```go
type TCPTransport struct {
    TCPTransportOpts   // ← embedded
    ...
}
```

All fields from `TCPTransportOpts` become directly accessible:

```go
t.ListenAddr      // works
t.HandshakeFunc   // works
t.Decoder         // works

// instead of:
t.TCPTransportOpts.ListenAddr  // not needed
```

### chan RPC — The Message Channel

```go
rpcch chan RPC
```

A **channel** is like a pipe between goroutines.

```text
handleConn goroutine    →→→→    rpcch    →→→→    main.go goroutine
   (writer)                    (pipe)               (reader)
```

Data goes in one side, comes out the other. Safe across goroutines.

---

## 6. FUNCTIONS

### NewTCPTransport

```go
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
    return &TCPTransport{
        TCPTransportOpts: opts,
        rpcch:            make(chan RPC),
    }
}
```

`make(chan RPC)` creates an **unbuffered channel**.

```text
Unbuffered = the writer blocks until someone reads.

Writer:  "here's a message" → WAITS
Reader:  "got it"           → Writer unblocks
```

This ensures no messages are lost or skipped.

### Consume()

```go
func (t *TCPTransport) Consume() <-chan RPC {
    return t.rpcch
}
```

Returns the channel, but as **read-only** (`<-chan RPC`).

The outside world can only READ from it. Cannot write to it.

Prevents misuse — only the transport itself should write to rpcch.

---

## 7. ListenAndAccept()

```go
func (t *TCPTransport) ListenAndAccept() error {
    var err error
    ln, err := net.Listen("tcp", t.ListenAddr)
    if err != nil {
        return err
    }
    t.listener = ln

    go t.startAcceptLoop()

    return nil
}
```

### Step by Step

```text
Step 1: net.Listen("tcp", ":3000")
    → Tells the OS: "I want port 3000"
    → OS opens the port
    → Returns a Listener

Step 2: t.listener = ln
    → Save the listener for later use

Step 3: go t.startAcceptLoop()
    → Launch accept loop in background (goroutine)
    → main function does NOT block here

Step 4: return nil
    → Success, no error
```

### Why goroutine here?

If we called `t.startAcceptLoop()` without `go`:

```text
ListenAndAccept() would block forever
main.go would never continue
```

With `go`:

```text
Accept loop runs in background
ListenAndAccept() returns immediately
main.go continues to set up other things
```

---

## 8. startAcceptLoop()

```go
func (t *TCPTransport) startAcceptLoop() {
    for {
        conn, err := t.listener.Accept()
        if err != nil {
            fmt.Printf("TCP accept error: %s\n", err)
        }
        fmt.Printf("new incoming connection %+v\n", conn)
        go t.handleConn(conn)
    }
}
```

Runs **forever** in a loop, waiting for connections.

### Accept() behavior

```go
conn, err := t.listener.Accept()
```

```text
Blocks (waits) until someone connects
When they connect → returns net.Conn
net.Conn = a private wire between us and that peer
```

### Why goroutine for handleConn?

```text
Without goroutine:
    Peer A connects → handleConn(A) runs → blocks forever reading A's messages
    Peer B can't connect until A disconnects

With goroutine:
    Peer A connects → goroutine for A
    Peer B connects → goroutine for B  ← immediately, no waiting
    Peer C connects → goroutine for C

All peers handled simultaneously
Accept loop goes back to waiting right away
```

---

## 9. handleConn()

```go
func (t *TCPTransport) handleConn(conn net.Conn) {
    var err error

    defer func() {
        fmt.Printf("Dropping peer connection: %s", err)
        conn.Close()
    }()

    peer := NewTCPPeer(conn, true)

    if err = t.HandshakeFunc(peer); err != nil {
        return
    }

    if t.OnPeer != nil {
        if err = t.OnPeer(peer); err != nil {
            return
        }
    }

    // Read Loop
    rpc := RPC{}
    for {
        if err := t.Decoder.Decode(conn, &rpc); err != nil {
            fmt.Printf("TCP error: %s\n", err)
            continue
        }
        rpc.From = conn.RemoteAddr()
        t.rpcch <- rpc
    }
}
```

This manages the **entire lifetime of one connection**.

### defer — Guaranteed Cleanup

```go
defer func() {
    fmt.Printf("Dropping peer connection: %s", err)
    conn.Close()
}()
```

`defer` runs when the function exits — whether normally or due to an error.

```text
Like Java's finally:

try {
    ...
} finally {
    conn.close(); // always runs
}
```

Ensures the connection is always closed when done.

### Steps inside handleConn

```text
1. defer setup → cleanup guaranteed at end

2. NewTCPPeer(conn, true)
       → wrap raw TCP connection in our TCPPeer struct

3. HandshakeFunc(peer)
       → verify the peer
       → if fails → function returns → defer closes conn

4. OnPeer(peer)
       → notify the app "new peer joined"
       → if fails → function returns → defer closes conn

5. Read Loop (infinite):
       → Decode(conn, &rpc)  → read bytes from TCP → fill RPC struct
       → rpc.From = conn.RemoteAddr()  → record who sent it
       → t.rpcch <- rpc  → push message into channel
       → go back to start
```

### The Read Loop

```go
for {
    if err := t.Decoder.Decode(conn, &rpc); err != nil {
        fmt.Printf("TCP error: %s\n", err)
        continue
    }
    rpc.From = conn.RemoteAddr()
    t.rpcch <- rpc
}
```

Reads messages until the connection dies.

```text
Decode error? → print it → continue (try reading again)
Success?      → set From → push to channel → loop again
```

`t.rpcch <- rpc` → **send** rpc into the channel. Blocks until someone reads it.

---

## 10. FULL FLOW DIAGRAM

```text
main.go
│
└── tr.ListenAndAccept()
         │
         ├── net.Listen(":3000")   → port open
         │
         └── go startAcceptLoop()
                    │
         ┌──────────▼────────────────────┐
         │  for {                        │
         │      conn = listener.Accept() │ ← blocks, waits
         │      go handleConn(conn)      │ ← new goroutine per peer
         │  }                            │
         └───────────────────────────────┘
                    │
         ┌──────────▼────────────────────┐
         │  handleConn(conn)             │
         │                               │
         │  peer = NewTCPPeer(conn)      │
         │  HandshakeFunc(peer)  ✓       │
         │  OnPeer(peer)         ✓       │
         │                               │
         │  for {                        │
         │      Decode(conn, &rpc)       │
         │      rpc.From = addr          │
         │      rpcch <- rpc       ──────┼──→  main.go reads message
         │  }                            │
         └───────────────────────────────┘
```

---

## 11. KEY GO CONCEPTS IN THIS FILE

| Concept | Example | What it does |
|---------|---------|--------------|
| Struct embedding | `TCPTransportOpts` inside `TCPTransport` | Reuse fields without inheritance |
| Pointer receiver | `func (p *TCPPeer) Close()` | Method modifies the original struct |
| Goroutine | `go t.startAcceptLoop()` | Run function concurrently |
| Channel | `rpcch chan RPC` | Pipe messages between goroutines |
| defer | `defer conn.Close()` | Always run at function end |
| nil check | `if t.OnPeer != nil` | Only call if it was set |
| Send to channel | `t.rpcch <- rpc` | Push data into channel |

---

# End Notes
