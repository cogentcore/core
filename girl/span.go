// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"errors"
	"fmt"
	"image/color"
	"sync"
	"unicode"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/mat32"
	"golang.org/x/image/font"
)

// Span contains fully explicit data needed for rendering a span of text
// as a slice of runes, with rune and Rune elements in one-to-one
// correspondence (but any nil values will use prior non-nil value -- first
// rune must have all non-nil). Text can be oriented in any direction -- the
// only constraint is that it starts from a single starting position.
// Typically only text within a span will obey kerning.  In standard
// Text context, each span is one line of text -- should not have new
// lines within the span itself.  In SVG special cases (e.g., TextPath), it
// can be anything.  It is NOT synonymous with the HTML <span> tag, as many
// styling applications of that tag can be accommodated within a larger
// span-as-line.  The first Rune RelPos for LR text should be at X=0
// (LastPos = 0 for RL) -- i.e., relpos positions are minimal for given span.
type Span struct {
	Text    []rune               `desc:"text as runes"`
	Render  []Rune               `desc:"render info for each rune in one-to-one correspondence"`
	RelPos  mat32.Vec2           `desc:"position for start of text relative to an absolute coordinate that is provided at the time of rendering -- this typically includes the baseline offset to align all rune rendering there -- individual rune RelPos are added to this plus the render-time offset to get the final position"`
	LastPos mat32.Vec2           `desc:"rune position for further edge of last rune -- for standard flat strings this is the overall length of the string -- used for size / layout computations -- you do not add RelPos to this -- it is in same Text relative coordinates"`
	Dir     gist.TextDirections  `desc:"where relevant, this is the (default, dominant) text direction for the span"`
	HasDeco gist.TextDecorations `desc:"mask of decorations that have been set on this span -- optimizes rendering passes"`
}

// Init initializes a new span with given capacity
func (sr *Span) Init(capsz int) {
	sr.Text = make([]rune, 0, capsz)
	sr.Render = make([]Rune, 0, capsz)
	sr.HasDeco = 0
}

// IsValid ensures that at least some text is represented and the sizes of
// Text and Render slices are the same, and that the first render info is non-nil
func (sr *Span) IsValid() error {
	if len(sr.Text) == 0 {
		return errors.New("gi.Text: Text is empty")
	}
	if len(sr.Text) != len(sr.Render) {
		return fmt.Errorf("gi.Text: Render length %v != Text length %v for text: %v", len(sr.Render), len(sr.Text), string(sr.Text))
	}
	return sr.Render[0].HasNil()
}

