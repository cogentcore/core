// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(js && goscript)

package gpu

import "github.com/oliverbestmann/webgpu/wgpu"

// sliceToBytes returns the bytes of the given slice, sharing its memory.
func sliceToBytes[E any](src []E) []byte {
	return wgpu.ToBytes(src)
}

// readToSlice copies the read buffer data of the given value
// directly into the memory of the given slice.
func readToSlice[E any](vl *Value, dest []E) error {
	return vl.ReadToBytes(wgpu.ToBytes(dest))
}
