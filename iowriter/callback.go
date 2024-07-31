package iowriter

import (
	"errors"
	"io"
)

// CallbackWriter is an io.Writer which calls a callback.
type CallbackWriter struct {
	cb func(p []byte) (n int, err error)
}

// NewCallbackWriter creates a new CallbackWriter with the given callback function.
func NewCallbackWriter(cb func(p []byte) (n int, err error)) *CallbackWriter {
	return &CallbackWriter{cb: cb}
}

// Write calls the callback function with the given byte slice.
// It returns an error if the callback is not defined.
func (w *CallbackWriter) Write(p []byte) (n int, err error) {
	if w.cb == nil {
		return 0, errors.New("writer cb is not defined")
	}
	return w.cb(p)
}

// _ is a type assertion
var _ io.Writer = ((*CallbackWriter)(nil))
