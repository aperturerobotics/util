//go:build js

package stream

import (
	"errors"
	"io"
	"sync"
	"syscall/js"
)

// ReadableStream implements io.ReadCloser for the response body.
type ReadableStream struct {
	stream    js.Value
	reader    js.Value
	closed    bool
	mu        sync.Mutex
	readError error
	buffer    []byte
}

func NewReadableStream(stream js.Value) *ReadableStream {
	return &ReadableStream{
		stream: stream,
		reader: stream.Call("getReader"),
	}
}

func (b *ReadableStream) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, io.EOF
	}

	if len(b.buffer) > 0 {
		n = copy(p, b.buffer)
		b.buffer = b.buffer[n:]
		return n, nil
	}

	resultChan := make(chan struct{}, 2)
	var result js.Value

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result = args[0]
		resultChan <- struct{}{}
		return nil
	})
	defer success.Release()

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		b.readError = errors.New(args[0].Get("message").String())
		resultChan <- struct{}{}
		return nil
	})
	defer failure.Release()

	b.reader.Call("read").Call("then", success).Call("catch", failure)
	<-resultChan

	if b.readError != nil {
		return 0, b.readError
	}

	if result.IsUndefined() || result.IsNull() {
		b.closed = true
		return 0, io.EOF
	}

	done := result.Get("done").Bool()
	if done {
		b.closed = true
		return 0, io.EOF
	}

	value := result.Get("value")
	if value.IsUndefined() || value.IsNull() {
		b.closed = true
		return 0, io.EOF
	}

	valueLength := value.Length()
	if valueLength == 0 {
		return 0, nil
	}

	b.buffer = make([]byte, valueLength)
	js.CopyBytesToGo(b.buffer, value)

	n = copy(p, b.buffer)
	b.buffer = b.buffer[n:]

	if len(b.buffer) == 0 {
		b.buffer = nil
	}

	return n, nil
}

func (b *ReadableStream) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.closed {
		b.closed = true
		b.reader.Call("cancel")
	}

	return nil
}
