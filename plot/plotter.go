// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

// Plotter is an interface that wraps the Plot method.
// Some standard implementations of Plotter can be found in plotters.
type Plotter interface {
	// Plot draws the data to the Plot Paint
	Plot(pt *Plot)

	// returns the data for this plot as X,Y points,
	// including corresponding pixel data.
	// This allows gui interface to inspect data etc.
	XYData() (data XYer, pixels XYer)

	// ApplyStyle runs the style functions.
	ApplyStyle()
}

// DataRanger wraps the DataRange method.
type DataRanger interface {
	// DataRange returns the range of X and Y values.
	DataRange(pt *Plot) (xmin, xmax, ymin, ymax float32)
}

// Stylers implements the styling functions for plotters.
type Stylers []func(p Plotter)

func (st *Stylers) Add(f func(p Plotter)) {
	*st = append(*st, f)
}

func (st *Stylers) Run(p Plotter) {
	for _, f := range *st {
		f(p)
	}
}
