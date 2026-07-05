# store_test.go — Testing the Storage Engine

---

## 1. WHAT IS THIS FILE DOING?

This file **tests** that `store.go` works correctly.

It checks:
- Does `CASPATHTransformFunc` produce the right hash and path?
- Can we write, read, and delete files correctly?
- Does `Has()` return the right answer?

---

## 2. HOW TESTING WORKS IN GO

```go
import "testing"

func TestSomething(t *testing.T) {
    // test code here
}
```

Rules:
```text
File must end in _test.go
Function must start with Test
Must take t *testing.T as argument

Run with:  go test ./...
           make test
```

`t *testing.T` = the test runner object. You use it to report failures.

---

## 3. TestPathTransformFunc

```go
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
```

### What is being tested?

That for key `"bestpics"`, the SHA1 hash function produces EXACTLY the expected hash and path.

SHA1 is deterministic:
```text
"bestpics" → always → "c565996f77ccab3a98f55f6546faa5b311ea674b"

If the function is broken, we'd get a different hash.
This test catches that.
```

### t.Errorf

```go
t.Errorf("Have %s want %s", pathKey.Pathname, expectedPathName)
```

```text
Marks the test as FAILED
Prints a readable message: "Have X want Y"
Does NOT stop the test (continues running)

Compare with t.Fatalf → marks failed AND stops immediately
```

---

## 4. TestStore — The Big Test

```go
func TestStore(t *testing.T) {
    s := newStore()
    defer teardown(t, s)

    for i := 0; i < 50; i++ {
        key := fmt.Sprintf("mygallery_%d", i)
        data := []byte("Some jpg bytes")

        // 1. Write
        if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
            t.Error(err)
        }

        // 2. Check it exists
        if ok := s.Has(key); !ok {
            t.Errorf("Expected to have key %s", key)
        }

        // 3. Read it back
        r, err := s.Read(key)
        if err != nil {
            t.Error(err)
        }
        b, _ := ioutil.ReadAll(r)
        if string(b) != string(data) {
            t.Errorf("want %s have %s", data, b)
        }

        // 4. Delete it
        if err := s.Delete(key); err != nil {
            t.Error(err)
        }

        // 5. Confirm it's gone
        if ok := s.Has(key); ok {
            t.Errorf("Expected to NOT have key %s", key)
        }
    }
}
```

### What is being tested?

50 times in a loop, for each file:

```text
1. Write the file to disk
2. Check it exists (Has = true)
3. Read it back → confirm the content matches
4. Delete the file
5. Confirm it's gone (Has = false)
```

This is a **full lifecycle test** — write → verify → read → delete → verify.

### Why 50 iterations?

To stress-test. One file working is luck. 50 files working is proof.

### bytes.NewReader(data)

```go
bytes.NewReader([]byte("Some jpg bytes"))
```

Creates an `io.Reader` from raw bytes.

```text
writeStream expects io.Reader
We have []byte
bytes.NewReader wraps []byte into io.Reader
```

### ioutil.ReadAll(r)

```go
b, _ := ioutil.ReadAll(r)
```

Reads ALL bytes from a reader into a `[]byte`.

```text
r (io.Reader) → read everything → b ([]byte)
```

`_` = ignore the error (only in tests where we trust the previous steps worked).

### string(b) comparison

```go
if string(b) != string(data) {
```

`string([]byte)` converts bytes to a string for easy comparison.

```text
data = []byte("Some jpg bytes")
b    = bytes read from disk

If they match → file was stored and retrieved correctly
```

---

## 5. HELPER FUNCTIONS

### newStore()

```go
func newStore() *Store {
    opts := StoreOpts{
        PathTransformFunc: CASPATHTransformFunc,
    }
    return NewStore(opts)
}
```

Creates a fresh Store for testing.

Note: no `Root` is set → `NewStore` will use the default `"storage"` folder.

### teardown()

```go
func teardown(t *testing.T, s *Store) {
    if err := s.Clear(); err != nil {
        t.Error(err)
    }
}
```

Cleans up after the test — deletes the entire storage folder.

Called with `defer`:

```go
defer teardown(t, s)
```

```text
defer = runs at the END of TestStore, no matter what
Even if the test fails partway through
The storage folder always gets cleaned up
```

Without teardown:
```text
After every test run, leftover files pile up on your disk.
Tests from a previous run could affect the next run.
```

---

## 6. TEST FLOW DIAGRAM

```text
TestStore starts
│
├── newStore() → creates Store with CAS path transform
│
├── defer teardown() → registered (runs at end)
│
└── for i = 0 to 49:
        │
        ├── writeStream("mygallery_0", bytes)
        │       → file created on disk
        │
        ├── Has("mygallery_0")
        │       → true ✓
        │
        ├── Read("mygallery_0")
        │       → bytes read from disk
        │       → compare with original ✓
        │
        ├── Delete("mygallery_0")
        │       → folder removed from disk
        │
        └── Has("mygallery_0")
                → false ✓
        
        ... repeat for i = 1 to 49

teardown() runs → s.Clear() → entire storage folder deleted
```

---

## 7. KEY GO TESTING CONCEPTS

| Concept | Example | What it does |
|---------|---------|--------------|
| Test function | `func TestXxx(t *testing.T)` | Marks a function as a test |
| `t.Error(err)` | `t.Error(err)` | Fail test, print error, continue |
| `t.Errorf(msg)` | `t.Errorf("Have %s want %s", ...)` | Fail test with formatted message |
| `defer teardown()` | `defer teardown(t, s)` | Always clean up after test |
| `bytes.NewReader` | `bytes.NewReader([]byte("data"))` | Wrap bytes as io.Reader |
| `ioutil.ReadAll` | `ioutil.ReadAll(r)` | Read all bytes from a reader |
| `string(b)` | `string([]byte{...})` | Convert bytes to string |

---

# End Notes
