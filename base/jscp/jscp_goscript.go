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
