// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"bufio"
	"bytes"
	"image"
	"io"
	"os"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
)

// SVGString returns an SVG representation of the plot as a string
func (pt *Plot) SVGString() string {
	b := &bytes.Buffer{}
	pt.Paint.SVGOut = b
	pt.svgDraw()
	pt.Paint.SVGOut = nil
	return b.String()
}

// svgDraw draws SVGOut writer that must already be set in Paint
func (pt *Plot) svgDraw() {
	pt.drawConfig()
	io.WriteString(pt.Paint.SVGOut, pt.Paint.SVGStart())
	pt.Draw()
	io.WriteString(pt.Paint.SVGOut, pt.Paint.SVGEnd())
}

// SVGToFile saves the SVG to given file
func (pt *Plot) SVGToFile(filename string) error {
	fp, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fp.Close()
	bw := bufio.NewWriter(fp)
	pt.Paint.SVGOut = bw
	pt.svgDraw()
	pt.Paint.SVGOut = nil
	return bw.Flush()
}

// drawConfig configures everything for drawing
func (pt *Plot) drawConfig() {
	pt.Resize(pt.Size) // ensure
	pt.Legend.TextStyle.openFont(pt)
	pt.Paint.ToDots()
}

// Draw draws the plot to image.
// Plotters are drawn in the order in which they were
// added to the plot.
func (pt *Plot) Draw() {
	ptw := float32(pt.Size.X)
	pth := float32(pt.Size.X)

	ptb := image.Rectangle{Max: pt.Size}
	pt.Paint.PushBounds(ptb)

	pt.drawConfig()
	if pt.Background != nil {
		pt.Paint.BlitBox(math32.Vector2{}, math32.Vector2FromPoint(pt.Size), colors.C(pt.Background))
	}

	if pt.Title.Text != "" {
		pt.Title.config(pt)
		pos := pt.Title.startPosX(ptw)
		pad := pt.Title.Style.Padding.Dots
		pos.Y = pad
		pt.Title.draw(pt, pos)
		th := pt.Title.paintText.Size.Y + 2*pad
		pth -= th
		ptb.Max.Y -= int(math32.Ceil(th))
	}

	pt.X.SanitizeRange()
	pt.Y.SanitizeRange()

	ywidth := pt.Y.size(pt)
	xheight := pt.X.size(pt)

	tb := ptb
	tb.Min.X += ywidth
	pt.Paint.PushBounds(tb)
	pt.X.draw(pt)
	pt.Paint.PopBounds()

	// tb = ptb
	// tb.Max.Y -= xheight
	// pt.Paint.PushBounds(tb)
	// pt.Y.draw(pt)
	// pt.Paint.PopBounds()

	tb = ptb
	tb.Min.X += ywidth
	tb.Max.Y -= xheight
	pt.Paint.PushBounds(tb)

	for _, plt := range pt.Plotters {
		plt.Plot(pt)
	}
	pt.Paint.PopBounds()

	pt.Legend.Draw(pt)
}

// DataCanvas returns a new draw.Canvas that
// is the subset of the given draw area into which
// the plot data will be drawn.
// func (pt *Plot) DataCanvas(da draw.Canvas) draw.Canvas {
// 	if pt.Title.Text != "" {
// 		rect := pt.Title.TextStyle.Rectangle(pt.Title.Text)
// 		da.Max.Y -= rect.Size().Y
// 		da.Max.Y -= pt.Title.Padding
// 	}
// 	pt.X.sanitizeRange()
// 	x := horizontalAxis{pt.X}
// 	pt.Y.sanitizeRange()
// 	y := verticalAxis{pt.Y}
// 	return padY(pt, padX(pt, draw.Crop(da, y.size(), 0, x.size(), 0)))
// }

