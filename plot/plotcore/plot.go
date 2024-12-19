// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

// Plot is a widget that renders a [plot.Plot] object.
// If it is not [states.ReadOnly], the user can pan and zoom the graph.
// See [PlotEditor] for an interactive interface for selecting columns to view.
type Plot struct {
	core.WidgetBase

	// Plot is the Plot to display in this widget
	Plot *plot.Plot `set:"-"`

	// SetRangesFunc, if set, is called to adjust the data ranges
	// after the point when these ranges are updated based on the plot data.
	SetRangesFunc func()
}

// SetPlot sets the plot to given Plot, and calls UpdatePlot to ensure it is
// drawn at the current size of this widget
func (pt *Plot) SetPlot(pl *plot.Plot) {
	if pl != nil && pt.Plot != nil && pt.Plot.Pixels != nil {
		pl.DPI = pt.Styles.UnitContext.DPI
		pl.SetPixels(pt.Plot.Pixels) // re-use the image!
	}
	pt.Plot = pl
	pt.updatePlot()
}

// updatePlot draws the current plot at the size of the current widget,
// and triggers a Render so the widget will be rendered.
func (pt *Plot) updatePlot() {
	if pt.Plot == nil {
		pt.NeedsRender()
		return
	}
	sz := pt.Geom.Size.Actual.Content.ToPoint()
	if sz == (image.Point{}) {
		return
	}
	pt.Plot.DPI = pt.Styles.UnitContext.DPI
	pt.Plot.Resize(sz)
	if pt.SetRangesFunc != nil {
		pt.SetRangesFunc()
	}
	pt.Plot.Draw()
	pt.NeedsRender()
}

func (pt *Plot) Init() {
	pt.WidgetBase.Init()
	pt.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))
		ro := pt.IsReadOnly()
		s.SetAbilities(!ro, abilities.Slideable, abilities.Activatable, abilities.Scrollable)
		if !ro {
			if s.Is(states.Active) {
				s.Cursor = cursors.Grabbing
				s.StateLayer = 0
			} else {
				s.Cursor = cursors.Grab
			}
		}
	})

	pt.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		if pt.Plot == nil {
			return
		}
		xf, yf := 1.0, 1.0
		if e.HasAnyModifier(key.Shift) {
			yf = 0
		} else if e.HasAnyModifier(key.Alt) {
			xf = 0
		}
		del := e.PrevDelta()
		dx := -float64(del.X) * (pt.Plot.X.Range.Range()) * 0.0008 * xf
		dy := float64(del.Y) * (pt.Plot.Y.Range.Range()) * 0.0008 * yf
		pt.Plot.PanZoom.XOffset += dx
		pt.Plot.PanZoom.YOffset += dy
		pt.updatePlot()
	})

	pt.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		if pt.Plot == nil {
			return
		}
		se := e.(*events.MouseScroll)
		sc := 1 + (float64(se.Delta.Y) * 0.002)
		xsc, ysc := sc, sc
		if e.HasAnyModifier(key.Shift) {
			ysc = 1
		} else if e.HasAnyModifier(key.Alt) {
			xsc = 1
		}
		pt.Plot.PanZoom.XScale *= xsc
		pt.Plot.PanZoom.YScale *= ysc
		pt.updatePlot()
	})
}

func (pt *Plot) WidgetTooltip(pos image.Point) (string, image.Point) {
	if pos == image.Pt(-1, -1) {
		return "_", image.Point{}
	}
	if pt.Plot == nil {
		return pt.Tooltip, pt.DefaultTooltipPos()
	}
	wpos := pos.Sub(pt.Geom.ContentBBox.Min)
	plt, _, idx, dist, _, data, legend := pt.Plot.ClosestDataToPixel(wpos.X, wpos.Y)
	if dist <= 10 {
		pt.Plot.HighlightPlotter = plt
		pt.Plot.HighlightIndex = idx
		pt.updatePlot()
		dx := 0.0
		if data[plot.X] != nil {
			dx = data[plot.X].Float1D(idx)
		}
		dy := 0.0
		if data[plot.Y] != nil {
			dy = data[plot.Y].Float1D(idx)
		}
		return fmt.Sprintf("%s[%d]: (%g, %g)", legend, idx, dx, dy), pos
	} else {
		if pt.Plot.HighlightPlotter != nil {
			pt.Plot.HighlightPlotter = nil
			pt.updatePlot()
		}
	}
	return pt.Tooltip, pt.DefaultTooltipPos()
}

func (pt *Plot) SizeFinal() {
	pt.WidgetBase.SizeFinal()
	pt.updatePlot()
}

func (pt *Plot) Render() {
	pt.WidgetBase.Render()

	r := pt.Geom.ContentBBox
	sp := pt.Geom.ScrollOffset()
	if pt.Plot == nil || pt.Plot.Pixels == nil {
		draw.Draw(pt.Scene.Pixels, r, colors.Scheme.Surface, sp, draw.Src)
		return
	}
	draw.Draw(pt.Scene.Pixels, r, pt.Plot.Pixels, sp, draw.Src)
}
