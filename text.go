// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"io"
	"math"

	"log"
	"unicode"
	"unicode/utf8"

	"github.com/chewxy/math32"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/net/html/charset"
)

// text.go contains all the core text rendering and formatting code -- see
// font.go for basic font-level style and management
//
// Styling, Formatting / Layout, and Rendering are each handled separately as
// three different levels in the stack -- simplifies many things to separate
// in this way, and makes the final render pass maximally efficient and
// high-performance, at the potential cost of some memory redundancy.

////////////////////////////////////////////////////////////////////////////////////////
// RuneRender

// RuneRender contains fully explicit data needed for rendering a single rune
// -- Face and Color can be nil after first element, in which case the last
// non-nil is used -- likely slightly more efficient to avoid setting all
// those pointers -- float32 values used to support better accuracy when
// transforming points
type RuneRender struct {
	Face    font.Face       `desc:"fully-specified font rendering info, includes fully computed font size -- this is exactly what will be drawn -- no further transforms"`
	Color   color.Color     `desc:"color to draw characters in"`
	BgColor color.Color     `desc:"background color to fill background of color -- for highlighting, <mark> tag, etc -- unlike Face, Color, this must be non-nil for every case that uses it, as nil is also used for default transparent background"`
	Deco    TextDecorations `desc:"additional decoration to apply -- underline, strike-through, etc -- also used for encoding a few special layout hints to pass info from styling tags to separate layout algorithms (e.g., <P> vs <BR>)"`
	RelPos  Vec2D           `desc:"relative position from start of TextRender for the lower-left baseline rendering position of the font character"`
	Size    Vec2D           `desc:"size of the rune itself, exclusive of spacing that might surround it"`
	RotRad  float32         `desc:"rotation in radians for this character, relative to its lower-left baseline rendering position"`
	ScaleX  float32         `desc:"scaling of the X dimension, in case of non-uniform scaling, 0 = no separate scaling"`
}

// HasNil returns error if any of the key info (face, color) is nil -- only
// the first element must be non-nil
func (rr *RuneRender) HasNil() error {
	if rr.Face == nil {
		return errors.New("gi.RuneRender: Face is nil")
	}
	if rr.Color == nil {
		return errors.New("gi.RuneRender: Color is nil")
	}
	// note: BgColor can be nil -- transparent
	return nil
}

// CurFace is convenience for updating current font face if non-nil
func (rr *RuneRender) CurFace(curFace font.Face) font.Face {
	if rr.Face != nil {
		return rr.Face
	}
	return curFace
}

// CurColor is convenience for updating current color if non-nil
func (rr *RuneRender) CurColor(curColor color.Color) color.Color {
	if rr.Color != nil {
		return rr.Color
	}
	return curColor
}

// RelPosAfterLR returns the relative position after given rune for LR order: RelPos.X + Size.X
func (rr *RuneRender) RelPosAfterLR() float32 {
	return rr.RelPos.X + rr.Size.X
}

// RelPosAfterRL returns the relative position after given rune for RL order: RelPos.X - Size.X
func (rr *RuneRender) RelPosAfterRL() float32 {
	return rr.RelPos.X - rr.Size.X
}

// RelPosAfterTB returns the relative position after given rune for TB order: RelPos.Y + Size.Y
func (rr *RuneRender) RelPosAfterTB() float32 {
	return rr.RelPos.Y + rr.Size.Y
}

//////////////////////////////////////////////////////////////////////////////////
//  SpanRender

// SpanRender contains fully explicit data needed for rendering a span of text
// as a slice of runes, with rune and RuneRender elements in one-to-one
// correspondence (but any nil values will use prior non-nil value -- first
// rune must have all non-nil). Text can be oriented in any direction -- the
// only constraint is that it starts from a single starting position.
// Typically only text within a span will obey kerning.  In standard
// TextRender context, each span is one line of text -- should not have new
// lines within the span itself.  In SVG special cases (e.g., TextPath), it
// can be anything.  It is NOT synonymous with the HTML <span> tag, as many
// styling applications of that tag can be accommodated within a larger
// span-as-line.  The first RuneRender RelPos for LR text should be at X=0
// (LastPos = 0 for RL) -- i.e., relpos positions are minimal for given span.
type SpanRender struct {
	Text    []rune          `desc:"text as runes"`
	Render  []RuneRender    `desc:"render info for each rune in one-to-one correspondence"`
	RelPos  Vec2D           `desc:"position for start of text relative to an absolute coordinate that is provided at the time of rendering -- individual rune RelPos are added to this plus the render-time offset to get the final position"`
	LastPos Vec2D           `desc:"rune position for further edge of last rune -- for standard flat strings this is the overall length of the string -- used for size / layout computations"`
	Dir     TextDirections  `desc:"where relevant, this is the (default, dominant) text direction for the span"`
	HasDeco TextDecorations `desc:"mask of decorations that have been set on this span -- optimizes rendering passes"`
}

