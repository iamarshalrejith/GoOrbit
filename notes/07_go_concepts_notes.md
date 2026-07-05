# Go Concepts — Everything Used in GoOrbit So Far

---

## 1. GOROUTINES

### What is it?

A goroutine is Go's way of doing **multiple things at the same time**.

Think of it like having multiple workers doing separate jobs simultaneously.

### Syntax

```go
go someFunction()   // runs in background
```

The `go` keyword launches a function in a new goroutine.

### Without goroutines

```text
Accept Peer A → handle A forever → can't accept Peer B
```

### With goroutines

```text
Accept Peer A → goroutine A ─┐
Accept Peer B → goroutine B ─┤  all run at same time
Accept Peer C → goroutine C ─┘
```

### In GoOrbit

```go
go t.startAcceptLoop()     // accept loop runs in background
go t.handleConn(conn)      // each peer gets its own goroutine
go func() { ... }()        // anonymous goroutine in main.go
```

### Important

```text
Goroutines are NOT threads.
They are much lighter (cheaper) than OS threads.
Go can run thousands of goroutines easily.
```

---

## 2. CHANNELS

### What is it?

A **pipe** that safely passes data between goroutines.

```text
Goroutine A  →→→→  channel  →→→→  Goroutine B
  (sender)          (pipe)          (receiver)
```

### Syntax

```go
ch := make(chan int)     // create a channel of ints

ch <- 42                 // send 42 into channel (blocks until someone reads)
value := <-ch            // receive from channel (blocks until something is sent)
```

### Read-only channel

```go
func something() <-chan RPC {   // caller can only READ from this
    return ch
}
```

### In GoOrbit

```go
rpcch chan RPC           // created in NewTCPTransport
t.rpcch <- rpc           // handleConn sends message into channel
msg := <-tr.Consume()    // main.go reads message from channel
```

### Why not just use a global variable?

```text
Global variable: multiple goroutines read/write at same time → crash (race condition)
Channel: designed for safe sharing between goroutines → no crash
```

---

## 3. INTERFACES

### What is it?

A **contract** — defines what methods a type must have.

Any type that has those methods automatically qualifies. No need to say "implements."

### Syntax

```go
type Animal interface {
    Sound() string
    Move() error
}

type Dog struct{}

func (d Dog) Sound() string { return "woof" }
func (d Dog) Move() error   { return nil }

// Dog automatically satisfies Animal interface
var a Animal = Dog{}   // works!
```

### In GoOrbit

```go
type Peer interface {
    Close() error
}

type Transport interface {
    ListenAndAccept() error
    Consume() <-chan RPC
}

type Decoder interface {
    Decode(io.Reader, *RPC) error
}
```

### Why useful?

```text
Code works with the interface, not the concrete type.
Swap implementations without changing other code.
Easy to test (use fake/mock implementations).
```

---

## 4. POINTERS

### What is it?

A pointer **stores the memory address** of a variable instead of the value itself.

### Without pointer (copy)

```go
func double(n int) {
    n = n * 2   // modifies the copy, not original
}

x := 5
double(x)
fmt.Println(x)  // still 5
```

### With pointer (original)

```go
func double(n *int) {
    *n = *n * 2   // modifies the original
}

x := 5
double(&x)
fmt.Println(x)  // now 10
```

### & and * explained

```text
&x   → "give me the address of x"   (creates a pointer)
*p   → "give me the value at address p"   (dereferences)
```

### In GoOrbit

```go
func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
    return &TCPPeer{...}   // return pointer to new struct
}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
    msg.Payload = buf[:n]   // modifies the original RPC
}
```

### Pointer receiver vs value receiver

```go
// Value receiver — works on a COPY
func (p TCPPeer) String() string { ... }

// Pointer receiver — works on the ORIGINAL
func (p *TCPPeer) Close() error { ... }
```

Use pointer receiver when the method needs to modify the struct, or when the struct is large (avoid copying).

---

## 5. DEFER

### What is it?

`defer` schedules a function to run **at the end of the current function**, no matter what.

### Syntax

```go
func example() {
    defer fmt.Println("I run last")
    fmt.Println("I run first")
    fmt.Println("I run second")
}

// Output:
// I run first
// I run second
// I run last
```

### Why is it useful?

Guarantees cleanup even if an error occurs.

