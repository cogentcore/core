// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// GlyphInfo returns info about a glyph.
type GlyphInfo struct {
	// Rune is the unicode rune as a string
	Rune string

	// RuneInt is the unicode code point, int number.
	RuneInt rune

	// RuneHex is the unicode code point, hexidecimal number.
	RuneHex rune `format:"%0X"`

	// GID is the glyph ID, specific to each Font.
	GID font.GID

	// HAdvance is the horizontal advance.
	HAdvance float32

	// Extents give the size of the glyph.
	Extents opentype.GlyphExtents

	// Extents are the horizontal font size parameters.
	HExtents font.FontExtents

	// Outline has the end points of each segment of the outline.
	Outline []math32.Vector2
}

func NewGlyphInfo(face *font.Face, r rune, gid font.GID) *GlyphInfo {
	gi := &GlyphInfo{}
	gi.Set(face, r, gid)
	return gi
}

// Set sets the info from given [font.Face] and gid.
func (gi *GlyphInfo) Set(face *font.Face, r rune, gid font.GID) {
	gi.Rune = string(r)
	gi.RuneInt = r
	gi.RuneHex = r
	gi.GID = gid
	gi.HAdvance = face.HorizontalAdvance(gid)
	gi.HExtents, _ = face.FontHExtents()
	gi.Extents, _ = face.GlyphExtents(gid)
}

// Glyph displays an individual glyph in the browser
type Glyph struct {
	core.Canvas

	// Rune is the rune to render.
	Rune rune

	// GID is the glyph ID of the Rune
	GID font.GID

	// Outline is the set of control points (end points only).
	Outline []math32.Vector2 `set:"-"`

	// Stroke only renders the outline of the glyph, not the standard fill.
	Stroke bool

	// Points plots the control points.
	Points bool

	Browser *Browser
}

func (gi *Glyph) Init() {
	gi.Canvas.Init()
	gi.Styler(func(s *styles.Style) {
		s.Min.Set(units.Em(3))
		s.SetTextWrap(false)
		s.Cursor = cursors.Pointer
		if gi.Browser == nil {
			return
		}
		s.SetAbilities(true, abilities.Clickable, abilities.Focusable, abilities.Activatable, abilities.Selectable)
		sty, tsty := s.NewRichText()
		fonts.Style(gi.Browser.Font, sty, tsty)
	})
	gi.OnClick(func(e events.Event) {
		if gi.Stroke || gi.Browser == nil || gi.Browser.Font == nil {
			return
		}
		gli := NewGlyphInfo(gi.Browser.Font, gi.Rune, gi.GID)
		gli.Outline = gi.Outline
		d := core.NewBody("Glyph Info")
		bg := NewGlyph(d).SetBrowser(gi.Browser).SetRune(gi.Rune).SetGID(gi.GID).
			SetStroke(true).SetPoints(true)
		bg.Styler(func(s *styles.Style) {
			s.Min.Set(units.Em(40))
		})
		core.NewForm(d).SetStruct(gli).StartFocus()
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddOK(bar)
		})
		d.RunWindowDialog(gi.Browser)
	})
	gi.SetDraw(gi.draw)
}

func (gi *Glyph) drawShaped(pc *paint.Painter) {
	sty, tsty := gi.Styles.NewRichText()
	fonts.Style(gi.Browser.Font, sty, tsty)
	sz := gi.Geom.Size.Actual.Content
	msz := min(sz.X, sz.Y)
	sty.Size = float32(msz) / tsty.FontSize.Dots
	sty.Size *= 0.85
	tx := rich.NewText(sty, []rune{gi.Rune})
	lns := gi.Scene.TextShaper().WrapLines(tx, sty, tsty, sz)
	off := math32.Vec2(0, 0)
	if msz > 200 {
		o := 0.2 * float32(msz)
		if gi.Browser.IsEmoji {
			off = math32.Vec2(0.5*o, -o)
		} else { // for bitmap fonts, kinda random
			off = math32.Vec2(o, o)
		}
	}
	pc.DrawText(lns, gi.Geom.Pos.Content.Add(off))
}

