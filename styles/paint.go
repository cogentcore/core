// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// Paint provides the styling parameters for SVG-style rendering,
// including the Path stroke and fill properties, and font and text
// properties.
type Paint struct { //types:add
	Path

	// Font selects font properties.
	Font rich.Style

	// Text has the text styling settings.
	Text text.Style

	// ClipPath is a clipping path for this item.
	ClipPath ppath.Path

	// Mask is a rendered image of the mask for this item.
	Mask image.Image
}

func NewPaint() *Paint {
	pc := &Paint{}
	pc.Defaults()
	return pc
}

// NewPaintWithContext returns a new Paint style with [units.Context]
// initialized from given. Pass the Styles context for example.
func NewPaintWithContext(uc *units.Context) *Paint {
	pc := NewPaint()
	pc.UnitContext = *uc
	return pc
}

func (pc *Paint) Defaults() {
	pc.Path.Defaults()
	pc.Font.Defaults()
	pc.Text.Defaults()
}

// CopyStyleFrom copies styles from another paint
func (pc *Paint) CopyStyleFrom(cp *Paint) {
	pc.Path.CopyStyleFrom(&cp.Path)
	pc.Font = cp.Font
	pc.Text = cp.Text
}

// InheritFields from parent
func (pc *Paint) InheritFields(parent *Paint) {
	pc.Font.InheritFields(&parent.Font)
	pc.Text.InheritFields(&parent.Text)
}

// SetProperties sets paint values based on given property map (name: value
// pairs), inheriting elements as appropriate from parent, and also having a
// default style for the "initial" setting
func (pc *Paint) SetProperties(parent *Paint, properties map[string]any, ctxt colors.Context) {
	if !pc.StyleSet && parent != nil { // first time
		pc.InheritFields(parent)
	}
	pc.fromProperties(parent, properties, ctxt)

	pc.PropertiesNil = (len(properties) == 0)
	pc.StyleSet = true
}

// GetProperties gets properties values from current style settings,
// for any non-default settings, setting name-value pairs in given map,
// which must be non-nil.
func (pc *Paint) GetProperties(properties map[string]any) {
	pc.toProperties(properties)
}

func (pc *Paint) FromStyle(st *Style) {
	pc.UnitContext = st.UnitContext
	st.SetRichText(&pc.Font, &pc.Text)
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDotsImpl(uc *units.Context) {
	pc.Path.ToDotsImpl(uc)
	// pc.Font.ToDots(uc)
	pc.Text.ToDots(uc)
}

// SetUnitContextExt sets the unit context for external usage of paint
// outside of Core Scene context, based on overall size of painting canvas.
// caches everything out in terms of raw pixel dots for rendering
// call at start of render.
func (pc *Paint) SetUnitContextExt(size image.Point) {
	if pc.UnitContext.DPI == 0 {
		pc.UnitContext.Defaults()
	}
	// TODO: maybe should have different values for these sizes?
	pc.UnitContext.SetSizes(float32(size.X), float32(size.Y), float32(size.X), float32(size.Y), float32(size.X), float32(size.Y))
	// todo: need a shaper here to get SetUnitContext call
	// pc.Font.SetUnitContext(&pc.UnitContext)
	pc.ToDotsImpl(&pc.UnitContext)
	pc.dotsSet = true
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDots() {
	if !(pc.dotsSet && pc.UnitContext == pc.lastUnCtxt && pc.PropertiesNil) {
		pc.ToDotsImpl(&pc.UnitContext)
		pc.dotsSet = true
		pc.lastUnCtxt = pc.UnitContext
	}
}
