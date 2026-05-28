//go:build js && (!tinygo || !bldr_tinygo_js_imports)

package fetch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/aperturerobotics/util/iocloser"
	"github.com/aperturerobotics/util/js/internal/tinygojs"
	stream "github.com/aperturerobotics/util/js/readable-stream"
)

// Fetch uses the JS Fetch API to make requests.
func Fetch(url string, opts *Opts) (*Response, error) {
	optsMap, err := mapOpts(opts)
	if err != nil {
		return nil, err
	}

	type fetchResponse struct {
		r *Response
		b js.Value
		e error
	}
	ch := make(chan *fetchResponse)
	if opts != nil && opts.Signal != nil {
		controller := tinygojs.New(js.Global().Get("AbortController"))
		signal := controller.Get("signal")
		optsMap["signal"] = signal
		abort := func() {
			tinygojs.Call(controller, "abort")
		}
		context.AfterFunc(opts.Signal, abort)
	}

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r := new(Response)
		resp := args[0]
		headersIt := tinygojs.Call(resp.Get("headers"), "entries")
		headers := Header{}
		for {
			n := tinygojs.Call(headersIt, "next")
			if n.Get("done").Bool() {
				break
			}
			pair := n.Get("value")
			key, value := pair.Index(0).String(), pair.Index(1).String()
			headers.Add(key, value)
		}
		r.Header = headers
		r.OK = resp.Get("ok").Bool()
		r.Redirected = resp.Get("redirected").Bool()
		r.StatusCode = resp.Get("status").Int()
		r.Status = strconv.Itoa(r.StatusCode) + " " + resp.Get("statusText").String()
		r.Type = resp.Get("type").String()
		r.URL = resp.Get("url").String()

		ch <- &fetchResponse{r: r, b: resp}
		return nil
	})

	defer success.Release()

	failure := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		msg := "fetch failed"
		if len(args) != 0 {
			msg = tinygojs.RejectionMessage(args[0])
		}
		ch <- &fetchResponse{e: errors.New(msg)}
		return nil
	})
	defer failure.Release()

	jsOptsMap := js.ValueOf(optsMap)
	promise := tinygojs.Call(js.Global(), "fetch", url, jsOptsMap)
	go tinygojs.AwaitPromise(promise, success, failure)

	r := <-ch
	if r.e != nil {
		return nil, r.e
	}

	bodyStream := r.b.Get("body")
	if !bodyStream.IsNull() && !bodyStream.IsUndefined() {
		r.r.Body = stream.NewReadableStream(bodyStream)

		// Read a small amount of data to ensure the stream is working
		testBuf := make([]byte, 1)
		n, err := r.r.Body.Read(testBuf)
		if err != nil && err != io.EOF {
			r.r.Body.Close()
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		// If we read some data, we need to put it back
		if n > 0 {
			r.r.Body = &prefixedReader{
				prefix: testBuf[:n],
				reader: r.r.Body,
			}
		}
	} else {
		r.r.Body = iocloser.NewReadCloser(bytes.NewReader(nil), nil)
	}

	return r.r, nil
}

// prefixedReader is a reader that first returns bytes from a prefix, then continues with the underlying reader
type prefixedReader struct {
	prefix []byte
	reader io.ReadCloser
}

func (pr *prefixedReader) Read(p []byte) (n int, err error) {
	if len(pr.prefix) > 0 {
		n = copy(p, pr.prefix)
		pr.prefix = pr.prefix[n:]
		return n, nil
	}
	return pr.reader.Read(p)
}

func (pr *prefixedReader) Close() error {
	return pr.reader.Close()
}

// oof.
func mapOpts(opts *Opts) (map[string]interface{}, error) {
	mp := map[string]interface{}{}
	if opts == nil {
		return mp, nil
	}

	if opts.Method != "" {
		mp["method"] = opts.Method
	}
	if opts.Header != nil {
		mp["headers"] = mapHeaders(opts.Header)
	}
	if opts.Mode != "" {
		mp["mode"] = opts.Mode
	}
	if opts.Credentials != "" {
		mp["credentials"] = opts.Credentials
	}
	if opts.Cache != "" {
		mp["cache"] = opts.Cache
	}
	if opts.Redirect != "" {
		mp["redirect"] = opts.Redirect
	}
	if opts.Referrer != "" {
		mp["referrer"] = opts.Referrer
	}
	if opts.ReferrerPolicy != "" {
		mp["referrerPolicy"] = opts.ReferrerPolicy
	}
	if opts.Integrity != "" {
		mp["integrity"] = opts.Integrity
	}
	if opts.KeepAlive != nil {
		mp["keepalive"] = *opts.KeepAlive
	}

	if opts.Body != nil {
		bts, err := io.ReadAll(opts.Body)
		if err != nil {
			return nil, err
		}

		uint8Array := tinygojs.New(js.Global().Get("Uint8Array"), len(bts))
		js.CopyBytesToJS(uint8Array, bts)
		mp["body"] = uint8Array
	}

	return mp, nil
}

func mapHeaders(mp Header) map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range mp {
		newMap[k] = strings.Join(v, ", ")
	}
	return newMap
}
