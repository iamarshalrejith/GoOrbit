# message.go — What is a Message?

---

## 1. WHAT IS THIS FILE DOING?

This file defines **what a message looks like** when it travels between two nodes.

One struct. Very simple. But very important.

---

## 2. THE CODE

```go
package p2p

import "net"

type RPC struct {
    From    net.Addr
    Payload []byte
}
```

That's the entire file.

---

## 3. WHAT IS RPC?

**RPC = Remote Procedure Call**

Fancy name. Simple idea.

```text
Node A wants to send data to Node B
That data, while traveling, is called an RPC
```

Think of it like a **letter in an envelope**:

```text
Envelope
│
├── From address    →  net.Addr  (who sent this?)
│
└── Letter content  →  []byte   (what is the data?)
```

---

## 4. WHAT IS net.Addr?

`net.Addr` is Go's built-in type for a **network address**.

Example values:

```text
"192.168.1.10:3000"
"127.0.0.1:5000"
"[::1]:8080"
```

It tells you **which computer sent the message**.

Without `From`, you receive a message but have no idea who sent it.

---

## 5. WHAT IS []byte?

`[]byte` = a **slice of bytes** = raw binary data.

All data on a computer is ultimately bytes (0s and 1s).

```text
Text "hello"        → bytes: [104, 101, 108, 108, 111]
Number 42           → bytes: [0, 0, 0, 42]
Image file          → bytes: [thousands of numbers...]
```

When data travels over a network, it becomes bytes.

When it arrives, you decode those bytes back into something useful.

### Why not use string?

```text
string   → only works for text
[]byte   → works for ANYTHING (text, files, images, audio)
```

In a file storage system, you need to send any type of file.

So `[]byte` is the right choice.

---

## 6. HOW RPC FLOWS THROUGH THE SYSTEM

```text
Node B sends some data
         │
         ▼
Raw bytes arrive at Node A's TCP connection
         │
         ▼
Decoder reads the bytes (encoding.go)
         │
         ▼
RPC struct is filled:
    From    = Node B's address
    Payload = the raw bytes
         │
         ▼
RPC is pushed into the rpcch channel
         │
         ▼
main.go reads the RPC and processes it
```

---

## 7. MENTAL MODEL

```text
Every message in GoOrbit is an RPC.

RPC is just a container:
    - Who sent it?
    - What did they send?

Simple. Clean. Reusable.
```

---

# End Notes
