package main

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpics"
	pathKey := CASPATHTransformFunc(key)
	expectedOriginalKey := "c565996f77ccab3a98f55f6546faa5b311ea674b"
	expectedPathName := "c5659/96f77/ccab3/a98f5/5f654/6faa5/b311e/a674b"
	if pathKey.Pathname != expectedPathName {
		t.Errorf("Have %s want %s", pathKey.Pathname, expectedPathName)
	}
	if pathKey.Filename != expectedOriginalKey {
		t.Errorf("Have %s want %s", pathKey.Filename, expectedOriginalKey)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPATHTransformFunc,
	}
	s := NewStore(opts)
	key := "mygallery"

	data := []byte("Some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("Expected to have key %s",key)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := ioutil.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPATHTransformFunc,
	}
	s := NewStore(opts)
	key := "mygallery"

	data := []byte("Some jpg bytes")
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}