// Init initializes a new span with given capacity
func (sr *SpanRender) Init(capsz int) {
	sr.Text = make([]rune, 0, capsz)
	sr.Render = make([]RuneRender, 0, capsz)
	sr.HasDeco = 0
}

// IsValid ensures that at least some text is represented and the sizes of
// Text and Render slices are the same, and that the first render info is non-nil
func (sr *SpanRender) IsValid() error {
	if len(sr.Text) == 0 {
		return errors.New("gi.TextRender: Text is empty")
	}
	if len(sr.Text) != len(sr.Render) {
		return errors.New("gi.TextRender: Render length != Text length")
	}
	return sr.Render[0].HasNil()
}

// SizeHV computes the size of the text span from the first char to the last
// position, which is valid for purely horizontal or vertical text lines --
// either X or Y will be zero depending on orientation
func (sr *SpanRender) SizeHV() Vec2D {
	if sr.IsValid() != nil {
		return Vec2D{}
	}
	sz := sr.Render[0].RelPos.Sub(sr.LastPos)
	if sz.X < 0 {
		sz.X = -sz.X
	}
	if sz.Y < 0 {
		sz.Y = -sz.Y
	}
	return sz
}

// AppendRune adds one rune and associated formatting info
func (sr *SpanRender) HasDecoUpdate(bg color.Color, deco TextDecorations) {
	sr.HasDeco |= deco
	if bg != nil {
		bitflag.Set32((*int32)(&sr.HasDeco), int(DecoBgColor))
	}
}

// AppendRune adds one rune and associated formatting info
func (sr *SpanRender) AppendRune(r rune, face font.Face, clr, bg color.Color, deco TextDecorations) {
	sr.Text = append(sr.Text, r)
	rr := RuneRender{Face: face, Color: clr, BgColor: bg, Deco: deco}
	sr.Render = append(sr.Render, rr)
	sr.HasDecoUpdate(bg, deco)
}

// AppendString adds string and associated formatting info, optimized with
// only first rune having non-nil face and color settings
func (sr *SpanRender) AppendString(str string, face font.Face, clr, bg color.Color, deco TextDecorations) {
	sz := len(str)
	if sz == 0 {
		return
	}
	sr.Text = append(sr.Text, []rune(str)...)
	rr := RuneRender{Face: face, Color: clr, BgColor: bg, Deco: deco}
	sr.HasDecoUpdate(bg, deco)
	sr.Render = append(sr.Render, rr)
	for i := 1; i < sz; i++ { // optimize by setting rest to nil for same
		rp := RuneRender{Deco: deco, BgColor: bg}
		sr.Render = append(sr.Render, rp)
	}
}

// SetRenders sets rendering parameters based on style
func (sr *SpanRender) SetRenders(sty *FontStyle, noBG bool, rot, scalex float32) {
	sz := len(sr.Text)
	if sz == 0 {
		return
	}

	bgc := (color.Color)(&sty.BgColor.Color)
	if noBG {
		bgc = nil
	}

	sr.HasDecoUpdate(bgc, sty.Deco)
	sr.Render = make([]RuneRender, sz)
	sr.Render[0].Face = sty.Face
	sr.Render[0].Color = sty.Color
	sr.Render[0].BgColor = bgc
	sr.Render[0].RotRad = rot
	sr.Render[0].ScaleX = scalex
	if bgc != nil {
		for i := range sr.Text {
			sr.Render[i].BgColor = bgc
		}
	}
	if rot != 0 || scalex != 0 {
		for i := range sr.Text {
			sr.Render[i].RotRad = rot
			sr.Render[i].ScaleX = scalex
		}
	}
	if sty.Deco != DecoNone {
		for i := range sr.Text {
			sr.Render[i].Deco = sty.Deco
		}
	}
}

// SetString initializes to given plain text string, with given default style
// parameters that are set for the first render element -- constructs Render
// slice of same size as Text
func (sr *SpanRender) SetString(str string, sty *FontStyle, noBG bool, rot, scalex float32) {
	sr.Text = []rune(str)
	sr.SetRenders(sty, noBG, rot, scalex)
}

// SetRunes initializes to given plain rune string, with given default style
// arameters that are set for the first render element -- constructs Render
// slice of same size as Text
func (sr *SpanRender) SetRunes(str []rune, sty *FontStyle, noBG bool, rot, scalex float32) {
	sr.Text = str
	sr.SetRenders(sty, noBG, rot, scalex)
}

