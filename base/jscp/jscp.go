//go:build js && !goscript

// Package jscp provides functions for copying data to JavaScript
package jscp

import (
	"syscall/js"
	"unsafe"
)

// BytesToJS converts the given bytes to a js Uint8ClampedArray
// by using the global wasm memory bytes. This avoids the
// copying present in [js.CopyBytesToJS]. The result is detached when the
// wasm memory grows, which can happen on any Go allocation, so it must be
// used right away: use [StableBytesToJS] for bytes that a js object keeps.
func BytesToJS(b []byte) js.Value {
	ptr := uintptr(unsafe.Pointer(&b[0]))
	// We directly pass the offset and length to the constructor to avoid calling subarray or slice,
	// thereby improving performance and safety (this fixes a detached array buffer crash).
	return js.Global().Get("Uint8ClampedArray").New(js.Global().Get("wasm").Get("instance").Get("exports").Get("mem").Get("buffer"), ptr, len(b))
}

// StableBytesToJS converts the given bytes to a js Uint8ClampedArray that
// stays valid after the call, for a js object that keeps it, such as an
// ImageData. On wasm, it is a copy of the bytes: the view of the wasm memory
// that [BytesToJS] returns is detached when that memory grows, which can
// happen on any Go allocation.
func StableBytesToJS(b []byte) js.Value {
	array := js.Global().Get("Uint8ClampedArray").New(len(b))
	js.CopyBytesToJS(array, b)
	return array
}