```go
func handleConn(conn net.Conn) {
    defer conn.Close()   // always closes, even if panic or error

    // ... do stuff
    // error happens here
    return   // conn.Close() still runs!
}
```

### In GoOrbit

```go
defer func() {
    fmt.Printf("Dropping peer connection: %s", err)
    conn.Close()
}()
```

Note the `()` at the end — deferred anonymous function must be called.

### Like Java's finally

```java
try {
    // ...
} finally {
    connection.close();  // always runs
}
```

Go's defer does the same thing, more cleanly.

---

## 6. STRUCT EMBEDDING

### What is it?

Including one struct inside another to **reuse its fields** without inheritance.

### Without embedding

```go
type TCPTransport struct {
    opts TCPTransportOpts
    listener net.Listener
}

// Access:
t.opts.ListenAddr    // verbose
```

### With embedding

```go
type TCPTransport struct {
    TCPTransportOpts   // embedded (no field name)
    listener net.Listener
}

// Access:
t.ListenAddr         // direct, clean
```

### In GoOrbit

```go
type TCPTransport struct {
    TCPTransportOpts        // all its fields become direct
    listener net.Listener
    rpcch    chan RPC
}
```

Now `t.ListenAddr`, `t.HandshakeFunc`, `t.Decoder`, `t.OnPeer` all work directly.

### Not the same as inheritance

```text
Java inheritance: Dog IS-A Animal, gets Animal's behavior
Go embedding:     TCPTransport HAS-A TCPTransportOpts, gets its fields
```

---

## 7. FUNCTION TYPES

### What is it?

In Go, functions are **first-class values**. You can store them in variables and pass them around.

### Syntax

```go
type HandshakeFunc func(Peer) error

// any function matching that signature is a HandshakeFunc
func myCheck(p Peer) error { return nil }

var h HandshakeFunc = myCheck   // store function in variable
h(somePeer)                     // call it
```

### Why useful?

Allows you to inject behavior:

```go
// For testing:
opts.HandshakeFunc = NOPHandshakeFunc  // skip

// For production:
opts.HandshakeFunc = FullCryptoHandshake  // real verification
```

---

## 8. ERROR HANDLING

Go has no exceptions (no try/catch). Errors are **return values**.

### Pattern

```go
func doSomething() error {
    // success
    return nil

    // failure
    return fmt.Errorf("something went wrong: %s", reason)
}

// caller checks:
if err := doSomething(); err != nil {
    // handle it
    log.Fatal(err)   // or return err, or continue, etc.
}
```

### In GoOrbit

```go
if err := tr.ListenAndAccept(); err != nil {
    log.Fatal(err)   // can't start → exit program
}

if err = t.HandshakeFunc(peer); err != nil {
    return   // handshake failed → drop peer
}
```

### log.Fatal vs return vs continue

```text
log.Fatal(err) → print error + EXIT program (unrecoverable)
return         → exit current function only
continue       → skip to next loop iteration
```

---

## 9. ANONYMOUS FUNCTIONS

### What is it?

A function with no name, defined and used inline.

### Syntax

```go
// named function
func greet() { fmt.Println("hello") }

// anonymous function
func() { fmt.Println("hello") }

// call immediately
func() { fmt.Println("hello") }()

// store in variable
greet := func() { fmt.Println("hello") }
greet()

// as goroutine
go func() {
    for { ... }
}()
```

### In GoOrbit (main.go)

```go
go func() {
    for {
        msg := <-tr.Consume()
        fmt.Printf("%+v\n", msg)
    }
}()
```

Anonymous goroutine — defined and launched in one place.

---

## 10. QUICK REFERENCE TABLE

| Concept | Keyword/Syntax | Purpose |
|---------|----------------|---------|
| Goroutine | `go func()` | Concurrent execution |
| Channel | `chan`, `<-` | Safe data passing between goroutines |
| Interface | `type X interface { }` | Contract/blueprint |
| Pointer | `&`, `*` | Work with original, not copy |
| Defer | `defer` | Guaranteed cleanup |
| Embedding | struct inside struct | Field reuse |
| Function type | `type F func(args) return` | Pluggable behavior |
| Error handling | `if err != nil` | Explicit error checking |
| Anonymous func | `func() { }()` | Inline function definition |

---

# End Notes