/*
// padX returns a draw.Canvas that is padded horizontally
// so that glyphs will no be clipped.
func padX(pt *Plot, c draw.Canvas) draw.Canvas {
	glyphs := pt.GlyphBoxes(pt)
	l := leftMost(&c, glyphs)
	xAxis := horizontalAxis{pt.X}
	glyphs = append(glyphs, xAxis.GlyphBoxes(pt)...)
	r := rightMost(&c, glyphs)

	minx := c.Min.X - l.Min.X
	maxx := c.Max.X - (r.Min.X + r.Size().X)
	lx := vg.Length(l.X)
	rx := vg.Length(r.X)
	n := (lx*maxx - rx*minx) / (lx - rx)
	m := ((lx-1)*maxx - rx*minx + minx) / (lx - rx)
	return draw.Canvas{
		Canvas: vg.Canvas(c),
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: n, Y: c.Min.Y},
			Max: vg.Point{X: m, Y: c.Max.Y},
		},
	}
}

// padY returns a draw.Canvas that is padded vertically
// so that glyphs will no be clipped.
func padY(pt *Plot, c draw.Canvas) draw.Canvas {
	glyphs := pt.GlyphBoxes(pt)
	b := bottomMost(&c, glyphs)
	yAxis := verticalAxis{pt.Y}
	glyphs = append(glyphs, yAxis.GlyphBoxes(pt)...)
	t := topMost(&c, glyphs)

	miny := c.Min.Y - b.Min.Y
	maxy := c.Max.Y - (t.Min.Y + t.Size().Y)
	by := vg.Length(b.Y)
	ty := vg.Length(t.Y)
	n := (by*maxy - ty*miny) / (by - ty)
	m := ((by-1)*maxy - ty*miny + miny) / (by - ty)
	return draw.Canvas{
		Canvas: vg.Canvas(c),
		Rectangle: vg.Rectangle{
			Min: vg.Point{Y: n, X: c.Min.X},
			Max: vg.Point{Y: m, X: c.Max.X},
		},
	}
}

// Transforms returns functions to transfrom
// from the x and y data coordinate system to
// the draw coordinate system of the given
// draw area.
func (pt *Plot) Transforms(c *draw.Canvas) (x, y func(float64) vg.Length) {
	x = func(x float64) vg.Length { return c.X(pt.X.Norm(x)) }
	y = func(y float64) vg.Length { return c.Y(pt.Y.Norm(y)) }
	return
}
*/

////////////////////////////////////////////////////////////////
//		Axis
// drawTicks returns true if the tick marks should be drawn.

func (ax *Axis) drawTicks() bool {
	return ax.TickLine.Width.Value > 0 && ax.TickLength.Value > 0
}

// size returns the Height of X axis or Width of Y axis
func (ax *Axis) size(pt *Plot) int {
	ax.ticks = ax.Ticker.Ticks(ax.Min, ax.Max)
	if ax.Axis == math32.X {
		return ax.sizeX(pt)
	} else {
		return ax.sizeY(pt)
	}
}

func (ax *Axis) sizeX(pt *Plot) int {
	h := float32(0)
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		ax.Label.config(pt)
		h += ax.Label.paintText.Size.Y
		h += ax.Label.Style.Padding.Dots
	}

	if len(ax.ticks) > 0 {
		if ax.drawTicks() {
			h += ax.TickLength.Dots
		}
		ax.TickText.Text = ax.longestTickLabel()
		if ax.TickText.Text != "" {
			ax.TickText.config(pt)
			h += ax.TickText.paintText.Size.Y
			h += ax.TickText.Style.Padding.Dots
		}
	}
	h += ax.Line.Width.Dots / 2
	h += ax.Padding.Dots

	return int(math32.Ceil(h))
}

func (ax *Axis) lastTickLabel() string {
	lst := ""
	for _, tk := range ax.ticks {
		if tk.Label != "" {
			lst = tk.Label
		}
	}
	return lst
}

func (ax *Axis) longestTickLabel() string {
	lst := ""
	for _, tk := range ax.ticks {
		if len(tk.Label) > len(lst) {
			lst = tk.Label
		}
	}
	return lst
}

func (ax *Axis) sizeY(pt *Plot) int {
	w := float32(0)
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		ax.Label.config(pt)
		w += ax.Label.paintText.Size.X
		w += ax.Label.Style.Padding.Dots
	}

	if len(ax.ticks) > 0 {
		if ax.drawTicks() {
			w += ax.TickLength.Dots
		}
		ax.TickText.Text = ax.longestTickLabel()
		if ax.TickText.Text != "" {
			ax.TickText.config(pt)
			w += ax.TickText.paintText.Size.X
			w += ax.TickText.Style.Padding.Dots
		}
	}
	w += ax.Line.Width.Dots / 2
	w += ax.Padding.Dots

	return int(math32.Ceil(w))
}

func (ax *Axis) draw(pt *Plot) {
	if ax.Axis == math32.X {
		ax.drawX(pt)
	} else {
		ax.drawY(pt)
	}
}

