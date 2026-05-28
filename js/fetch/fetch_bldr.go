//go:build js && tinygo && bldr_tinygo_js_imports

package fetch

import (
	"bytes"
	"context"
	"io"
	"runtime"
	"strconv"
	"sync"
	"unsafe"

	"github.com/aperturerobotics/fastjson"
	"github.com/pkg/errors"
)

type bldrFetchResult struct {
	metaID  uint32
	metaLen uint32
	bodyID  uint32
	bodyLen uint32
	err     error
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

//go:wasmimport gojs bldr.tinygo.dropStoredBytes
func bldrTinyGoDropStoredBytes(bytesID uint32) uint32

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
		return finishBldrFetchResult(result)
	case <-ctx.Done():
		deleteBldrFetchOp(opID)
		bldrTinyGoFetchAbort(opID)
		select {
		case result := <-ch:
			return finishBldrFetchResult(result)
		default:
		}
		return nil, ctx.Err()
	}
}

func buildBldrFetchRequest(opts *Opts) ([]byte, []byte, error) {
	req := []byte{'{'}
	first := true
	var body []byte
	if opts != nil {
		req = appendJSONStringField(req, &first, "method", opts.Method)
		req = appendJSONHeaderField(req, &first, "header", opts.Header)
		req = appendJSONStringField(req, &first, "mode", opts.Mode)
		req = appendJSONStringField(req, &first, "credentials", opts.Credentials)
		req = appendJSONStringField(req, &first, "cache", opts.Cache)
		req = appendJSONStringField(req, &first, "redirect", opts.Redirect)
		req = appendJSONStringField(req, &first, "referrer", opts.Referrer)
		req = appendJSONStringField(req, &first, "referrerPolicy", opts.ReferrerPolicy)
		req = appendJSONStringField(req, &first, "integrity", opts.Integrity)
		if opts.KeepAlive != nil {
			req = appendJSONBoolField(req, &first, "keepAlive", *opts.KeepAlive)
		}
		if opts.Signal != nil {
			req = appendJSONBoolField(req, &first, "signal", true)
		}
		if opts.Body != nil {
			var err error
			body, err = io.ReadAll(opts.Body)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	req = append(req, '}')
	return req, body, nil
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

func sendBldrFetchResult(opID uint32, result bldrFetchResult) bool {
	bldrFetchMu.Lock()
	ch := bldrFetchOps[opID]
	sent := false
	if ch != nil {
		select {
		case ch <- result:
			sent = true
		default:
		}
	}
	bldrFetchMu.Unlock()
	return sent
}

//go:wasmexport BLDR_TINYGO_FETCH_RESOLVE
func bldrTinyGoFetchResolve(opID uint32, metaID uint32, metaLen uint32, bodyID uint32, bodyLen uint32) {
	if sendBldrFetchResult(opID, bldrFetchResult{
		metaID:  metaID,
		metaLen: metaLen,
		bodyID:  bodyID,
		bodyLen: bodyLen,
	}) {
		return
	}
	dropBldrStoredBytes(metaID)
	dropBldrStoredBytes(bodyID)
}

func finishBldrFetchResult(result bldrFetchResult) (*Response, error) {
	if result.err != nil {
		return nil, result.err
	}
	metaBytes, ok := takeBldrStoredBytes(result.metaID, result.metaLen)
	if !ok {
		dropBldrStoredBytes(result.bodyID)
		return nil, errors.New("fetch metadata unavailable")
	}
	bodyBytes, ok := takeBldrStoredBytes(result.bodyID, result.bodyLen)
	if !ok {
		return nil, errors.New("fetch body unavailable")
	}
	return buildBldrFetchResponse(metaBytes, bodyBytes)
}

func buildBldrFetchResponse(metaBytes []byte, bodyBytes []byte) (*Response, error) {
	var parser fastjson.Parser
	meta, err := parser.ParseBytes(metaBytes)
	if err != nil {
		return nil, errors.Wrap(err, "decode fetch metadata")
	}

	headers := Header{}
	for _, item := range meta.GetArray("header") {
		key := item.GetStringBytes("key")
		if key == nil {
			continue
		}
		headers.Add(string(key), string(item.GetStringBytes("value")))
	}
	statusText := string(meta.GetStringBytes("statusText"))
	statusCode := meta.GetInt("statusCode")
	status := strconv.Itoa(statusCode)
	if statusText != "" {
		status += " " + statusText
	}
	return &Response{
		OK:         meta.GetBool("ok"),
		Header:     headers,
		Redirected: meta.GetBool("redirected"),
		StatusCode: statusCode,
		Status:     status,
		Type:       string(meta.GetStringBytes("type")),
		URL:        string(meta.GetStringBytes("url")),
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
	}, nil
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

func dropBldrStoredBytes(bytesID uint32) {
	bldrTinyGoDropStoredBytes(bytesID)
}

func appendJSONFieldName(dst []byte, first *bool, name string) []byte {
	if *first {
		*first = false
	} else {
		dst = append(dst, ',')
	}
	dst = strconv.AppendQuote(dst, name)
	dst = append(dst, ':')
	return dst
}

func appendJSONStringField(dst []byte, first *bool, name string, value string) []byte {
	if value == "" {
		return dst
	}
	dst = appendJSONFieldName(dst, first, name)
	return strconv.AppendQuote(dst, value)
}

func appendJSONBoolField(dst []byte, first *bool, name string, value bool) []byte {
	dst = appendJSONFieldName(dst, first, name)
	if value {
		return append(dst, "true"...)
	}
	return append(dst, "false"...)
}

func appendJSONHeaderField(dst []byte, first *bool, name string, header Header) []byte {
	if len(header) == 0 {
		return dst
	}
	dst = appendJSONFieldName(dst, first, name)
	dst = append(dst, '{')
	headerFirst := true
	for key, values := range header {
		if headerFirst {
			headerFirst = false
		} else {
			dst = append(dst, ',')
		}
		dst = strconv.AppendQuote(dst, key)
		dst = append(dst, ':', '[')
		for idx, value := range values {
			if idx != 0 {
				dst = append(dst, ',')
			}
			dst = strconv.AppendQuote(dst, value)
		}
		dst = append(dst, ']')
	}
	dst = append(dst, '}')
	return dst
}

func bytesPtr(bytes []byte) unsafe.Pointer {
	if len(bytes) == 0 {
		return nil
	}
	return unsafe.Pointer(&bytes[0])
}