func (gi *Glyph) draw(pc *paint.Painter) {
	if gi.Browser == nil || gi.Browser.Font == nil {
		return
	}
	face := gi.Browser.Font
	data := face.GlyphData(gi.GID)
	gd, ok := data.(font.GlyphOutline)
	if !ok {
		gi.drawShaped(pc)
		return
	}
	scale := 0.7 / float32(face.Upem())
	x := float32(0.1)
	y := float32(0.8)
	gi.Outline = slicesx.SetLength(gi.Outline, len(gd.Segments))
	pc.Fill.Color = colors.Scheme.Surface
	if gi.StateIs(states.Active) || gi.StateIs(states.Focused) || gi.StateIs(states.Selected) {
		pc.Fill.Color = colors.Scheme.Select.Container
	}
	pc.Stroke.Color = colors.Scheme.OnSurface
	pc.Rectangle(0, 0, 1, 1)
	pc.Draw()
	pc.Fill.Color = nil
	pc.Line(0, y, 1, y)
	pc.Draw()
	if gi.Stroke {
		pc.Stroke.Width.Dp(2)
		pc.Stroke.Color = colors.Scheme.OnSurface
		pc.Fill.Color = nil
	} else {
		pc.Stroke.Color = nil
		pc.Fill.Color = colors.Scheme.OnSurface
	}
	ext, _ := face.GlyphExtents(gi.GID)
	if ext.XBearing < 0 {
		x -= scale * ext.XBearing
	}
	var gp ppath.Path
	for i, s := range gd.Segments {
		px := s.Args[0].X*scale + x
		py := -s.Args[0].Y*scale + y
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			gp.MoveTo(px, py)
			gi.Outline[i] = math32.Vec2(px, py)
		case opentype.SegmentOpLineTo:
			gp.LineTo(px, py)
			gi.Outline[i] = math32.Vec2(px, py)
		case opentype.SegmentOpQuadTo:
			p1x := s.Args[1].X*scale + x
			p1y := -s.Args[1].Y*scale + y
			gp.QuadTo(px, py, p1x, p1y)
			gi.Outline[i] = math32.Vec2(p1x, p1y)
		case opentype.SegmentOpCubeTo:
			p1x := s.Args[1].X*scale + x
			p1y := -s.Args[1].Y*scale + y
			p2x := s.Args[2].X*scale + x
			p2y := -s.Args[2].Y*scale + y
			gp.CubeTo(px, py, p1x, p1y, p2x, p2y)
			gi.Outline[i] = math32.Vec2(p2x, p2y)
		}
	}
	bb := gp.FastBounds()
	sx := float32(1)
	sy := float32(1)
	if bb.Max.X >= 0.98 {
		sx = 0.9 / bb.Max.X
	}
	if bb.Min.Y < 0 {
		sy = 0.9 * (1 + bb.Min.Y) / 1.0
		gp = gp.Translate(0, -bb.Min.Y/sy)
		y -= bb.Min.Y / sy
	}
	if bb.Max.Y > 1 {
		sy *= 0.9 / bb.Max.Y
	}
	if sx != 1 || sy != 1 {
		gp = gp.Scale(sx, sy)
	}
	pc.State.Path = gp
	pc.Draw()

	// Points
	if !gi.Points {
		return
	}
	pc.Stroke.Color = nil
	pc.Fill.Color = colors.Scheme.Primary.Base
	radius := float32(0.01)
	for _, s := range gd.Segments {
		px := sx * (s.Args[0].X*scale + x)
		py := sy * (-s.Args[0].Y*scale + y)
		switch s.Op {
		case opentype.SegmentOpMoveTo, opentype.SegmentOpLineTo:
			pc.Circle(px, py, radius)
		case opentype.SegmentOpQuadTo:
			p1x := sx * (s.Args[1].X*scale + x)
			p1y := sy * (-s.Args[1].Y*scale + y)
			pc.Circle(p1x, p1y, radius)
		case opentype.SegmentOpCubeTo:
			p2x := sx * (s.Args[2].X*scale + x)
			p2y := sy * (-s.Args[2].Y*scale + y)
			pc.Circle(p2x, p2y, radius)
		}
	}
	pc.Draw()

	radius *= 0.8
	pc.Stroke.Color = nil
	pc.Fill.Color = colors.Scheme.Error.Base
	for _, s := range gd.Segments {
		px := sx * (s.Args[0].X*scale + x)
		py := sy * (-s.Args[0].Y*scale + y)
		switch s.Op {
		case opentype.SegmentOpQuadTo:
			pc.Circle(px, py, radius)
		case opentype.SegmentOpCubeTo:
			p1x := sx * (s.Args[1].X*scale + x)
			p1y := sy * (-s.Args[1].Y*scale + y)
			pc.Circle(px, py, radius)
			pc.Circle(p1x, p1y, radius)
		}
	}
	pc.Draw()

	pc.Stroke.Color = colors.Scheme.Error.Base
	pc.Fill.Color = nil
	for _, s := range gd.Segments {
		px := sx * (s.Args[0].X*scale + x)
		py := sy * (-s.Args[0].Y*scale + y)
		switch s.Op {
		case opentype.SegmentOpQuadTo:
			p1x := sx * (s.Args[1].X*scale + x)
			p1y := sy * (-s.Args[1].Y*scale + y)
			pc.Line(p1x, p1y, px, py)
		case opentype.SegmentOpCubeTo:
			p1x := sx * (s.Args[1].X*scale + x)
			p1y := sy * (-s.Args[1].Y*scale + y)
			p2x := sx * (s.Args[2].X*scale + x)
			p2y := sy * (-s.Args[2].Y*scale + y)
			pc.Line(px, py, p2x, p2y)
			pc.Line(p1x, p1y, p2x, p2y)
		}
	}
	pc.Draw()

}
