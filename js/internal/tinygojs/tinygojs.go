//go:build js

// Package tinygojs centralizes TinyGo-safe syscall/js helper calls.
package tinygojs

import (
	"runtime"
	"strconv"
	"syscall/js"
)

const (
	tinyGoJSCall       = "BLDR_TINYGO_JS_CALL"
	tinyGoJSNew        = "BLDR_TINYGO_JS_NEW"
	tinyGoPromiseAwait = "BLDR_TINYGO_PROMISE_AWAIT"
)

// Call calls target[method](...args), using a JS-owned helper when available.
func Call(target js.Value, method string, args ...any) js.Value {
	if UseHelpers() {
		call := js.Global().Get(tinyGoJSCall)
		if Available(call) {
			helperArgs := make([]any, 0, len(args)+2)
			helperArgs = append(helperArgs, target, method)
			helperArgs = append(helperArgs, args...)
			return call.Invoke(helperArgs...)
		}
	}
	return target.Call(method, args...)
}

// New constructs ctor with args, using a JS-owned helper when available.
func New(ctor js.Value, args ...any) js.Value {
	if UseHelpers() {
		newValue := js.Global().Get(tinyGoJSNew)
		if Available(newValue) {
			helperArgs := make([]any, 0, len(args)+1)
			helperArgs = append(helperArgs, ctor)
			helperArgs = append(helperArgs, args...)
			return newValue.Invoke(helperArgs...)
		}
	}
	return ctor.New(args...)
}

// AwaitPromise attaches promise completion callbacks through a JS-owned helper
// when available. The helper defers callback execution away from the current JS
// event frame for TinyGo asyncify builds.
func AwaitPromise(promise js.Value, thenCb, catchCb js.Func) {
	if UseHelpers() {
		await := js.Global().Get(tinyGoPromiseAwait)
		if Available(await) {
			await.Invoke(promise, thenCb, catchCb)
			return
		}
	}
	Call(Call(promise, "then", thenCb), "catch", catchCb)
}

// RejectionMessage returns a conservative error message for a JS promise
// rejection. Bldr's TinyGo helper passes numeric error classes instead of raw
// browser Error objects to avoid TinyGo stringifying arbitrary JS values.
func RejectionMessage(value js.Value) string {
	if value.IsUndefined() || value.IsNull() {
		return "js promise rejected"
	}
	if value.Type() == js.TypeNumber {
		return "js promise rejected with code " + strconv.Itoa(value.Int())
	}
	message := value.Get("message")
	if !message.IsUndefined() && !message.IsNull() && message.Type() == js.TypeString {
		return message.String()
	}
	if UseHelpers() {
		return "js promise rejected"
	}
	return value.String()
}

// UseHelpers reports whether TinyGo-specific JavaScript helpers may be used.
func UseHelpers() bool {
	return runtime.Compiler == "tinygo"
}

// Available returns true if fn is a callable JavaScript value.
func Available(fn js.Value) bool {
	return !fn.IsUndefined() && !fn.IsNull() && fn.Type() == js.TypeFunction
}
