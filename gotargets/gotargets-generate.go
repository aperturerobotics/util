//go:build generate
// +build generate

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type GoDistEntry struct {
	GOOS   string `json:"GOOS"`
	GOARCH string `json:"GOARCH"`
}

func main() {
	output, err := exec.Command("go", "tool", "dist", "list", "-json").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running 'go tool dist list -json': %v\n", err)
		os.Exit(1)
	}

	var entries []GoDistEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling JSON output: %v\n", err)
		os.Exit(1)
	}

	var out bytes.Buffer
	out.WriteString("package gotargets\n\n")
	out.WriteString("type GoDistEntry struct {\n\tGOOS   string `json:\"GOOS\"`\n\tGOARCH string `json:\"GOARCH\"`\n}\n\n")
	out.WriteString("var KnownGoDists = []*GoDistEntry{\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "\t{\n")
		fmt.Fprintf(&out, "\t\tGOOS: %q,\n", entry.GOOS)
		fmt.Fprintf(&out, "\t\tGOARCH: %q,\n", entry.GOARCH)
		fmt.Fprintf(&out, "\t},\n")
	}
	out.WriteString("}\n")

	err = os.WriteFile("gotargets.gen.go", out.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing gotargets.gen.go: %v\n", err)
		os.Exit(1)
	}
}
