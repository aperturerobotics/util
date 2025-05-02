## Utilities

[![GoDoc Widget]][GoDoc] [![Go Report Card Widget]][Go Report Card]

[GoDoc]: https://godoc.org/github.com/aperturerobotics/util
[GoDoc Widget]: https://godoc.org/github.com/aperturerobotics/util?status.svg
[Go Report Card Widget]: https://goreportcard.com/badge/github.com/aperturerobotics/util
[Go Report Card]: https://goreportcard.com/report/github.com/aperturerobotics/util
[![DeepWiki](https://img.shields.io/badge/DeepWiki-aperturerobotics%2Futil-blue.svg?logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACwAAAAyCAYAAAAnWDnqAAAAAXNSR0IArs4c6QAAA05JREFUaEPtmUtyEzEQhtWTQyQLHNak2AB7ZnyXZMEjXMGeK/AIi+QuHrMnbChYY7MIh8g01fJoopFb0uhhEqqcbWTp06/uv1saEDv4O3n3dV60RfP947Mm9/SQc0ICFQgzfc4CYZoTPAswgSJCCUJUnAAoRHOAUOcATwbmVLWdGoH//PB8mnKqScAhsD0kYP3j/Yt5LPQe2KvcXmGvRHcDnpxfL2zOYJ1mFwrryWTz0advv1Ut4CJgf5uhDuDj5eUcAUoahrdY/56ebRWeraTjMt/00Sh3UDtjgHtQNHwcRGOC98BJEAEymycmYcWwOprTgcB6VZ5JK5TAJ+fXGLBm3FDAmn6oPPjR4rKCAoJCal2eAiQp2x0vxTPB3ALO2CRkwmDy5WohzBDwSEFKRwPbknEggCPB/imwrycgxX2NzoMCHhPkDwqYMr9tRcP5qNrMZHkVnOjRMWwLCcr8ohBVb1OMjxLwGCvjTikrsBOiA6fNyCrm8V1rP93iVPpwaE+gO0SsWmPiXB+jikdf6SizrT5qKasx5j8ABbHpFTx+vFXp9EnYQmLx02h1QTTrl6eDqxLnGjporxl3NL3agEvXdT0WmEost648sQOYAeJS9Q7bfUVoMGnjo4AZdUMQku50McDcMWcBPvr0SzbTAFDfvJqwLzgxwATnCgnp4wDl6Aa+Ax283gghmj+vj7feE2KBBRMW3FzOpLOADl0Isb5587h/U4gGvkt5v60Z1VLG8BhYjbzRwyQZemwAd6cCR5/XFWLYZRIMpX39AR0tjaGGiGzLVyhse5C9RKC6ai42ppWPKiBagOvaYk8lO7DajerabOZP46Lby5wKjw1HCRx7p9sVMOWGzb/vA1hwiWc6jm3MvQDTogQkiqIhJV0nBQBTU+3okKCFDy9WwferkHjtxib7t3xIUQtHxnIwtx4mpg26/HfwVNVDb4oI9RHmx5WGelRVlrtiw43zboCLaxv46AZeB3IlTkwouebTr1y2NjSpHz68WNFjHvupy3q8TFn3Hos2IAk4Ju5dCo8B3wP7VPr/FGaKiG+T+v+TQqIrOqMTL1VdWV1DdmcbO8KXBz6esmYWYKPwDL5b5FA1a0hwapHiom0r/cKaoqr+27/XcrS5UwSMbQAAAABJRU5ErkJggg==)](https://deepwiki.com/aperturerobotics/util)

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
