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
 - [cqueue]: concurrent atomic queues (LIFO)
 - [conc]: concurrent processing queue
 - [csync]: sync primitives supporting context arguments
 - [debounce-fswatcher]: debounce fs watcher events
 - [exec]: wrapper around Go os exec
 - [fsutil]: utilities for os filesystem
 - [iocloser]: wrap reader/writer with a close function
 - [iosizer]: read/writer with metrics for size
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
[csync]: ./csync
[cqueue]: ./cqueue
[debounce-fswatcher]: ./debounce-fswatcher
[exec]: ./exec
[fsutil]: ./fsutil
[iocloser]: ./iocloser
[iosizer]: ./iosizer
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

## License

MIT
