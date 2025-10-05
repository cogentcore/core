// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate core generate -add-types

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/paginate/pagesizes"
	"cogentcore.org/core/text/rich"
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

	// FontFamily specifies the default font family to apply
	// to all core.Text elements.
	FontFamily rich.Family

	// FontSize specifies the default font size to apply
	// to all core.Text elements, if non-zero.
	FontSize units.Value

	// Title generates the title contents for the first page,
	// into the given page body frame.
	Title func(frame *core.Frame, opts *Options)

	// Header generates the header contents for the page, into the given
	// frame that represents the entire top margin.
	// See examples in runners.go
	Header func(frame *core.Frame, opts *Options, pageNo int)

	// Footer generates the footer contents for the page, into the given
	// frame that represents the entire top margin.
	// See examples in runners.go
	Footer func(frame *core.Frame, opts *Options, pageNo int)

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
	o.Footer = CenteredPageNumber
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
