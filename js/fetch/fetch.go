//go:build js

package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"syscall/js"

	stream "github.com/aperturerobotics/util/js/readable-stream"
)

// Opts are the options for Fetch.
type Opts struct {
	// CommonOpts are the common Fetch options.
	CommonOpts

	// Method specifies the HTTP method (GET, POST, PUT, etc.).
	// For client requests, an empty string means GET.
	// constants are copied from net/http to avoid import
	Method string

	// Header contains the request header fields.
	//
	// Example:
	//
	//	Host: example.com
	//	accept-encoding: gzip, deflate
	//	Accept-Language: en-us
	//	fOO: Bar
	//	foo: two
	//
	// then
	//
	//	Header = map[string][]string{
	//		"Accept-Encoding": {"gzip, deflate"},
	//		"Accept-Language": {"en-us"},
	//		"Foo": {"Bar", "two"},
	//	}
	//
	// HTTP defines that header names are case-insensitive. The
	// request parser implements this by using CanonicalHeaderKey,
	// making the first character and any characters following a
	// hyphen uppercase and the rest lowercase.
	//
	// Certain headers such as Content-Length and Connection are automatically
	// written when needed and values in Header may be ignored.
	Header Header

	// Body is the request's body.
	//
	// For client requests, a nil body means the request has no
	// body, such as a GET request. The HTTP Client's Transport
	// is responsible for calling the Close method.
	//
	// Body must allow Read to be called concurrently with Close.
	// In particular, calling Close should unblock a Read waiting
	// for input.
	Body io.Reader

	// Signal docs https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
	Signal context.Context
}

// CommonOpts are opts for Fetch that can be reused between requests.
type CommonOpts struct {
	// Mode docs https://developer.mozilla.org/en-US/docs/Web/API/Request/mode
	Mode string

	// Credentials docs https://developer.mozilla.org/en-US/docs/Web/API/Request/credentials
	Credentials string

	// Cache docs https://developer.mozilla.org/en-US/docs/Web/API/Request/cache
	Cache string

	// Redirect docs https://developer.mozilla.org/en-US/docs/Web/API/WindowOrWorkerGlobalScope/fetch
	Redirect string

	// Referrer docs https://developer.mozilla.org/en-US/docs/Web/API/Request/referrer
	Referrer string

	// ReferrerPolicy docs https://developer.mozilla.org/en-US/docs/Web/API/WindowOrWorkerGlobalScope/fetch
	ReferrerPolicy string

	// Integrity docs https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
	Integrity string

	// KeepAlive docs https://developer.mozilla.org/en-US/docs/Web/API/WindowOrWorkerGlobalScope/fetch
	KeepAlive *bool
}

// Clone clones the opts, excluding the Body field.
func (o *Opts) Clone() *Opts {
	if o == nil {
		return nil
	}

	clone := &Opts{
		CommonOpts: o.CommonOpts,
		Method:     o.Method,
		Header:     o.Header.Clone(),
		Signal:     o.Signal,
	}

	if o.CommonOpts.KeepAlive != nil {
		keepAliveValue := *o.CommonOpts.KeepAlive
		clone.CommonOpts.KeepAlive = &keepAliveValue
	}

	return clone
}

// Response is the response that returns from the fetch promise.
type Response struct {
	// OK indicates if the response status code indicates success or not.
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/ok
	OK bool

	// Header maps header keys to values. If the response had multiple
	// headers with the same key, they may be concatenated, with comma
	// delimiters.  (RFC 7230, section 3.2.2 requires that multiple headers
	// be semantically equivalent to a comma-delimited sequence.) When
	// Header values are duplicated by other fields in this struct (e.g.,
	// ContentLength, TransferEncoding, Trailer), the field values are
	// authoritative.
	//
	// Keys in the map are canonicalized (see CanonicalHeaderKey).
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/headers
	Header Header

	// Redirected indicates the request was redirected one or more times.
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/redirected
	Redirected bool

	// StatusCode is the HTTP status code.
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/status
	StatusCode int

	// Status is the HTTP status text e.g. 200 OK.
	//
	// For compatibility this is {StatusCode} {statusText}
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/statusText
	Status string

	// Type has the type of the response.
	//
	// basic: Normal, same origin response, with all headers exposed except "Set-Cookie".
	// cors: Response was received from a valid cross-origin request. Certain headers and the body may be accessed.
	// error: Network error. No useful information describing the error is available. The Response's status is 0, headers are empty and immutable. This is the type for a Response obtained from Response.error().
	// opaque: Response for "no-cors" request to cross-origin resource. Severely restricted.
	// opaqueredirect: The fetch request was made with redirect: "manual". The Response's status is 0, headers are empty, body is null and trailer is empty.
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/type
	Type string

	// The url read-only property of the Response interface contains the URL of
	// the response. The value of the url property will be the final URL
	// obtained after any redirects.
	//
	// https://developer.mozilla.org/en-US/docs/Web/API/Response/url
	URL string

	// Body is the body of the response.
	Body io.ReadCloser
}

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
		controller := js.Global().Get("AbortController").New()
		signal := controller.Get("signal")
		optsMap["signal"] = signal
		abort := func() {
			controller.Call("abort")
		}
		defer abort()
		defer context.AfterFunc(opts.Signal, abort)()
	}

	success := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r := new(Response)
		resp := args[0]
		headersIt := resp.Get("headers").Call("entries")
		headers := Header{}
		for {
			n := headersIt.Call("next")
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
		msg := args[0].Get("message").String()
		ch <- &fetchResponse{e: errors.New(msg)}
		return nil
	})
	defer failure.Release()

	go js.Global().Call("fetch", url, optsMap).Call("then", success).Call("catch", failure)

	r := <-ch
	if r.e != nil {
		return nil, r.e
	}

	bodyStream := r.b.Get("body")
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

		mp["body"] = string(bts)
	}

	return mp, nil
}

func mapHeaders(mp Header) map[string]interface{} {
	newMap := map[string]interface{}{}
	for k, v := range mp {
		newMap[k] = v
	}
	return newMap
}
