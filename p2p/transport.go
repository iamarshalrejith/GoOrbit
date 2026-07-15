package p2p

import "net"

// Peer is a interface that represents the remote node
type Peer interface {
	Send([]byte) error
	net.Conn
	CloseStream()
}

// Transport is anything that handles the communication
// between the nodes in the network.
// This can be in the form of TCP,UDP, websockets, ...
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
