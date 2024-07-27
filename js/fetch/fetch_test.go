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

func TestMerge(t *testing.T) {
	t.Run("CommonOpts.Merge", func(t *testing.T) {
		base := &CommonOpts{
			Mode:        "cors",
			Credentials: "include",
			Cache:       "no-cache",
		}
		other := &CommonOpts{
			Mode:           "no-cors",
			ReferrerPolicy: "no-referrer",
			KeepAlive:      &[]bool{true}[0],
		}

		base.Merge(other)

		if base.Mode != "no-cors" {
			t.Errorf("Expected Mode to be 'no-cors', got '%s'", base.Mode)
		}
		if base.Credentials != "include" {
			t.Errorf("Expected Credentials to be 'include', got '%s'", base.Credentials)
		}
		if base.Cache != "no-cache" {
			t.Errorf("Expected Cache to be 'no-cache', got '%s'", base.Cache)
		}
		if base.ReferrerPolicy != "no-referrer" {
			t.Errorf("Expected ReferrerPolicy to be 'no-referrer', got '%s'", base.ReferrerPolicy)
		}
		if base.KeepAlive == nil || *base.KeepAlive != true {
			t.Errorf("Expected KeepAlive to be true, got %v", base.KeepAlive)
		}
	})

	t.Run("Opts.Merge", func(t *testing.T) {
		base := &Opts{
			Method: "GET",
			Header: Header{"Content-Type": []string{"application/json"}},
			CommonOpts: CommonOpts{
				Mode: "cors",
			},
		}
		other := &Opts{
			Method: "POST",
			Header: Header{"Authorization": []string{"Bearer token"}},
			Body:   strings.NewReader("test body"),
			CommonOpts: CommonOpts{
				Credentials: "include",
			},
		}

		base.Merge(other)

		if base.Method != "POST" {
			t.Errorf("Expected Method to be 'POST', got '%s'", base.Method)
		}
		if len(base.Header) != 2 {
			t.Errorf("Expected 2 headers, got %d", len(base.Header))
		}
		if base.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be 'application/json', got '%s'", base.Header.Get("Content-Type"))
		}
		if base.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Expected Authorization header to be 'Bearer token', got '%s'", base.Header.Get("Authorization"))
		}
		if base.Body == nil {
			t.Error("Expected Body to be set, got nil")
		}
		if base.CommonOpts.Mode != "cors" {
			t.Errorf("Expected Mode to be 'cors', got '%s'", base.CommonOpts.Mode)
		}
		if base.CommonOpts.Credentials != "include" {
			t.Errorf("Expected Credentials to be 'include', got '%s'", base.CommonOpts.Credentials)
		}
	})
}