// SetRunePosLR sets relative positions of each rune using a flat
// left-to-right text layout, based on font size info and additional extra
// letter and word spacing parameters (which can be negative)
func (sr *SpanRender) SetRunePosLR(letterSpace, wordSpace float32) {
	if err := sr.IsValid(); err != nil {
		// log.Println(err)
		return
	}
	sr.Dir = LRTB
	sz := len(sr.Text)
	prevR := rune(-1)
	lspc := letterSpace
	wspc := wordSpace
	var fpos float32
	curFace := sr.Render[0].Face
	for i, r := range sr.Text {
		rr := &(sr.Render[i])
		curFace = rr.CurFace(curFace)
		fht := curFace.Metrics().Ascent + curFace.Metrics().Descent
		if prevR >= 0 {
			fpos += FixedToFloat32(curFace.Kern(prevR, r))
		}
		rr.RelPos.X = fpos
		rr.RelPos.Y = 0

		if bitflag.Has32(int32(rr.Deco), int(DecoSuper)) {
			rr.RelPos.Y = -0.45 * FixedToFloat32(curFace.Metrics().Ascent)
		}
		if bitflag.Has32(int32(rr.Deco), int(DecoSub)) {
			rr.RelPos.Y = 0.15 * FixedToFloat32(curFace.Metrics().Ascent)
		}

		a, ok := curFace.GlyphAdvance(r)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		a32 := FixedToFloat32(a)
		rr.Size = Vec2D{a32, FixedToFloat32(fht)}
		fpos += a32
		if i < sz-1 {
			fpos += lspc
			if unicode.IsSpace(r) {
				fpos += wspc
			}
		}
		prevR = r
	}
	sr.LastPos.X = fpos
	sr.LastPos.Y = 0
}

// FindWrapPosLR finds a position to do word wrapping to fit within trgSize --
// RelPos positions must have already been set (e.g., SetRunePosLR)
func (sr *SpanRender) FindWrapPosLR(trgSize, curSize float32) int {
	sz := len(sr.Text)
	idx := int(float32(sz) * (trgSize / curSize))
	if idx >= sz {
		idx = sz - 1
	}
	for idx >= 0 && !unicode.IsSpace(sr.Text[idx]) {
		idx--
	}
	csz := sr.Render[idx].RelPos.X
	lstgoodi := -1
	if csz <= trgSize {
		lstgoodi = idx
	}
	for {
		if csz > trgSize {
			for idx > 0 {
				if unicode.IsSpace(sr.Text[idx]) {
					csz = sr.Render[idx].RelPos.X
					if csz <= trgSize {
						return idx
					}
				}
				idx--
			}
			return -1 // oops.
		} else { // too small, go up
			for idx < sz {
				if unicode.IsSpace(sr.Text[idx]) {
					csz = sr.Render[idx].RelPos.X
					if csz <= trgSize {
						if csz == trgSize {
							return idx
						}
						lstgoodi = idx
					} else if lstgoodi > 0 {
						return lstgoodi
					} else {
						break // go back down
					}
				}
				idx++
			}
			return lstgoodi
		}
	}
	return -1
}

// ZeroPos ensures that the positions start at 0, for LR direction
func (sr *SpanRender) ZeroPosLR() {
	sz := len(sr.Text)
	if sz == 0 {
		return
	}
	sx := sr.Render[0].RelPos.X
	if sx == 0 {
		return
	}
	for i, _ := range sr.Render {
		sr.Render[i].RelPos.X -= sx
	}
	sr.LastPos.X -= sx
}

// TrimSpace trims leading and trailing space elements from span, and updates
// the relative positions accordingly, for LR direction
func (sr *SpanRender) TrimSpaceLR() {
	for range sr.Text {
		if unicode.IsSpace(sr.Text[0]) {
			sr.Text = sr.Text[1:]
			sr.Render = sr.Render[1:]
		} else {
			break
		}
	}
	sr.ZeroPosLR()
	for range sr.Text {
		lidx := len(sr.Text) - 1
		if unicode.IsSpace(sr.Text[lidx]) {
			sr.Text = sr.Text[:lidx]
			sr.Render = sr.Render[:lidx]
			lidx--
			if lidx >= 0 {
				sr.LastPos.X = sr.Render[lidx].RelPosAfterLR()
			} else {
				sr.LastPos.X = sr.Render[0].Size.X
			}
		} else {
			break
		}
	}
}

// SplitAt splits current span at given index, returning a new span with
// remainder after index -- space is trimmed from both spans and relative
// positions updated, for LR direction
func (sr *SpanRender) SplitAtLR(idx int) *SpanRender {
	if idx <= 0 || idx >= len(sr.Text)-1 { // shouldn't happen
		return nil
	}
	nsr := SpanRender{Text: sr.Text[idx:], Render: sr.Render[idx:], Dir: sr.Dir, HasDeco: sr.HasDeco}
	sr.Text = sr.Text[:idx]
	sr.Render = sr.Render[:idx]
	sr.LastPos.X = sr.Render[idx-1].RelPosAfterLR()
	sr.TrimSpaceLR()
	nsr.TrimSpaceLR()
	// go back and find latest face and color -- each sr must start with valid one
	nrr0 := &(nsr.Render[0])
	for i := len(sr.Render) - 1; i >= 0; i-- {
		srr := sr.Render[i]
		if nrr0.Face == nil && srr.Face != nil {
			nrr0.Face = srr.Face
		}
		if nrr0.Color == nil && srr.Color != nil {
			nrr0.Color = srr.Color
		}
	}
	return &nsr
}

