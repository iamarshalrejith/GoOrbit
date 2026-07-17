package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// ======================================================
// Constants
// ======================================================

const defaultRootFolderName = "storage"

// ======================================================
// Path Types
// ======================================================

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename) // instead of printing to the console, it returns the formatted string.
}

type PathTransformFunc func(string) PathKey

// ======================================================
// Path Transform Functions
// ======================================================

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

func CASPATHTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashStr) / blocksize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

// ======================================================
// Store Configuration
// ======================================================

type StoreOpts struct {
	// Root is the folder name of the root, containing all the folders/files of the system.
	Root              string
	PathTransformFunc PathTransformFunc
}

// ======================================================
// Store
// ======================================================

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

// ======================================================
// Store Operations
// ======================================================

func (s *Store) Has(id string,key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root,id,pathKey.FullPath())

	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}


func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	size, r, err := s.readStream(id, key)
	if err != nil {
		return 0, nil, err
	}
	defer r.Close() // close the OS file handle immediately instead of leaking it to the caller

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return 0, nil, err
	}

	return size, buf, nil
}

func (s *Store) Write(id string,key string, r io.Reader) (int64,error) {
	return s.writeStream(id,key,r)
}

func (s *Store) Delete(id string,key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {
		log.Printf("Deleted [%s] from Disk", pathKey.Filename)
	}()

	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

// ======================================================
// Internal Helpers
// ======================================================

func (s *Store) readStream(id string,key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id ,pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err!=nil{
		return 0,nil,err
	}

	fi, err := file.Stat()
	if err!=nil{
		return 0,nil,err
	}

	return fi.Size(), file, nil
}

func (s *Store) writeDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error){
	f, err := s.openFileForWriting(id,key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	
	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) openFileForWriting(id string, key string) (*os.File,error){
	// hashed key for path
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root,id,pathKey.Pathname)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPath())

	return os.Create(fullPathWithRoot)
}

func (s *Store) writeStream(id string,key string, r io.Reader) (int64,error) {
	f, err := s.openFileForWriting(id,key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

