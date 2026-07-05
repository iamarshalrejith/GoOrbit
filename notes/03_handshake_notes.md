# handshake.go — Peer Verification

---

## 1. WHAT IS THIS FILE DOING?

Before two GoOrbit nodes start sharing data, they need to verify each other.

```text
"Are you really a GoOrbit node?"
"Are you running a compatible version?"
"Do I trust you?"
```

This process is called a **Handshake**.

This file defines the TYPE of function that performs the handshake, and a default placeholder.

---

## 2. THE CODE

```go
package p2p

type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
```

Small file. But has important Go concepts inside.

---

## 3. WHAT IS A FUNCTION TYPE?

In Go, **functions are values**. You can store a function in a variable.

```go
type HandshakeFunc func(Peer) error
```

This creates a **new type** called `HandshakeFunc`.

Any function that:
- takes a `Peer` as input
- returns an `error`

...is automatically a `HandshakeFunc`.

### Example

```go
// This is a HandshakeFunc
func myCheck(p Peer) error {
    // verify the peer
    return nil
}

// You can store it in a variable
var h HandshakeFunc = myCheck

// You can pass it to a struct
opts := TCPTransportOpts{
    HandshakeFunc: myCheck,
}
```

### Why is this useful?

Because now `TCPTransportOpts` can accept **any** handshake logic.

```text
TCPTransportOpts
│
├── HandshakeFunc: NOPHandshakeFunc    ← for testing (do nothing)
├── HandshakeFunc: VersionCheckFunc    ← check protocol version
└── HandshakeFunc: CryptoHandshake     ← full encryption handshake
```

Swap the function. Change the behavior. No other code changes.

---

## 4. WHAT IS NOPHandshakeFunc?

```go
func NOPHandshakeFunc(Peer) error { return nil }
```

**NOP = No Operation**

It does absolutely nothing. Just returns `nil` (meaning: no error, success).

### Why have a function that does nothing?

Because right now we are still building the project.

```text
Real handshake is complex:
    - Exchange version info
    - Cryptographic challenge
    - Check compatibility

We don't want to build that yet.
We just want the networking to work first.

NOPHandshakeFunc is a placeholder.
It says "skip handshake for now."
```

Think of it as:

```text
Security Guard at office entrance
Today: door is open, no ID check (NOPHandshake)
Later: proper ID badge scanner (real handshake)
```

---

## 5. WHAT IS error IN GO?

`error` is a built-in interface in Go.

```go
func someFunc() error {
    return nil     // success, no error
}

func someFunc() error {
    return fmt.Errorf("something went wrong") // failure
}
```

Convention in Go:

```text
nil   = success
error = something went wrong, here's what
```

Callers always check:

```go
if err := someFunc(); err != nil {
    // handle the problem
}
```

---

## 6. HOW HANDSHAKE IS USED IN TCP TRANSPORT

```go
// Inside handleConn in tcp_transport.go:

peer := NewTCPPeer(conn, true)

if err = t.HandshakeFunc(peer); err != nil {
    return  // handshake failed → reject peer → close connection
}

// if we reach here → handshake passed → continue
```

Flow:

```text
New peer connects
       │
       ▼
Run HandshakeFunc(peer)
       │
   ┌───┴───┐
   │       │
 Pass     Fail
   │       │
   ▼       ▼
Continue  Close connection
talking   Drop peer
```

---

## 7. MENTAL MODEL

```text
HandshakeFunc = a pluggable verification step

Right now:     NOPHandshakeFunc (do nothing)
In the future: real security checks

The rest of the code doesn't care WHICH function it is.
It just calls it and checks the result.
```

---

# End Notes
