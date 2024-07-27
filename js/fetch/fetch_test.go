//go:build js && webtests

package fetch

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func TestFetchHttpBin(t *testing.T) {
	url := "https://httpbin.org/get"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := &Opts{
		Method: "GET",
		Signal: ctx,
	}

	resp, err := Fetch(url, opts)
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	url, ok := result["url"].(string)
	if !ok {
		t.Fatalf("Response does not contain a 'url' field of type string")
	}

	if url != "https://httpbin.org/get" {
		t.Errorf("Expected URL to be 'https://httpbin.org/get', got '%s'", url)
	}

	t.Logf("Received response from URL: %s", url)
}
