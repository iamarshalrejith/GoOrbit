package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/iamarshalrejith/GoOrbit/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{} // Just used for signaling
}

// ======================================================
// Message Types
// ======================================================

// This will be travelling over the transport and payload can be anything - so today we have DataMessage for the payload.Tommorow it may change - we can have any kind of paylaods like pingmessage. So we are having "any" in Payload
type Message struct {
	From    string
	Payload any
}

type DataMessage struct {
	Key  string
	Data []byte
}

// ======================================================
// Constructor
// ======================================================

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}), // An empty struct also uses zero bytes of storage, making it a common choice for signaling in Go.
		peers:          make(map[string]p2p.Peer),
	}
}

// ======================================================
// Server Lifecycle
// ======================================================

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	if len(s.BootstrapNodes) != 0 {
		s.bootstrapNetwork()
	}
	s.loop()
	return nil
}

func (s *FileServer) Stop() {
	close(s.quitch) // When a channel is closed, every goroutine waiting on it immediately wakes up.
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
				log.Println(err)
			}

			if err := s.handleMessage(&m); err != nil {
				log.Println(err)
			}
		case <-s.quitch:
			return
		}
	}
}

// ======================================================
// Networking with other nodes
// ======================================================

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Println("Attempting to connect with remote: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("Dial Error: ", err)
			}
		}(addr)
	}
	return nil
}

// ======================================================
// Peer Management
// ======================================================

// Adds the peer to the map for the peers reference for node
func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("Connected with remote %s", p.RemoteAddr())

	return nil
}

// ======================================================
// Sending / Broadcasting
// ======================================================

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// 1.Store this file to disk
	// 2. Broadcast this file to all known peers in the network
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := s.store.Write(key, tee); err != nil {
		return err
	}

	// After reading, the reader is empty, so we will create a buffer and store data in it

	p := &DataMessage{
		Key:  key,
		Data: buf.Bytes(),
	}
	fmt.Println(buf.Bytes())
	return s.broadcast(&Message{
		From:    "todo",
		Payload: p,
	})
}

func (s *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{} // Anything that can accept bytes. so here we are making peers to accept bytes
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

// ======================================================
// Receiving
// ======================================================

func (s *FileServer) handleMessage(msg *Message) error {
	switch v := msg.Payload.(type) {
	case *DataMessage:
		fmt.Printf("Recieved Data %+v\n", v)
	}
	return nil
}
