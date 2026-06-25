package p2p

// Message holds any arbitrary data that is being sent over the each transport b/w two nodes in network

type Message struct{
	Payload []byte
}