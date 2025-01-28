// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/scanx:
// Copyright 2018 by the scanx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package scan

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/colors"
)

const (
	m         = 1<<16 - 1
	mp        = 0x100 * m
	pa uint32 = 0x101
	q  uint32 = 0xFF00
)

// ImgSpanner is a Spanner that draws Spans onto an [*image.RGBA] image.
// It uses either a color function as a the color source, or a fgColor
// if colFunc is nil.
type ImgSpanner struct {
	BaseSpanner
	Pix        []uint8
	Stride     int
	ColorImage image.Image
}

// LinkListSpanner is a Spanner that draws Spans onto a draw.Image
// interface satisfying struct but it is optimized for [*image.RGBA].
// It uses a solid Color only for fg and bg and does not support a color function
// used by gradients. Spans are accumulated into a set of linked lists, one for
// every horizontal line in the image. After the spans for the image are accumulated,
// use the DrawToImage function to write the spans to an image.
type LinkListSpanner struct {
	BaseSpanner
	Spans   []SpanCell
	BgColor color.RGBA
	LastY   int
	LastP   int
}

// SpanCell represents a span cell.
type SpanCell struct {
	X0   int
	X1   int
	Next int
	Clr  color.RGBA
}

// BaseSpanner contains base spanner information extended by [ImgSpanner] and [LinkListSpanner].
type BaseSpanner struct {
	// drawing is done with Bounds.Min as the origin
	Bounds image.Rectangle

	// Op is how pixels are overlayed
	Op      draw.Op
	FgColor color.RGBA
}

// Clear clears the current spans
func (x *LinkListSpanner) Clear() {
	x.LastY, x.LastP = 0, 0
	x.Spans = x.Spans[0:0]
	width := x.Bounds.Dy()
	for i := 0; i < width; i++ {
		// The first cells are indexed according to the y values
		// to create y separate linked lists corresponding to the
		// image y length. Since index 0 is used by the first of these sentinel cells
		// 0 can and is used for the end of list value by the spanner linked list.
		x.Spans = append(x.Spans, SpanCell{})
	}
}

func (x *LinkListSpanner) SpansToImage(img draw.Image) {
	for y := 0; y < x.Bounds.Dy(); y++ {
		p := x.Spans[y].Next
		for p != 0 {
			spCell := x.Spans[p]
			clr := spCell.Clr
			x0, x1 := spCell.X0, spCell.X1
			for x := x0; x < x1; x++ {
				img.Set(y, x, clr)
			}
			p = spCell.Next
		}
	}
}

func (x *LinkListSpanner) SpansToPix(pix []uint8, stride int) {
	for y := 0; y < x.Bounds.Dy(); y++ {
		yo := y * stride
		p := x.Spans[y].Next
		for p != 0 {
			spCell := x.Spans[p]
			i0 := yo + spCell.X0*4
			i1 := i0 + (spCell.X1-spCell.X0)*4
			r, g, b, a := spCell.Clr.R, spCell.Clr.G, spCell.Clr.B, spCell.Clr.A
			for i := i0; i < i1; i += 4 {
				pix[i+0] = r
				pix[i+1] = g
				pix[i+2] = b
				pix[i+3] = a
			}
			p = spCell.Next
		}
	}
}

// DrawToImage draws the accumulated y spans onto the img
func (x *LinkListSpanner) DrawToImage(img image.Image) {
	switch img := img.(type) {
	case *image.RGBA:
		x.SpansToPix(img.Pix, img.Stride)
	case draw.Image:
		x.SpansToImage(img)
	}
}

// SetBounds sets the spanner boundaries
func (x *LinkListSpanner) SetBounds(bounds image.Rectangle) {
	x.Bounds = bounds
	x.Clear()
}

func (x *LinkListSpanner) BlendColor(under color.RGBA, ma uint32) color.RGBA {
	if ma == 0 {
		return under
	}
	rma := uint32(x.FgColor.R) * ma
	gma := uint32(x.FgColor.G) * ma
	bma := uint32(x.FgColor.B) * ma
	ama := uint32(x.FgColor.A) * ma
	if x.Op != draw.Over || under.A == 0 || ama == m*0xFF {
		return color.RGBA{
			uint8(rma / q),
			uint8(gma / q),
			uint8(bma / q),
			uint8(ama / q)}
	}
	a := m - (ama / (m >> 8))
	cc := color.RGBA{
		uint8((uint32(under.R)*a + rma) / q),
		uint8((uint32(under.G)*a + gma) / q),
		uint8((uint32(under.B)*a + bma) / q),
		uint8((uint32(under.A)*a + ama) / q)}
	return cc
}

