// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import "image"

type imageImpl struct {
	// buf should always be equal to (i.e. the same ptr, len, cap as) rgba.Pix.
	// It is a separate, redundant field in order to detect modifications to
	// the rgba field that are invalid as per the oswin.Image documentation.
	buf  []byte
	rgba image.RGBA
	size image.Point
}

func (b *imageImpl) Release()                {}
func (b *imageImpl) Size() image.Point       { return b.size }
func (b *imageImpl) Bounds() image.Rectangle { return image.Rectangle{Max: b.size} }
func (b *imageImpl) RGBA() *image.RGBA       { return &b.rgba }

func (b *imageImpl) preUpload() {
	// Check that the program hasn't tried to modify the rgba field via the
	// pointer returned by the imageImpl.RGBA method. This check doesn't catch
	// 100% of all cases; it simply tries to detect some invalid uses of a
	// oswin.Image such as:
	//	*image.RGBA() = anotherImageRGBA
	if len(b.buf) != 0 && len(b.rgba.Pix) != 0 && &b.buf[0] != &b.rgba.Pix[0] {
		panic("macdriver: invalid Image.RGBA modification")
	}
}
