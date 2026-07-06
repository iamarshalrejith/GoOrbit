package p2p

import "net"

// Peer is a interface that represents the remote node
type Peer interface {
	RemoteAddr() net.Addr
	Close() error
}

// Transport is anything that handles the communication
// between the nodes in the network.
// This can be in the form of TCP,UDP, websockets, ...
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}
