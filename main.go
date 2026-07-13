package main

import (
	"bytes"
	"fmt"
	// "io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/iamarshalrejith/GoOrbit/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       strings.ReplaceAll(listenAddr, ":", "") + "_network",
		PathTransformFunc: CASPATHTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")
	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(3 * time.Second)
	go s2.Start()

	time.Sleep(3 * time.Second)

	for i := 0;i<10;i++{
		data := bytes.NewReader([]byte("My big data file here!"))
		s2.Store(fmt.Sprintf("myprivatedata_%d",i), data)
		time.Sleep(5*time.Millisecond)
	}
	

	// r, err := s2.Get("myprivatedata")
	// if err != nil{
	// 	log.Fatal(err)
	// }

	// b, err := ioutil.ReadAll(r)
	// if err != nil{
	// 	log.Fatal(err)
	// }

	// fmt.Println(string(b))

	select {}
}
