package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

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
	Key  string
	Size int64
}

// This is used to ask the peers if the file is not found locally
type MessageGetFile struct{
	Key string
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
		log.Println("File server stopped due to error or user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("Decoding Error: ",err)
			}
			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("Handling Message Error: ",err)
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
// Store file to Disk & Sending / Broadcasting to Peers
// ======================================================

func (s *FileServer) Store(key string, r io.Reader) error {
	// 1.Store this file to disk
	// 2. Broadcast this file to all known peers in the network

	// Store the file bytes to the local storage
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)
	size, err := s.store.Write(key, tee)

	if err != nil {
		return err
	}

	// Distributing the Gob Message and File Bytes to the Known Peers

	// Sending the Gob Message
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}

	if err:= s.broadcast(&msg); err!= nil{
		return err
	}

	time.Sleep(time.Second * 3)

	// Sends Raw File Bytes to the peers
	// ToDo : Use a multiwriter
	for _, peer := range s.peers {
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		fmt.Println("Received and written bytes to disk: ", n)
	}

	return nil
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{} // Anything that can accept bytes. so here we are making peers to accept bytes
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (s *FileServer) broadcast(msg *Message) error{
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	// Broadcast

	for _, peer := range s.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// ======================================================
// File Retrieval
// ======================================================

func (s *FileServer) Get(key string) (io.Reader, error) {
	// Check the file on local disk
    if s.store.Has(key) {
        return s.store.Read(key)
    }
	fmt.Printf("Dont have file (%s) locally, Fetching from Network...\n", key)

   // So if the file doesnt exist in local disk, first check if the file exist in the peers
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err!=nil{
		return nil, err
	}

	for _, peer := range s.peers {
		fmt.Println("Receiving stream from peer: ",peer.RemoteAddr())
		fileBuffer := new(bytes.Buffer) 
		n, err := io.CopyN(fileBuffer,peer,22)
		if err !=nil{
			return nil, err
		}
		fmt.Printf("Recieved Bytes over the network: %d", n)
		fmt.Println(fileBuffer.String())
	}

	select{}
  
	return nil, nil
}

// ======================================================
// Message Processing
// ======================================================

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peer list", from)
	}

	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("Written %d bytes to disk\n", n)
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error{
	if !s.store.Has(msg.Key){
		return fmt.Errorf("Need to serve a file (%s) but it does not exist on disk", msg.Key)
	}

	fmt.Printf("Serving File (%s) over the network\n",msg.Key)
	r, err := s.store.Read(msg.Key)
	if err!= nil{
		return err
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("Peer %s not in map", from)
	}

	n, err := io.Copy(peer,r)

	fmt.Printf("Written %d bytes over the network to %s\n",n,from)
	return nil
}

// ======================================================
// Gob Type Registration
// ======================================================

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
