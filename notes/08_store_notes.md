# store.go — File Storage Engine

---

## 1. WHAT IS THIS FILE DOING?

This file handles **storing and retrieving files on disk**.

Think of it as the hard drive manager of GoOrbit.

It answers questions like:
```text
"Save this file with the key 'mypicture.jpg'"
"Give me back the file for key 'mypicture.jpg'"
"Does a file with this key exist?"
"Delete the file for this key"
```

But it does it in a **smart way** — using content-addressable storage.

---

## 2. WHAT IS CONTENT-ADDRESSABLE STORAGE (CAS)?

Normal storage:
```text
File name: "vacation.jpg"
Stored at: /files/vacation.jpg
```

Content-addressable storage:
```text
File content → run through a hash function → get a unique ID
Stored at: /files/a3f9c/1b2d4/...
```

### Why is this better?

```text
Normal:
    Two files named "vacation.jpg" → conflict!
    Rename the file → it's now a different file!

CAS:
    Two identical files → same hash → same location (no duplicate)
    Rename doesn't matter → content determines location
    Like Git — files are stored by their SHA hash
```

### Real World Analogy

```text
Normal filing cabinet:
    Labelled "Invoices 2024" → if label falls off, file is lost

CAS:
    Label IS the content of the document
    Can never be mixed up
    Same document always in same drawer
```

---

## 3. THE HASH FUNCTION — CASPATHTransformFunc

```go
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
```

### Step by Step — Input: `"bestpics"`

```text
Step 1: sha1.Sum([]byte("bestpics"))
    → SHA1 hash (20 raw bytes)

Step 2: hex.EncodeToString(hash[:])
    → "c565996f77ccab3a98f55f6546faa5b311ea674b"
       (40 character hex string)

Step 3: blocksize = 5
        sliceLen = 40 / 5 = 8 blocks

Step 4: Split into blocks of 5 characters
    → ["c5659", "96f77", "ccab3", "a98f5", "5f654", "6faa5", "b311e", "a674b"]

Step 5: Join with "/"
    → "c5659/96f77/ccab3/a98f5/5f654/6faa5/b311e/a674b"
```

### Result

```text
Pathname: "c5659/96f77/ccab3/a98f5/5f654/6faa5/b311e/a674b"
Filename: "c565996f77ccab3a98f55f6546faa5b311ea674b"
```

### Why split into folders?

```text
Without splitting:
    /storage/c565996f77ccab3a98f55f6546faa5b311ea674b
    → flat directory, millions of files, very slow

With splitting:
    /storage/c5659/96f77/ccab3/.../c565996f77...
    → nested directories, faster lookup (like Git does it)
```

---

## 4. WHAT IS sha1.Sum?

SHA1 = **Secure Hash Algorithm 1**

```text
Input:  any string or bytes
Output: always a 20-byte (160-bit) fixed-length hash

"hello"     → da39a3ee5e6b...
"hello!"    → f572d396fae9...
```

Properties:
```text
Same input  → always same output
Tiny change → completely different output
Cannot reverse (one-way)
```

Used here to create a **unique, consistent location** for every key.

---

## 5. WHAT IS hex.EncodeToString?

SHA1 gives 20 raw bytes. Bytes look like: `[197, 101, 153, ...]`

Hex encoding converts each byte to 2 readable characters:

```text
197 → "c5"
101 → "65"
153 → "99"

[197, 101, 153, ...] → "c56599..."
```

Result: a 40-character string we can use as a file path.

---

## 6. PathKey STRUCT

```go
type PathKey struct {
    Pathname string   // folder structure: "c5659/96f77/..."
    Filename string   // full hash:        "c565996f77ccab3a..."
}
```

Two helper methods:

### FirstPathName()

```go
func (p PathKey) FirstPathName() string {
    paths := strings.Split(p.Pathname, "/")
    return paths[0]
}
```

```text
Input:  "c5659/96f77/ccab3/..."
Output: "c5659"
```

Used when **deleting** — removes the entire top-level folder.

### FullPath()

```go
func (p PathKey) FullPath() string {
    return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}
```

```text
Input:  Pathname="c5659/96f77", Filename="c565996f77..."
Output: "c5659/96f77/c565996f77..."
```

The exact path to the file on disk.

---

## 7. PathTransformFunc TYPE

```go
type PathTransformFunc func(string) PathKey
```

A **function type** — just like `HandshakeFunc` in the networking layer.

Lets you plug in any hashing/path logic:

```text
CASPATHTransformFunc     ← SHA1 based (production)
DefaultPathTransformFunc ← key used as-is (simple/testing)
```

---

## 8. StoreOpts and Store

```go
type StoreOpts struct {
    Root              string
    PathTransformFunc PathTransformFunc
}

type Store struct {
    StoreOpts   // embedded
}
```

`Root` = the top-level folder where all files go.

```text
Root = "3000_network"
→ all files stored inside "3000_network/" folder
```

### NewStore — Constructor

```go
func NewStore(opts StoreOpts) *Store {
    if opts.PathTransformFunc == nil {
        opts.PathTransformFunc = DefaultPathTransformFunc
    }
    if len(opts.Root) == 0 {
        opts.Root = defaultRootFolderName   // "storage"
    }
    return &Store{StoreOpts: opts}
}
```

