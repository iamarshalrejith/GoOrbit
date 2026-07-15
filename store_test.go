package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "bestpics"
	pathKey := CASPATHTransformFunc(key)
	expectedFilename := "c565996f77ccab3a98f55f6546faa5b311ea674b"
	expectedPathName := "c5659/96f77/ccab3/a98f5/5f654/6faa5/b311e/a674b"
	if pathKey.Pathname != expectedPathName {
		t.Errorf("Have %s want %s", pathKey.Pathname, expectedPathName)
	}
	if pathKey.Filename != expectedFilename {
		t.Errorf("Have %s want %s", pathKey.Filename, expectedFilename)
	}
}

func TestStore(t *testing.T) {
	s := newStore()
	defer teardown(t,s)

	for i:=0;i<50;i++{
	key := fmt.Sprintf("mygallery_%d",i)

	data := []byte("Some jpg bytes")
	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
    t.Error(err)
}

	if ok := s.Has(key); !ok {
		t.Errorf("Expected to have key %s",key)
	}

	_, r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error(err)
	}

	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}

	if err:= s.Delete(key);err!= nil{
		t.Error(err)
	}

	if ok := s.Has(key);ok{
		t.Errorf("Expected to Not have a key %s",key)
	}
	}
}



// -------------Helper Functions-------------

func newStore() *Store{
	opts := StoreOpts{
		PathTransformFunc : CASPATHTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store){
	if err:= s.Clear(); err != nil{
		t.Error(err)
	}
}