// SizeHV computes the size of the text span from the first char to the last
// position, which is valid for purely horizontal or vertical text lines --
// either X or Y will be zero depending on orientation
func (sr *Span) SizeHV() mat32.Vec2 {
	if sr.IsValid() != nil {
		return mat32.Vec2{}
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

// RuneRelPos returns the relative (starting) position of the given rune index
// (adds Span RelPos and rune RelPos) -- this is typically the baseline
// position where rendering will start, not the upper left corner. if index >
// length, then uses LastPos
func (sr *Span) RuneRelPos(idx int) mat32.Vec2 {
	if idx >= len(sr.Render) {
		return sr.LastPos
	}
	return sr.RelPos.Add(sr.Render[idx].RelPos)
}

// RuneEndPos returns the relative ending position of the given rune index
// (adds Span RelPos and rune RelPos + rune Size.X for LR writing). If index >
// length, then uses LastPos
func (sr *Span) RuneEndPos(idx int) mat32.Vec2 {
	if idx >= len(sr.Render) {
		return sr.LastPos
	}
	spos := sr.RelPos.Add(sr.Render[idx].RelPos)
	spos.X += sr.Render[idx].Size.X
	return spos
}

// AppendRune adds one rune and associated formatting info
func (sr *Span) HasDecoUpdate(bg color.Color, deco gist.TextDecorations) {
	sr.HasDeco |= deco
	if bg != nil {
		bitflag.Set32((*int32)(&sr.HasDeco), int(gist.DecoBgColor))
	}
}

// IsNewPara returns true if this span starts a new paragraph
func (sr *Span) IsNewPara() bool {
	if len(sr.Render) == 0 {
		return false
	}
	return bitflag.Has32(int32(sr.Render[0].Deco), int(gist.DecoParaStart))
}

// SetNewPara sets this as starting a new paragraph
func (sr *Span) SetNewPara() {
	if len(sr.Render) > 0 {
		bitflag.Set32((*int32)(&sr.Render[0].Deco), int(gist.DecoParaStart))
	}
}

// AppendRune adds one rune and associated formatting info
func (sr *Span) AppendRune(r rune, face font.Face, clr, bg color.Color, deco gist.TextDecorations) {
	sr.Text = append(sr.Text, r)
	rr := Rune{Face: face, Color: clr, BgColor: bg, Deco: deco}
	sr.Render = append(sr.Render, rr)
	sr.HasDecoUpdate(bg, deco)
}

// AppendString adds string and associated formatting info, optimized with
// only first rune having non-nil face and color settings
func (sr *Span) AppendString(str string, face font.Face, clr, bg color.Color, deco gist.TextDecorations, sty *gist.Font, ctxt *units.Context) {
	if len(str) == 0 {
		return
	}
	ucfont := &gist.Font{}
	if oswin.TheApp != nil && oswin.TheApp.Platform() == oswin.MacOS {
		ucfont.Family = "Arial Unicode"
	} else {
		ucfont.Family = "Arial"
	}
	ucfont.Size = sty.Size
	OpenFont(ucfont, ctxt)

	nwr := []rune(str)
	sz := len(nwr)
	sr.Text = append(sr.Text, nwr...)
	rr := Rune{Face: face, Color: clr, BgColor: bg, Deco: deco}
	r := nwr[0]
	lastUc := false
	if r > 0xFF && unicode.IsSymbol(r) {
		rr.Face = ucfont.Face.Face
		lastUc = true
	}
	sr.HasDecoUpdate(bg, deco)
	sr.Render = append(sr.Render, rr)
	for i := 1; i < sz; i++ { // optimize by setting rest to nil for same
		rp := Rune{Deco: deco, BgColor: bg}
		r := nwr[i]
		if oswin.TheApp != nil && oswin.TheApp.Platform() == oswin.MacOS {
			if r > 0xFF && unicode.IsSymbol(r) {
				if !lastUc {
					rp.Face = ucfont.Face.Face
					lastUc = true
				}
			} else {
				if lastUc {
					rp.Face = face
					lastUc = false
				}
			}
		}
		sr.Render = append(sr.Render, rp)
	}
}

// SetRenders sets rendering parameters based on style
func (sr *Span) SetRenders(sty *gist.Font, ctxt *units.Context, noBG bool, rot, scalex float32) {
	sz := len(sr.Text)
	if sz == 0 {
		return
	}

	bgc := (color.Color)(&sty.BgColor.Color)
	if noBG {
		bgc = nil
	}

	ucfont := &gist.Font{}
	ucfont.Family = "Arial Unicode"
	ucfont.Size = sty.Size
	OpenFont(ucfont, ctxt)

	sr.HasDecoUpdate(bgc, sty.Deco)
	sr.Render = make([]Rune, sz)
	if sty.Face == nil {
		sr.Render[0].Face = ucfont.Face.Face
	} else {
		sr.Render[0].Face = sty.Face.Face
	}
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
	if sty.Deco != gist.DecoNone {
		for i := range sr.Text {
			sr.Render[i].Deco = sty.Deco
		}
	}
	// use unicode font for all non-ascii symbols
	lastUc := false
	for i, r := range sr.Text {
		if r > 0xFF && unicode.IsSymbol(r) {
			if !lastUc {
				sr.Render[i].Face = ucfont.Face.Face
				lastUc = true
			}
		} else {
			if lastUc {
				sr.Render[i].Face = sty.Face.Face
				lastUc = false
			}
		}
	}
}

// SetString initializes to given plain text string, with given default style
// parameters that are set for the first render element -- constructs Render
// slice of same size as Text
func (sr *Span) SetString(str string, sty *gist.Font, ctxt *units.Context, noBG bool, rot, scalex float32) {
	sr.Text = []rune(str)
	sr.SetRenders(sty, ctxt, noBG, rot, scalex)
}

// SetRunes initializes to given plain rune string, with given default style
// parameters that are set for the first render element -- constructs Render
// slice of same size as Text
func (sr *Span) SetRunes(str []rune, sty *gist.Font, ctxt *units.Context, noBG bool, rot, scalex float32) {
	sr.Text = str
	sr.SetRenders(sty, ctxt, noBG, rot, scalex)
}

// TextFontRenderMu mutex is required because multiple different goroutines
// associated with different windows can (and often will be) call font stuff
// at the same time (curFace.GlyphAdvance, rendering font) at the same time, on
// the same font face -- and that turns out not to work!
var TextFontRenderMu sync.Mutex

// SetRunePosLR sets relative positions of each rune using a flat
// left-to-right text layout, based on font size info and additional extra
// letter and word spacing parameters (which can be negative)
func (sr *Span) SetRunePosLR(letterSpace, wordSpace, chsz float32, tabSize int) {
	if err := sr.IsValid(); err != nil {
		// log.Println(err)
		return
	}
	sr.Dir = gist.LRTB
	sz := len(sr.Text)
	prevR := rune(-1)
	lspc := letterSpace
	wspc := wordSpace
	if tabSize == 0 {
		tabSize = 4
	}
	var fpos float32
	curFace := sr.Render[0].Face
	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()
	for i, r := range sr.Text {
		rr := &(sr.Render[i])
		curFace = rr.CurFace(curFace)

		fht := mat32.FromFixed(curFace.Metrics().Height)
		if prevR >= 0 {
			fpos += mat32.FromFixed(curFace.Kern(prevR, r))
		}
		rr.RelPos.X = fpos
		rr.RelPos.Y = 0

		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSuper)) {
			rr.RelPos.Y = -0.45 * mat32.FromFixed(curFace.Metrics().Ascent)
		}
		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSub)) {
			rr.RelPos.Y = 0.15 * mat32.FromFixed(curFace.Metrics().Ascent)
		}

		// todo: could check for various types of special unicode space chars here
		a, _ := curFace.GlyphAdvance(r)
		a32 := mat32.FromFixed(a)
		if a32 == 0 {
			a32 = .1 * fht // something..
		}
		rr.Size = mat32.Vec2{a32, fht}

		if r == '\t' {
			col := int(mat32.Ceil(fpos / chsz))
			curtab := col / tabSize
			curtab++
			col = curtab * tabSize
			cpos := chsz * float32(col)
			if cpos > fpos {
				fpos = cpos
			}
		} else {
			fpos += a32
			if i < sz-1 {
				fpos += lspc
				if unicode.IsSpace(r) {
					fpos += wspc
				}
			}
		}
		prevR = r
	}
	sr.LastPos.X = fpos
	sr.LastPos.Y = 0
}

