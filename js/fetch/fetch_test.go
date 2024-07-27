//go:build js && webtests

package fetch

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func TestFetchHttpBin(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
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
	})

	t.Run("POST request with body", func(t *testing.T) {
		url := "https://httpbin.org/post"
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		postBody := strings.NewReader(`{"key": "value"}`)
		opts := &Opts{
			Method: "POST",
			Signal: ctx,
			Body:   postBody,
			Header: Header{
				"Content-Type": []string{"application/json"},
			},
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

		jsonData, ok := result["json"].(map[string]interface{})
		if !ok {
			t.Fatalf("Response does not contain a 'json' field of type map[string]interface{}")
		}

		value, ok := jsonData["key"].(string)
		if !ok || value != "value" {
			t.Errorf("Expected JSON data to contain {'key': 'value'}, got %v", jsonData)
		}

		t.Logf("Received response with correct POST body")
	})
}
