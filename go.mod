module github.com/aperturerobotics/util

go 1.22.0

toolchain go1.23.4

require (
	github.com/aperturerobotics/common v0.20.2 // latest
	github.com/aperturerobotics/json-iterator-lite v1.0.1-0.20240713111131-be6bf89c3008 // indirect
	github.com/aperturerobotics/protobuf-go-lite v0.8.0 // latest
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/fsnotify/fsnotify v1.8.0
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.9.3
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67
)

require golang.org/x/sys v0.13.0 // indirect