// SetRunePosTB sets relative positions of each rune using a flat
// top-to-bottom text layout -- i.e., letters are in their normal
// upright orientation, but arranged vertically.
func (sr *Span) SetRunePosTB(letterSpace, wordSpace, chsz float32, tabSize int) {
	if err := sr.IsValid(); err != nil {
		// log.Println(err)
		return
	}
	sr.Dir = gist.TB
	sz := len(sr.Text)
	lspc := letterSpace
	wspc := wordSpace
	if tabSize == 0 {
		tabSize = 4
	}
	var fpos float32
	curFace := sr.Render[0].Face
	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()
	col := 0 // current column position -- todo: does NOT deal with indent
	for i, r := range sr.Text {
		rr := &(sr.Render[i])
		curFace = rr.CurFace(curFace)

		fht := mat32.FromFixed(curFace.Metrics().Height)
		rr.RelPos.X = 0
		rr.RelPos.Y = fpos

		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSuper)) {
			rr.RelPos.Y = -0.45 * mat32.FromFixed(curFace.Metrics().Ascent)
		}
		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSub)) {
			rr.RelPos.Y = 0.15 * mat32.FromFixed(curFace.Metrics().Ascent)
		}

		// todo: could check for various types of special unicode space chars here
		a, _ := curFace.GlyphAdvance(r)
		a32 := mat32.FromFixed(a)
		if a32 == 0 {
			a32 = .1 * fht // something..
		}
		rr.Size = mat32.Vec2{a32, fht}

		if r == '\t' {
			curtab := col / tabSize
			curtab++
			col = curtab * tabSize
			cpos := chsz * float32(col)
			if cpos > fpos {
				fpos = cpos
			}
		} else {
			fpos += fht
			col++
			if i < sz-1 {
				fpos += lspc
				if unicode.IsSpace(r) {
					fpos += wspc
				}
			}
		}
	}
	sr.LastPos.Y = fpos
	sr.LastPos.X = 0
}

