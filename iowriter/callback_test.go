package iowriter

import (
	"bytes"
	"errors"
	"testing"
)

func TestCallbackWriter(t *testing.T) {
	t.Run("Write with valid callback", func(t *testing.T) {
		var buf bytes.Buffer
		cw := NewCallbackWriter(func(p []byte) (int, error) {
			return buf.Write(p)
		})

		testData := []byte("Hello, World!")
		n, err := cw.Write(testData)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if n != len(testData) {
			t.Errorf("Expected %d bytes written, got %d", len(testData), n)
		}
		if buf.String() != string(testData) {
			t.Errorf("Expected %q, got %q", string(testData), buf.String())
		}
	})

	t.Run("Write with nil callback", func(t *testing.T) {
		cw := &CallbackWriter{cb: nil}

		_, err := cw.Write([]byte("Test"))

		if err == nil {
			t.Error("Expected an error, got nil")
		}
		if err.Error() != "writer cb is not defined" {
			t.Errorf("Expected error message 'writer cb is not defined', got %q", err.Error())
		}
	})

	t.Run("Write with error-returning callback", func(t *testing.T) {
		expectedError := errors.New("test error")
		cw := NewCallbackWriter(func(p []byte) (int, error) {
			return 0, expectedError
		})

		_, err := cw.Write([]byte("Test"))

		if err != expectedError {
			t.Errorf("Expected error %v, got %v", expectedError, err)
		}
	})
}
