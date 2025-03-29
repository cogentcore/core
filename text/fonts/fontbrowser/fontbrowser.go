// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -add-types -setters

import (
	"bytes"
	"os"
	"strconv"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/tree"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// GlyphInfo returns info about a glyph.
type GlyphInfo struct {
	Rune     rune
	GID      font.GID
	HAdvance float32
	Extents  opentype.GlyphExtents
	Outline  []math32.Vector2
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
		s.SetAbilities(true, abilities.Clickable)
		fonts.FontStyle(gi.Browser.Font, &s.Font, &s.Text)
	})
	gi.OnClick(func(e events.Event) {
		if gi.Browser == nil || gi.Browser.Font == nil {
			return
		}
		gli := NewGlyphInfo(gi.Browser.Font, gi.Rune, gi.GID)
		d := core.NewBody("Glyph Info")
		bg := NewGlyph(d).SetBrowser(gi.Browser).SetRune(gi.Rune).SetGID(gi.GID)
		bg.Styler(func(s *styles.Style) {
			s.Min.Set(units.Em(20))
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

// Browser is a font browser.
type Browser struct {
	core.Frame

	Filename core.Filename
	Font     *font.Face
	RuneMap  *keylist.List[rune, font.GID]
}

var _ tree.Node = (*Browser)(nil)

// OpenFile opens a font file.
func (fb *Browser) OpenFile(fname core.Filename) error { //types:add
	b, err := os.ReadFile(string(fname))
	if errors.Log(err) != nil {
		return err
	}
	fb.Filename = fname
	return fb.OpenFontData(b)
}

// OpenFontData opens given font data.
func (fb *Browser) OpenFontData(b []byte) error {
	faces, err := font.ParseTTC(bytes.NewReader(b))
	if errors.Log(err) != nil {
		return err
	}
	fb.Font = faces[0]
	fb.UpdateRuneMap()
	fb.Update()
	return nil
}

func (fb *Browser) UpdateRuneMap() {
	fb.RuneMap = keylist.New[rune, font.GID]()
	if fb.Font == nil {
		return
	}
	for _, pr := range unicode.PrintRanges {
		for _, rv := range pr.R16 {
			for r := rv.Lo; r <= rv.Hi; r += rv.Stride {
				gid, has := fb.Font.NominalGlyph(rune(r))
				if !has {
					continue
				}
				fb.RuneMap.Add(rune(r), gid)
			}
		}
	}
}

func (fb *Browser) Init() {
	fb.Frame.Init()
	fb.Styler(func(s *styles.Style) {
		// s.Display = styles.Flex
		// s.Wrap = true
		// s.Direction = styles.Row
		s.Display = styles.Grid
		s.Columns = 32
	})
	fb.Maker(func(p *tree.Plan) {
		if fb.Font == nil {
			return
		}
		for i, gid := range fb.RuneMap.Values {
			r := fb.RuneMap.Keys[i]
			nm := string(r) + "_" + strconv.Itoa(int(r))
			tree.AddAt(p, nm, func(w *Glyph) {
				w.SetBrowser(fb).SetRune(r).SetGID(gid)
			})
		}
	})
}

func (fb *Browser) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.OpenFile).SetIcon(icons.Open).SetKey(keymap.Open)
		w.Args[0].SetValue(fb.Filename).SetTag(`extension:".ttf"`)
	})
}