// SetRunePosTBRot sets relative positions of each rune using a flat
// top-to-bottom text layout, with characters rotated 90 degress
// based on font size info and additional extra letter and word spacing
// parameters (which can be negative)
func (sr *Span) SetRunePosTBRot(letterSpace, wordSpace, chsz float32, tabSize int) {
	if err := sr.IsValid(); err != nil {
		// log.Println(err)
		return
	}
	sr.Dir = gist.TB
	sz := len(sr.Text)
	prevR := rune(-1)
	lspc := letterSpace
	wspc := wordSpace
	if tabSize == 0 {
		tabSize = 4
	}
	var fpos float32
	curFace := sr.Render[0].Face
	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()
	col := 0 // current column position -- todo: does NOT deal with indent
	for i, r := range sr.Text {
		rr := &(sr.Render[i])
		rr.RotRad = mat32.Pi / 2
		curFace = rr.CurFace(curFace)

		fht := mat32.FromFixed(curFace.Metrics().Height)
		if prevR >= 0 {
			fpos += mat32.FromFixed(curFace.Kern(prevR, r))
		}
		rr.RelPos.Y = fpos
		rr.RelPos.X = 0

		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSuper)) {
			rr.RelPos.X = -0.45 * mat32.FromFixed(curFace.Metrics().Ascent)
		}
		if bitflag.Has32(int32(rr.Deco), int(gist.DecoSub)) {
			rr.RelPos.X = 0.15 * mat32.FromFixed(curFace.Metrics().Ascent)
		}

		// todo: could check for various types of special unicode space chars here
		a, _ := curFace.GlyphAdvance(r)
		a32 := mat32.FromFixed(a)
		if a32 == 0 {
			a32 = .1 * fht // something..
		}
		rr.Size = mat32.Vec2{fht, a32}

		if r == '\t' {
			curtab := col / tabSize
			curtab++
			col = curtab * tabSize
			cpos := chsz * float32(col)
			if cpos > fpos {
				fpos = cpos
			}
		} else {
			fpos += a32
			col++
			if i < sz-1 {
				fpos += lspc
				if unicode.IsSpace(r) {
					fpos += wspc
				}
			}
		}
		prevR = r
	}
	sr.LastPos.Y = fpos
	sr.LastPos.X = 0
}

