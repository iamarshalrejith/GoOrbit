# TCP Transport - Beginner Notes

---

## 1. WHAT IS THIS FILE DOING?

This file defines the **Network Layer** of GoOrbit.

### GoOrbit Architecture

```text
GoOrbit
│
├── Storage Layer
├── Encryption Layer
├── Chunking Layer
└── Network Layer
      │
      └── TCPTransport
```

### Responsibilities of TCPTransport

* ✓ Listen for incoming connections
* ✓ Connect to other nodes
* ✓ Track connected peers
* ✓ Send data
* ✓ Receive data

Think of **TCPTransport** as the **Network Manager** of a node.

---

## 2. WHAT IS A STRUCT?

### Java

```java
class Student {
    String name;
    int age;
}
```

### Go

```go
type Student struct {
    name string
    age  int
}
```

### Usage

```go
s := Student{
    name: "Arshal",
    age: 21,
}
```

In this file:

```go
type TCPTransport struct {...}
```

is simply a custom data type that stores all network-related information.

---

## 3. WHAT IS net.Listener?

A **Listener** waits for incoming network connections.

### Real Life Analogy

```text
Customer arrives
       ↓
  Door opens
       ↓
 Customer enters
```

### Networking

```text
Connection arrives
        ↓
 Listener accepts
        ↓
 Connection established
```

### Example

```go
listener, err := net.Listen("tcp", ":3000")
```

This opens Port **3000** and waits for connections.

### Visual

```text
Waiting...
Waiting...
Waiting...
Connection Received!
```

---

## 4. WHAT IS A PEER?

In Distributed Systems:

```text
Node A
Node B
Node C
```

Every node sees other nodes as **Peers**.

### Example

```text
A <------> B
```

For A:

```text
Peer = B
```

For B:

```text
Peer = A
```

A Peer simply means:

> "Another machine participating in the network"

---

## 5. WHAT IS map[net.Addr]Peer?

### Java

```java
HashMap<String, User>
```

### Go

```go
map[string]User
```

### Current Code

```go
map[net.Addr]Peer
```

Meaning:

```text
Network Address -> Peer
```

### Example

```text
192.168.1.10:3000 -> PeerA
192.168.1.11:3000 -> PeerB
192.168.1.12:3000 -> PeerC
```

### Purpose

Keep track of connected nodes.

---

## 6. WHAT IS sync.RWMutex?

**Mutex = Mutual Exclusion**

Used to prevent multiple goroutines from modifying shared data at the same time.

### Problem

```text
Goroutine 1 -> Add Peer
Goroutine 2 -> Remove Peer
Goroutine 3 -> Read Peer
```

All running simultaneously.

### Without Mutex

```text
Race Condition
Corrupted Data
Crash
```

### Solution

```go
mu.Lock()
```

Only one goroutine can enter.

```text
G1 enters
G2 waits
G3 waits
```

After:

```go
mu.Unlock()
```

Next goroutine proceeds.

### RWMutex = Read Write Mutex

Allows:

```text
Multiple Readers ✓
Single Writer ✓
```

### Methods

```go
mu.RLock()
mu.RUnlock()

mu.Lock()
mu.Unlock()
```

---

## 7. WHY USE A CONSTRUCTOR FUNCTION?

### Java

```java
TCPTransport t =
    new TCPTransport(":3000");
```

Go does not have constructors.

Convention:

```go
func NewTCPTransport(...) {...}
```

The **New** prefix is Go's standard way of creating objects.

---

## 8. WHAT DOES &TCPTransport{} MEAN?

### Without '&'

```go
t := TCPTransport{}
```

Creates the actual struct value.

### With '&'

```go
t := &TCPTransport{}
```

Creates the struct and returns its address.

### Visual

```text
TCPTransport
Address: 0x12345
```

Variable stores:

```text
0x12345
```

instead of the full object.

This is called a **Pointer**.

---

## 9. WHY RETURN Transport INSTEAD OF TCPTransport?

Function:

```go
func NewTCPTransport(...) Transport
```

returns:

```go
&TCPTransport{}
```

### Reason

TCPTransport implements Transport.

### Similar Java Example

```java
Animal a = new Dog();
```

### Go Equivalent

```go
var t Transport = &TCPTransport{}
```

### Benefits

Later we can create:

```text
TCPTransport
UDPTransport
WebsocketTransport
```

and use all of them through the same interface.

---

## 10. MENTAL MODEL

Whenever you see TCPTransport, imagine:

```text
TCPTransport
│
├── Address
│     ":3000"
│
├── TCP Listener
│     Waits for connections
│
├── Mutex
│     Protects shared data
│
└── Peer Map
      Stores connected nodes
```

This struct is essentially the **Network Manager** for one GoOrbit node.

---

# End Notes
