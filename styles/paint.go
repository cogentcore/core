// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/paint/path"
	"cogentcore.org/core/styles/units"
)

// Paint provides the styling parameters for SVG-style rendering,
// including the Path stroke and fill properties, and font and text
// properties.
type Paint struct { //types:add
	Path

	// FontStyle selects font properties and also has a global opacity setting,
	// along with generic color, background-color settings, which can be copied
	// into stroke / fill as needed.
	FontStyle FontRender

	// TextStyle has the text styling settings.
	TextStyle Text

	//	ClipPath is a clipping path for this item.
	ClipPath path.Path

	//	Mask is a rendered image of the mask for this item.
	Mask image.Image
}

func NewPaint() *Paint {
	pc := &Paint{}
	pc.Defaults()
	return pc
}

func (pc *Paint) Defaults() {
	pc.Path.Defaults()
	pc.FontStyle.Defaults()
	pc.TextStyle.Defaults()
}

// CopyStyleFrom copies styles from another paint
func (pc *Paint) CopyStyleFrom(cp *Paint) {
	pc.Path.CopyStyleFrom(&cp.Path)
	pc.FontStyle = cp.FontStyle
	pc.TextStyle = cp.TextStyle
}

// InheritFields from parent
func (pc *Paint) InheritFields(parent *Paint) {
	pc.FontStyle.InheritFields(&parent.FontStyle)
	pc.TextStyle.InheritFields(&parent.TextStyle)
}

// SetStyleProperties sets paint values based on given property map (name: value
// pairs), inheriting elements as appropriate from parent, and also having a
// default style for the "initial" setting
func (pc *Paint) SetStyleProperties(parent *Paint, properties map[string]any, ctxt colors.Context) {
	if !pc.StyleSet && parent != nil { // first time
		pc.InheritFields(parent)
	}
	pc.styleFromProperties(parent, properties, ctxt)

	pc.PropertiesNil = (len(properties) == 0)
	pc.StyleSet = true
}

func (pc *Paint) FromStyle(st *Style) {
	pc.UnitContext = st.UnitContext
	pc.FontStyle = *st.FontRender()
	pc.TextStyle = st.Text
}

// ToDotsImpl runs ToDots on unit values, to compile down to raw pixels
func (pc *Paint) ToDotsImpl(uc *units.Context) {
	pc.Path.ToDotsImpl(uc)
	pc.FontStyle.ToDots(uc)
	pc.TextStyle.ToDots(uc)
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
	pc.FontStyle.SetUnitContext(&pc.UnitContext)
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