// todo: TB, RL cases -- layout is complicated.. with unicode-bidi, direction,
// writing-mode styles all interacting: https://www.w3.org/TR/SVG11/text.html#TextLayout

//////////////////////////////////////////////////////////////////////////////////
//  TextRender

// TextRender contains one or more SpanRender elements, typically with each
// representing a separate line of text (but they can be anything).
type TextRender struct {
	Spans []SpanRender
	Size  Vec2D          `desc:"last size of overall rendered text"`
	Dir   TextDirections `desc:"where relevant, this is the (default, dominant) text direction for the span"`
}

// InsertSpan inserts a new span at given index
func (tr *TextRender) InsertSpan(at int, ns *SpanRender) {
	sz := len(tr.Spans)
	tr.Spans = append(tr.Spans, SpanRender{})
	if at > sz-1 {
		tr.Spans[sz] = *ns
		return
	}
	copy(tr.Spans[at+1:], tr.Spans[at:])
	tr.Spans[at] = *ns
}

// Render does text rendering into given image, within given bounds, at given
// absolute position offset (specifying position of text baseline) -- any
// applicable transforms (aside from the char-specific rotation in Render)
// must be applied in advance in computing the relative positions of the
// runes, and the overall font size, etc.  todo: does not currently support
// stroking, only filling of text -- probably need to grab path from font and
// use paint rendering for stroking
func (tr *TextRender) Render(rs *RenderState, pos Vec2D) {
	pr := prof.Start("RenderText")
	defer pr.End()

	rs.BackupPaint()
	defer rs.RestorePaint()

	rs.PushXForm(Identity2D()) // needed for SVG
	defer rs.PopXForm()
	rs.XForm = Identity2D()

	for _, sr := range tr.Spans {
		if sr.IsValid() != nil {
			continue
		}
		curFace := sr.Render[0].Face
		curColor := sr.Render[0].Color
		tpos := pos.Add(sr.RelPos)

		d := &font.Drawer{
			Dst:  rs.Image,
			Src:  image.NewUniform(curColor),
			Face: curFace,
		}

		// todo: cache flags if these are actually needed
		if bitflag.Has32(int32(sr.HasDeco), int(DecoBgColor)) {
			sr.RenderBg(rs, tpos)
		}
		if bitflag.HasAny32(int32(sr.HasDeco), int(DecoUnderline), int(DecoDottedUnderline)) {
			sr.RenderUnderline(rs, tpos)
		}
		if bitflag.Has32(int32(sr.HasDeco), int(DecoOverline)) {
			sr.RenderLine(rs, tpos, DecoOverline, 1.1)
		}

		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			curFace = rr.CurFace(curFace)
			dsc32 := FixedToFloat32(curFace.Metrics().Descent)
			rp := tpos.Add(rr.RelPos)
			scx := float32(1)
			if rr.ScaleX != 0 {
				scx = rr.ScaleX
			}
			tx := Scale2D(scx, 1).Rotate(rr.RotRad)
			ll := rp.Add(tx.TransformVectorVec2D(Vec2D{0, dsc32}))
			ur := ll.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, -rr.Size.Y}))
			if int(math32.Floor(ll.X)) > rs.Bounds.Max.X || int(math32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
				int(math32.Ceil(ur.X)) < rs.Bounds.Min.X || int(math32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
				continue
			}
			if rr.Color != nil {
				curColor = rr.Color
				d.Src = image.NewUniform(curColor)
			}
			d.Face = curFace
			d.Dot = rp.Fixed()
			dr, mask, maskp, _, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				continue
			}
			idr := dr.Intersect(rs.Bounds)
			soff := idr.Min.Sub(dr.Min)
			if rr.RotRad == 0 && (rr.ScaleX == 0 || rr.ScaleX == 1) {
				draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
			} else {
				srect := dr.Sub(dr.Min)
				dbase := Vec2D{rp.X - float32(dr.Min.X), rp.Y - float32(dr.Min.Y)}

				transformer := draw.BiLinear
				fx, fy := float32(dr.Min.X), float32(dr.Min.Y)
				m := Translate2D(fx+dbase.X, fy+dbase.Y).Scale(scx, 1).Rotate(rr.RotRad).Translate(-dbase.X, -dbase.Y)
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, srect, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
		}
		if bitflag.Has32(int32(sr.HasDeco), int(DecoLineThrough)) {
			sr.RenderLine(rs, tpos, DecoLineThrough, 0.25)
		}
	}
}

