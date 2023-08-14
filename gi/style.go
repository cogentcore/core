// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
)

// CustomConfigStyles is the custom, global style configuration function
// that is called on all widgets to configure their style functions.
// By default, it is nil. If you set it, you should mostly call
// AddStyleFunc within it. For reference on
// how you should structure your CustomStyleFunc, you
// should look at https://goki.dev/docs/gi/styling.
var CustomConfigStyles func(w *WidgetBase)

// Pre-configured box shadow values, based on
// those in Material 3
var (
	// ShadowElevation0 contains the shadows
	// to be used on Elevation 0 elements.
	// There are no shadows part of ShadowElevation0,
	// so applying it is purely semantic.
	ShadowElevation0 = []gist.Shadow{}
	// ShadowElevation1 contains the shadows
	// to be used on Elevation 1 elements.
	ShadowElevation1 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(1),
			Spread:  units.Px(-2),
			Color:   ColorScheme.Shadow.WithA(0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(2),
			Spread:  units.Px(0),
			Color:   ColorScheme.Shadow.WithA(0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   ColorScheme.Shadow.WithA(0.12),
		},
	}
	// ShadowElevation2 contains the shadows
	// to be used on Elevation 2 elements.
	ShadowElevation2 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(2),
			Blur:    units.Px(4),
			Spread:  units.Px(-1),
			Color:   ColorScheme.Shadow.WithA(0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(4),
			Blur:    units.Px(5),
			Spread:  units.Px(0),
			Color:   ColorScheme.Shadow.WithA(0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(1),
			Blur:    units.Px(10),
			Spread:  units.Px(0),
			Color:   ColorScheme.Shadow.WithA(0.12),
		},
	}
	// TODO: figure out why 3 and 4 are the same

	// ShadowElevation3 contains the shadows
	// to be used on Elevation 3 elements.
	ShadowElevation3 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   ColorScheme.Shadow.WithA(0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   ColorScheme.Shadow.WithA(0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   ColorScheme.Shadow.WithA(0.12),
		},
	}
	// ShadowElevation4 contains the shadows
	// to be used on Elevation 4 elements.
	ShadowElevation4 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(5),
			Blur:    units.Px(5),
			Spread:  units.Px(-3),
			Color:   ColorScheme.Shadow.WithA(0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(1),
			Color:   ColorScheme.Shadow.WithA(0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(3),
			Blur:    units.Px(14),
			Spread:  units.Px(2),
			Color:   ColorScheme.Shadow.WithA(0.12),
		},
	}
	// ShadowElevation5 contains the shadows
	// to be used on Elevation 5 elements.
	ShadowElevation5 = []gist.Shadow{
		{
			HOffset: units.Px(0),
			VOffset: units.Px(8),
			Blur:    units.Px(10),
			Spread:  units.Px(-6),
			Color:   ColorScheme.Shadow.WithA(0.2),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(16),
			Blur:    units.Px(24),
			Spread:  units.Px(2),
			Color:   ColorScheme.Shadow.WithA(0.14),
		},
		{
			HOffset: units.Px(0),
			VOffset: units.Px(6),
			Blur:    units.Px(30),
			Spread:  units.Px(5),
			Color:   ColorScheme.Shadow.WithA(0.12),
		},
	}
)
