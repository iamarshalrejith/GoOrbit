# transport.go — Interfaces (Blueprints)

---

## 1. WHAT IS THIS FILE DOING?

This is the **first file you should read** in the whole project.

It defines the **contracts** — rules that every networking component in GoOrbit must follow.

It does NOT contain any real implementation. Just the rules.

### Real World Analogy

```text
Government issues a Driver's License requirement:

    Must pass written test
    Must pass driving test
    Must have working eyesight

Anyone who passes all three = Licensed Driver
Doesn't matter if you drive a car, truck, or bus
```

Same idea here:

```text
GoOrbit defines:

    Peer must have Close()
    Transport must have ListenAndAccept() and Consume()

Any type that has those methods = qualifies automatically
```

---

## 2. THE PEER INTERFACE

```go
type Peer interface {
    Close() error
}
```

### What is a Peer?

In a distributed system, every other computer you are connected to is a **Peer**.

```text
[Node A] <-----------> [Node B]

For A:   B is a Peer
For B:   A is a Peer
```

### What does Close() do?

When you are done talking to a peer, you close the connection.

`Close() error` means:

```text
Try to close
If something goes wrong, return an error
If everything is fine, return nil
```

### Why an Interface instead of a Struct?

Because a Peer could be:

```text
TCPPeer     → connected over TCP
UDPPeer     → connected over UDP
MockPeer    → fake peer for testing
```

All of them can be treated as a `Peer` as long as they have `Close()`.

---

## 3. THE TRANSPORT INTERFACE

```go
type Transport interface {
    ListenAndAccept() error
    Consume() <-chan RPC
}
```

Transport is anything that handles communication between nodes.

### ListenAndAccept() error

```text
Open a port
Start waiting for connections
If it fails → return the error
```

### Consume() <-chan RPC

```text
Return a channel
Through which incoming messages can be read
```

`<-chan RPC` means **read-only channel of RPC messages**.

The caller can only READ from it. Cannot write to it.

### Visual

```text
Transport
│
├── ListenAndAccept()   →   Opens door for peers to connect
│
└── Consume()           →   Pipe that delivers incoming messages
```

---

## 4. HOW GO INTERFACES WORK (Important!)

### Java / Python Style (explicit)

```java
class TCPTransport implements Transport {
    ...
}
```

You have to say "I implement Transport."

### Go Style (implicit)

```go
type TCPTransport struct { ... }

func (t *TCPTransport) ListenAndAccept() error { ... }
func (t *TCPTransport) Consume() <-chan RPC    { ... }
```

You never say "implements Transport."

Go sees that TCPTransport has both required methods.

Go automatically treats it as a Transport.

This is called **implicit interface satisfaction**.

---

## 5. WHY DO WE NEED INTERFACES?

Without interfaces:

```text
Code is tightly coupled to TCPTransport
Want to switch to UDP? Change 100 places in code
Want to test? Must use real TCP
```

With interfaces:

```text
Code only knows about Transport (the contract)
Swap TCPTransport for UDPTransport → zero changes elsewhere
Testing? Pass in a fake MockTransport
```

### Future Transport Options

```text
Transport Interface
       │
       ├── TCPTransport       ← built now
       ├── UDPTransport       ← could add later
       └── WebSocketTransport ← could add later
```

---

## 6. MENTAL MODEL

```text
transport.go = Rulebook

Rule 1: To be a Peer  → you must have Close()
Rule 2: To be a Transport → you must have ListenAndAccept() and Consume()

Everyone else in the project uses these rules
Nobody cares about implementation details
```

---

# End Notes
