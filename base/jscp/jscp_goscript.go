// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package jscp

import "syscall/js"

// BytesToJS converts the given bytes to a js Uint8ClampedArray.
// Under goscript a []byte is already a js Uint8Array, so the result
// is a view of the same memory, without copying.
func BytesToJS(b []byte) js.Value {
	array := js.ValueOf(b)
	return js.Global().Get("Uint8ClampedArray").New(array.Get("buffer"), array.Get("byteOffset"), len(b))
}

// StableBytesToJS converts the given bytes to a js Uint8ClampedArray that
// stays valid after the call, for a js object that keeps it, such as an
// ImageData. Under goscript the memory of a []byte is never detached, so it
// is the same view as [BytesToJS]; on wasm it is a copy.
func StableBytesToJS(b []byte) js.Value {
	return BytesToJS(b)
}
