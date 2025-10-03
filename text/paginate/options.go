// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate -add-types

package paginate

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/paginate/pagesizes"
)

// Options has the parameters for pagination.
type Options struct {
	// PageSize specifies a standard page size, or Custom.
	PageSize pagesizes.Sizes

	// Units are the units in which size is specified.
	// Will automatically be set if PageSize != Custom.
	Units units.Units

	// Size is the size in given units.
	// Will automatically be set if PageSize != Custom.
	Size math32.Vector2

	// Margins specify the page margins in the size units.
	Margins sides.Floats `display:"inline"`

	// Header is the header template string, with #
	// replaced with the page number
	// <stretch> adds a stretch element that can be used to accomplish
	// justification: at start = right justify, at start and end = center
	Header string

	// Footer is the footer template string, with #
	// replaced with the page number.
	// <stretch> adds a stretch element that can be used to accomplish
	// justification: at start = right justify, at start and end = center
	Footer string

	sizeDots math32.Vector2 // total size in dots
	bodyDots math32.Vector2 // body (content) size in dots
	margDots sides.Floats   // margin sizes in dots
}

func NewOptions() Options {
	o := Options{}
	o.Defaults()
	return o
}

func (o *Options) Defaults() {
	// todo: make this contingent on localization somehow!
	o.PageSize = pagesizes.A4
	o.Margins.Set(25) // basically one inch
	o.Footer = "<stretch>#<stretch>"
	o.Update()
}

func (o *Options) Update() {
	if o.PageSize != pagesizes.Custom {
		o.Units, o.Size = o.PageSize.Size()
	}
}

func (o *Options) ToDots(un *units.Context) {
	sc := un.ToDots(1, o.Units)
	o.sizeDots = o.Size.MulScalar(sc)
	o.margDots = o.Margins.MulScalar(sc)
	o.bodyDots.X = o.sizeDots.X - (o.margDots.Left + o.margDots.Right)
	o.bodyDots.Y = o.sizeDots.Y - (o.margDots.Top + o.margDots.Bottom)
}
