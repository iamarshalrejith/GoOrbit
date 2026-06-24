package p2p

// Peer is a interface that represents the remote node
type Peer interface{

}

// Transport is anything that handles the communication
// between the nodes in the network.
// This can be in the form of TCP,UDP, websockets, ...
type Transport interface{
	ListenAndAccept() error
}