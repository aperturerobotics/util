## Utilities

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

[GoDoc]: https://godoc.org/github.com/aperturerobotics/util
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/util?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/util
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/util

Various utilities for Go and TypeScript including:

 - [backoff]: configurable backoff
 - [broadcast]: channel-based broadcast (similar to sync.Cond)
 - [ccall]: call a set of functions concurrently and wait for error or exit
 - [ccontainer]: concurrent container for objects
 - [commonprefix]: find common prefix between strings
 - [conc]: concurrent processing queue
 - [cqueue]: concurrent atomic queues (LIFO)
 - [csync]: sync primitives supporting context arguments
 - [debounce-fswatcher]: debounce fs watcher events
 - [enabled]: three-way boolean proto enum
 - [exec]: wrapper around Go os exec
 - [fsutil]: utilities for os filesystem
 - [gitroot]: git repository root finder
 - [httplog/fetch]: JS Fetch API wrapper with logging for WASM
 - [httplog]: HTTP request and response logging utilities
 - [iocloser]: wrap reader/writer with a close function
 - [iowriter]: io.Writer implementation with callback function
 - [iosizer]: read/writer with metrics for size
 - [js/fetch]: Fetch API wrapper for WASM
 - [js/readable-stream]: ReadableStream wrapper for WASM
 - [keyed]: key/value based routine management
 - [linkedlist]: linked list with head/tail
 - [memo]: memoize a function: call it once and remember results
 - [padding]: pad / unpad a byte array slice
 - [prng]: psuedorandom generator with seed
 - [promise]: promise mechanics for Go (like JS)
 - [refcount]: reference counter ccontainer
 - [routine]: start, stop, restart, reset a goroutine
 - [scrub]: zero a buffer after usage
 - [unique]: deduplicated list of items by key

[backoff]: ./backoff
[broadcast]: ./broadcast
[ccall]: ./ccall
[ccontainer]: ./ccontainer
[commonprefix]: ./commonprefix
[conc]: ./conc
[cqueue]: ./cqueue
[csync]: ./csync
[debounce-fswatcher]: ./debounce-fswatcher
[exec]: ./exec
[fsutil]: ./fsutil
[httplog/fetch]: ./httplog/fetch
[httplog]: ./httplog
[iocloser]: ./iocloser
[iowriter]: ./iowriter
[iosizer]: ./iosizer
[js/fetch]: ./js/fetch
[js/readable-stream]: ./js/readable-stream
[keyed]: ./keyed
[linkedlist]: ./linkedlist
[memo]: ./memo
[padding]: ./padding
[prng]: ./prng
[promise]: ./promise
[refcount]: ./refcount
[routine]: ./routine
[scrub]: ./scrub
[unique]: ./unique
[vmime]: ./vmime
[vmime]: ./vmime

## License

MIT
