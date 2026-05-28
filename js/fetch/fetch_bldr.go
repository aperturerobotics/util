//go:build js && tinygo && bldr_tinygo_js_imports

package fetch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"runtime"
	"strconv"
	"sync"
	"unsafe"

	"github.com/pkg/errors"
)

type bldrFetchRequest struct {
	Method         string `json:"method,omitempty"`
	Header         Header `json:"header,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Credentials    string `json:"credentials,omitempty"`
	Cache          string `json:"cache,omitempty"`
	Redirect       string `json:"redirect,omitempty"`
	Referrer       string `json:"referrer,omitempty"`
	ReferrerPolicy string `json:"referrerPolicy,omitempty"`
	Integrity      string `json:"integrity,omitempty"`
	KeepAlive      *bool  `json:"keepAlive,omitempty"`
	Signal         bool   `json:"signal,omitempty"`
}

type bldrFetchMeta struct {
	OK         bool              `json:"ok"`
	Header     []bldrFetchHeader `json:"header"`
	Redirected bool              `json:"redirected"`
	StatusCode int               `json:"statusCode"`
	StatusText string            `json:"statusText"`
	Type       string            `json:"type"`
	URL        string            `json:"url"`
}

type bldrFetchHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type bldrFetchResult struct {
	resp *Response
	err  error
}

var (
	bldrFetchMu       sync.Mutex
	bldrFetchNextOpID uint32
	bldrFetchOps      = map[uint32]chan bldrFetchResult{}
)

//go:wasmimport gojs bldr.tinygo.fetch
func bldrTinyGoFetch(opID uint32, urlPtr unsafe.Pointer, urlLen uint32, reqPtr unsafe.Pointer, reqLen uint32, bodyPtr unsafe.Pointer, bodyLen uint32)

//go:wasmimport gojs bldr.tinygo.fetchAbort
func bldrTinyGoFetchAbort(opID uint32) uint32

//go:wasmimport gojs bldr.tinygo.takeStoredBytes
func bldrTinyGoTakeStoredBytes(bytesID uint32, ptr unsafe.Pointer, len uint32) uint32

// Fetch uses Bldr's TinyGo fetch import to make requests without raw
// syscall/js callbacks or response-object decoding in Go.
func Fetch(url string, opts *Opts) (*Response, error) {
	reqBytes, bodyBytes, err := buildBldrFetchRequest(opts)
	if err != nil {
		return nil, err
	}

	opID, ch := registerBldrFetchOp()
	defer deleteBldrFetchOp(opID)

	ctx := context.Background()
	if opts != nil && opts.Signal != nil {
		ctx = opts.Signal
		stopAbort := context.AfterFunc(ctx, func() {
			bldrTinyGoFetchAbort(opID)
		})
		defer stopAbort()
	}

	urlBytes := []byte(url)
	bldrTinyGoFetch(
		opID,
		bytesPtr(urlBytes),
		uint32(len(urlBytes)),
		bytesPtr(reqBytes),
		uint32(len(reqBytes)),
		bytesPtr(bodyBytes),
		uint32(len(bodyBytes)),
	)
	runtime.KeepAlive(urlBytes)
	runtime.KeepAlive(reqBytes)
	runtime.KeepAlive(bodyBytes)

	select {
	case result := <-ch:
		return result.resp, result.err
	case <-ctx.Done():
		bldrTinyGoFetchAbort(opID)
		return nil, ctx.Err()
	}
}

func buildBldrFetchRequest(opts *Opts) ([]byte, []byte, error) {
	req := bldrFetchRequest{}
	var body []byte
	if opts != nil {
		req = bldrFetchRequest{
			Method:         opts.Method,
			Header:         opts.Header,
			Mode:           opts.Mode,
			Credentials:    opts.Credentials,
			Cache:          opts.Cache,
			Redirect:       opts.Redirect,
			Referrer:       opts.Referrer,
			ReferrerPolicy: opts.ReferrerPolicy,
			Integrity:      opts.Integrity,
			KeepAlive:      opts.KeepAlive,
			Signal:         opts.Signal != nil,
		}
		if opts.Body != nil {
			var err error
			body, err = io.ReadAll(opts.Body)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	return reqBytes, body, nil
}

func registerBldrFetchOp() (uint32, chan bldrFetchResult) {
	ch := make(chan bldrFetchResult, 1)
	bldrFetchMu.Lock()
	bldrFetchNextOpID++
	if bldrFetchNextOpID == 0 {
		bldrFetchNextOpID++
	}
	opID := bldrFetchNextOpID
	bldrFetchOps[opID] = ch
	bldrFetchMu.Unlock()
	return opID, ch
}

func deleteBldrFetchOp(opID uint32) {
	bldrFetchMu.Lock()
	delete(bldrFetchOps, opID)
	bldrFetchMu.Unlock()
}

func sendBldrFetchResult(opID uint32, result bldrFetchResult) {
	bldrFetchMu.Lock()
	ch := bldrFetchOps[opID]
	bldrFetchMu.Unlock()
	if ch == nil {
		return
	}
	ch <- result
}

//go:wasmexport BLDR_TINYGO_FETCH_RESOLVE
func bldrTinyGoFetchResolve(opID uint32, metaID uint32, metaLen uint32, bodyID uint32, bodyLen uint32) {
	metaBytes, ok := takeBldrStoredBytes(metaID, metaLen)
	if !ok {
		sendBldrFetchResult(opID, bldrFetchResult{err: errors.New("fetch metadata unavailable")})
		return
	}
	bodyBytes, ok := takeBldrStoredBytes(bodyID, bodyLen)
	if !ok {
		sendBldrFetchResult(opID, bldrFetchResult{err: errors.New("fetch body unavailable")})
		return
	}

	var meta bldrFetchMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		sendBldrFetchResult(opID, bldrFetchResult{err: errors.Wrap(err, "decode fetch metadata")})
		return
	}

	headers := Header{}
	for _, header := range meta.Header {
		headers.Add(header.Key, header.Value)
	}
	statusText := meta.StatusText
	status := strconv.Itoa(meta.StatusCode)
	if statusText != "" {
		status += " " + statusText
	}
	sendBldrFetchResult(opID, bldrFetchResult{
		resp: &Response{
			OK:         meta.OK,
			Header:     headers,
			Redirected: meta.Redirected,
			StatusCode: meta.StatusCode,
			Status:     status,
			Type:       meta.Type,
			URL:        meta.URL,
			Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		},
	})
}

//go:wasmexport BLDR_TINYGO_FETCH_REJECT
func bldrTinyGoFetchReject(opID uint32, code uint32) {
	sendBldrFetchResult(opID, bldrFetchResult{err: errors.Errorf("fetch failed with code %d", code)})
}

func takeBldrStoredBytes(bytesID uint32, length uint32) ([]byte, bool) {
	if length == 0 {
		return nil, bldrTinyGoTakeStoredBytes(bytesID, nil, 0) != 0
	}
	bytes := make([]byte, int(length))
	if bldrTinyGoTakeStoredBytes(bytesID, unsafe.Pointer(&bytes[0]), length) == 0 {
		return nil, false
	}
	return bytes, true
}

func bytesPtr(bytes []byte) unsafe.Pointer {
	if len(bytes) == 0 {
		return nil
	}
	return unsafe.Pointer(&bytes[0])
}