// RenderBg renders the background behind chars
func (sr *SpanRender) RenderBg(rs *RenderState, tpos Vec2D) {
	curFace := sr.Render[0].Face
	didLast := false
	// first := true
	pc := &rs.Paint

	for i := range sr.Text {
		rr := &(sr.Render[i])
		if rr.BgColor == nil {
			if didLast {
				pc.Fill(rs)
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := FixedToFloat32(curFace.Metrics().Descent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.TransformVectorVec2D(Vec2D{0, dsc32}))
		ur := ll.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, -rr.Size.Y}))
		if int(math32.Floor(ll.X)) > rs.Bounds.Max.X || int(math32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(math32.Ceil(ur.X)) < rs.Bounds.Min.X || int(math32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
			if didLast {
				pc.Fill(rs)
			}
			didLast = false
			continue
		}
		pc.FillStyle.Color.SetColor(rr.BgColor)
		szt := Vec2D{rr.Size.X, -rr.Size.Y}
		sp := rp.Add(tx.TransformVectorVec2D(Vec2D{0, dsc32}))
		ul := sp.Add(tx.TransformVectorVec2D(Vec2D{0, szt.Y}))
		lr := sp.Add(tx.TransformVectorVec2D(Vec2D{szt.X, 0}))
		pc.DrawPolygon(rs, []Vec2D{sp, ul, ur, lr})
		didLast = true
	}
	if didLast {
		pc.Fill(rs)
	}
}

// RenderUnderline renders the underline for span -- ensures continuity to do it all at once
func (sr *SpanRender) RenderUnderline(rs *RenderState, tpos Vec2D) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	didLast := false
	pc := &rs.Paint

	for i := range sr.Text {
		rr := &(sr.Render[i])
		if !bitflag.HasAny32(int32(rr.Deco), int(DecoUnderline), int(DecoDottedUnderline)) {
			if didLast {
				pc.Stroke(rs)
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := FixedToFloat32(curFace.Metrics().Descent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.TransformVectorVec2D(Vec2D{0, dsc32}))
		ur := ll.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, -rr.Size.Y}))
		if int(math32.Floor(ll.X)) > rs.Bounds.Max.X || int(math32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(math32.Ceil(ur.X)) < rs.Bounds.Min.X || int(math32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
			if didLast {
				pc.Stroke(rs)
			}
			continue
		}
		if rr.Color != nil {
			curColor = rr.Color
		}
		dw := .05 * rr.Size.Y
		if !didLast {
			pc.StrokeStyle.Width.Dots = dw
			pc.StrokeStyle.Color.SetColor(curColor)
		}
		if bitflag.Has32(int32(rr.Deco), int(DecoDottedUnderline)) {
			pc.StrokeStyle.Dashes = []float64{float64(dw), float64(dw)}
		}
		sp := rp.Add(tx.TransformVectorVec2D(Vec2D{0, 2 * dw}))
		ep := rp.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, 2 * dw}))

		if didLast {
			pc.LineTo(rs, sp.X, sp.Y)
		} else {
			pc.NewSubPath(rs)
			pc.MoveTo(rs, sp.X, sp.Y)
		}
		pc.LineTo(rs, ep.X, ep.Y)
		didLast = true
	}
	if didLast {
		pc.Stroke(rs)
	}
	pc.StrokeStyle.Dashes = nil
}

// RenderLine renders overline or line-through -- anything that is a function of ascent
func (sr *SpanRender) RenderLine(rs *RenderState, tpos Vec2D, deco TextDecorations, ascPct float32) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	didLast := false
	pc := &rs.Paint

	for i := range sr.Text {
		rr := &(sr.Render[i])
		if !bitflag.Has32(int32(rr.Deco), int(deco)) {
			if didLast {
				pc.Stroke(rs)
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := FixedToFloat32(curFace.Metrics().Descent)
		asc32 := FixedToFloat32(curFace.Metrics().Ascent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.TransformVectorVec2D(Vec2D{0, dsc32}))
		ur := ll.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, -rr.Size.Y}))
		if int(math32.Floor(ll.X)) > rs.Bounds.Max.X || int(math32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(math32.Ceil(ur.X)) < rs.Bounds.Min.X || int(math32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
			if didLast {
				pc.Stroke(rs)
			}
			continue
		}
		if rr.Color != nil {
			curColor = rr.Color
		}
		dw := 0.05 * rr.Size.Y
		if !didLast {
			pc.StrokeStyle.Width.Dots = dw
			pc.StrokeStyle.Color.SetColor(curColor)
		}
		yo := ascPct * asc32
		sp := rp.Add(tx.TransformVectorVec2D(Vec2D{0, -yo}))
		ep := rp.Add(tx.TransformVectorVec2D(Vec2D{rr.Size.X, -yo}))

		if didLast {
			pc.LineTo(rs, sp.X, sp.Y)
		} else {
			pc.NewSubPath(rs)
			pc.MoveTo(rs, sp.X, sp.Y)
		}
		pc.LineTo(rs, ep.X, ep.Y)
		didLast = true
	}
	if didLast {
		pc.Stroke(rs)
	}
}

