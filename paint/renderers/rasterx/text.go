// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rasterx

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg" // load image formats for users of the API
	_ "image/png"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/shaped"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/tiff" // load image formats for users of the API
)

// TextLines rasterizes the given shaped.Lines.
// The text will be drawn starting at the start pixel position, which specifies the
// left baseline location of the first text item..
func (rs *Renderer) TextLines(lns *shaped.Lines) {
	start := lns.Position
	for li := range lns.Lines {
		ln := &lns.Lines[li]
		rs.TextLine(ln, lns.FontSize, lns.Color, start) // todo: start + offset
	}
}

// TextLine rasterizes the given shaped.Line.
func (rs *Renderer) TextLine(ln *shaped.Line, fsz float32, clr color.Color, start math32.Vector2) {
	off := start.Add(ln.Offset)
	for ri := range ln.Runs {
		run := &ln.Runs[ri]
		rs.TextRun(run, fsz, clr, off)
	}
}

// TextRun rasterizes the given text run into the output image using the
// font face set in the shaping.
// The text will be drawn starting at the start pixel position.
func (rs *Renderer) TextRun(run *shaping.Output, fsz float32, clr color.Color, start math32.Vector2) {
	x := start.X
	y := start.Y
	// todo: render bg, render decoration
	for gi := range run.Glyphs {
		g := &run.Glyphs[gi]
		xPos := x + math32.FromFixed(g.XOffset)
		yPos := y - math32.FromFixed(g.YOffset)
		top := yPos - math32.FromFixed(g.YBearing)
		bottom := top - math32.FromFixed(g.Height)
		right := xPos + math32.FromFixed(g.Width)
		rect := image.Rect(int(xPos)-2, int(top)-2, int(right)+2, int(bottom)+2) // don't cut off
		data := run.Face.GlyphData(g.GlyphID)
		switch format := data.(type) {
		case font.GlyphOutline:
			rs.GlyphOutline(run, g, format, fsz, clr, rect, xPos, yPos)
		case font.GlyphBitmap:
			rs.GlyphBitmap(run, g, format, fsz, clr, rect, xPos, yPos)
		case font.GlyphSVG:
			fmt.Println("svg", format)
			// 	_ = rs.GlyphSVG(g, format, clr, xPos, yPos)
		}

		x += math32.FromFixed(g.XAdvance)
	}
	// todo: render strikethrough
}

func (rs *Renderer) GlyphOutline(run *shaping.Output, g *shaping.Glyph, bitmap font.GlyphOutline, fsz float32, clr color.Color, rect image.Rectangle, x, y float32) {
	rs.Raster.SetColor(colors.Uniform(clr))
	rf := &rs.Raster.Filler

	scale := fsz / float32(run.Face.Upem())
	rs.Scanner.SetClip(rect)
	rf.SetWinding(true)

	// todo: use stroke vs. fill color
	for _, s := range bitmap.Segments {
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			rf.Start(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpLineTo:
			rf.Line(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)})
		case opentype.SegmentOpQuadTo:
			rf.QuadBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)})
		case opentype.SegmentOpCubeTo:
			rf.CubeBezier(fixed.Point26_6{X: math32.ToFixed(s.Args[0].X*scale + x), Y: math32.ToFixed(-s.Args[0].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[1].X*scale + x), Y: math32.ToFixed(-s.Args[1].Y*scale + y)},
				fixed.Point26_6{X: math32.ToFixed(s.Args[2].X*scale + x), Y: math32.ToFixed(-s.Args[2].Y*scale + y)})
		}
	}
	rf.Stop(true)
	rf.Draw()
	rf.Clear()
}

func (rs *Renderer) GlyphBitmap(run *shaping.Output, g *shaping.Glyph, bitmap font.GlyphBitmap, fsz float32, clr color.Color, rect image.Rectangle, x, y float32) error {
	// scaled glyph rect content
	top := y - math32.FromFixed(g.YBearing)
	switch bitmap.Format {
	case font.BlackAndWhite:
		rec := image.Rect(0, 0, bitmap.Width, bitmap.Height)
		sub := image.NewPaletted(rec, color.Palette{color.Transparent, clr})

		for i := range sub.Pix {
			sub.Pix[i] = bitAt(bitmap.Data, i)
		}
		// todo: does it need scale? presumably not
		// scale.NearestNeighbor.Scale(img, rect, sub, sub.Bounds(), int(top)}, draw.Over, nil)
		draw.Draw(rs.image, sub.Bounds(), sub, image.Point{int(x), int(top)}, draw.Over)
	case font.JPG, font.PNG, font.TIFF:
		fmt.Println("img")
		// todo: how often?
		pix, _, err := image.Decode(bytes.NewReader(bitmap.Data))
		if err != nil {
			return err
		}
		// scale.BiLinear.Scale(img, rect, pix, pix.Bounds(), draw.Over, nil)
		draw.Draw(rs.image, pix.Bounds(), pix, image.Point{int(x), int(top)}, draw.Over)
	}

	if bitmap.Outline != nil {
		rs.GlyphOutline(run, g, *bitmap.Outline, fsz, clr, rect, x, y)
	}
	return nil
}

// bitAt returns the bit at the given index in the byte slice.
func bitAt(b []byte, i int) byte {
	return (b[i/8] >> (7 - i%8)) & 1
}
