// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"errors"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles/units"
)

// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Labels implements the Plotter interface,
// drawing a set of labels at specified points.
type Labels struct {
	// XYs is a copy of the points for labels
	plot.XYs

	// PXYs is the actual pixel plotting coordinates for each XY value.
	PXYs plot.XYs

	// Labels is the set of labels corresponding to each point.
	Labels []string

	// TextStyle is the style of the label text.
	// Each label can have a different text style, but
	// by default they share a common one (len = 1)
	TextStyle []plot.TextStyle

	// Offset is added directly to the final label location.
	Offset units.XY
}

// NewLabels returns a new Labels using defaults
func NewLabels(d XYLabeler) (*Labels, error) {
	xys, err := plot.CopyXYs(d)
	if err != nil {
		return nil, err
	}

	if d.Len() != len(xys) {
		return nil, errors.New("plotter: number of points does not match the number of labels")
	}

	strs := make([]string, d.Len())
	for i := range strs {
		strs[i] = d.Label(i)
	}

	styles := make([]plot.TextStyle, 1)
	for i := range styles {
		styles[i].Defaults()
	}

	return &Labels{
		XYs:       xys,
		Labels:    strs,
		TextStyle: styles,
	}, nil
}

func (l *Labels) XYData() (data plot.XYer, pixels plot.XYer) {
	data = l.XYs
	pixels = l.PXYs
	return
}

// Plot implements the Plotter interface, drawing labels.
func (l *Labels) Plot(plt *plot.Plot) {
	pc := plt.Paint
	uc := &pc.UnitContext
	ps := plot.PlotXYs(plt, l.XYs)

	l.Offset.ToDots(uc)
	np := len(l.XYs)
	customStyles := len(l.TextStyle) == np

	for i := range l.TextStyle {
		l.TextStyle[i].ToDots(uc)
	}

	var ltxt plot.Text
	for i, label := range l.Labels {
		if customStyles {
			ltxt.Style = l.TextStyle[i]
		} else {
			ltxt.Style = l.TextStyle[0]
		}
		ltxt.Text = label
		ltxt.Config(plt)
		tht := ltxt.PaintText.BBox.Size().Y
		ltxt.Draw(plt, math32.Vec2(ps[i].X+l.Offset.X.Dots, ps[i].Y+l.Offset.Y.Dots-tht))
	}
}

// DataRange returns the minimum and maximum X and Y values
func (l *Labels) DataRange() (xmin, xmax, ymin, ymax float32) {
	return plot.XYRange(l)
}

// XYLabeler combines the [plot.XYer] and [plot.Labeler] types.
type XYLabeler interface {
	plot.XYer
	plot.Labeler
}

// XYLabels holds XY data with labels.
// The ith label corresponds to the ith XY.
type XYLabels struct {
	plot.XYs
	Labels []string
}

// Label returns the label for point index i.
func (l XYLabels) Label(i int) string {
	return l.Labels[i]
}

var _ XYLabeler = (*XYLabels)(nil)
