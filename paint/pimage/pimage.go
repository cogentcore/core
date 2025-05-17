// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pimage

//go:generate core generate

import (
	"image"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/math32"
	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// Cmds are possible commands to perform for [Params].
type Cmds int32 //enums:enum

const (
	// Draw Source image using draw.Draw equivalent function,
	// without any transformation. If Mask is non-nil it is used.
	Draw Cmds = iota

	// Draw Source image with transform. If Mask is non-nil, it is used.
	Transform

	// blurs the Rect region with the given blur radius.
	// The blur radius passed to this function is the actual Gaussian
	// standard deviation (σ).
	Blur

	// Sets pixel from Source image at Pos
	SetPixel
)

// Params for image operations. This is a Render Item.
type Params struct {
	// Command to perform.
	Cmd Cmds

	// Rect is the rectangle to draw into. This is the bounds for Transform source.
	// If empty, the entire destination image Bounds() are used.
	Rect image.Rectangle

	// SourcePos is the position for the source image in Draw,
	// and the location for SetPixel.
	SourcePos image.Point

	// Draw operation: Src or Over
	Op draw.Op

	// Source to draw.
	Source image.Image

	// Mask, used if non-nil.
	Mask image.Image

	// MaskPos is the position for the mask
	MaskPos image.Point

	// Transform for image transform.
	Transform math32.Matrix2

	// BlurRadius is the Gaussian standard deviation for Blur function
	BlurRadius float32
}

func (pr *Params) IsRenderItem() {}

// NewClear returns a new Clear that renders entire image with given source image.
func NewClear(src image.Image, sp image.Point, op draw.Op) *Params {
	pr := &Params{Cmd: Draw, Rect: image.Rectangle{}, Source: imagex.WrapJS(src), SourcePos: sp, Op: op}
	return pr
}

// NewDraw returns a new Draw operation with given parameters.
// Does nothing if rect is empty.
func NewDraw(rect image.Rectangle, src image.Image, sp image.Point, op draw.Op) *Params {
	if rect == (image.Rectangle{}) {
		return nil
	}
	pr := &Params{Cmd: Draw, Rect: rect, Source: imagex.WrapJS(src), SourcePos: sp, Op: op}
	return pr
}

// NewDrawMask returns a new DrawMask operation with given parameters.
// Does nothing if rect is empty.
func NewDrawMask(rect image.Rectangle, src image.Image, sp image.Point, op draw.Op, mask image.Image, mp image.Point) *Params {
	if rect == (image.Rectangle{}) {
		return nil
	}
	pr := &Params{Cmd: Draw, Rect: rect, Source: imagex.WrapJS(src), SourcePos: sp, Op: op, Mask: imagex.WrapJS(mask), MaskPos: mp}
	return pr
}

// NewTransform returns a new Transform operation with given parameters.
// Does nothing if rect is empty.
func NewTransform(m math32.Matrix2, rect image.Rectangle, src image.Image, op draw.Op) *Params {
	if rect == (image.Rectangle{}) {
		return nil
	}
	pr := &Params{Cmd: Transform, Transform: m, Rect: rect, Source: imagex.WrapJS(src), Op: op}
	return pr
}

// NewTransformMask returns a new Transform Mask operation with given parameters.
// Does nothing if rect is empty.
func NewTransformMask(m math32.Matrix2, rect image.Rectangle, src image.Image, op draw.Op, mask image.Image, mp image.Point) *Params {
	if rect == (image.Rectangle{}) {
		return nil
	}
	pr := &Params{Cmd: Transform, Transform: m, Rect: rect, Source: imagex.WrapJS(src), Op: op, Mask: imagex.WrapJS(mask), MaskPos: mp}
	return pr
}

// NewBlur returns a new Blur operation with given parameters.
// Does nothing if rect is empty.
func NewBlur(rect image.Rectangle, blurRadius float32) *Params {
	if rect == (image.Rectangle{}) {
		return nil
	}
	pr := &Params{Cmd: Blur, Rect: rect, BlurRadius: blurRadius}
	return pr
}

// NewSetPixel returns a new SetPixel operation with given parameters.
func NewSetPixel(at image.Point, clr image.Image) *Params {
	pr := &Params{Cmd: SetPixel, SourcePos: at, Source: clr}
	return pr
}

// Render performs the image operation on given destination image.
func (pr *Params) Render(dest *image.RGBA) {
	switch pr.Cmd {
	case Draw:
		if pr.Rect == (image.Rectangle{}) {
			pr.Rect = dest.Bounds()
		}
		if pr.Mask != nil {
			draw.DrawMask(dest, pr.Rect, imagex.Unwrap(pr.Source), pr.SourcePos, imagex.Unwrap(pr.Mask), pr.MaskPos, pr.Op)
		} else {
			if pr.Source == nil {
				return
			}
			draw.Draw(dest, pr.Rect, imagex.Unwrap(pr.Source), pr.SourcePos, pr.Op)
		}
	case Transform:
		m := pr.Transform
		s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
		tdraw := draw.BiLinear
		if pr.Mask != nil {
			tdraw.Transform(dest, s2d, imagex.Unwrap(pr.Source), pr.Rect, pr.Op, &draw.Options{
				DstMask:  imagex.Unwrap(pr.Mask),
				DstMaskP: pr.MaskPos,
			})
		} else {
			tdraw.Transform(dest, s2d, imagex.Unwrap(pr.Source), pr.Rect, pr.Op, nil)
		}
	case Blur:
		sub := dest.SubImage(pr.Rect)
		sub = GaussianBlur(sub, float64(pr.BlurRadius))
		draw.Draw(dest, pr.Rect, sub, pr.Rect.Min, draw.Src)
	case SetPixel:
		x := pr.SourcePos.X
		y := pr.SourcePos.Y
		dest.Set(x, y, imagex.Unwrap(pr.Source).At(x, y))
	}
}