Provides **sensible defaults** — if you forget to set something, it still works.

---

## 9. THE FIVE OPERATIONS

### Has(key) — Does this file exist?

```go
func (s *Store) Has(key string) bool {
    pathKey := s.PathTransformFunc(key)
    fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
    _, err := os.Stat(fullPathWithRoot)
    return !errors.Is(err, os.ErrNotExist)
}
```

`os.Stat(path)` → checks if a file exists at that path.

```text
File exists    → err is nil   → return true
File not found → err is ErrNotExist → return false
```

---

### Delete(key) — Remove a file

```go
func (s *Store) Delete(key string) error {
    pathKey := s.PathTransformFunc(key)
    defer func() {
        log.Printf("Deleted [%s] from Disk", pathKey.Filename)
    }()
    firstPathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathName())
    return os.RemoveAll(firstPathNameWithRoot)
}
```

Deletes the **entire top-level folder** (`c5659/`), not just the file.

Why? Avoids leaving empty folders behind.

```text
Before: storage/c5659/96f77/ccab3/.../c565996f77...
After:  (c5659 folder and everything inside it is gone)
```

`os.RemoveAll` = delete a folder and all its contents (like `rm -rf`).

---

### Write(key, r) — Save a file

```go
func (s *Store) Write(key string, r io.Reader) error {
    return s.writeStream(key, r)
}
```

Public wrapper for `writeStream`.

### writeStream — The actual writing

```go
func (s *Store) writeStream(key string, r io.Reader) error {
    pathKey := s.PathTransformFunc(key)
    pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Pathname)

    if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
        return err
    }

    fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())

    f, err := os.Create(fullPathWithRoot)
    if err != nil {
        return err
    }
    defer f.Close()

    n, err := io.Copy(f, r)
    if err != nil {
        return err
    }

    log.Printf("written (%d) bytes to disk: %s", n, fullPathWithRoot)
    return nil
}
```

Step by step:

```text
1. Transform key → PathKey (get folder + filename)

2. os.MkdirAll(path)
       → Create all nested folders if they don't exist
       → Like "mkdir -p" in terminal
       → storage/c5659/96f77/ccab3/... (all at once)

3. os.Create(fullPath)
       → Create the actual file

4. io.Copy(file, reader)
       → Read bytes from 'r' (the data source)
       → Write them to 'f' (the file)
       → Returns number of bytes written

5. Log how many bytes were written
```

### What is io.Reader here?

The file content is passed as `io.Reader` — a stream of bytes.

```text
Could be:
    bytes.NewReader([]byte("some data"))   ← in-memory bytes
    os.Open("somefile.jpg")               ← a file on disk
    a TCP connection                       ← data from network

The Store doesn't care which one. It just reads from it.
```

---

### Read(key) — Get a file back

```go
func (s *Store) Read(key string) (io.Reader, error) {
    f, err := s.readStream(key)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    buf := new(bytes.Buffer)
    _, err = io.Copy(buf, f)
    return buf, err
}
```

Opens the file, reads all bytes into a `bytes.Buffer`, closes the file.

Returns the buffer as an `io.Reader`.

### readStream — Opens the file

```go
func (s *Store) readStream(key string) (io.ReadCloser, error) {
    pathKey := s.PathTransformFunc(key)
    fullPathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FullPath())
    return os.Open(fullPathWithRoot)
}
```

`os.Open` → opens file for reading.

Returns `io.ReadCloser` = something you can read AND close.

---

### Clear() — Wipe everything

```go
func (s *Store) Clear() error {
    return os.RemoveAll(s.Root)
}
```

Deletes the entire root folder.

Used in tests to clean up after testing.

---

## 10. THE FULL DISK STRUCTURE

When you store key `"bestpics"`:

```text
3000_network/                      ← Root folder
└── c5659/                         ← FirstPathName (top-level hash block)
    └── 96f77/
        └── ccab3/
            └── a98f5/
                └── 5f654/
                    └── 6faa5/
                        └── b311e/
                            └── a674b/
                                └── c565996f77ccab3a98f55f6546faa5b311ea674b
                                                        ↑
                                                     actual file
```

---

## 11. KEY Go CONCEPTS IN THIS FILE

| Concept | Example | What it does |
|---------|---------|--------------|
| `sha1.Sum` | `sha1.Sum([]byte(key))` | Hash function — unique fingerprint |
| `hex.EncodeToString` | converts bytes to hex string | Make hash readable |
| `os.MkdirAll` | create nested folders | Like `mkdir -p` |
| `os.Create` | create a file | Open for writing |
| `os.Open` | open existing file | Open for reading |
| `os.RemoveAll` | delete folder+contents | Like `rm -rf` |
| `os.Stat` | check if file exists | File system probe |
| `io.Copy` | copy data stream | Efficient data transfer |
| `bytes.Buffer` | in-memory byte buffer | Read file into memory |
| `fmt.Sprintf` | `"%s/%s", a, b` | Build file paths as strings |

---

# End Notes
