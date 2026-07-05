package main

import (
	"log"
	"time"

	"github.com/iamarshalrejith/GoOrbit/p2p"
)

func main() {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder: p2p.DefaultDecoder{},
		// TODO: OnPeer func
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPATHTransformFunc,
		Transport:    tcpTransport,
	}

	s := NewFileServer(fileServerOpts)

	go func(){
		time.Sleep(time.Second * 3) // Wait 3 seconds
		s.Stop()
	}()

	if err := s.Start(); err!=nil{
		log.Fatal(err)
	}
}