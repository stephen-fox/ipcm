# lock

## What is it?
A Go library that provides tooling for orchestrating inter-process
communication (IPC) on different operating systems.

## API

#### `Mutex`
The primary feature of this library is a Mutex for communicating exclusion
across process boundaries. It functions in a similar manner to `sync.Mutex` in
that a call to `Lock()` will block until the mutex is locked. Once locked, the
Mutex owner is responsible for releasing control by calling `Unlock()`.
