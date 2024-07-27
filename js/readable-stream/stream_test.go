//go:build js && webtests

package stream

import (
	"io"
	"strings"
	"syscall/js"
	"testing"
)

func TestReadableStream(t *testing.T) {
	t.Run("Read entire stream", func(t *testing.T) {
		mockStream := newMockStream("Hello, World!")
		reader := NewReadableStream(mockStream)

		result, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to read from BodyReader: %v", err)
		}

		if string(result) != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", string(result))
		}
	})

	t.Run("Read in chunks", func(t *testing.T) {
		mockStream := newMockStream("Hello, World!")
		reader := NewReadableStream(mockStream)

		result := make([]byte, 0, 13)
		buf := make([]byte, 5)
		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("Failed to read chunk: %v", err)
			}
			result = append(result, buf[:n]...)
		}

		if string(result) != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got '%s'", string(result))
		}
	})

	t.Run("Close stream", func(t *testing.T) {
		mockStream := newMockStream("Hello, World!")
		reader := NewReadableStream(mockStream)

		err := reader.Close()
		if err != nil {
			t.Fatalf("Failed to close BodyReader: %v", err)
		}

		_, err = reader.Read(make([]byte, 1))
		if err != io.EOF {
			t.Errorf("Expected EOF after closing, got: %v", err)
		}
	})
}

func newMockStream(content string) js.Value {
	return js.Global().Get("ReadableStream").New(map[string]interface{}{
		"start": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			controller := args[0]
			go func() {
				for _, chunk := range strings.Split(content, "") {
					controller.Call("enqueue", js.Global().Get("TextEncoder").New().Call("encode", chunk))
				}
				controller.Call("close")
			}()
			return nil
		}),
	})
}
