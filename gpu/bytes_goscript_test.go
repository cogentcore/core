// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package gpu

import (
	"encoding/binary"
	"math"
	"testing"
)

type bytesTestVertex struct {
	Pos   [2]float32
	Index uint32
	_     uint32
}

func appendGeneric[E any](s []E, v ...E) []E {
	return append(s, v...)
}

func TestSliceBytesShared(t *testing.T) {
	// The bytes of a float32 slice are its memory, so writes on either
	// side are seen by the other.
	f := []float32{1, -2.5, 0.25}
	if _, ok := sliceBytes(f); !ok {
		t.Skip("this goscript holds numeric slices in plain arrays")
	}
	b := sliceToBytes(f)
	if len(b) != 12 {
		t.Fatalf("len = %d, want 12", len(b))
	}
	if got := math.Float32frombits(binary.LittleEndian.Uint32(b[4:])); got != -2.5 {
		t.Errorf("element 1 = %g, want -2.5", got)
	}
	f[2] = 8
	if got := math.Float32frombits(binary.LittleEndian.Uint32(b[8:])); got != 8 {
		t.Errorf("after Go write, element 2 = %g, want 8", got)
	}
	binary.LittleEndian.PutUint32(b, math.Float32bits(3))
	if f[0] != 3 {
		t.Errorf("after byte write, f[0] = %g, want 3", f[0])
	}

	// A window views only its own elements.
	u := []uint32{1, 2, 3, 4}
	w := sliceToBytes(u[1:3])
	if len(w) != 8 || binary.LittleEndian.Uint32(w) != 2 {
		t.Errorf("window bytes = %v", w)
	}
	if _, ok := sliceBytes(u); !ok {
		t.Errorf("uint32 slice is not viewed")
	}
}

func TestSliceBytesFallback(t *testing.T) {
	// A slice of structs is encoded with its layout, padding included.
	v := []bytesTestVertex{{Pos: [2]float32{1, 2}, Index: 7}}
	if _, ok := sliceBytes(v); ok {
		t.Errorf("struct slice is viewed")
	}
	b := sliceToBytes(v)
	if len(b) != 16 || binary.LittleEndian.Uint32(b[8:]) != 7 ||
		math.Float32frombits(binary.LittleEndian.Uint32(b[4:])) != 2 {
		t.Errorf("struct bytes = %v", b)
	}

	// A float32 slice that generic code appends to nil is held in a plain
	// array, which is encoded instead.
	g := appendGeneric[float32](nil, 0, 0.5)
	gb := sliceToBytes(g)
	if len(gb) != 8 || math.Float32frombits(binary.LittleEndian.Uint32(gb[4:])) != 0.5 {
		t.Errorf("generic bytes = %v", gb)
	}

	// Decoding fills a slice of structs.
	out := make([]bytesTestVertex, 1)
	if err := bytesToSlice(b, out); err != nil || out[0] != v[0] {
		t.Errorf("decoded %v, %v", out, err)
	}
}

// readBytesInto copies bytes read from a GPU buffer into dest, as
// readToSlice does.
func readBytesInto[E any](dest []E, src []byte) {
	if b, ok := sliceBytes(dest); ok {
		copy(b, src)
		return
	}
	bytesToSlice(src, dest)
}
