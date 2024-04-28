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
		pos := pt.Title.posX(ptw)
		pad := pt.Title.Style.Padding.Dots
		pos.Y = pad
		pt.Title.draw(pt, pos)
		th := pt.Title.paintText.Size.Y + 2*pad
		pth -= th
		ptb.Min.Y += int(math32.Ceil(th))
	}

	pt.X.SanitizeRange()
	pt.Y.SanitizeRange()

	ywidth, tickWidth, tpad, bpad := pt.Y.sizeY(pt)
	xheight, lpad, rpad := pt.X.sizeX(pt, float32(pt.Size.X-int(ywidth)))

	tb := ptb
	tb.Min.X += ywidth
	pt.Paint.PushBounds(tb)
	pt.X.drawX(pt, lpad, rpad)
	pt.Paint.PopBounds()

	tb = ptb
	tb.Max.Y -= xheight
	pt.Paint.PushBounds(tb)
	pt.Y.drawY(pt, tickWidth, tpad, bpad)
	pt.Paint.PopBounds()

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
		ax.Label.config(pt)
		h += ax.Label.paintText.Size.Y
		h += ax.Label.Style.Padding.Dots
	}
	lw := ax.Line.Width.Dots
	lpad = int(math32.Ceil(lw)) + 2
	rpad = int(math32.Ceil(lw)) + 2
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
			tht = max(tht, ax.TickText.paintText.Size.Y)
		}
		ltk := ax.lastTickLabel()
		if ltk.Label != "" {
			px, wd := ax.tickPosX(pt, ltk, axw)
			if px+wd > axw {
				rpad += int(math32.Ceil((px + wd) - axw))
			}
			tht = max(tht, ax.TickText.paintText.Size.Y)
		}
		ax.TickText.Text = ax.longestTickLabel()
		if ax.TickText.Text != "" {
			ax.TickText.config(pt)
			tht = max(tht, ax.TickText.paintText.Size.Y)
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
	ax.TickText.config(pt)
	pos := ax.TickText.posX(0)
	px = pos.X + x
	wd = ax.TickText.paintText.Size.X
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
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		ax.Label.configRot(pt, ax.Label.Style.Rotation)
		w += ax.Label.paintText.Size.Y
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
			ax.TickText.config(pt)
			w += ax.TickText.paintText.Size.X
			tickWidth = int(math32.Ceil(ax.TickText.paintText.Size.X))
			w += ax.TickText.Style.Padding.Dots
			tht := int(math32.Ceil(0.5 * ax.TickText.paintText.Size.X))
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
		ax.Label.config(pt)
		pos := ax.Label.posX(axw)
		pos.X += float32(ab.Min.X)
		th := ax.Label.paintText.Size.Y
		pos.Y = float32(ab.Max.Y) - th
		ax.Label.draw(pt, pos)
		ab.Max.Y -= int(math32.Ceil(th + ax.Label.Style.Padding.Dots))
	}

	tickHt := float32(0)
	for _, t := range ax.ticks {
		x := axw * float32(ax.Norm(t.Value))
		if x < 0 || x > axw || t.IsMinor() {
			continue
		}
		ax.TickText.Text = t.Label
		ax.TickText.config(pt)
		pos := ax.TickText.posX(0)
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
			ax.TickLine.draw(pt, math32.Vec2(x, float32(ab.Max.Y)-yoff), math32.Vec2(x, float32(ab.Max.Y)-ln))
		}
		ab.Max.Y -= int(ln - 0.5*ax.Line.Width.Dots)
	}

	ax.Line.draw(pt, math32.Vec2(float32(ab.Min.X), float32(ab.Max.Y)), math32.Vec2(float32(ab.Min.X)+axw, float32(ab.Max.Y)))
}

// drawY draws the Y axis along the left side
func (ax *Axis) drawY(pt *Plot, tickWidth, tpad, bpad int) {
	ab := pt.Paint.Bounds
	ab.Min.Y += tpad
	ab.Max.Y -= bpad
	axh := float32(ab.Size().Y)
	if ax.Label.Text != "" {
		pos := ax.Label.posX(axh)
		pos.Y = float32(ab.Min.Y) + pos.X + ax.Label.paintText.Size.X
		pos.X = float32(ab.Min.X) + ax.Label.paintText.Size.Y
		tw := ax.Label.paintText.Size.Y
		ax.Label.draw(pt, pos)
		ab.Min.X += int(math32.Ceil(tw + ax.Label.Style.Padding.Dots))
	}

	tickWd := float32(0)
	for _, t := range ax.ticks {
		y := axh * float32(ax.Norm(t.Value))
		if y < 0 || y > axh || t.IsMinor() {
			continue
		}
		ax.TickText.Text = t.Label
		ax.TickText.config(pt)
		pos := ax.TickText.posX(float32(tickWidth))
		pos.X += float32(ab.Min.X)
		pos.Y = float32(ab.Min.Y) + y - 0.5*ax.TickText.paintText.Size.Y
		tickWd = max(tickWd, ax.TickText.paintText.Size.X+ax.TickText.Style.Padding.Dots)
		ax.TickText.draw(pt, pos)
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
			y := axh * float32(ax.Norm(t.Value))
			if y < 0 || y > axh {
				continue
			}
			y += float32(ab.Min.Y)
			ax.TickLine.draw(pt, math32.Vec2(float32(ab.Min.X)+xoff, y), math32.Vec2(float32(ab.Min.X)+ln, y))
		}
		ab.Min.X += int(ln + 0.5*ax.Line.Width.Dots)
	}

	ax.Line.draw(pt, math32.Vec2(float32(ab.Min.X), float32(ab.Min.Y)), math32.Vec2(float32(ab.Min.X), float32(ab.Max.Y)))
}
