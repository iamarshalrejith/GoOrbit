# server.go — The File Server (Brain of GoOrbit)

---

## 1. WHAT IS THIS FILE DOING?

`server.go` is the **central coordinator** of GoOrbit.

It connects the two major parts built so far:

```text
Network Layer (p2p/)          Storage Layer (store.go)
      │                               │
      └──────────── FileServer ───────┘
```

The `FileServer`:
- Listens for messages from peers (networking)
- Stores/retrieves files on disk (storage)
- Can be gracefully stopped

---

## 2. THE CODE

```go
type FileServerOpts struct {
    StorageRoot       string
    PathTransformFunc PathTransformFunc
    Transport         p2p.Transport
}

type FileServer struct {
    FileServerOpts
    store  *Store
    quitch chan struct{}
}
```

---

## 3. FileServerOpts — Configuration

```go
type FileServerOpts struct {
    StorageRoot       string             // which folder to store files in
    PathTransformFunc PathTransformFunc  // how to hash file paths
    Transport         p2p.Transport      // the network layer (TCP, UDP, etc.)
}
```

Same pattern as `TCPTransportOpts` — group all settings into one struct.

`Transport p2p.Transport` is an **interface** — FileServer doesn't care if it's TCP, UDP, or anything else. It just talks to the interface.

---

## 4. FileServer STRUCT

```go
type FileServer struct {
    FileServerOpts         // embedded: StorageRoot, PathTransformFunc, Transport
    store  *Store          // the disk storage engine
    quitch chan struct{}    // signal channel for stopping
}
```

### quitch — The Stop Signal Channel

```go
quitch chan struct{}
```

`struct{}` = an **empty struct**. Takes zero bytes of memory.

```text
Why not chan bool or chan int?
    We don't need to send a value.
    We just need to SIGNAL "stop now".
    Empty struct is idiomatic Go for pure signaling.
```

---

## 5. NewFileServer — Constructor

```go
func NewFileServer(opts FileServerOpts) *FileServer {
    storeOpts := StoreOpts{
        Root:              opts.StorageRoot,
        PathTransformFunc: opts.PathTransformFunc,
    }

    return &FileServer{
        FileServerOpts: opts,
        store:          NewStore(storeOpts),
        quitch:         make(chan struct{}),
    }
}
```

### What happens here

```text
1. Take the FileServerOpts
2. Build StoreOpts from the relevant fields
3. Create a new Store (disk storage engine)
4. Create the FileServer, wiring together:
       - opts (Transport, StorageRoot, etc.)
       - store (disk layer)
       - quitch (empty channel, ready to receive stop signal)
```

---

## 6. Stop() — Graceful Shutdown

```go
func (s *FileServer) Stop() {
    close(s.quitch)
}
```

`close(channel)` → **closes** the channel.

When a channel is closed:
```text
All goroutines currently waiting on that channel
immediately wake up and receive the zero value.
```

### How is this used?

In `main.go`:
```go
go func() {
    time.Sleep(3 * time.Second)
    s.Stop()   // closes quitch after 3 seconds
}()
```

```text
Goroutine waits 3 seconds
→ calls Stop()
→ closes quitch channel
→ loop() in FileServer wakes up
→ FileServer shuts down cleanly
```

---

## 7. loop() — The Heart of the Server

```go
func (s *FileServer) loop() {
    defer func() {
        log.Println("File server stopped due to user quit action")
        s.Transport.Close()
    }()

    for {
        select {
        case msg := <-s.Transport.Consume():
            fmt.Println(msg)
        case <-s.quitch:
            return
        }
    }
}
```

This is the **main event loop** of the server.

### What is select?

```go
select {
case msg := <-channelA:
    // handle msg
case <-channelB:
    // handle this
}
```

`select` is like a `switch` for channels.

```text
It waits for WHICHEVER channel is ready first.
If both are ready at the same time, one is chosen randomly.
If neither is ready, it blocks (waits).
```

### This specific select

```go
select {
case msg := <-s.Transport.Consume():  // received a message from a peer
    fmt.Println(msg)

case <-s.quitch:                       // stop signal received
    return                             // exit loop() → defer runs → server stops
}
```

```text
Option A: A peer sends a message
    → receive it from the Transport channel
    → print it (for now)
    → loop again

Option B: Stop() is called
    → quitch channel closes
    → this case triggers
    → return from loop()
    → defer runs → logs "stopped" + closes Transport
```

### defer in loop()

```go
defer func() {
    log.Println("File server stopped due to user quit action")
    s.Transport.Close()
}()
```

When loop() exits (via `return`):
```text
→ log "File server stopped"
→ close the Transport (stop accepting connections)
```

Clean shutdown, every time.

---

## 8. Start() — Launch the Server

```go
func (s *FileServer) Start() error {
    if err := s.Transport.ListenAndAccept(); err != nil {
        return err
    }
    s.loop()
    return nil
}
```

### Step by step

```text
1. s.Transport.ListenAndAccept()
       → Opens the TCP port
       → Starts the accept loop (in a goroutine inside Transport)
       → Returns immediately if successful

2. s.loop()
       → Runs the main event loop
       → BLOCKS here until Stop() is called
       → This is intentional — the server stays alive

3. return nil
       → Only reached after loop() exits (after Stop())
```

### Why does loop() block?

```text
Start() is called from main.go
If loop() didn't block, main() would reach the end and exit
All goroutines would die
Server would stop immediately

loop() staying alive = server staying alive
```

---

## 9. THE CONNECTION TO main.go

```go
// main.go
s := NewFileServer(fileServerOpts)

go func() {
    time.Sleep(3 * time.Second)
    s.Stop()   // stop after 3 seconds
}()

s.Start()   // blocks here
```

Timeline:
```text
t=0s   Start() called
           → ListenAndAccept() → port open
           → loop() starts → blocking

t=0s   (background goroutine starts counting)

t=3s   s.Stop() called
           → quitch channel closed
           → loop() select unblocks on <-s.quitch
           → loop() returns
           → defer: log + Transport.Close()

t=3s   Start() returns nil
t=3s   main() exits
```

---

## 10. THE BIG PICTURE — All Layers Working Together

```text
main.go
│
├── Creates TCPTransport (Network Layer)
│       │
│       └── listens on :3000
│           accepts connections
│           reads bytes → RPC → pushes to channel
│
├── Creates FileServer (Coordinator)
│       │
│       ├── Transport = TCPTransport
│       │
│       ├── Store = disk storage engine
│       │
│       └── loop():
│               waits on Transport.Consume()
│               → gets messages from peers
│               → (future: stores to disk)
│
└── goroutine: Stop() after 3 seconds
```

---

## 11. WHAT'S MISSING (Coming Next)

Right now `loop()` just prints messages:
```go
case msg := <-s.Transport.Consume():
    fmt.Println(msg)   // ← placeholder
```

In the future this will:
```text
Parse the message
Decide: is this a file upload? a download request? a delete?
Call s.store.Write() or s.store.Read()
Send response back to the peer
```

---

## 12. KEY GO CONCEPTS IN THIS FILE

| Concept | Example | What it does |
|---------|---------|--------------|
| `chan struct{}` | `quitch chan struct{}` | Zero-cost signal channel |
| `close(ch)` | `close(s.quitch)` | Wake up all waiters on channel |
| `select` | `select { case ...: }` | Wait on multiple channels |
| Blocking call | `s.loop()` | Keeps server alive |
| `defer` in loop | `defer Transport.Close()` | Clean shutdown guaranteed |
| Interface field | `Transport p2p.Transport` | Any transport implementation works |

---

# End Notes
