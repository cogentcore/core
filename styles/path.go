// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles/units"
)

// Path provides the styling parameters for path-level rendering:
// Stroke and Fill.
type Path struct { //types:add
	// Off indicates that node and everything below it are off, non-rendering.
	// This is auto-updated based on other settings.
	Off bool

	// Display is the user-settable flag that determines if this item
	// should be displayed.
	Display bool

	// Stroke (line drawing) parameters.
	Stroke Stroke `display:"add-fields"`

	// Fill (region filling) parameters.
	Fill Fill

	// Opacity is a global transparency alpha factor that applies to stroke and fill.
	Opacity float32

	// Transform has our additions to the transform stack.
	Transform math32.Matrix2 `display:"inline"`

	// VectorEffect has various rendering special effects settings.
	VectorEffect ppath.VectorEffects

	// UnitContext has parameters necessary for determining unit sizes.
	UnitContext units.Context `display:"-"`

	// StyleSet indicates if the styles already been set.
	StyleSet bool `display:"-"`

	PropertiesNil bool `display:"-"`
	dotsSet       bool
	lastUnCtxt    units.Context
}

func (pc *Path) Defaults() {
	pc.Off = false
	pc.Display = true
	pc.Stroke.Defaults()
	pc.Fill.Defaults()
	pc.Opacity = 1
	pc.Transform = math32.Identity2()
	pc.StyleSet = false
}

// CopyStyleFrom copies styles from another paint
func (pc *Path) CopyStyleFrom(cp *Path) {
	pc.Off = cp.Off
	pc.UnitContext = cp.UnitContext
	pc.Stroke = cp.Stroke
	pc.Fill = cp.Fill
	pc.VectorEffect = cp.VectorEffect
}

// SetProperties sets path values based on given property map (name: value
// pairs), inheriting elements as appropriate from parent, and also having a
// default style for the "initial" setting
func (pc *Path) SetProperties(parent *Path, properties map[string]any, ctxt colors.Context) {
	pc.fromProperties(parent, properties, ctxt)
	pc.PropertiesNil = (len(properties) == 0)
	pc.StyleSet = true
}

// GetProperties gets properties values from current style settings,
// for any non-default settings, setting name-value pairs in given map,
// which must be non-nil.
func (pc *Path) GetProperties(properties map[string]any) {
	pc.toProperties(properties)
}

func (pc *Path) FromStyle(st *Style) {
	pc.UnitContext = st.UnitContext
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (pc *Path) ToDotsImpl(uc *units.Context) {
	pc.Stroke.ToDots(uc)
	pc.Fill.ToDots(uc)
}

func (pc *Path) HasFill() bool {
	return !pc.Off && pc.Fill.Color != nil && pc.Fill.Opacity > 0
}

func (pc *Path) HasStroke() bool {
	return !pc.Off && pc.Stroke.Color != nil && pc.Stroke.Width.Dots > 0 && pc.Stroke.Opacity > 0
}

//////// Stroke and Fill Styles

// IMPORTANT: any changes here must be updated in StyleFillFuncs

// Fill contains all the properties for filling a region.
type Fill struct {

	// Color to use in filling; filling is off if nil.
	Color image.Image

	// Fill alpha opacity / transparency factor between 0 and 1.
	// This applies in addition to any alpha specified in the Color.
	Opacity float32

	// Rule for how to fill more complex shapes with crossing lines.
	Rule ppath.FillRules
}

// Defaults initializes default values for paint fill
func (pf *Fill) Defaults() {
	pf.Color = colors.Uniform(color.Black)
	pf.Rule = ppath.NonZero
	pf.Opacity = 1.0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Fill) ToDots(uc *units.Context) {
}

//////// Stroke

// IMPORTANT: any changes here must be updated below in StyleStrokeFuncs

// Stroke contains all the properties for painting a line.
type Stroke struct {

	// stroke color image specification; stroking is off if nil
	Color image.Image

	// global alpha opacity / transparency factor between 0 and 1
	Opacity float32

	// line width
	Width units.Value

	// Dashes are the dashes of the stroke. Each pair of values specifies
	// the amount to paint and then the amount to skip.
	Dashes []float32

	// DashOffset is the starting offset for the dashes.
	DashOffset float32

	// Cap specifies how to draw the end cap of stroked lines.
	Cap ppath.Caps

	// Join specifies how to join line segments.
	Join ppath.Joins

	// MiterLimit is the limit of how far to miter: must be 1 or larger.
	MiterLimit float32 `min:"1"`
}

// Defaults initializes default values for paint stroke
func (ss *Stroke) Defaults() {
	// stroking is off by default in svg
	ss.Color = nil
	ss.Width.Dp(1)
	ss.Cap = ppath.CapButt
	ss.Join = ppath.JoinMiter
	ss.MiterLimit = 10.0
	ss.Opacity = 1.0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (ss *Stroke) ToDots(uc *units.Context) {
	ss.Width.ToDots(uc)
}

// ApplyBorderStyle applies the given border style to the stroke style.
func (ss *Stroke) ApplyBorderStyle(bs BorderStyles) {
	switch bs {
	case BorderNone:
		ss.Color = nil
	case BorderDotted:
		ss.Dashes = []float32{0, 12}
		ss.Cap = ppath.CapRound
	case BorderDashed:
		ss.Dashes = []float32{8, 6}
	}
}
