package p2p

// RPC holds any arbitrary data that is being sent over the
//  each transport b/w two nodes in network
type RPC struct {
	From    string
	Payload []byte
}
