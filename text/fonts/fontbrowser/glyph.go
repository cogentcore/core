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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/fonts"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// GlyphInfo returns info about a glyph.
type GlyphInfo struct {
	// Rune is the unicode code point.
	Rune rune

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
	gi.Rune = r
	gi.GID = gid
	gi.HAdvance = face.HorizontalAdvance(gid)
	gi.HExtents, _ = face.FontHExtents()
	gi.Extents, _ = face.GlyphExtents(gid)
}

// Glyph displays an individual glyph in the browser
type Glyph struct {
	core.Canvas

	Rune    rune
	GID     font.GID
	Outline []math32.Vector2
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
		fonts.FontStyle(gi.Browser.Font, &s.Font, &s.Text)
	})
	gi.OnClick(func(e events.Event) {
		if gi.Browser == nil || gi.Browser.Font == nil {
			return
		}
		gli := NewGlyphInfo(gi.Browser.Font, gi.Rune, gi.GID)
		gli.Outline = gi.Outline
		d := core.NewBody("Glyph Info")
		bg := NewGlyph(d).SetBrowser(gi.Browser).SetRune(gi.Rune).SetGID(gi.GID)
		bg.Styler(func(s *styles.Style) {
			s.Min.Set(units.Em(40))
		})
		core.NewForm(d).SetStruct(gli)
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddOK(bar)
		})
		d.RunDialog(gi.Browser)
	})
	gi.SetDraw(func(pc *paint.Painter) {
		if gi.Browser == nil || gi.Browser.Font == nil {
			return
		}
		data := gi.Browser.Font.GlyphData(gi.GID)
		gd, ok := data.(font.GlyphOutline)
		if !ok {
			return
		}
		scale := 0.7 / float32(gi.Browser.Font.Upem())
		x := float32(0.1)
		y := float32(0.8)
		gi.Outline = slicesx.SetLength(gi.Outline, len(gd.Segments))
		pc.Fill.Color = colors.Scheme.Surface
		if gi.StateIs(states.Active) || gi.StateIs(states.Focused) || gi.StateIs(states.Selected) {
			pc.Fill.Color = colors.Scheme.Select.Container
		}
		pc.Stroke.Color = colors.Scheme.OnSurface
		pc.Rectangle(0, 0, 1, 1)
		pc.PathDone()
		pc.Fill.Color = nil
		pc.Line(0, y, 1, y)
		pc.PathDone()
		pc.Stroke.Color = nil
		pc.Fill.Color = colors.Scheme.OnSurface
		for i, s := range gd.Segments {
			px := s.Args[0].X*scale + x
			py := -s.Args[0].Y*scale + y
			switch s.Op {
			case opentype.SegmentOpMoveTo:
				pc.MoveTo(px, py)
				gi.Outline[i] = math32.Vec2(px, py)
			case opentype.SegmentOpLineTo:
				pc.LineTo(px, py)
				gi.Outline[i] = math32.Vec2(px, py)
			case opentype.SegmentOpQuadTo:
				p1x := s.Args[1].X*scale + x
				p1y := -s.Args[1].Y*scale + y
				pc.QuadTo(px, py, p1x, p1y)
				gi.Outline[i] = math32.Vec2(p1x, p1y)
			case opentype.SegmentOpCubeTo:
				p1x := s.Args[1].X*scale + x
				p1y := -s.Args[1].Y*scale + y
				p2x := s.Args[2].X*scale + x
				p2y := -s.Args[2].Y*scale + y
				pc.CubeTo(px, py, p1x, p1y, p2x, p2y)
				gi.Outline[i] = math32.Vec2(p2x, p2y)
			}
		}
		pc.PathDone()
	})
}
