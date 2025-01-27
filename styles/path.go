// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
)

// Path provides the styling parameters for path-level rendering:
// Stroke and Fill.
type Path struct { //types:add
	// Off indicates that node and everything below it are off, non-rendering.
	Off bool

	// Stroke (line drawing) parameters.
	Stroke Stroke

	// Fill (region filling) parameters.
	Fill Fill

	// Transform has our additions to the transform stack.
	Transform math32.Matrix2
}

func (pc *Path) Defaults() {
	pc.Off = false
	pc.Stroke.Defaults()
	pc.Fill.Defaults()
	pc.Transform = math32.Identity2()
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (pc *Path) ToDotsImpl(uc *units.Context) {
	pc.Stroke.ToDots(uc)
	pc.Fill.ToDots(uc)
}

type FillRules int32 //enums:enum -trim-prefix FillRule -transform lower

const (
	FillRuleNonZero FillRules = iota
	FillRuleEvenOdd
)

// VectorEffects contains special effects for rendering
type VectorEffects int32 //enums:enum -trim-prefix VectorEffect -transform kebab

const (
	VectorEffectNone VectorEffects = iota

	// VectorEffectNonScalingStroke means that the stroke width is not affected by
	// transform properties
	VectorEffectNonScalingStroke
)

// IMPORTANT: any changes here must be updated in StyleFillFuncs

// Fill contains all the properties for filling a region.
type Fill struct {

	// fill color image specification; filling is off if nil
	Color image.Image

	// global alpha opacity / transparency factor between 0 and 1
	Opacity float32

	// rule for how to fill more complex shapes with crossing lines
	Rule FillRules
}

// Defaults initializes default values for paint fill
func (pf *Fill) Defaults() {
	pf.Color = colors.Uniform(color.Black)
	pf.Rule = FillRuleNonZero
	pf.Opacity = 1.0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Fill) ToDots(uc *units.Context) {
}

//////// Stroke

// LineCaps specifies end-cap of a line: stroke-linecap property in SVG
type LineCaps int32 //enums:enum -trim-prefix LineCap -transform kebab

const (
	// LineCapButt indicates to draw no line caps; it draws a
	// line with the length of the specified length.
	LineCapButt LineCaps = iota

	// LineCapRound indicates to draw a semicircle on each line
	// end with a diameter of the stroke width.
	LineCapRound

	// LineCapSquare indicates to draw a rectangle on each line end
	// with a height of the stroke width and a width of half of the
	// stroke width.
	LineCapSquare
)

// the way in which lines are joined together: stroke-linejoin property in SVG
type LineJoins int32 //enums:enum -trim-prefix LineJoin -transform kebab

const (
	LineJoinMiter LineJoins = iota
	LineJoinMiterClip
	LineJoinRound
	LineJoinBevel
	LineJoinArcs
	// rasterx extension
	LineJoinArcsClip
)

// IMPORTANT: any changes here must be updated below in StyleStrokeFuncs

// Stroke contains all the properties for painting a line
type Stroke struct {

	// stroke color image specification; stroking is off if nil
	Color image.Image

	// global alpha opacity / transparency factor between 0 and 1
	Opacity float32

	// line width
	Width units.Value

	// minimum line width used for rendering -- if width is > 0, then this is the smallest line width -- this value is NOT subject to transforms so is in absolute dot values, and is ignored if vector-effects non-scaling-stroke is used -- this is an extension of the SVG / CSS standard
	MinWidth units.Value

	// Dashes are the dashes of the stroke. Each pair of values specifies
	// the amount to paint and then the amount to skip.
	Dashes []float32

	// how to draw the end cap of lines
	Cap LineCaps

	// how to join line segments
	Join LineJoins

	// limit of how far to miter -- must be 1 or larger
	MiterLimit float32 `min:"1"`
}

// Defaults initializes default values for paint stroke
func (ss *Stroke) Defaults() {
	// stroking is off by default in svg
	ss.Color = nil
	ss.Width.Dp(1)
	ss.MinWidth.Dot(.5)
	ss.Cap = LineCapButt
	ss.Join = LineJoinMiter // Miter not yet supported, but that is the default -- falls back on bevel
	ss.MiterLimit = 10.0
	ss.Opacity = 1.0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ss *Stroke) ToDots(uc *units.Context) {
	ss.Width.ToDots(uc)
	ss.MinWidth.ToDots(uc)
}

// ApplyBorderStyle applies the given border style to the stroke style.
func (ss *Stroke) ApplyBorderStyle(bs BorderStyles) {
	switch bs {
	case BorderNone:
		ss.Color = nil
	case BorderDotted:
		ss.Dashes = []float32{0, 12}
		ss.Cap = LineCapRound
	case BorderDashed:
		ss.Dashes = []float32{8, 6}
	}
}
