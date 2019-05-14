// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd
// +build !3d

package x11driver

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"
	"unsafe"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"github.com/goki/gi/oswin/driver/internal/swizzle"
)

type imageImpl struct {
	app *appImpl

	addr unsafe.Pointer
	buf  []byte
	rgba image.RGBA
	size image.Point
	xs   shm.Seg

	mu        sync.Mutex
	nUpload   uint32
	released  bool
	cleanedUp bool
}

func (b *imageImpl) degenerate() bool        { return b.size.X == 0 || b.size.Y == 0 }
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
		panic("x11driver: invalid Image.RGBA modification")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.released {
		panic("x11driver: Image.Upload called after Image.Release")
	}
	if b.nUpload == 0 {
		swizzle.BGRA(b.buf)
	}
	b.nUpload++
}

func (b *imageImpl) postUpload() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nUpload--
	if b.nUpload != 0 {
		return
	}

	if b.released {
		go b.cleanUp()
	} else {
		swizzle.BGRA(b.buf)
	}
}

func (b *imageImpl) Release() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.released && b.nUpload == 0 {
		go b.cleanUp()
	}
	b.released = true
}

func (b *imageImpl) cleanUp() {
	b.mu.Lock()
	if b.cleanedUp {
		b.mu.Unlock()
		panic("x11driver: Image clean-up occurred twice")
	}
	b.cleanedUp = true
	b.mu.Unlock()

	b.app.mu.Lock()
	delete(b.app.images, b.xs)
	b.app.mu.Unlock()

	if b.degenerate() {
		return
	}
	shm.Detach(b.app.xc, b.xs)
	if err := shmClose(b.addr); err != nil {
		log.Printf("x11driver: shmClose: %v", err)
	}
}

func (b *imageImpl) upload(xd xproto.Drawable, xg xproto.Gcontext, depth uint8, dp image.Point, sr image.Rectangle) {
	originalSRMin := sr.Min
	sr = sr.Intersect(b.Bounds())
	if sr.Empty() {
		return
	}
	dp = dp.Add(sr.Min.Sub(originalSRMin))
	b.preUpload()

	b.app.mu.Lock()
	b.app.nPendingUploads++
	b.app.mu.Unlock()

	cookie := shm.PutImage(
		b.app.xc, xd, xg,
		uint16(b.size.X), uint16(b.size.Y), // TotalWidth, TotalHeight,
		uint16(sr.Min.X), uint16(sr.Min.Y), // SrcX, SrcY,
		uint16(sr.Dx()), uint16(sr.Dy()), // SrcWidth, SrcHeight,
		int16(dp.X), int16(dp.Y), // DstX, DstY,
		depth, xproto.ImageFormatZPixmap,
		1, b.xs, 0, // 1 means send a completion event, 0 means a zero offset.
	)

	completion := make(chan struct{})

	b.app.mu.Lock()
	b.app.uploads[cookie.Sequence] = completion
	b.app.nPendingUploads--
	b.app.handleCompletions()
	b.app.mu.Unlock()

	<-completion

	b.postUpload()
}

func fill(xc *xgb.Conn, xp render.Picture, dr image.Rectangle, src color.Color, op draw.Op) {
	r, g, b, a := src.RGBA()
	c := render.Color{
		Red:   uint16(r),
		Green: uint16(g),
		Blue:  uint16(b),
		Alpha: uint16(a),
	}
	x, y := dr.Min.X, dr.Min.Y
	if x < -0x8000 || 0x7fff < x || y < -0x8000 || 0x7fff < y {
		return
	}
	dx, dy := dr.Dx(), dr.Dy()
	if dx < 0 || 0xffff < dx || dy < 0 || 0xffff < dy {
		return
	}
	render.FillRectangles(xc, renderOp(op), xp, c, []xproto.Rectangle{{
		X:      int16(x),
		Y:      int16(y),
		Width:  uint16(dx),
		Height: uint16(dy),
	}})
}
