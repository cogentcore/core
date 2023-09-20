// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
)

// CustomConfigStyles is the custom, global style configuration function
// that is called on all widgets to configure their style functions.
// By default, it is nil. If you set it, you should mostly call
// AddStyleFunc within it. For reference on
// how you should structure your CustomStyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
var CustomConfigStyles func(w *WidgetBase)

// Styler is a fuction that can be used to style an element.
// They are the building blocks of the GoGi styling system.
// They can be used as a closure and capture surrounding context,
// but they are passed the widget base and style for convenience
// and so that they can be used for multiple elements if desired;
// you can get all of the information you need from the function.
// A Styler should be added to a widget through the [WidgetBase.AddStyler]
// method. We use stylers for styling because they give you complete
// control and full programming logic without any CSS-selector magic.
type Styler func(w *WidgetBase, s *gist.Style)

// Pre-configured box shadow values, based on
// those in Material 3. They are in gi because
// they need to access the color scheme.
var (
	// BoxShadow0 contains the shadows
	// to be used on Elevation 0 elements.
	// There are no shadows part of BoxShadow0,
	// so applying it is purely semantic.
	BoxShadow0 = []gist.Shadow{}
	// BoxShadow1 contains the shadows
	// to be used on Elevation 1 elements.
	BoxShadow1 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(1),
			Spread:  units.Px(-2),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(2),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.12),
		},
	}
	// BoxShadow2 contains the shadows
	// to be used on Elevation 2 elements.
	BoxShadow2 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(4),
			Spread:  units.Px(-1),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(4),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(10),
			Spread:  units.Px(0),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.12),
		},
	}
	// TODO: figure out why 3 and 4 are the same

	// BoxShadow3 contains the shadows
	// to be used on Elevation 3 elements.
	BoxShadow3 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.12),
		},
	}
	// BoxShadow4 contains the shadows
	// to be used on Elevation 4 elements.
	BoxShadow4 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.12),
		},
	}
	// BoxShadow5 contains the shadows
	// to be used on Elevation 5 elements.
	BoxShadow5 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(-6),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(16),
			Blur:    units.Px(24),
			Spread:  units.Px(2),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(6),
			Blur:    units.Px(30),
			Spread:  units.Px(5),
			Color:   colors.SetAF32(ColorScheme.Shadow, 0.12),
		},
	}
)
