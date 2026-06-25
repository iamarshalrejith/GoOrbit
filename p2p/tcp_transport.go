package p2p

import (
	"fmt"
	"net"
	"sync"
)


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

type TCPTransportOpts struct{
        ListenAddr string
        HandshakeFunc HandshakeFunc
        Decoder Decoder
}


// TCPTransport manages peer-to-peer communication over TCP.
type TCPTransport struct {
    TCPTransportOpts
	listenAddress string // Address this node listens on (e.g. ":3000")
	listener net.Listener // TCP listener that accepts incoming connections.
    shakeHands HandshakeFunc
    decoder Decoder

	mu sync.RWMutex // Protects concurrent access to peers.
	peers map[net.Addr]Peer // Connected peers indexed by network address.
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport{
    return &TCPTransport{
        TCPTransportOpts: opts,
    }
}

func (t *TCPTransport) ListenAndAccept() error{
    var err error
    ln, err := net.Listen("tcp",t.ListenAddr) 
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
        fmt.Printf("new incoming connection %+v\n",conn)
        go t.handleConn(conn)
     }
}

func (t *TCPTransport) handleConn(conn net.Conn){
    peer := NewTCPPeer(conn, false)

    if err := t.HandshakeFunc(peer); err !=nil {
        conn.Close()
        fmt.Printf("TCP HandShake Error: %s\n",err)
        return 
    }
  
    // Read Loop
    msg := &Message{}
for {
    if err := t.Decoder.Decode(conn, msg); err!= nil{
        fmt.Printf("TCP error: %s\n",err)
        continue
    }

    fmt.Printf("Message: %+v\n", msg)
}
}

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

// 2. startAcceptLoop
/*
After opening the TCP port, our job is still not finished.

Think of a receptionist at a hotel.

Opening the hotel doors doesn't automatically
check guests into their rooms.

Someone must continuously wait at the front desk.

Whenever a guest arrives:

    Accept the guest
    Register them
    Hand them over to the hotel staff

Networking works exactly the same way.

--------------------------------------------------
Without an Accept Loop
--------------------------------------------------

Arshal opens port 3000.

Bob tries to connect.

Nobody is waiting to receive Bob.

Result:

    Connection eventually times out.

--------------------------------------------------
What Accept() Does
--------------------------------------------------

conn, err := listener.Accept()

Accept blocks (waits) until a remote peer
tries to connect.

Once a peer connects:

    Accept() returns a brand-new net.Conn.

Think of it as:

    "A private telephone line between
     this node and that peer."

Every peer gets its own connection.

--------------------------------------------------
Why launch handleConn as a goroutine?
--------------------------------------------------

Suppose three peers connect.

If we handled them one by one:

    Peer A ---- processing...
    Peer B ---- waiting...
    Peer C ---- waiting...

Bad.

Instead:

    Peer A --> goroutine
    Peer B --> goroutine
    Peer C --> goroutine

Now every peer can communicate independently.

Meanwhile,

startAcceptLoop immediately goes back to waiting
for the next incoming connection.

This allows our node to accept unlimited peers
concurrently.
*/

// 3. handleConn
/*
handleConn is responsible for managing the
entire lifetime of one TCP connection.

Think of it as assigning one employee
to one customer.

From the moment the customer walks in
until they leave,
that employee handles everything.

--------------------------------------------------
Step 1 : Wrap the raw TCP connection
--------------------------------------------------

peer := NewTCPPeer(conn, false)

net.Conn is just a TCP socket.

We wrap it inside our own TCPPeer object
so we can attach GoOrbit-specific behavior
and metadata.

Here:

    outbound = false

because this connection was accepted,
not dialed.

--------------------------------------------------
Step 2 : Perform the Handshake
--------------------------------------------------

Before two GoOrbit nodes trust each other,
they should verify that both sides speak
the same protocol.

This is called the handshake.

If the handshake fails:

    Close the connection
    Reject the peer

No further communication happens.

--------------------------------------------------
Step 3 : Read Messages Forever
--------------------------------------------------

Once the handshake succeeds,
the connection is considered established.

Now we repeatedly do:

    Wait for a message
    Decode it
    Process it
    Wait for the next one

This is called the read loop.

--------------------------------------------------
Why an Infinite Loop?
--------------------------------------------------

A TCP connection stays alive.

A peer may send:

    Block #1

...two seconds later...

    Transaction #54

...ten minutes later...

    Ping

We don't know when messages will arrive.

So we keep reading until the connection
is closed or an error occurs.

This is why networking code almost always
contains a read loop.
*/

