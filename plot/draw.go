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

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
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
	pt.drawConfig()
	pc := pt.Paint
	ptw := float32(pt.Size.X)
	pth := float32(pt.Size.X)

	ptb := image.Rectangle{Max: pt.Size}
	pc.PushBounds(ptb)

	if pt.Background != nil {
		pc.BlitBox(math32.Vector2{}, math32.FromPoint(pt.Size), pt.Background)
	}

	if pt.Title.Text != "" {
		pt.Title.Config(pt)
		pos := pt.Title.PosX(ptw)
		pad := pt.Title.Style.Padding.Dots
		pos.Y = pad
		pt.Title.Draw(pt, pos)
		th := pt.Title.PaintText.BBox.Size().Y + 2*pad
		pth -= th
		ptb.Min.Y += int(math32.Ceil(th))
	}

	pt.X.SanitizeRange()
	pt.Y.SanitizeRange()

	ywidth, tickWidth, tpad, bpad := pt.Y.sizeY(pt)
	xheight, lpad, rpad := pt.X.sizeX(pt, float32(pt.Size.X-int(ywidth)))

	tb := ptb
	tb.Min.X += ywidth
	pc.PushBounds(tb)
	pt.X.drawX(pt, lpad, rpad)
	pc.PopBounds()

	tb = ptb
	tb.Max.Y -= xheight
	pc.PushBounds(tb)
	pt.Y.drawY(pt, tickWidth, tpad, bpad)
	pc.PopBounds()

	tb = ptb
	tb.Min.X += ywidth + lpad
	tb.Max.X -= rpad
	tb.Max.Y -= xheight + bpad
	tb.Min.Y += tpad
	pt.PlotBox.SetFromRect(tb)

	// don't cut off lines
	tb.Min.X -= 2
	tb.Min.Y -= 2
	tb.Max.X += 2
	tb.Max.Y += 2
	pc.PushBounds(tb)

	for _, plt := range pt.Plotters {
		plt.Plot(pt)
	}

	pt.Legend.draw(pt)
	pc.PopBounds()
	pc.PopBounds() // global
}

////////////////////////////////////////////////////////////////
//		Axis

// drawTicks returns true if the tick marks should be drawn.
func (ax *Axis) drawTicks() bool {
	return ax.TickLine.Width.Value > 0 && ax.TickLength.Value > 0
}

// sizeX returns the total height of the axis, left and right padding
func (ax *Axis) sizeX(pt *Plot, axw float32) (ht, lpad, rpad int) {
	pc := pt.Paint
	uc := &pc.UnitContext
	ax.TickLength.ToDots(uc)
	ax.ticks = ax.Ticker.Ticks(ax.Min, ax.Max)
	h := float32(0)
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		ax.Label.Config(pt)
		h += ax.Label.PaintText.BBox.Size().Y
		h += ax.Label.Style.Padding.Dots
	}
	lw := ax.Line.Width.Dots
	lpad = int(math32.Ceil(lw)) + 2
	rpad = int(math32.Ceil(lw)) + 10
	tht := float32(0)
	if len(ax.ticks) > 0 {
		if ax.drawTicks() {
			h += ax.TickLength.Dots
		}
		ftk := ax.firstTickLabel()
		if ftk.Label != "" {
			px, _ := ax.tickPosX(pt, ftk, axw)
			if px < 0 {
				lpad += int(math32.Ceil(-px))
			}
			tht = max(tht, ax.TickText.PaintText.BBox.Size().Y)
		}
		ltk := ax.lastTickLabel()
		if ltk.Label != "" {
			px, wd := ax.tickPosX(pt, ltk, axw)
			if px+wd > axw {
				rpad += int(math32.Ceil((px + wd) - axw))
			}
			tht = max(tht, ax.TickText.PaintText.BBox.Size().Y)
		}
		ax.TickText.Text = ax.longestTickLabel()
		if ax.TickText.Text != "" {
			ax.TickText.Config(pt)
			tht = max(tht, ax.TickText.PaintText.BBox.Size().Y)
		}
		h += ax.TickText.Style.Padding.Dots
	}
	h += tht + lw + ax.Padding.Dots

	ht = int(math32.Ceil(h))
	return
}

// tickLabelPosX returns the relative position and width for given tick along X axis
// for given total axis width
func (ax *Axis) tickPosX(pt *Plot, t Tick, axw float32) (px, wd float32) {
	x := axw * float32(ax.Norm(t.Value))
	if x < 0 || x > axw {
		return
	}
	ax.TickText.Text = t.Label
	ax.TickText.Config(pt)
	pos := ax.TickText.PosX(0)
	px = pos.X + x
	wd = ax.TickText.PaintText.BBox.Size().X
	return
}

