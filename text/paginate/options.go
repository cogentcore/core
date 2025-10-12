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
	"cogentcore.org/core/text/printer"
)

// Options has the parameters for pagination.
type Options struct {
	// FontScale is an additional font scaling factor to apply.
	// This is used in content to reverse the DocsFontSize factor, for example.
	FontScale float32

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

	// SizeDots is the total size in dots. Set automatically, but needs to be readable
	// so is exported.
	SizeDots math32.Vector2 `edit:"-"`

	// BodyDots (content) size in dots.
	BodyDots math32.Vector2 `edit:"-"`

	// MargDots is the margin sizes in dots.
	MargDots sides.Floats `edit:"-"`
}

func NewOptions() Options {
	o := Options{}
	o.Defaults()
	return o
}

func (o *Options) Defaults() {
	o.FontScale = 1
	o.Footer = CenteredPageNumber
}

func (o *Options) ToDots(un *units.Context) {
	o.SizeDots, o.BodyDots, o.MargDots = printer.Settings.ToDots(un)
}