// Render at given top position -- uses first font info to compute baseline
// offset and calls overall Render -- convenience for simple widget rendering
// without layouts
func (tr *TextRender) RenderTopPos(rs *RenderState, tpos Vec2D) {
	if len(tr.Spans) == 0 {
		return
	}
	sr := &(tr.Spans[0])
	if sr.IsValid() != nil {
		return
	}
	curFace := sr.Render[0].Face
	pos := tpos
	pos.Y += FixedToFloat32(curFace.Metrics().Ascent)
	tr.Render(rs, pos)
}

// SetString is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single SpanRender with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after.  Be sure that LoadFont has been run so a
// valid Face is available.  noBG ignores any BgColor in font style, and never
// renders background color
func (tr *TextRender) SetString(str string, fontSty *FontStyle, txtSty *TextStyle, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]SpanRender, 1)
	}
	sr := &(tr.Spans[0])
	sr.SetString(str, fontSty, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Metrics().Height
	tr.Size = Vec2D{ssz.X, FixedToFloat32(vht)}

}

// SetRunes is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single SpanRender with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after Be sure that LoadFont has been run so a
// valid Face is available.  noBG ignores any BgColor in font style, and never
// renders background color
func (tr *TextRender) SetRunes(str []rune, fontSty *FontStyle, txtSty *TextStyle, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]SpanRender, 1)
	}
	sr := &(tr.Spans[0])
	sr.SetRunes(str, fontSty, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Metrics().Height
	tr.Size = Vec2D{ssz.X, FixedToFloat32(vht)}
}