func (ax *Axis) firstTickLabel() Tick {
	for _, tk := range ax.ticks {
		if tk.Label != "" {
			return tk
		}
	}
	return Tick{}
}

func (ax *Axis) lastTickLabel() Tick {
	n := len(ax.ticks)
	for i := n - 1; i >= 0; i-- {
		tk := ax.ticks[i]
		if tk.Label != "" {
			return tk
		}
	}
	return Tick{}
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

func (ax *Axis) sizeY(pt *Plot) (ywidth, tickWidth, tpad, bpad int) {
	pc := pt.Paint
	uc := &pc.UnitContext
	ax.ticks = ax.Ticker.Ticks(ax.Min, ax.Max)
	ax.TickLength.ToDots(uc)

	w := float32(0)
	if ax.Label.Text != "" {
		ax.Label.Config(pt)
		w += ax.Label.PaintText.BBox.Size().X
		w += ax.Label.Style.Padding.Dots
	}

	lw := ax.Line.Width.Dots
	tpad = int(math32.Ceil(lw)) + 2
	bpad = int(math32.Ceil(lw)) + 2

	if len(ax.ticks) > 0 {
		if ax.drawTicks() {
			w += ax.TickLength.Dots
		}
		ax.TickText.Text = ax.longestTickLabel()
		if ax.TickText.Text != "" {
			ax.TickText.Config(pt)
			tw := ax.TickText.PaintText.BBox.Size().X
			w += tw
			tickWidth = int(math32.Ceil(tw))
			w += ax.TickText.Style.Padding.Dots
			tht := int(math32.Ceil(0.5 * ax.TickText.PaintText.BBox.Size().X))
			tpad += tht
			bpad += tht
		}
	}
	w += lw + ax.Padding.Dots
	ywidth = int(math32.Ceil(w))
	return
}

// drawX draws the horizontal axis
func (ax *Axis) drawX(pt *Plot, lpad, rpad int) {
	ab := pt.Paint.Bounds
	ab.Min.X += lpad
	ab.Max.X -= rpad
	axw := float32(ab.Size().X)
	// axh := float32(ab.Size().Y) // height of entire plot
	if ax.Label.Text != "" {
		ax.Label.Config(pt)
		pos := ax.Label.PosX(axw)
		pos.X += float32(ab.Min.X)
		th := ax.Label.PaintText.BBox.Size().Y
		pos.Y = float32(ab.Max.Y) - th
		ax.Label.Draw(pt, pos)
		ab.Max.Y -= int(math32.Ceil(th + ax.Label.Style.Padding.Dots))
	}

	tickHt := float32(0)
	for _, t := range ax.ticks {
		x := axw * float32(ax.Norm(t.Value))
		if x < 0 || x > axw || t.IsMinor() {
			continue
		}
		ax.TickText.Text = t.Label
		ax.TickText.Config(pt)
		pos := ax.TickText.PosX(0)
		pos.X += x + float32(ab.Min.X)
		tickHt = ax.TickText.PaintText.BBox.Size().Y + ax.TickText.Style.Padding.Dots
		pos.Y += float32(ab.Max.Y) - tickHt
		ax.TickText.Draw(pt, pos)
	}

	if len(ax.ticks) > 0 {
		ab.Max.Y -= int(math32.Ceil(tickHt))
		// } else {
		// 	y += ax.Width / 2
	}

	if len(ax.ticks) > 0 && ax.drawTicks() {
		ln := ax.TickLength.Dots
		for _, t := range ax.ticks {
			yoff := float32(0)
			if t.IsMinor() {
				yoff = 0.5 * ln
			}
			x := axw * float32(ax.Norm(t.Value))
			if x < 0 || x > axw {
				continue
			}
			x += float32(ab.Min.X)
			ax.TickLine.Draw(pt, math32.Vec2(x, float32(ab.Max.Y)-yoff), math32.Vec2(x, float32(ab.Max.Y)-ln))
		}
		ab.Max.Y -= int(ln - 0.5*ax.Line.Width.Dots)
	}

	ax.Line.Draw(pt, math32.Vec2(float32(ab.Min.X), float32(ab.Max.Y)), math32.Vec2(float32(ab.Min.X)+axw, float32(ab.Max.Y)))
}

// drawY draws the Y axis along the left side
func (ax *Axis) drawY(pt *Plot, tickWidth, tpad, bpad int) {
	ab := pt.Paint.Bounds
	ab.Min.Y += tpad
	ab.Max.Y -= bpad
	axh := float32(ab.Size().Y)
	if ax.Label.Text != "" {
		ax.Label.Style.Align = styles.Center
		pos := ax.Label.PosY(axh)
		tw := ax.Label.PaintText.BBox.Size().X
		pos.Y += float32(ab.Min.Y) + ax.Label.PaintText.BBox.Size().Y
		pos.X = float32(ab.Min.X)
		ax.Label.Draw(pt, pos)
		ab.Min.X += int(math32.Ceil(tw + ax.Label.Style.Padding.Dots))
	}

	tickWd := float32(0)
	for _, t := range ax.ticks {
		y := axh * (1 - float32(ax.Norm(t.Value)))
		if y < 0 || y > axh || t.IsMinor() {
			continue
		}
		ax.TickText.Text = t.Label
		ax.TickText.Config(pt)
		pos := ax.TickText.PosX(float32(tickWidth))
		pos.X += float32(ab.Min.X)
		pos.Y = float32(ab.Min.Y) + y - 0.5*ax.TickText.PaintText.BBox.Size().Y
		tickWd = max(tickWd, ax.TickText.PaintText.BBox.Size().X+ax.TickText.Style.Padding.Dots)
		ax.TickText.Draw(pt, pos)
	}

	if len(ax.ticks) > 0 {
		ab.Min.X += int(math32.Ceil(tickWd))
		// } else {
		// 	y += ax.Width / 2
	}

	if len(ax.ticks) > 0 && ax.drawTicks() {
		ln := ax.TickLength.Dots
		for _, t := range ax.ticks {
			xoff := float32(0)
			if t.IsMinor() {
				xoff = 0.5 * ln
			}
			y := axh * (1 - float32(ax.Norm(t.Value)))
			if y < 0 || y > axh {
				continue
			}
			y += float32(ab.Min.Y)
			ax.TickLine.Draw(pt, math32.Vec2(float32(ab.Min.X)+xoff, y), math32.Vec2(float32(ab.Min.X)+ln, y))
		}
		ab.Min.X += int(ln + 0.5*ax.Line.Width.Dots)
	}

	ax.Line.Draw(pt, math32.Vec2(float32(ab.Min.X), float32(ab.Min.Y)), math32.Vec2(float32(ab.Min.X), float32(ab.Max.Y)))
}

////////////////////////////////////////////////
//		Legend

// draw draws the legend
func (lg *Legend) draw(pt *Plot) {
	pc := pt.Paint
	uc := &pc.UnitContext
	ptb := pc.Bounds

	lg.ThumbnailWidth.ToDots(uc)
	lg.TextStyle.ToDots(uc)
	lg.Position.XOffs.ToDots(uc)
	lg.Position.YOffs.ToDots(uc)
	lg.TextStyle.openFont(pt)

	em := lg.TextStyle.Font.Face.Metrics.Em
	pad := math32.Ceil(lg.TextStyle.Padding.Dots)

	var ltxt Text
	ltxt.Style = lg.TextStyle
	var sz image.Point
	maxTht := 0
	for _, e := range lg.Entries {
		ltxt.Text = e.Text
		ltxt.Config(pt)
		sz.X = max(sz.X, int(math32.Ceil(ltxt.PaintText.BBox.Size().X)))
		tht := int(math32.Ceil(ltxt.PaintText.BBox.Size().Y + pad))
		maxTht = max(tht, maxTht)
	}
	sz.X += int(em)
	sz.Y = len(lg.Entries) * maxTht
	txsz := sz
	sz.X += int(lg.ThumbnailWidth.Dots)

	pos := ptb.Min
	if lg.Position.Left {
		pos.X += int(lg.Position.XOffs.Dots)
	} else {
		pos.X = ptb.Max.X - sz.X - int(lg.Position.XOffs.Dots)
	}
	if lg.Position.Top {
		pos.Y += int(lg.Position.YOffs.Dots)
	} else {
		pos.Y = ptb.Max.Y - sz.Y - int(lg.Position.YOffs.Dots)
	}

	if lg.Fill != nil {
		pc.FillBox(math32.FromPoint(pos), math32.FromPoint(sz), lg.Fill)
	}
	cp := pos
	thsz := image.Point{X: int(lg.ThumbnailWidth.Dots), Y: maxTht - 2*int(pad)}
	for _, e := range lg.Entries {
		tp := cp
		tp.X += int(txsz.X)
		tp.Y += int(pad)
		tb := image.Rectangle{Min: tp, Max: tp.Add(thsz)}
		pc.PushBounds(tb)
		for _, t := range e.Thumbs {
			t.Thumbnail(pt)
		}
		pc.PopBounds()
		ltxt.Text = e.Text
		ltxt.Config(pt)
		ltxt.Draw(pt, math32.FromPoint(cp))
		cp.Y += maxTht
	}
}
