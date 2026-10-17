// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package gpu

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"

	"cogentcore.org/core/math32"
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

// bytesTestPadded has padding before B, which encoding/binary leaves out.
type bytesTestPadded struct {
	A uint8
	B float32
}

// reflectBytes encodes the given slice with the reflect encoder.
func reflectBytes[E any](src []E) []byte {
	sv := reflect.ValueOf(src)
	size := int(sv.Type().Elem().Size())
	b := make([]byte, size*len(src))
	for i := range len(src) {
		encodeValue(b[i*size:], sv.Index(i))
	}
	return b
}

func TestSliceBytesStructs(t *testing.T) {
	// Structs of nested structs and arrays encode as the reflect encoder
	// does, whether or not encoding/binary handles their slices.
	obs := make([]benchObject, 3)
	for i := range obs {
		obs[i].Color = math32.Vec4(float32(i), 0.5, -1, 1)
		obs[i].Offset.Y = float32(i) / 3
		obs[i].Matrix.SetIdentity()
		obs[i].WorldMatrix[14] = float32(-i)
	}
	b := sliceToBytes(obs)
	if !bytes.Equal(b, reflectBytes(obs)) {
		t.Errorf("object bytes differ from the reflect encoding")
	}
	t.Logf("encoding/binary layout: %v", elementInfoOf[benchObject]().binarySize == 192)

	// Decoding gives the structs back.
	out := make([]benchObject, 3)
	if err := bytesToSlice(b, out); err != nil || !reflect.DeepEqual(out, obs) {
		t.Errorf("decoded %v, %v", out, err)
	}

	// A struct with padding keeps its layout through the reflect encoder.
	if elementInfoOf[bytesTestPadded]().binarySize != 0 {
		t.Errorf("padded struct has the encoding/binary layout")
	}
	p := []bytesTestPadded{{A: 3, B: 1.5}}
	pb := sliceToBytes(p)
	if len(pb) != 8 || pb[0] != 3 || math.Float32frombits(binary.LittleEndian.Uint32(pb[4:])) != 1.5 {
		t.Errorf("padded bytes = %v", pb)
	}
}
