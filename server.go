package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/iamarshalrejith/GoOrbit/p2p"
)

type FileServerOpts struct {
	EncKey 			  []byte
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
type MessageGetFile struct {
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
				log.Println("Decoding Error: ", err)
			}
			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Println("Handling Message Error: ", err)
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
			fmt.Printf("[%s] attempting to connect with remote %s \n ",s.Transport.Addr() ,addr)
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
			Size: size + 16,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 2)

	// Sends Raw File Bytes to the peers
	// Using a multiwriter
	peers := []io.Writer{}
	for _, peer := range s.peers{
		peers = append(peers,peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EncKey, fileBuffer, mw)
		if err != nil {
			return err
		}
	fmt.Printf("[%s] received and written (%d) bytes to disk\n", s.Transport.Addr(),n)
	return nil
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	// Broadcast

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
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
		fmt.Printf("[%s] serving file (%s) from local disk.\n", s.Transport.Addr(), key)
		_, r,err := s.store.Read(key) 
		return r, err
	}
	fmt.Printf("[%s] dont have file (%s) locally, Fetching from Network...\n", s.Transport.Addr(), key)

	// So if the file doesnt exist in local disk, first check if the file exist in the peers
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)
	for _, peer := range s.peers {
		// First read the file size so we can limit the amount of bytes that we read from the connection, so it will not keep hanging
		var fileSize int64
		binary.Read(peer, binary.LittleEndian,&fileSize)

		n, err := s.store.writeDecrypt(s.EncKey, key, io.LimitReader(peer,fileSize))
		if err != nil {
			return nil, err
		}
		
		fmt.Printf("[%s] recieved (%d) bytes over the network from (%s)\n", s.Transport.Addr(), n, peer.RemoteAddr())
		peer.CloseStream()
	}
	_, r, err := s.store.Read(key)
	return r, err
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

	fmt.Printf("[%s] written %d bytes to disk\n", s.Transport.Addr(), n)

	peer.CloseStream()
	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("[%s] need to serve a file (%s) but it does not exist on disk", s.Transport.Addr(), msg.Key)
	}

	fmt.Printf("[%s] serving File (%s) over the network\n", s.Transport.Addr(), msg.Key)

	fileSize, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	if rc, ok := r.(io.ReadCloser); ok{
		fmt.Println("Closing ReadCloser")
		defer rc.Close()
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("Peer %s not in map", from)
	}

	// First send the "IncomingStream" byte to the peer and then we can send the file size as an int64.
	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes over the network to %s\n", s.Transport.Addr(), n, from)
	return nil
}

// ======================================================
// Gob Type Registration
// ======================================================

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
