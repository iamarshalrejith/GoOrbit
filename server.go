package main

import (
	"fmt"
	"log"

	"github.com/iamarshalrejith/GoOrbit/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts
	store  *Store
	quitch chan struct{} // Just used for signaling
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}), // An empty struct also uses zero bytes of storage, making it a common choice for signaling in Go.
	}
}

func (s *FileServer) Stop() {
	close(s.quitch) // When a channel is closed, every goroutine waiting on it immediately wakes up.
}

func (s *FileServer) loop() { // blocks main
	defer func(){
		log.Println("File server stopped due to user quit action")
		s.Transport.Close()
	}()

	for{
		select{
		case msg := <- s.Transport.Consume():
			fmt.Println(msg)
		case <- s.quitch:
			return
		}
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.loop()
	return nil
}
