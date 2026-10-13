// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package jscp

import "syscall/js"

// BytesToJS converts the given bytes to a js Uint8ClampedArray.
// Under goscript there is no wasm memory to view, so the bytes
// are copied with [js.CopyBytesToJS].
func BytesToJS(b []byte) js.Value {
	array := js.Global().Get("Uint8Array").New(len(b))
	js.CopyBytesToJS(array, b)
	return js.Global().Get("Uint8ClampedArray").New(array.Get("buffer"), array.Get("byteOffset"), len(b))
}