func (x *LinkListSpanner) AddLink(x0, x1, next, pp int, underColor color.RGBA, alpha uint32) (p int) {
	clr := x.BlendColor(underColor, alpha)
	if pp >= x.Bounds.Dy() && x.Spans[pp].X1 >= x0 && ((clr.A == 0 && x.Spans[pp].Clr.A == 0) || clr == x.Spans[pp].Clr) {
		// Just extend the prev span; a new one is not required
		x.Spans[pp].X1 = x1
		return pp
	}
	x.Spans = append(x.Spans, SpanCell{X0: x0, X1: x1, Next: next, Clr: clr})
	p = len(x.Spans) - 1
	x.Spans[pp].Next = p
	return
}

// GetSpanFunc returns the function that consumes a span described by the parameters.
func (x *LinkListSpanner) GetSpanFunc() SpanFunc {
	x.LastY = -1 // x within a y list may no longer be ordered, so this ensures a reset.
	return x.SpanOver
}

// SpanOver adds the span into an array of linked lists of spans using the fgColor and Porter-Duff composition
// ma is the accumulated alpha coverage. This function also assumes usage sorted x inputs for each y and so if
// inputs for x in y are not monotonically increasing, then lastY should be set to -1.
func (x *LinkListSpanner) SpanOver(yi, xi0, xi1 int, ma uint32) {
	if yi != x.LastY { // If the y place has changed, start at the list beginning
		x.LastP = yi
		x.LastY = yi
	}
	// since spans are sorted, we can start from x.lastP
	pp := x.LastP
	p := x.Spans[pp].Next
	for p != 0 && xi0 < xi1 {
		sp := x.Spans[p]
		if sp.X1 <= xi0 { //sp is before new span
			pp = p
			p = sp.Next
			continue
		}
		if sp.X0 >= xi1 { //new span is before sp
			x.LastP = x.AddLink(xi0, xi1, p, pp, x.BgColor, ma)
			return
		}
		// left span
		if xi0 < sp.X0 {
			pp = x.AddLink(xi0, sp.X0, p, pp, x.BgColor, ma)
			xi0 = sp.X0
		} else if xi0 > sp.X0 {
			pp = x.AddLink(sp.X0, xi0, p, pp, sp.Clr, 0)
		}

		clr := x.BlendColor(sp.Clr, ma)
		sameClrs := pp >= x.Bounds.Dy() && ((clr.A == 0 && x.Spans[pp].Clr.A == 0) || clr == x.Spans[pp].Clr)
		if xi1 < sp.X1 { // span does not go beyond sp
			// merge with left span
			if x.Spans[pp].X1 >= xi0 && sameClrs {
				x.Spans[pp].X1 = xi1
				x.Spans[pp].Next = sp.Next
				// Suffices not to advance lastP ?!? Testing says NO!
				x.LastP = yi // We need to go back, so let's just go to start of the list next time
				p = pp
			} else {
				// middle span; replaces sp
				x.Spans[p] = SpanCell{X0: xi0, X1: xi1, Next: sp.Next, Clr: clr}
				x.LastP = pp
			}
			x.AddLink(xi1, sp.X1, sp.Next, p, sp.Clr, 0)
			return
		}
		if x.Spans[pp].X1 >= xi0 && sameClrs { // Extend and merge with previous
			x.Spans[pp].X1 = sp.X1
			x.Spans[pp].Next = sp.Next
			p = sp.Next // clip out the current span from the list
			xi0 = sp.X1 // set remaining to start for next loop
			continue
		}
		// Set current span to start of new span and combined color
		x.Spans[p] = SpanCell{X0: xi0, X1: sp.X1, Next: sp.Next, Clr: clr}
		xi0 = sp.X1 // any remaining span starts at sp.x1
		pp = p
		p = sp.Next
	}
	x.LastP = pp
	if xi0 < xi1 { // add any remaining span to the end of the chain
		x.AddLink(xi0, xi1, 0, pp, x.BgColor, ma)
	}
}

// SetBgColor sets the background color for blending to the first pixel of the given color
func (x *LinkListSpanner) SetBgColor(c image.Image) {
	x.BgColor = colors.AsRGBA(colors.ToUniform(c))
}

// SetColor sets the color of x to the first pixel of the given color
func (x *LinkListSpanner) SetColor(c image.Image) {
	x.FgColor = colors.AsRGBA(colors.ToUniform(c))
}

// NewImgSpanner returns an ImgSpanner set to draw to the given [*image.RGBA].
func NewImgSpanner(img *image.RGBA) (x *ImgSpanner) {
	x = &ImgSpanner{}
	x.SetImage(img)
	return
}

// SetImage set the [*image.RGBA] that the ImgSpanner will draw onto.
func (x *ImgSpanner) SetImage(img *image.RGBA) {
	x.Pix = img.Pix
	x.Stride = img.Stride
	x.Bounds = img.Bounds()
}

