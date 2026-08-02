package autobun

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildBunZip writes a zip containing a bun binary entry with the given data.
func buildBunZip(t *testing.T, data []byte) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("bun-dist/" + GetBunBinaryName())
	if err != nil {
		t.Fatal(err.Error())
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err.Error())
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err.Error())
	}
	zipPath := filepath.Join(t.TempDir(), "bun.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err.Error())
	}
	return zipPath
}

// TestExtractBunFromZip checks a normal-sized entry extracts successfully.
func TestExtractBunFromZip(t *testing.T) {
	data := []byte("#!/bin/sh\necho bun\n")
	zipPath := buildBunZip(t, data)
	destDir := t.TempDir()
	if err := extractBunFromZip(zipPath, destDir); err != nil {
		t.Fatal(err.Error())
	}
	out, err := os.ReadFile(filepath.Join(destDir, GetBunBinaryName()))
	if err != nil {
		t.Fatal(err.Error())
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("extracted data mismatch: %q", out)
	}
}

// TestCopyCappedBoundary checks both sides of the extraction size cap.
func TestCopyCappedBoundary(t *testing.T) {
	const limit = 8

	var under bytes.Buffer
	if err := copyCapped(&under, strings.NewReader("1234567"), limit); err != nil {
		t.Fatal(err.Error())
	}
	if under.String() != "1234567" {
		t.Fatalf("under-cap copy mismatch: %q", under.String())
	}

	var at bytes.Buffer
	if err := copyCapped(&at, strings.NewReader("12345678"), limit); err == nil {
		t.Fatal("expected error for source at the cap")
	}

	var over bytes.Buffer
	if err := copyCapped(&over, strings.NewReader("123456789"), limit); err == nil {
		t.Fatal("expected error for source over the cap")
	}
}