// drawX draws the horizontal axis
func (ax *Axis) drawX(pt *Plot) {
	pc := pt.Paint
	uc := &pc.UnitContext
	ab := pt.Paint.Bounds
	axw := float32(ab.Size().X)
	// axh := float32(ab.Size().Y) // height of entire plot
	if ax.Label.Text != "" {
		ax.Label.config(pt)
		pos := ax.Label.startPosX(axw)
		pos.X += float32(ab.Min.X)
		th := ax.Label.paintText.Size.Y
		pos.Y = float32(ab.Max.Y) - th
		ax.Label.draw(pt, pos)
		ab.Max.Y -= int(math32.Ceil(th + ax.Label.Style.Padding.Dots))
	}

	tickHt := float32(0)
	for _, t := range ax.ticks {
		x := axw * float32(ax.Norm(t.Value))
		if x < 0 || x >= axw || t.IsMinor() {
			continue
		}
		ax.TickText.Text = t.Label
		ax.TickText.config(pt)
		pos := ax.TickText.startPosX(0)
		pos.X += x + float32(ab.Min.X)
		tickHt = ax.TickText.paintText.Size.Y + ax.TickText.Style.Padding.Dots
		pos.Y = float32(ab.Max.Y) - tickHt
		ax.TickText.draw(pt, pos)
	}

	if len(ax.ticks) > 0 {
		ab.Max.Y -= int(math32.Ceil(tickHt))
		// } else {
		// 	y += ax.Width / 2
	}

	if len(ax.ticks) > 0 && ax.drawTicks() {
		ax.TickLength.ToDots(uc)
		ln := ax.TickLength.Dots
		for _, t := range ax.ticks {
			x := axw * float32(ax.Norm(t.Value))
			if x < 0 || x >= axw {
				continue
			}
			x += float32(ab.Min.X)
			ax.TickLine.draw(pt, math32.Vec2(x, float32(ab.Max.Y)), math32.Vec2(x, float32(ab.Max.Y)-ln))
		}
		ab.Max.Y -= int(0.5 * ln)
	}

	ax.Line.draw(pt, math32.Vec2(float32(ab.Min.X), float32(ab.Max.Y)), math32.Vec2(float32(ab.Min.X)+axw, float32(ab.Max.Y)))
}

/*
// GlyphBoxes returns the GlyphBoxes for the tick labels.
func (ax horizontalAxis) GlyphBoxes(p *Plot) []GlyphBox {
	var (
		boxes []GlyphBox
		yoff  font.Length
	)

	if ax.Label.Text != "" {
		x := ax.Norm(p.X.Max)
		switch ax.Label.Position {
		case draw.PosCenter:
			x = ax.Norm(0.5 * (p.X.Max + p.X.Min))
		case draw.PosRight:
			x -= ax.Norm(0.5 * ax.Label.TextStyle.Width(ax.Label.Text).Points()) // FIXME(sbinet): want data coordinates
		}
		descent := ax.Label.TextStyle.FontExtents().Descent
		boxes = append(boxes, GlyphBox{
			X:         x,
			Rectangle: ax.Label.TextStyle.Rectangle(ax.Label.Text).Add(vg.Point{Y: yoff + descent}),
		})
		yoff += ax.Label.TextStyle.Height(ax.Label.Text)
		yoff += ax.Label.Padding
	}

	var (
		ax.ticks   = ax.Ticker.Ticks(ax.Min, ax.Max)
		height  = tickLabelHeight(ax.Tick.Label, ax.ticks)
		descent = ax.Tick.Label.FontExtents().Descent
	)
	for _, t := range ax.ticks {
		if t.IsMinor() {
			continue
		}
		box := GlyphBox{
			X:         ax.Norm(t.Value),
			Rectangle: ax.Tick.Label.Rectangle(t.Label).Add(vg.Point{Y: yoff + height + descent}),
		}
		boxes = append(boxes, box)
	}
	return boxes
}
*/

// drawY draws the Y axis along the left side
func (ax *Axis) drawY(pt *Plot) {
	/*
		var (
			x = c.Min.X
			y vg.Length
		)
		if ax.Label.Text != "" {
			sty := ax.Label.TextStyle
			sty.Rotation += math.Pi / 2
			x += ax.Label.TextStyle.Height(ax.Label.Text)
			switch ax.Label.Position {
			case draw.PosCenter:
				y = c.Center().Y
			case draw.PosTop:
				y = c.Max.Y
				y -= ax.Label.TextStyle.Width(ax.Label.Text) / 2
			}
			descent := ax.Label.TextStyle.FontExtents().Descent
			c.FillText(sty, vg.Point{X: x - descent, Y: y}, ax.Label.Text)
			x += descent
			x += ax.Label.Padding
		}
		ax.ticks := ax.Ticker.Ticks(ax.Min, ax.Max)
		if w := tickLabelWidth(ax.Tick.Label, ax.ticks); len(ax.ticks) > 0 && w > 0 {
			x += w
		}

		major := false
		descent := ax.Tick.Label.FontExtents().Descent
		for _, t := range ax.ticks {
			y := c.Y(ax.Norm(t.Value))
			if !c.ContainsY(y) || t.IsMinor() {
				continue
			}
			c.FillText(ax.Tick.Label, vg.Point{X: x, Y: y + descent}, t.Label)
			major = true
		}
		if major {
			x += ax.Tick.Label.Width(" ")
		}
		if ax.drawTicks() && len(ax.ticks) > 0 {
			len := ax.Tick.Length
			for _, t := range ax.ticks {
				y := c.Y(ax.Norm(t.Value))
				if !c.ContainsY(y) {
					continue
				}
				start := t.lengthOffset(len)
				c.StrokeLine2(ax.Tick.LineStyle, x+start, y, x+len, y)
			}
			x += len
		}

		c.StrokeLine2(ax.LineStyle, x, c.Min.Y, x, c.Max.Y)
	*/
}
