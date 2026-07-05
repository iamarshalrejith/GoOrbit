# encoding.go — Decoding Raw Network Bytes

---

## 1. WHAT IS THIS FILE DOING?

When data arrives over TCP, it's **raw bytes**.

You need to convert those bytes into an `RPC` struct that your code can work with.

That conversion is called **decoding**.

This file defines HOW decoding happens.

---

## 2. THE CODE

```go
package p2p

import (
    "encoding/gob"
    "io"
)

type Decoder interface {
    Decode(io.Reader, *RPC) error
}

type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
    return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
    buf := make([]byte, 1028)
    n, err := r.Read(buf)
    if err != nil {
        return err
    }
    msg.Payload = buf[:n]
    return nil
}
```

---

## 3. THE DECODER INTERFACE

```go
type Decoder interface {
    Decode(io.Reader, *RPC) error
}
```

Any type that has a `Decode` method (with those exact inputs/outputs) IS a Decoder.

Same pattern as `Transport` and `Peer` — a contract.

This means you can swap out decoding logic later without breaking anything.

---

## 4. WHAT IS io.Reader?

`io.Reader` is Go's built-in interface for anything you can **read from**.

```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

Things that implement io.Reader:

```text
net.Conn    → a TCP connection
os.File     → a file on disk
bytes.Buffer → an in-memory buffer
strings.Reader → a string
```

By accepting `io.Reader` instead of `net.Conn`, the Decoder works with ANY data source.

### Why is this smart?

```text
During real use:
    Decode(tcpConnection, &rpc)

During testing:
    Decode(strings.NewReader("test data"), &rpc)

Same function. Different sources. No code changes.
```

---

## 5. WHAT IS *RPC?

```go
Decode(r io.Reader, msg *RPC) error
```

`*RPC` = a **pointer** to an RPC struct.

### Without pointer

```go
func fill(msg RPC) {
    msg.Payload = []byte("hello") // modifies a COPY
}
// original RPC is unchanged
```

### With pointer

```go
func fill(msg *RPC) {
    msg.Payload = []byte("hello") // modifies the ORIGINAL
}
// original RPC is now updated
```

We pass `*RPC` so the Decode function can actually fill in the data and the caller sees the result.

---

## 6. DefaultDecoder — Used Right Now

```go
type DefaultDecoder struct{}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
    buf := make([]byte, 1028)   // Step 1: create empty buffer of 1028 bytes
    n, err := r.Read(buf)       // Step 2: read bytes from the connection
    if err != nil {
        return err              // Step 3: if read failed, return the error
    }
    msg.Payload = buf[:n]       // Step 4: put the bytes into RPC's Payload
    return nil                  // Step 5: success
}
```

### Step by Step

```text
TCP connection has bytes: [72, 101, 108, 108, 111]  ("Hello")
                                    │
                                    ▼
            buf := make([]byte, 1028)   → empty [0,0,0,...,0] (1028 zeros)
                                    │
                                    ▼
            r.Read(buf)             → buf is now [72,101,108,108,111,0,...,0]
                                      n = 5 (read 5 bytes)
                                    │
                                    ▼
            msg.Payload = buf[:5]   → [72,101,108,108,111]
```

### What is buf[:n]?

Go slice syntax — take only the first `n` elements.

```text
buf   = [72,101,108,108,111, 0, 0, 0, ... 0]   (1028 items)
buf[:5] = [72,101,108,108,111]                  (just the real data)
```

We don't want to include the trailing zeros.

---

## 7. GOBDecoder — For Later

```go
type GOBDecoder struct{}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
    return gob.NewDecoder(r).Decode(msg)
}
```

**GOB = Go's Binary format** — like JSON but designed for Go structs, more efficient.

Used when you want to send **structured data** (a full RPC struct with both `From` and `Payload` properly encoded).

Right now we use `DefaultDecoder` because we're just testing with raw bytes.

Later, `GOBDecoder` will let us encode/decode full structured messages.

### Comparison

```text
DefaultDecoder
    Input:  raw bytes (anything)
    Output: those bytes go into RPC.Payload
    Use:    simple/testing

GOBDecoder
    Input:  a GOB-encoded RPC struct
    Output: fully decoded RPC with all fields
    Use:    production, structured messages
```

---

## 8. MENTAL MODEL

```text
encoding.go = Translator

Raw bytes arrive from network
         │
         ▼
Decoder translates them into RPC struct
         │
         ▼
Rest of the code works with clean RPC objects

DefaultDecoder = simple translator (put bytes in Payload)
GOBDecoder     = smart translator (decode full Go struct)
```

---

# End Notes
