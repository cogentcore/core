//go:build js

// Package jscp provides functions for copying data to JavaScript
package jscp

import (
	"syscall/js"
	"unsafe"
)

// BytesToJS converts the given bytes to a js Uint8ClampedArray
// by using the global wasm memory bytes. This avoids the
// copying present in [js.CopyBytesToJS].
func BytesToJS(b []byte) js.Value {
	ptr := uintptr(unsafe.Pointer(&b[0]))
	// We directly pass the offset and length to the constructor to avoid calling subarray or slice,
	// thereby improving performance and safety (this fixes a detached array buffer crash).
	return js.Global().Get("Uint8ClampedArray").New(js.Global().Get("wasm").Get("instance").Get("exports").Get("mem").Get("buffer"), ptr, len(b))
}
