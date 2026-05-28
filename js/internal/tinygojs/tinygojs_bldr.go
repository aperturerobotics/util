//go:build js && tinygo && bldr_tinygo_js_imports

// Package tinygojs centralizes TinyGo-safe syscall/js helper calls.
package tinygojs

import (
	"fmt"
	"runtime"
	"strconv"
	"syscall/js"
	"unsafe"
)

const (
	nanHead      = 0x7ff80000
	typeFlagNone = 0
)

type tinyGoJSRef uint64

type tinyGoJSValue struct {
	_     [0]func()
	ref   tinyGoJSRef
	gcPtr *tinyGoJSRef
}

//go:wasmimport gojs bldr.tinygo.jsCall0
func bldrTinyGoJSCall0(targetRef uint64, methodPtr unsafe.Pointer, methodLen uint32) uint64

//go:wasmimport gojs bldr.tinygo.jsCall1Value
func bldrTinyGoJSCall1Value(targetRef uint64, methodPtr unsafe.Pointer, methodLen uint32, arg0Ref uint64) uint64

//go:wasmimport gojs bldr.tinygo.jsCall2StringValue
func bldrTinyGoJSCall2StringValue(targetRef uint64, methodPtr unsafe.Pointer, methodLen uint32, arg0Ptr unsafe.Pointer, arg0Len uint32, arg1Ref uint64) uint64

//go:wasmimport gojs bldr.tinygo.jsNew0
func bldrTinyGoJSNew0(ctorRef uint64) uint64

//go:wasmimport gojs bldr.tinygo.jsNew1Int
func bldrTinyGoJSNew1Int(ctorRef uint64, arg0 uint32) uint64

//go:wasmimport gojs bldr.tinygo.promiseAwait
func bldrTinyGoPromiseAwait(promiseRef uint64, thenRef uint64, catchRef uint64)

//go:wasmimport gojs syscall/js.finalizeRef
func tinyGoFinalizeRef(ref tinyGoJSRef)

// Call calls target[method](...args) through Bldr's TinyGo import table.
func Call(target js.Value, method string, args ...any) js.Value {
	methodBytes := []byte(method)
	methodPtr := bytesPtr(methodBytes)
	methodLen := uint32(len(methodBytes))
	targetRef := jsRef(target)
	switch len(args) {
	case 0:
		ref := bldrTinyGoJSCall0(targetRef, methodPtr, methodLen)
		runtime.KeepAlive(methodBytes)
		runtime.KeepAlive(target)
		return valueFromRef(ref)
	case 1:
		switch arg := args[0].(type) {
		case js.Value:
			ref := bldrTinyGoJSCall1Value(targetRef, methodPtr, methodLen, jsRef(arg))
			runtime.KeepAlive(methodBytes)
			runtime.KeepAlive(target)
			runtime.KeepAlive(arg)
			return valueFromRef(ref)
		case js.Func:
			ref := bldrTinyGoJSCall1Value(targetRef, methodPtr, methodLen, jsRef(arg.Value))
			runtime.KeepAlive(methodBytes)
			runtime.KeepAlive(target)
			runtime.KeepAlive(arg)
			return valueFromRef(ref)
		}
	case 2:
		arg0, ok := args[0].(string)
		if ok {
			arg0Bytes := []byte(arg0)
			arg0Ptr := bytesPtr(arg0Bytes)
			arg0Len := uint32(len(arg0Bytes))
			switch arg1 := args[1].(type) {
			case js.Value:
				ref := bldrTinyGoJSCall2StringValue(targetRef, methodPtr, methodLen, arg0Ptr, arg0Len, jsRef(arg1))
				runtime.KeepAlive(methodBytes)
				runtime.KeepAlive(arg0Bytes)
				runtime.KeepAlive(target)
				runtime.KeepAlive(arg1)
				return valueFromRef(ref)
			case js.Func:
				ref := bldrTinyGoJSCall2StringValue(targetRef, methodPtr, methodLen, arg0Ptr, arg0Len, jsRef(arg1.Value))
				runtime.KeepAlive(methodBytes)
				runtime.KeepAlive(arg0Bytes)
				runtime.KeepAlive(target)
				runtime.KeepAlive(arg1)
				return valueFromRef(ref)
			}
		}
	}
	panic(fmt.Sprintf("unsupported Bldr TinyGo JavaScript call %q with %d argument(s)", method, len(args)))
}

// New constructs ctor through Bldr's TinyGo import table.
func New(ctor js.Value, args ...any) js.Value {
	ctorRef := jsRef(ctor)
	switch len(args) {
	case 0:
		ref := bldrTinyGoJSNew0(ctorRef)
		runtime.KeepAlive(ctor)
		return valueFromRef(ref)
	case 1:
		arg0, ok := uint32Arg(args[0])
		if ok {
			ref := bldrTinyGoJSNew1Int(ctorRef, arg0)
			runtime.KeepAlive(ctor)
			return valueFromRef(ref)
		}
	}
	panic(fmt.Sprintf("unsupported Bldr TinyGo JavaScript constructor with %d argument(s)", len(args)))
}

// AwaitPromise attaches promise completion callbacks through Bldr's TinyGo
// import table. The JavaScript side defers callback execution away from the
// current JS event frame for TinyGo asyncify builds.
func AwaitPromise(promise js.Value, thenCb, catchCb js.Func) {
	bldrTinyGoPromiseAwait(jsRef(promise), jsRef(thenCb.Value), jsRef(catchCb.Value))
	runtime.KeepAlive(promise)
	runtime.KeepAlive(thenCb)
	runtime.KeepAlive(catchCb)
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
	return "js promise rejected"
}

// UseHelpers reports whether TinyGo-specific JavaScript helpers may be used.
func UseHelpers() bool {
	return true
}

// Available returns true if fn is a callable JavaScript value.
func Available(fn js.Value) bool {
	return !fn.IsUndefined() && !fn.IsNull() && fn.Type() == js.TypeFunction
}

func jsRef(value js.Value) uint64 {
	return uint64((*tinyGoJSValue)(unsafe.Pointer(&value)).ref)
}

func valueFromRef(rawRef uint64) js.Value {
	ref := tinyGoJSRef(rawRef)
	var gcPtr *tinyGoJSRef
	typeFlag := (ref >> 32) & 7
	if (ref>>32)&nanHead == nanHead && typeFlag != typeFlagNone {
		gcPtr = new(tinyGoJSRef)
		*gcPtr = ref
		runtime.SetFinalizer(gcPtr, func(p *tinyGoJSRef) {
			tinyGoFinalizeRef(*p)
		})
	}
	return *(*js.Value)(unsafe.Pointer(&tinyGoJSValue{ref: ref, gcPtr: gcPtr}))
}

func bytesPtr(bytes []byte) unsafe.Pointer {
	if len(bytes) == 0 {
		return nil
	}
	return unsafe.Pointer(&bytes[0])
}

func uint32Arg(value any) (uint32, bool) {
	switch v := value.(type) {
	case int:
		if v < 0 || uint64(v) > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(v), true
	case uint:
		if uint64(v) > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(v), true
	case int32:
		if v < 0 {
			return 0, false
		}
		return uint32(v), true
	case uint32:
		return v, true
	default:
		return 0, false
	}
}
