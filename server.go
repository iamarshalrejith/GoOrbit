package main

import (
	"bytes"
	"encoding/gob"
	"time"
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

// This will be travelling over the transport and payload can be anything
type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key string
	Size int64
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

// It should handle the file bytes.
func (s *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println(err)
				return
			}
			if err := s.handleMessage(rpc.From,&msg); err!=nil{
				log.Println(err)
				return
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

	// Store
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	size, err := s.store.Write(key, tee)
	if err!= nil{
		return err
	}
	
	msg := Message{
		Payload: MessageStoreFile{
			Key:key,
			Size: size,
		},
	}

	msgBuf := new(bytes.Buffer)
	if err:= gob.NewEncoder(msgBuf).Encode(msg); err!= nil{
		return err
	}

	// Broadcast

	for _, peer := range s.peers{
		if err := peer.Send(msgBuf.Bytes());err!=nil{
			return err
		}
	}

	time.Sleep(time.Second * 3)

	for _, peer := range s.peers{
		n, err := io.Copy(peer,buf)
		if err!= nil {
			return err
		}
		fmt.Println("Received and written bytes to disk: ",n)
	}

	return nil
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
// Message Processing
// ======================================================

func (s *FileServer) handleMessage(from string,msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from,v)
	}
	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error{
	peer, ok := s.peers[from]
	if !ok{
		return fmt.Errorf("peer (%s) could not be found in the peer list",from)
	}

	if _,err := s.store.Write(msg.Key,io.LimitReader(peer,int64(msg.Size))); err != nil {
		return err
	}
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

// ======================================================
// Gob Type Registration
// ======================================================

func init(){
	gob.Register(MessageStoreFile{})
}