// SetColor sets the color of x to the given color image
func (x *ImgSpanner) SetColor(c image.Image) {
	if u, ok := c.(*image.Uniform); ok {
		x.FgColor = colors.AsRGBA(u.C)
		x.ColorImage = nil
		return
	}
	x.FgColor = color.RGBA{}
	x.ColorImage = c
}

// GetSpanFunc returns the function that consumes a span described by the parameters.
// The next four func declarations are all slightly different
// but in order to reduce code redundancy, this method is used
// to dispatch the function in the draw method.
func (x *ImgSpanner) GetSpanFunc() SpanFunc {
	var (
		useColorFunc = x.ColorImage != nil
		drawOver     = x.Op == draw.Over
	)
	switch {
	case useColorFunc && drawOver:
		return x.SpanColorFunc
	case useColorFunc && !drawOver:
		return x.SpanColorFuncR
	case !useColorFunc && !drawOver:
		return x.SpanFgColorR
	default:
		return x.SpanFgColor
	}
}

// SpanColorFuncR draw the span using a colorFunc and replaces the previous values.
func (x *ImgSpanner) SpanColorFuncR(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.Stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cx := xi0
	for i := i0; i < i1; i += 4 {
		rcr, rcg, rcb, rca := x.ColorImage.At(cx, yi).RGBA()
		cx++
		x.Pix[i+0] = uint8(rcr * ma / mp)
		x.Pix[i+1] = uint8(rcg * ma / mp)
		x.Pix[i+2] = uint8(rcb * ma / mp)
		x.Pix[i+3] = uint8(rca * ma / mp)
	}
}

// SpanFgColorR draws the span with the fore ground color and replaces the previous values.
func (x *ImgSpanner) SpanFgColorR(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.Stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cr, cg, cb, ca := x.FgColor.RGBA()
	rma := uint8(cr * ma / mp)
	gma := uint8(cg * ma / mp)
	bma := uint8(cb * ma / mp)
	ama := uint8(ca * ma / mp)
	for i := i0; i < i1; i += 4 {
		x.Pix[i+0] = rma
		x.Pix[i+1] = gma
		x.Pix[i+2] = bma
		x.Pix[i+3] = ama
	}
}

// SpanColorFunc draws the span using a colorFunc and the  Porter-Duff composition operator.
func (x *ImgSpanner) SpanColorFunc(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.Stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	cx := xi0

	for i := i0; i < i1; i += 4 {
		// uses the Porter-Duff composition operator.
		rcr, rcg, rcb, rca := x.ColorImage.At(cx, yi).RGBA()
		cx++
		a := (m - (rca * ma / m)) * pa
		dr := uint32(x.Pix[i+0])
		dg := uint32(x.Pix[i+1])
		db := uint32(x.Pix[i+2])
		da := uint32(x.Pix[i+3])
		x.Pix[i+0] = uint8((dr*a + rcr*ma) / mp)
		x.Pix[i+1] = uint8((dg*a + rcg*ma) / mp)
		x.Pix[i+2] = uint8((db*a + rcb*ma) / mp)
		x.Pix[i+3] = uint8((da*a + rca*ma) / mp)
	}
}

// SpanFgColor draw the span using the fore ground color and the Porter-Duff composition operator.
func (x *ImgSpanner) SpanFgColor(yi, xi0, xi1 int, ma uint32) {
	i0 := (yi)*x.Stride + (xi0)*4
	i1 := i0 + (xi1-xi0)*4
	// uses the Porter-Duff composition operator.
	cr, cg, cb, ca := x.FgColor.RGBA()
	ama := ca * ma
	if ama == 0xFFFF*0xFFFF { // undercolor is ignored
		rmb := uint8(cr * ma / mp)
		gmb := uint8(cg * ma / mp)
		bmb := uint8(cb * ma / mp)
		amb := uint8(ama / mp)
		for i := i0; i < i1; i += 4 {
			x.Pix[i+0] = rmb
			x.Pix[i+1] = gmb
			x.Pix[i+2] = bmb
			x.Pix[i+3] = amb
		}
		return
	}
	rma := cr * ma
	gma := cg * ma
	bma := cb * ma
	a := (m - (ama / m)) * pa
	for i := i0; i < i1; i += 4 {
		x.Pix[i+0] = uint8((uint32(x.Pix[i+0])*a + rma) / mp)
		x.Pix[i+1] = uint8((uint32(x.Pix[i+1])*a + gma) / mp)
		x.Pix[i+2] = uint8((uint32(x.Pix[i+2])*a + bma) / mp)
		x.Pix[i+3] = uint8((uint32(x.Pix[i+3])*a + ama) / mp)
	}
}