// SetHTML sets text by decoding all standard inline HTML text style
// formatting tags in the string and sets the per-character font information
// appropriately, using given font style info.  <P> and <BR> tags create new
// spans, with <P> marking start of subsequent span with DecoParaStart.
// Critically, it does NOT deal at all with layout (positioning) -- only sets
// font, color, and decoration info, and strips out the tags it processes --
// result can then be processed by different layout algorithms as needed.
// cssAgg, if non-nil, should contain CSSAgg properties -- will be tested for
// special css styling of each element
func (tr *TextRender) SetHTML(str string, font *FontStyle, ctxt *units.Context, cssAgg ki.Props) {
	sz := len(str)
	if sz == 0 {
		return
	}
	tr.Spans = make([]SpanRender, 1)
	curSp := &(tr.Spans[0])
	initsz := kit.MinInt(sz, 1020)
	curSp.Init(initsz)

	spcstr := bytes.Join(bytes.Fields([]byte(str)), []byte(" "))

	reader := bytes.NewReader(spcstr)
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel

	font.LoadFont(ctxt, "")

	// set when a </p> is encountered
	nextIsParaStart := false

	fstack := make([]*FontStyle, 1, 10)
	fstack[0] = font
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("gi.TextRender DecodeHTML parsing error: %v\n", err)
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			curf := fstack[len(fstack)-1]
			fs := *curf
			nm := se.Name.Local
			// https://www.w3schools.com/cssref/css_default_values.asp
			switch nm {
			case "b", "strong":
				fs.Weight = WeightBold
				fs.LoadFont(ctxt, "")
			case "i", "em", "var", "cite":
				fs.Style = FontItalic
				fs.LoadFont(ctxt, "")
			case "ins":
				fallthrough
			case "u":
				fs.SetDeco(DecoUnderline)
			case "s", "del", "strike":
				fs.SetDeco(DecoLineThrough)
			case "sup":
				fs.SetDeco(DecoSuper)
				curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
				curpts -= 2
				fs.Size = units.NewValue(float32(curpts), units.Pt)
				fs.Size.ToDots(ctxt)
				fs.LoadFont(ctxt, "")
			case "sub":
				fs.SetDeco(DecoSub)
				fallthrough
			case "small":
				curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
				curpts -= 2
				fs.Size = units.NewValue(float32(curpts), units.Pt)
				fs.Size.ToDots(ctxt)
				fs.LoadFont(ctxt, "")
			case "big":
				curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
				curpts += 2
				fs.Size = units.NewValue(float32(curpts), units.Pt)
				fs.Size.ToDots(ctxt)
				fs.LoadFont(ctxt, "")
			case "xx-small", "x-small", "smallf", "medium", "large", "x-large", "xx-large":
				fs.Size = units.NewValue(FontSizePoints[nm], units.Pt)
				fs.Size.ToDots(ctxt)
				fs.LoadFont(ctxt, "")
			case "mark":
				fs.BgColor.SetColor(Prefs.HighlightColor)
			case "abbr", "acronym":
				fs.SetDeco(DecoDottedUnderline)
			case "tt", "kbd", "samp", "code":
				fs.Family = "monospace"
				fs.LoadFont(ctxt, "")
			case "span":
				if len(se.Attr) > 0 {
					sprop := make(ki.Props, len(se.Attr))
					for _, attr := range se.Attr {
						if attr.Name.Local == "style" {
							SetStylePropsXML(attr.Value, sprop)
						} else {
							sprop[attr.Name.Local] = attr.Value
						}
					}
					fs.SetStyleProps(nil, sprop)
					fs.LoadFont(ctxt, "")
				}
			case "q":
				curf := fstack[len(fstack)-1]
				atStart := len(curSp.Text) == 0
				curSp.AppendRune('“', curf.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
				if nextIsParaStart && atStart {
					bitflag.Set32((*int32)(&(curSp.Render[0].Deco)), int(DecoParaStart))
				}
				nextIsParaStart = false
			case "dfn":
				// no default styling
			case "bdo":
				// bidirectional override..
			case "p":
			case "br":
			default:
				log.Printf("gi.TextRender SetHTML tag not recognized: %v\n", nm)
			}
			if cssAgg != nil {
				fs.StyleCSS(nm, cssAgg, ctxt)
			}
			fstack = append(fstack, &fs)
		case xml.EndElement:
			switch se.Name.Local {
			case "p":
				tr.Spans = append(tr.Spans, SpanRender{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
				nextIsParaStart = true
			case "br":
				tr.Spans = append(tr.Spans, SpanRender{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
			case "q":
				curf := fstack[len(fstack)-1]
				curSp.AppendRune('”', curf.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
			}
			if len(fstack) > 1 {
				fstack = fstack[:len(fstack)-1]
			}
		case xml.CharData:
			curf := fstack[len(fstack)-1]
			atStart := len(curSp.Text) == 0
			curSp.AppendString(string(se), curf.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
			if nextIsParaStart && atStart {
				bitflag.Set32((*int32)(&(curSp.Render[0].Deco)), int(DecoParaStart))
			}
			nextIsParaStart = false
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  TextStyle

// TextStyle is used for layout-level (widget, html-style) text styling --
// FontStyle contains all the lower-level text rendering info used in SVG --
// most of these are inherited
type TextStyle struct {
	Align            Align          `xml:"text-align" inherit:"true" desc:"how to align text, horizontally"`
	AlignV           Align          `xml:"-" json:"-" desc:"vertical alignment of text -- copied from layout style AlignV"`
	Anchor           TextAnchors    `xml:"text-anchor" inherit:"true" desc:"for svg rendering only: determines the alignment relative to text position coordinate: for RTL start is right, not left, and start is top for TB"`
	LetterSpacing    units.Value    `xml:"letter-spacing" desc:"spacing between characters and lines"`
	WordSpacing      units.Value    `xml:"word-spacing" inherit:"true" desc:"extra space to add between words"`
	LineHeight       float32        `xml:"line-height" inherit:"true" desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (todo: specific values such as pixels are not supported, in order to properly support percentage) -- text is centered within the overall lineheight"`
	UnicodeBidi      UnicodeBidi    `xml:"unicode-bidi" inherit:"true" desc:"determines how to treat unicode bidirectional information"`
	Direction        TextDirections `xml:"direction" inherit:"true" desc:"direction of text -- only applicable for unicode-bidi = bidi-override or embed -- applies to all text elements"`
	WritingMode      TextDirections `xml:"writing-mode" inherit:"true" desc:"overall writing mode -- only for text elements, not tspan"`
	OrientationVert  float32        `xml:"glyph-orientation-vertical" inherit:"true" desc:"for TBRL writing mode (only), determines orientation of alphabetic characters -- 90 is default (rotated) -- 0 means keep upright"`
	OrientationHoriz float32        `xml:"glyph-orientation-horizontal" inherit:"true" desc:"for horizontal LR/RL writing mode (only), determines orientation of all characters -- 0 is default (upright)"`
	Indent           units.Value    `xml:"text-indent" inherit:"true" desc:"how much to indent the first line in a paragraph"`
	TabSize          units.Value    `xml:"tab-size" inherit:"true" desc:"tab size"`
	WordWrap         bool           `xml:"word-wrap" inherit:"true" desc:"wrap text within a given size"`
	// todo:
	// page-break options
	// text-justify  inherit:"true" -- how to justify text
	// text-overflow -- clip, ellipsis, string..
	// text-shadow  inherit:"true"
	// text-transform --  inherit:"true" uppercase, lowercase, capitalize
	// user-select -- can user select text?
	// white-space -- what to do with white-space  inherit:"true"
	// word-break  inherit:"true"
}

func (ts *TextStyle) Defaults() {
	ts.WordWrap = false
	ts.Align = AlignLeft
	ts.AlignV = AlignBaseline
	ts.Direction = LTR
	ts.OrientationVert = 90
}

// SetStylePost applies any updates after generic xml-tag property setting
func (ts *TextStyle) SetStylePost(props ki.Props) {
}

// EffLineHeight returns the effective line height (taking into account 0 value)
func (ts *TextStyle) EffLineHeight() float32 {
	if ts.LineHeight == 0 {
		return 1.0
	}
	return ts.LineHeight
}

// AlignFactors gets basic text alignment factors
func (ts *TextStyle) AlignFactors() (ax, ay float32) {
	ax = 0.0
	ay = 0.0
	hal := ts.Align
	switch {
	case IsAlignMiddle(hal):
		ax = 0.5 // todo: determine if font is horiz or vert..
	case IsAlignEnd(hal):
		ax = 1.0
	}
	val := ts.AlignV
	switch {
	case val == AlignSub:
		ay = -0.15 // todo: fixme -- need actual font metrics
	case val == AlignSuper:
		ay = 0.65 // todo: fixme
	case IsAlignStart(val):
		ay = 0.9 // todo: need to find out actual baseline
	case IsAlignMiddle(val):
		ay = 0.45 // todo: determine if font is horiz or vert..
	case IsAlignEnd(val):
		ay = -0.1 // todo: need actual baseline
	}
	return
}

//////////////////////////////////////////////////////////////////////////////////
//  TextStyle-based Layout Routines

// LayoutStdLR does basic standard layout of text in LR direction, assigning
// relative positions to spans and runes according to given styles, and given
// size overall box (nonzero values used to constrain). Returns total
// resulting size box for text.  Font face in FontStyle is used for
// determining line spacing here -- other versions can do more expensive
// calculations of variable line spacing as needed.
func (tr *TextRender) LayoutStdLR(txtSty *TextStyle, fontSty *FontStyle, ctxt *units.Context, size Vec2D) Vec2D {
	if len(tr.Spans) == 0 {
		return Vec2DZero
	}

	pr := prof.Start("TextRenderLayout")
	defer pr.End()

	tr.Dir = LRTB
	fontSty.LoadFont(ctxt, "")
	asc := FixedToFloat32(fontSty.Face.Metrics().Ascent)
	dsc := FixedToFloat32(fontSty.Face.Metrics().Descent)
	fht := asc + dsc
	lspc := fontSty.Height * txtSty.EffLineHeight()
	lpad := (lspc - fht) / 2 // padding above / below text box for centering in line

	maxw := float32(0)

	// first pass gets rune positions and wraps text as needed, and gets max width
	si := 0
	for si < len(tr.Spans) {
		sr := &(tr.Spans[si])
		if sr.IsValid() != nil {
			si++
			continue
		}
		if sr.LastPos.X == 0 { // don't re-do unless necessary
			sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots)
		}
		ssz := sr.SizeHV()
		if size.X > 0 && ssz.X > size.X && txtSty.WordWrap {
			for {
				wp := sr.FindWrapPosLR(size.X, ssz.X)
				if wp > 0 {
					nsr := sr.SplitAtLR(wp)
					tr.InsertSpan(si+1, nsr)
					ssz = sr.SizeHV()
					if ssz.X > maxw {
						maxw = ssz.X
					}
					si++
					sr = &(tr.Spans[si]) // keep going with nsr
					sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots)
					ssz = sr.SizeHV()
					if ssz.X <= size.X {
						if ssz.X > maxw {
							maxw = ssz.X
						}
						break
					}
				} else {
					if ssz.X > maxw {
						maxw = ssz.X
					}
					break
				}
			}
		} else {
			if ssz.X > maxw {
				maxw = ssz.X
			}
		}
		si++
	}
	// have maxw, can do alignment cases..

	if maxw > size.X {
		size.X = maxw
	}

	// vertical alignment
	vht := lspc * float32(len(tr.Spans))
	if vht > size.Y {
		size.Y = vht
	}

	tr.Size = Vec2D{maxw, vht}

	vpad := float32(0) // padding at top to achieve vertical alignment
	vextra := size.Y - vht
	if vextra > 0 {
		switch {
		case IsAlignMiddle(txtSty.AlignV):
			vpad = vextra / 2
		case IsAlignEnd(txtSty.AlignV):
			vpad = vextra
		}
	}

	vbaseoff := lspc - lpad - dsc // offset of baseline within overall line
	vpos := vpad + vbaseoff

	for si := range tr.Spans {
		sr := &(tr.Spans[si])
		sr.RelPos.Y = vpos
		sr.RelPos.X = 0
		// todo: handle indent here look at para start -- sets +X -- also need in size above
		ssz := sr.SizeHV()
		hextra := size.X - ssz.X
		if hextra > 0 {
			switch {
			case IsAlignMiddle(txtSty.Align):
				sr.RelPos.X = hextra / 2
			case IsAlignEnd(txtSty.Align):
				sr.RelPos.X = hextra
			}
		}
		vpos += lspc
	}
	return size
}

//////////////////////////////////////////////////////////////////////////////////
//  Utilities

// NextRuneAt returns the next rune starting from given index -- could be at
// that index or some point thereafter -- returns utf8.RuneError if no valid
// rune could be found -- this should be a standard function!
func NextRuneAt(str string, idx int) rune {
	r, _ := utf8.DecodeRuneInString(str[idx:])
	return r
}
