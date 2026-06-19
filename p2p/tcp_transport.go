package p2p

import (
	"fmt"
	"net"
	"sync"
)

/*
- Each GoOrbit node needs a networking component.
- We chose TCP as the communication protocol.
- TCPTransport is the networking component of a node.
- It stores networking-related information and contains networking-related behavior.
*/

/*
If we already have TCP, why create TCPTransport?
Ans: TCP is just a protocol.TCPTransport is OUR object that USES TCP.
*/

// TCPTransport manages peer-to-peer communication over TCP.
type TCPTransport struct {
	listenAddress string // Address this node listens on (e.g. ":3000")
	listener net.Listener // TCP listener that accepts incoming connections.

	mu sync.RWMutex // Protects concurrent access to peers.
	peers map[net.Addr]Peer // Connected peers indexed by network address.
}

func NewTCPTransport(listenAddr string) *TCPTransport{
    return &TCPTransport{
        listenAddress: listenAddr,
    }
}

func (t *TCPTransport) ListenAndAccept() error{
    var err error
    ln, err := net.Listen("tcp",t.listenAddress)
    if err != nil{
        return err
}
    t.listener = ln

    go t.startAcceptLoop()

    return nil
}


func (t *TCPTransport) startAcceptLoop(){
     for {
        conn, err := t.listener.Accept()
        if err != nil {
            fmt.Printf("TCP accept error: %s\n",err)
        }
        go t.handleConn(conn)
     }
}

func (t *TCPTransport) handleConn(conn net.Conn){

}

















/*
===============================================================================
BEGINNER NOTES - TCP TRANSPORT
===============================================================================

1. WHAT IS THIS FILE DOING?
---------------------------

This file defines the Network Layer of GoOrbit.

GoOrbit Architecture:

    GoOrbit
    │
    ├── Storage Layer
    ├── Encryption Layer
    ├── Chunking Layer
    └── Network Layer
          │
          └── TCPTransport

Responsibilities of TCPTransport:

    ✓ Listen for incoming connections
    ✓ Connect to other nodes
    ✓ Track connected peers
    ✓ Send data
    ✓ Receive data

Think of TCPTransport as the Network Manager of a node.


2. WHAT IS A STRUCT?
--------------------

Java:

    class Student {
        String name;
        int age;
    }

Go:

    type Student struct {
        name string
        age  int
    }

Usage:

    s := Student{
        name: "Arshal",
        age: 21,
    }

In this file:

    type TCPTransport struct {...}

is simply a custom data type that stores all network-related information.


3. WHAT IS net.Listener?
------------------------

A Listener waits for incoming network connections.

Real Life Analogy:

    Customer arrives
           ↓
      Door opens
           ↓
     Customer enters

Networking:

    Connection arrives
            ↓
     Listener accepts
            ↓
     Connection established

Example:

    listener, err := net.Listen("tcp", ":3000")

This opens Port 3000 and waits for connections.

Visual:

    Waiting...
    Waiting...
    Waiting...
    Connection Received!


4. WHAT IS A PEER?
------------------

In Distributed Systems:

    Node A
    Node B
    Node C

Every node sees other nodes as Peers.

Example:

    A <------> B

For A:

    Peer = B

For B:

    Peer = A

A Peer simply means:

    "Another machine participating in the network"


5. WHAT IS map[net.Addr]Peer?
-----------------------------

Java:

    HashMap<String, User>

Go:

    map[string]User

Current Code:

    map[net.Addr]Peer

Meaning:

    Network Address -> Peer

Example:

    192.168.1.10:3000 -> PeerA
    192.168.1.11:3000 -> PeerB
    192.168.1.12:3000 -> PeerC

Purpose:

    Keep track of connected nodes.


6. WHAT IS sync.RWMutex?
------------------------

Mutex = Mutual Exclusion

Used to prevent multiple goroutines from modifying
shared data at the same time.

Problem:

    Goroutine 1 -> Add Peer
    Goroutine 2 -> Remove Peer
    Goroutine 3 -> Read Peer

All running simultaneously.

Without Mutex:

    Race Condition
    Corrupted Data
    Crash

Solution:

    mu.Lock()

Only one goroutine can enter.

    G1 enters
    G2 waits
    G3 waits

After:

    mu.Unlock()

Next goroutine proceeds.

RWMutex = Read Write Mutex

Allows:

    Multiple Readers ✓
    Single Writer ✓

Methods:

    mu.RLock()
    mu.RUnlock()

    mu.Lock()
    mu.Unlock()


7. WHY USE A CONSTRUCTOR FUNCTION?
----------------------------------

Java:

    TCPTransport t =
        new TCPTransport(":3000");

Go does not have constructors.

Convention:

    func NewTCPTransport(...) {...}

The "New" prefix is Go's standard way of creating objects.


8. WHAT DOES &TCPTransport{} MEAN?
----------------------------------

Without '&':

    t := TCPTransport{}

Creates the actual struct value.

With '&':

    t := &TCPTransport{}

Creates the struct and returns its address.

Visual:

    TCPTransport
    Address: 0x12345

Variable stores:

    0x12345

instead of the full object.

This is called a Pointer.


9. WHY RETURN Transport INSTEAD OF TCPTransport?
------------------------------------------------

Function:

    func NewTCPTransport(...) Transport

returns:

    &TCPTransport{}

Reason:

    TCPTransport implements Transport.

Similar Java Example:

    Animal a = new Dog();

Go Equivalent:

    var t Transport = &TCPTransport{}

Benefits:

    Later we can create:

        TCPTransport
        UDPTransport
        WebsocketTransport

and use all of them through the same interface.


10. MENTAL MODEL
----------------

Whenever you see TCPTransport, imagine:

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

This struct is essentially the Network Manager
for one GoOrbit node.

===============================================================================
END NOTES
===============================================================================
*/