// FindWrapPosLR finds a position to do word wrapping to fit within trgSize --
// RelPos positions must have already been set (e.g., SetRunePosLR)
func (sr *Span) FindWrapPosLR(trgSize, curSize float32) int {
	sz := len(sr.Text)
	if sz == 0 {
		return -1
	}
	idx := int(float32(sz) * (trgSize / curSize))
	if idx >= sz {
		idx = sz - 1
	}
	// find starting index that is just within size
	csz := sr.RelPos.X + sr.Render[idx].RelPosAfterLR()
	if csz > trgSize {
		for idx > 0 {
			csz = sr.RelPos.X + sr.Render[idx].RelPosAfterLR()
			if csz <= trgSize {
				break
			}
			idx--
		}
	} else {
		for idx < sz-1 {
			nsz := sr.RelPos.X + sr.Render[idx+1].RelPosAfterLR()
			if nsz > trgSize {
				break
			}
			csz = nsz
			idx++
		}
	}
	if unicode.IsSpace(sr.Text[idx]) {
		idx++
		for idx < sz && unicode.IsSpace(sr.Text[idx]) { // break at END of whitespace
			idx++
		}
		return idx
	}
	// find earlier space
	for idx > 0 && !unicode.IsSpace(sr.Text[idx-1]) {
		idx--
	}
	if idx > 0 {
		return idx
	}
	// no spaces within size -- find next break going up
	for idx < sz && !unicode.IsSpace(sr.Text[idx]) {
		idx++
	}
	if idx == sz-1 { // unbreakable
		return -1
	}
	idx++
	for idx < sz && unicode.IsSpace(sr.Text[idx]) { // break at END of whitespace
		idx++
	}
	return idx
}

// ZeroPos ensures that the positions start at 0, for LR direction
func (sr *Span) ZeroPosLR() {
	sz := len(sr.Text)
	if sz == 0 {
		return
	}
	sx := sr.Render[0].RelPos.X
	if sx == 0 {
		return
	}
	for i := range sr.Render {
		sr.Render[i].RelPos.X -= sx
	}
	sr.LastPos.X -= sx
}

// TrimSpaceLeft trims leading space elements from span, and updates the
// relative positions accordingly, for LR direction
func (sr *Span) TrimSpaceLeftLR() {
	srr0 := sr.Render[0]
	for range sr.Text {
		if unicode.IsSpace(sr.Text[0]) {
			sr.Text = sr.Text[1:]
			sr.Render = sr.Render[1:]
			if len(sr.Render) > 0 {
				if sr.Render[0].Face == nil {
					sr.Render[0].Face = srr0.Face
				}
				if sr.Render[0].Color == nil {
					sr.Render[0].Color = srr0.Color
				}
			}
		} else {
			break
		}
	}
	sr.ZeroPosLR()
}

// TrimSpaceRight trims trailing space elements from span, and updates the
// relative positions accordingly, for LR direction
func (sr *Span) TrimSpaceRightLR() {
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

// TrimSpace trims leading and trailing space elements from span, and updates
// the relative positions accordingly, for LR direction
func (sr *Span) TrimSpaceLR() {
	sr.TrimSpaceLeftLR()
	sr.TrimSpaceRightLR()
}

// SplitAt splits current span at given index, returning a new span with
// remainder after index -- space is trimmed from both spans and relative
// positions updated, for LR direction
func (sr *Span) SplitAtLR(idx int) *Span {
	if idx <= 0 || idx >= len(sr.Text)-1 { // shouldn't happen
		return nil
	}
	nsr := Span{Text: sr.Text[idx:], Render: sr.Render[idx:], Dir: sr.Dir, HasDeco: sr.HasDeco}
	sr.Text = sr.Text[:idx]
	sr.Render = sr.Render[:idx]
	sr.LastPos.X = sr.Render[idx-1].RelPosAfterLR()
	// sr.TrimSpaceLR()
	// nsr.TrimSpaceLeftLR() // don't trim right!
	// go back and find latest face and color -- each sr must start with valid one
	if len(nsr.Render) > 0 {
		nrr0 := &(nsr.Render[0])
		face, color := sr.LastFont()
		if nrr0.Face == nil {
			nrr0.Face = face
		}
		if nrr0.Color == nil {
			nrr0.Color = color
		}
	}
	return &nsr
}

// LastFont finds the last font and color from given span
func (sr *Span) LastFont() (face font.Face, color color.Color) {
	for i := len(sr.Render) - 1; i >= 0; i-- {
		srr := sr.Render[i]
		if face == nil && srr.Face != nil {
			face = srr.Face
			if face != nil && color != nil {
				break
			}
		}
		if color == nil && srr.Color != nil {
			color = srr.Color
			if face != nil && color != nil {
				break
			}
		}
	}
	return
}

// RenderBg renders the background behind chars
func (sr *Span) RenderBg(rs *State, tpos mat32.Vec2) {
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
		dsc32 := mat32.FromFixed(curFace.Metrics().Descent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := mat32.Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, dsc32}))
		ur := ll.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, -rr.Size.Y}))
		if int(mat32.Floor(ll.X)) > rs.Bounds.Max.X || int(mat32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(mat32.Ceil(ur.X)) < rs.Bounds.Min.X || int(mat32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
			if didLast {
				pc.Fill(rs)
			}
			didLast = false
			continue
		}
		pc.FillStyle.Color.SetColor(rr.BgColor)
		szt := mat32.Vec2{rr.Size.X, -rr.Size.Y}
		sp := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, dsc32}))
		ul := sp.Add(tx.MulVec2AsVec(mat32.Vec2{0, szt.Y}))
		lr := sp.Add(tx.MulVec2AsVec(mat32.Vec2{szt.X, 0}))
		pc.DrawPolygon(rs, []mat32.Vec2{sp, ul, ur, lr})
		didLast = true
	}
	if didLast {
		pc.Fill(rs)
	}
}

