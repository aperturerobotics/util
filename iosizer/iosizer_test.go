package iosizer

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestSizeReadWriter(t *testing.T) {
	// Test data
	testData := "hello world"
	reader := strings.NewReader(testData)
	writer := &bytes.Buffer{}

	// Create SizeReadWriter
	srw := NewSizeReadWriter(reader, writer)

	// Test initial size
	if size := srw.TotalSize(); size != 0 {
		t.Fatalf("expected initial size 0, got %d", size)
	}

	// Test reading
	buf := make([]byte, 5)
	n, err := srw.Read(buf)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected to read 5 bytes, got %d", n)
	}
	if string(buf) != "hello" {
		t.Fatalf("expected 'hello', got '%s'", string(buf))
	}
	if size := srw.TotalSize(); size != 5 {
		t.Fatalf("expected size 5 after read, got %d", size)
	}

	// Test writing
	writeData := []byte("test")
	n, err = srw.Write(writeData)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected to write 4 bytes, got %d", n)
	}
	if writer.String() != "test" {
		t.Fatalf("expected writer to contain 'test', got '%s'", writer.String())
	}
	if size := srw.TotalSize(); size != 9 {
		t.Fatalf("expected total size 9 after read+write, got %d", size)
	}

	// Test nil reader/writer
	nilSrw := NewSizeReadWriter(nil, nil)

	_, err = nilSrw.Read(buf)
	if err != io.EOF {
		t.Fatalf("expected EOF for nil reader, got %v", err)
	}

	_, err = nilSrw.Write(writeData)
	if err != io.EOF {
		t.Fatalf("expected EOF for nil writer, got %v", err)
	}
}

func TestLargeDataTransfer(t *testing.T) {
	// Create large test data
	largeData := make([]byte, 1<<20) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	reader := bytes.NewReader(largeData)
	writer := &bytes.Buffer{}
	srw := NewSizeReadWriter(reader, writer)

	// Test reading in chunks
	buf := make([]byte, 64*1024) // 64KB chunks
	totalRead := 0
	for {
		n, err := srw.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		totalRead += n
	}

	if totalRead != len(largeData) {
		t.Fatalf("expected to read %d bytes, got %d", len(largeData), totalRead)
	}

	if size := srw.TotalSize(); size != uint64(len(largeData)) {
		t.Fatalf("expected size %d after large read, got %d", len(largeData), size)
	}

	// Test writing large data
	writer.Reset()
	totalWritten := 0
	reader = bytes.NewReader(largeData)
	for {
		n, err := io.CopyN(srw, reader, 64*1024) // Write in 64KB chunks
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected write error: %v", err)
		}
		totalWritten += int(n)
	}

	if totalWritten != len(largeData) {
		t.Fatalf("expected to write %d bytes, got %d", len(largeData), totalWritten)
	}

	expectedSize := uint64(len(largeData)) * 2 // Both read and write
	if size := srw.TotalSize(); size != expectedSize {
		t.Fatalf("expected total size %d after large read+write, got %d", expectedSize, size)
	}

	// Verify written data matches
	if !bytes.Equal(writer.Bytes(), largeData) {
		t.Fatal("written data does not match original data")
	}
}
