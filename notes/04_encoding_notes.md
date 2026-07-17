# encoding.go — Decoding Network Messages

---

# 1. WHAT IS THIS FILE DOING?

Whenever data arrives over a TCP connection, it is just a stream of bytes.

The rest of GoOrbit doesn't work directly with raw bytes—it works with `RPC` objects.

The job of a **Decoder** is to translate incoming network bytes into an `RPC`.

```text
TCP Connection

↓

Raw Bytes

↓

Decoder

↓

RPC

↓

FileServer
```

The networking layer only understands bytes.

The application layer understands RPCs.

The Decoder connects these two layers.

---

# 2. THE CODE

```go
type Decoder interface {
    Decode(io.Reader, *RPC) error
}
```

GoOrbit currently provides two implementations:

```text
Decoder

├── DefaultDecoder
│      Used by the project
│
└── GOBDecoder
       Alternative implementation
```

---

# 3. THE DECODER INTERFACE

```go
type Decoder interface {
    Decode(io.Reader, *RPC) error
}
```

Like the other interfaces (`Peer` and `Transport`), this defines a **contract**.

Any type that implements

```go
Decode(io.Reader, *RPC) error
```

automatically becomes a Decoder.

This allows GoOrbit to swap different decoding strategies without changing the Transport.

```text
Transport

↓

Decoder Interface

↓

DefaultDecoder

or

GOBDecoder

or

CustomDecoder
```

---

# 4. WHAT IS io.Reader?

```go
Decode(io.Reader, *RPC)
```

`io.Reader` is one of Go's most important interfaces.

Anything that can produce bytes implements it.

Examples:

```text
net.Conn

↓

TCP connection
```

```text
os.File

↓

File on disk
```

```text
bytes.Buffer

↓

Memory buffer
```

```text
strings.Reader

↓

String
```

All of these satisfy

```go
Read([]byte)
```

Because the Decoder accepts an `io.Reader`, it doesn't care where the bytes come from.

During production:

```go
Decode(conn, &rpc)
```

During testing:

```go
Decode(bytes.NewReader(data), &rpc)
```

Same Decoder.

Different data source.

No code changes.

---

# 5. WHY *RPC?

```go
Decode(io.Reader, *RPC)
```

The Decoder needs to fill in the RPC.

If we passed the struct by value:

```go
func Decode(r io.Reader, msg RPC)
```

the Decoder would modify only a copy.

The caller would never see those changes.

Instead, we pass a pointer.

```go
func Decode(r io.Reader, msg *RPC)
```

Now the Decoder modifies the original RPC.

```text
Caller

↓

RPC

↓

Pointer

↓

Decoder fills fields

↓

Caller sees updated RPC
```

---

# 6. DefaultDecoder

This is the Decoder used by GoOrbit.

Its job is to determine what kind of data has arrived.

There are two possibilities:

1. A normal RPC message
2. A file stream

---

# 7. Step 1 — Read the First Byte

```go
peekBuf := make([]byte, 1)

io.ReadFull(r, peekBuf)
```

The Decoder first reads **exactly one byte**.

This byte acts like a small header.

```text
Incoming TCP Data

↓

Read first byte

↓

What kind of data is this?
```

Why `ReadFull`?

Because we need exactly one byte.

If zero bytes are read, we cannot determine what follows.

---

# 8. Detecting a File Stream

```go
stream := peekBuf[0] == IncomingStream
```

GoOrbit reserves one special byte value to indicate:

> "The following data is not an RPC message—it is a file stream."

If that byte matches `IncomingStream`:

```go
msg.Stream = true
return nil
```

Notice something important.

The Decoder **does not read the file**.

It only informs the rest of the application that a stream is beginning.

```text
Read first byte

↓

IncomingStream ?

├── Yes

│

│   RPC.Stream = true

│

│   Return immediately

│

└── No

↓

Continue decoding message
```

---

# 9. Why Doesn't the Decoder Read the File?

Imagine receiving a 5 GB video.

Putting all of that inside

```go
msg.Payload
```

would require enormous memory.

Instead:

```text
Decoder

↓

Detect stream

↓

Tell FileServer

↓

FileServer streams bytes directly

↓

Store on disk
```

The file is processed while it is arriving.

This is called **streaming**.

Benefits:

- Low memory usage
- Supports huge files
- Faster transfers
- No unnecessary buffering

---

# 10. Normal RPC Messages

If the first byte is **not** `IncomingStream`,

the Decoder assumes a normal message.

```go
buf := make([]byte, 1028)

n, err := r.Read(buf)

msg.Payload = buf[:n]
```

Step by step:

```text
Allocate buffer

↓

Read bytes

↓

Read 85 bytes

↓

Store only those 85 bytes

↓

RPC.Payload ready
```

Notice:

```go
buf[:n]
```

If we didn't slice,

```text
Payload

[real data]

+

943 useless zeros
```

would be stored.

Instead,

```text
Only actual message bytes

↓

Payload
```

---

# 11. GOBDecoder

Go also provides another implementation.

```go
func (dec GOBDecoder) Decode(...)
```

This uses Go's built-in

```
encoding/gob
```

package.

Flow:

```text
Go Struct

↓

GOB Encoder

↓

Binary bytes

↓

Network

↓

GOB Decoder

↓

Original Struct
```

GOB is useful when transmitting Go structures directly.

Although GoOrbit currently uses `DefaultDecoder`, the interface allows switching to `GOBDecoder` without changing the Transport.

---

# 12. Why Separate the Decoder?

Suppose TCPTransport also contained decoding logic.

Then one file would be responsible for:

- sockets
- accepting peers
- reading bytes
- decoding messages

That mixes multiple responsibilities.

Instead:

```text
TCPTransport

↓

Receive bytes

↓

Decoder

↓

Produce RPC

↓

FileServer

↓

Business logic
```

Each component has exactly one responsibility.

This follows the **Single Responsibility Principle (SRP).**

---

# 13. Message vs File Stream

GoOrbit transfers two different kinds of data.

## RPC Messages

Small pieces of information.

Examples:

```text
Store this file

Get this file

Peer joined

Metadata
```

These become an `RPC`.

---

## File Streams

Large file contents.

Examples:

```text
movie.mp4

backup.zip

photo.jpg
```

These are **not copied into `RPC.Payload`.**

Instead,

```text
IncomingStream marker

↓

RPC.Stream = true

↓

FileServer handles stream

↓

Store file
```

This makes GoOrbit efficient even for very large files.

---

# 14. Complete Flow

```text
TCP Connection

↓

Read first byte

↓

IncomingStream ?

├── Yes

│

│   RPC.Stream = true

│

│   FileServer handles file stream

│

└── No

↓

Read remaining bytes

↓

Fill RPC.Payload

↓

Push RPC into channel

↓

FileServer processes message
```

---

# 15. Mental Model

```text
TCP Connection

↓

Bytes arrive

↓

Decoder

↓

RPC

↓

Transport Channel

↓

FileServer

↓

Application Logic
```

Think of the Decoder as a translator.

The network speaks **bytes**.

The application speaks **RPCs**.

The Decoder translates between them.

---

# Key Takeaway

`encoding.go` separates raw networking from application logic.

It converts incoming TCP data into an `RPC`, detects whether the incoming data is a normal message or a file stream, and leaves higher-level processing to the `FileServer`.

By keeping decoding separate from the Transport, GoOrbit remains modular, reusable, and easy to extend with new decoding strategies in the future.

---

# End Notes