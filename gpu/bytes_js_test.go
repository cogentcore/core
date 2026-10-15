// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package gpu

import (
	"syscall/js"
	"testing"
)

// benchFloats is the number of float32 values of the upload and readback
// benchmarks: 4 MB of data.
const benchFloats = 1 << 20

// BenchmarkUploadFloats measures what writing a float32 slice to a GPU
// buffer costs on the Go side: its bytes, and their copy into a JavaScript
// Uint8Array, as Queue.WriteBuffer does.
func BenchmarkUploadFloats(b *testing.B) {
	f := make([]float32, benchFloats)
	for i := range f {
		f[i] = float32(i)
	}
	array := js.Global().Get("Uint8Array").New(4 * benchFloats)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		js.CopyBytesToJS(array, sliceToBytes(f))
	}
}

// BenchmarkReadbackFloats measures what reading a float32 slice back from a
// mapped GPU buffer costs on the Go side: the copy of the mapped range into
// Go bytes, as Buffer.GetMappedRange does, and into the slice.
func BenchmarkReadbackFloats(b *testing.B) {
	f := make([]float32, benchFloats)
	src := js.Global().Get("Uint8Array").New(4 * benchFloats)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapped := make([]byte, 4*benchFloats)
		js.CopyBytesToGo(mapped, src)
		readBytesInto(f, mapped)
	}
}

// benchStruct is a GPU struct of four float32 fields.
type benchStruct struct {
	A, B, C, D float32
}

// BenchmarkUploadStructs measures the bytes of 10000 structs, which GoScript
// encodes field by field, where native and wasm builds view their memory.
func BenchmarkUploadStructs(b *testing.B) {
	v := make([]benchStruct, 10000)
	for i := range v {
		v[i].A = float32(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sliceToBytes(v)
	}
}
