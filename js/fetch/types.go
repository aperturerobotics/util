//go:build js

package fetch

import (
	"context"
	"io"
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

// Merge merges another CommonOpts into the current CommonOpts, overwriting fields if set.
func (c *CommonOpts) Merge(other *CommonOpts) {
	if other == nil {
		return
	}

	if other.Mode != "" {
		c.Mode = other.Mode
	}
	if other.Credentials != "" {
		c.Credentials = other.Credentials
	}
	if other.Cache != "" {
		c.Cache = other.Cache
	}
	if other.Redirect != "" {
		c.Redirect = other.Redirect
	}
	if other.Referrer != "" {
		c.Referrer = other.Referrer
	}
	if other.ReferrerPolicy != "" {
		c.ReferrerPolicy = other.ReferrerPolicy
	}
	if other.Integrity != "" {
		c.Integrity = other.Integrity
	}
	if other.KeepAlive != nil {
		keepAliveValue := *other.KeepAlive
		c.KeepAlive = &keepAliveValue
	}
}

// Merge merges another Opts into the current Opts, overwriting fields if set.
func (o *Opts) Merge(other *Opts) {
	if other == nil {
		return
	}

	if other.Method != "" {
		o.Method = other.Method
	}
	if other.Header != nil {
		if o.Header == nil {
			o.Header = make(Header)
		}
		for k, v := range other.Header {
			o.Header[k] = v
		}
	}
	if other.Body != nil {
		o.Body = other.Body
	}
	if other.Signal != nil {
		o.Signal = other.Signal
	}

	// Merge CommonOpts
	o.CommonOpts.Merge(&other.CommonOpts)
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
	// opaqueredirect: The fetch request was made with redirect: "manual". The Response's status is 0, headers are empty and trailer is empty.
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
