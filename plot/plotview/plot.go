// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotview

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
)

// Plot is a Widget that renders a [plot.Plot] object.
// If it is not [states.ReadOnly], the user can pan and zoom the graph.
// See [ConfigPlotToolbar] for a toolbar with panning, selecting, and I/O buttons,
// and PlotView for an interactive interface for selecting columns to view.
type Plot struct {
	core.WidgetBase

	// Scale multiplies the plot DPI value, to change the overall scale
	// of the rendered plot.  Larger numbers produce larger scaling.
	// Typically use larger numbers when generating plots for inclusion in
	// documents or other cases where the overall plot size will be small.
	Scale float32

	// Plot is the Plot to display in this widget
	Plot *plot.Plot `set:"-"`
}

// SetPlot sets the plot to given Plot, and calls UpdatePlot to ensure it is
// drawn at the current size of this widget
func (pt *Plot) SetPlot(pl *plot.Plot) {
	if pl != nil && pt.Plot != nil && pt.Plot.Pixels != nil {
		pl.SetPixels(pt.Plot.Pixels) // re-use the image!
	}
	pt.Plot = pl
	pt.UpdatePlot()
}

// UpdatePlot draws the current plot at the size of the current widget,
// and triggers a Render so the widget will be rendered.
func (pt *Plot) UpdatePlot() {
	if pt.Plot == nil {
		pt.NeedsRender()
		return
	}
	sz := pt.Geom.Size.Actual.Content.ToPoint()
	if sz == (image.Point{}) {
		return
	}
	pt.Plot.DPI = pt.Scale * pt.Styles.UnitContext.DPI
	pt.Plot.Resize(sz)
	pt.Plot.Draw()
	pt.NeedsRender()
}

func (pt *Plot) OnInit() {
	pt.WidgetBase.OnInit()
	pt.Scale = 1
	pt.SetStyles()
	pt.HandleEvents()
}

func (pt *Plot) SetStyles() {
	pt.Style(func(s *styles.Style) {
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
}

func (pt *Plot) HandleEvents() {
	pt.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		if pt.Plot == nil {
			return
		}
		del := e.PrevDelta()
		dx := -float32(del.X) * (pt.Plot.X.Max - pt.Plot.X.Min) * 0.0008
		dy := float32(del.Y) * (pt.Plot.Y.Max - pt.Plot.Y.Min) * 0.0008
		pt.Plot.X.Min += dx
		pt.Plot.X.Max += dx
		pt.Plot.Y.Min += dy
		pt.Plot.Y.Max += dy
		pt.UpdatePlot()
		pt.NeedsRender()
	})

	pt.On(events.Scroll, func(e events.Event) {
		e.SetHandled()
		if pt.Plot == nil {
			return
		}
		se := e.(*events.MouseScroll)
		sc := 1 + (float32(se.Delta.Y) * 0.002)
		pt.Plot.X.Min *= sc
		pt.Plot.X.Max *= sc
		pt.Plot.Y.Min *= sc
		pt.Plot.Y.Max *= sc
		pt.UpdatePlot()
		pt.NeedsRender()
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
	_, idx, dist, data, _ := pt.Plot.ClosestDataToPixel(wpos.X, wpos.Y)
	if dist <= 10 {
		return fmt.Sprintf("[%d]: (%g, %g)", idx, data.X, data.Y), pos
	}
	return pt.Tooltip, pt.DefaultTooltipPos()
}

// SaveSVG saves the current Plot to an SVG file
func (pt *Plot) SavePlot(filename core.Filename) error { //types:add
	// return sv.Plot.SaveXML(string(filename))
	return nil
}

// SavePNG saves the current rendered Plot image to an PNG image file.
func (pt *Plot) SavePNG(filename core.Filename) error { //types:add
	// return sv.Plot.SavePNG(string(filename))
	return nil
}

func (pt *Plot) SizeFinal() {
	pt.WidgetBase.SizeFinal()
	pt.UpdatePlot()
}

func (pt *Plot) Render() {
	pt.WidgetBase.Render()

	r := pt.Geom.ContentBBox
	sp := pt.Geom.ScrollOffset()
	if pt.Plot == nil || pt.Plot.Pixels == nil {
		draw.Draw(pt.Scene.Pixels, r, colors.C(colors.Scheme.Surface), sp, draw.Src)
		return
	}
	draw.Draw(pt.Scene.Pixels, r, pt.Plot.Pixels, sp, draw.Src)
}
