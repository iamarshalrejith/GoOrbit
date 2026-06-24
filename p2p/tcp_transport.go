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

// TCP Peer represents the remote node over a TCP established connection
type TCPPeer struct {
    // conn is the underlying connection of the peer
    conn net.Conn

    // If we dial and retrieve a connection => outbound == true
    // If we accept and retrive a connection => outbound == false
    outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer{
    return &TCPPeer{
        conn : conn,
        outbound : outbound,
    }
}


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
    peer := NewTCPPeer(conn, true)
    fmt.Printf("new incoming connection %+v\n", peer)
}


// Function Explanations
// 1. ListenAndAccept
/*
Imagine:

    Arshal's Node  <---------------->  Bob's Node

Bob wants to connect to Arshal.

Question:
    How does Bob know Arshal is ready?

Suppose:
    ✓ Arshal's laptop is ON
        -> Not enough

    ✓ GoOrbit is installed
        -> Not enough

    ✓ GoOrbit process is running
        -> Still not enough

Why?

Because the Operating System does not know
which application should receive incoming
TCP connections.

--------------------------------------------------
Real World Analogy
--------------------------------------------------

Bob wants to visit your house.

Before Bob arrives:
    You must open the door.

If the door is closed:
    Bob knocks
    No response
    Visit fails

--------------------------------------------------
Networking Equivalent
--------------------------------------------------

Before Bob can connect:

    Arshal must:
        1. Open a TCP port
        2. Start listening on that port

This is exactly what:

    net.Listen("tcp", ":3000")

does.

It tells the OS:

    "If anyone connects to port 3000,
     send those connections to me."

What Problem Does It Solve ?

Without it:

GoOrbit starts
but nobody can connect to it

With it:

GoOrbit starts
opens TCP port
waits for peers
accepts peers

*/