// RenderUnderline renders the underline for span -- ensures continuity to do it all at once
func (sr *Span) RenderUnderline(rs *State, tpos mat32.Vec2) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	didLast := false
	pc := &rs.Paint

	for i, r := range sr.Text {
		if !unicode.IsPrint(r) {
			continue
		}
		rr := &(sr.Render[i])
		if !bitflag.HasAny32(int32(rr.Deco), int(gist.DecoUnderline), int(gist.DecoDottedUnderline)) {
			if didLast {
				pc.Stroke(rs)
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		if rr.Color != nil {
			curColor = rr.Color
		}
		dsc32 := mat32.FromFixed(curFace.Metrics().Descent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := mat32.Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, dsc32}))
		ur := ll.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, -rr.Size.Y}))
		if int(mat32.Floor(ll.X)) > rs.Bounds.Max.X || int(mat32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(mat32.Ceil(ur.X)) < rs.Bounds.Min.X || int(mat32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
			if didLast {
				pc.Stroke(rs)
			}
			continue
		}
		dw := .05 * rr.Size.Y
		if !didLast {
			pc.StrokeStyle.Width.Dots = dw
			pc.StrokeStyle.Color.SetColor(curColor)
		}
		if bitflag.Has32(int32(rr.Deco), int(gist.DecoDottedUnderline)) {
			pc.StrokeStyle.Dashes = []float64{2, 2}
		}
		sp := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, 2 * dw}))
		ep := rp.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, 2 * dw}))

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
func (sr *Span) RenderLine(rs *State, tpos mat32.Vec2, deco gist.TextDecorations, ascPct float32) {
	curFace := sr.Render[0].Face
	curColor := sr.Render[0].Color
	didLast := false
	pc := &rs.Paint

	for i, r := range sr.Text {
		if !unicode.IsPrint(r) {
			continue
		}
		rr := &(sr.Render[i])
		if !bitflag.Has32(int32(rr.Deco), int(deco)) {
			if didLast {
				pc.Stroke(rs)
			}
			didLast = false
			continue
		}
		curFace = rr.CurFace(curFace)
		dsc32 := mat32.FromFixed(curFace.Metrics().Descent)
		asc32 := mat32.FromFixed(curFace.Metrics().Ascent)
		rp := tpos.Add(rr.RelPos)
		scx := float32(1)
		if rr.ScaleX != 0 {
			scx = rr.ScaleX
		}
		tx := mat32.Scale2D(scx, 1).Rotate(rr.RotRad)
		ll := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, dsc32}))
		ur := ll.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, -rr.Size.Y}))
		if int(mat32.Floor(ll.X)) > rs.Bounds.Max.X || int(mat32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
			int(mat32.Ceil(ur.X)) < rs.Bounds.Min.X || int(mat32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
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
		sp := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, -yo}))
		ep := rp.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, -yo}))

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
