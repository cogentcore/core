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

	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/chewxy/math32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
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
// those pointers
type RuneRender struct {
	Face   font.Face       `desc:"fully-specified font rendering info, includes fully computed font size -- this is exactly what will be drawn -- no further transforms"`
	Color  color.Color     `desc:"color to draw characters in"`
	Deco   TextDecorations `desc:"additional decoration to apply -- underline, strike-through, etc -- also used for encoding a few special layout hints to pass info from styling tags to separate layout algorithms (e.g., <P> vs <BR>)"`
	RelPos Vec2D           `desc:"relative position from start of TextRender for the lower-left baseline rendering position of the font character"`
	RotRad float32         `desc:"rotation in radians for this character, relative to its lower-left baseline rendering position"`
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

//////////////////////////////////////////////////////////////////////////////////
//  SpanRender

// SpanRender contains fully explicit data needed for rendering a span of text
// as a slice of runes, with rune and RuneRender elements in one-to-one
// correspondence (but any nil values will use prior non-nil value) -- text
// can be oriented in any direction -- the only constraint is that it starts
// from a single starting position.  In standard TextRender context, each span
// is one line of text -- should not have new lines within the span itself.
// In SVG special cases (e.g., TextPath), it can be anything.  It is NOT
// synonymous with the HTML <span> tag, as many styling applications of that
// tag can be accommodated within a larger span-as-line.  The first RuneRender
// RelPos for LR text should be at X=0 (LastPos = 0 for RL) -- i.e., relpos
// positions are minimal for given span.
type SpanRender struct {
	Text    []rune       `desc:"text as runes"`
	Render  []RuneRender `desc:"render info for each rune in one-to-one correspondence"`
	RelPos  Vec2D        `desc:"position for start of text relative to an absolute coordinate that is provided at the time of rendering -- individual rune RelPos are added to this plus the render-time offset to get the final position"`
	LastPos Vec2D        `desc:"rune position for right-hand-side of last rune -- for standard flat strings this is the overall length of the string -- not used for rendering but should be set to allow quick size computations"`
}

// Init initializes a new span with given capacity
func (sr *SpanRender) Init(capsz int) {
	sr.Text = make([]rune, 0, capsz)
	sr.Render = make([]RuneRender, 0, capsz)
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
	if IsValid() != nil {
		return Vec2DZero
	}
	sz := sr.Render[0].RelPos.Sub(sr.LastPos).Abs()
	return sz
}

// AppendRune adds one rune and associated formatting info
func (sr *SpanRender) AppendRune(r rune, face font.Face, clr color.Color, deco TextDecorations) {
	sr.Text = append(sr.Ttext, r)
	rr := RuneRender{Face: face, Color: clr, Deco: deco}
	sr.Render = append(sr.Render, rr)
}

// AppendString adds string and associated formatting info, optimized with
// only first rune having non-nil face and color settings
func (sr *SpanRender) AppendString(str string, face font.Face, clr color.Color, deco TextDecorations) {
	sz := len(str)
	if sz == 0 {
		return
	}
	sr.Text = append(sr.Ttext, []rune(str)...)
	rr := RuneRender{Face: face, Color: clr, Deco: deco}
	sr.Render = append(sr.Render, rr)
	for i = 1; i < sz; i++ { // optimize by setting rest to nil for same
		rp := RuneRender{Deco: deco}
		sr.Render = append(sr.Render, rp)
	}
}

// SetString initializes to given plain text string, with given default
// rendering parameters that are set for the first render element --
// constructs Render slice of same size as Text -- see also SetHTML
func (sr *SpanRender) SetString(str string, face font.Face, clr color.Color) {
	sr.Text = []rune(str)
	sz := len(sr.Text)
	if sz == 0 {
		return
	}
	sr.Render = make([]RuneRender, sz)
	sr.Render[0].Face = face
	sr.Render[0].Color = clr
}

// SetRunePosLR sets relative positions of each rune using a flat
// left-to-right text layout, based on font size info and additional extra
// letter and word spacing parameters (which can be negative)
func (sr *SpanRender) SetRunePosLR(letterSpace, wordSpace float32) {
	if err := sr.IsValid(); err != nil {
		log.Println(err)
		return
	}
	sz := len(sr.Text)
	prevR := rune(-1)
	lspc := Float32ToFixed(letterSpace)
	wspc := Float32ToFixed(wordSpace)
	var fpos fixed.Int26_6
	curFace := sr.Render[0].Face
	for i, r := range sr.Text {
		sr.Render[i].RelPos.X = FixedToFloat32(fpos)
		sr.Render[i].RelPos.Y = 0
		curFace := sr.Render[i].CurFace(curFace)
		if prevR >= 0 {
			fpos += curFace.Kern(prevR, r)
		}
		a, ok := curFace.GlyphAdvance(r)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		fpos += a
		if i < sz-1 {
			fpos += lspc
			if unicode.IsSpace(r) {
				fpos += wspc
			}
		}
		prevR = r
	}
	sr.LastPos.X = FixedToFloat32(fpos)
	sr.LastPos.Y = 0
}

// FindWrapPosLR finds a position to do word wrapping to fit within trgSize --
// RelPos positions must have already been set (e.g., SetRunePosLR)
func (sr *SpanRender) FindWrapPosLR(trgSize, curSize float32) int {
	sz := len(sr.Text)
	sti := int(float32(sz) * (trgSize / curSize))
	csz := sr.Render[sti].RelPos.X
	lstgoodi := -1
	for {
		if csz > trgSize {
			for sti > 0 {
				if unicode.IsSpace(sr.Text[sri]) {
					csz = sr.Render[sti].RelPos.X
					if csz < size.X {
						return sti
					}
				}
				sti--
			}
			return -1 // oops.
		} else {
			for sti < sz {
				if unicode.IsSpace(sr.Text[sri]) {
					csz := sr.Render[sti].RelPos.X
					if csz < size.X {
						lstgoodi = sti
					} else if lstgoodi > 0 {
						return lstgoodi
					} else {
						break // try the other way
					}
				}
				sti++
			}
		}
	}
	return -1
}

// ZeroPos ensures that the positions start at 0 -- todo: need direction info
// -- this is LR specific
func (sr *SpanRender) ZeroPos() {
	sz := len(sr.Text)
	if sz == 0 {
		return
	}
	sx := sr.Render[0].RelPos.X
	if sx == 0 {
		return
	}
	for i, _ := range sr.Render {
		sr.Render[i].RelPos.X -= sz
	}
	sr.LastPos.X -= sz
}

// TrimSpace trims leading and trailing space elements from span
func (sr *SpanRender) TrimSpace(idx int) *SpanRender {
	for i, _ := range sr.Text {
		if unicode.IsSpace(sr.Text[0]) {
			sr.Text = sr.Text[1:]
			sr.Render = sr.Render[1:]
		} else {
			break
		}
	}
	for i, _ := range sr.Text {
		if unicode.IsSpace(sr.Text[len(sr.Text)-1]) {
			sr.Text = sr.Text[:len(sr.Text)-1]
			sr.Render = sr.Render[:len(sr.Render)-1]
		} else {
			break
		}
	}
	sr.ZeroPos()
}

// SplitAt splits current span at given index, returning a new span with
// remainder after index -- space is trimmed from both spans
func (sr *SpanRender) SplitAt(idx int) *SpanRender {
	nsr := SpanRender{Text: sr.Text[idx:], Render: sr.Render[idx:]}
	sr.Text = sr.Text[:idx]
	sr.Render = sr.Render[:idx]
	sr.TrimSpace()
	nsr.TrimSpace()
	return &nsr
}

// todo: TB, RL cases -- layout is complicated.. with unicode-bidi, direction,
// writing-mode styles all interacting: https://www.w3.org/TR/SVG11/text.html#TextLayout

// Transform transforms relative coordinates -- todo: need origin
func (sr *SpanRender) Transform(xform XFormMatrix2D, pos Vec2D) {
	// if err := tr.IsValid(); err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// sz := len(tr.Text)
}

//////////////////////////////////////////////////////////////////////////////////
//  TextRender

// TextRender contains one or more SpanRender elements, typically with each
// representing a separate line of text (but they can be anything).
type TextRender struct {
	Spans []SpanRender
}

// Render does text rendering into given image, within given bounds, at given
// absolute position offset -- any applicable transforms (aside from the
// char-specific rotation in Render) must be applied in advance in computing
// the relative positions of the runes, and the overall font size, etc.  todo:
// does not currently support stroking, only filling of text -- probably need
// to grab path from font and use paint rendering for stroking
func (tr *TextRender) Render(im *image.RGBA, bounds image.Rectangle, pos Vec2D) {
	pr := prof.Start("RenderText")
	for _, sr := range tr.Spans {
		if sr.IsValid() != nil {
			continue
		}
		curFace := sr.Render[0].Face
		curColor := sr.Render[0].Color
		tpos := pos.Add(sr.RelPos)

		fht := curFace.Metrics().Ascent // just for bounds checking

		d := &font.Drawer{
			Dst:  im,
			Src:  image.NewUniform(curColor),
			Face: curFace,
			Dot:  Float32ToFixedPoint(tpos.X, tpos.Y),
		}

		// based on Drawer.DrawString() in golang.org/x/image/font/font.go
		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			rp := tpos.Add(rr.RelPos)
			if int(rp.X) > bounds.Max.X || int(rp.Y) < bounds.Min.Y {
				continue
			}
			d.Dot = Float32ToFixedPoint(rp.X, rp.Y)
			dr, mask, maskp, advance, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				continue
			}
			rx := d.Dot.X + advance
			ty := d.Dot.Y - fht
			if rx.Floor() < bounds.Min.X || ty.Ceil() > bounds.Max.Y {
				continue
			}
			sr := dr.Sub(dr.Min)

			// todo: decoration!
			if rr.RotRad == 0 {
				draw.Draw(d.Dst, sr, d.Src, image.ZP, draw.Over) // todo mask?
			} else {
				transformer := draw.BiLinear
				m := Rotate2D(rr.RotRad) // todo: maybe have to back the position out first..
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, sr, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
		}
	}
	pr.End()
}

// SetHTML sets text by decoding basic HTML text style formatting tags in the
// string and sets the per-character font information appropriately, using
// given font style info.  <P> and <BR> tags create new spans, with <P>
// marking start of subsequent span with DecoParaStart.  Critically, it does
// NOT deal at all with layout (positioning) -- only sets font, color, and
// decoration info, and strips out the tags it processes -- result can then be
// processed by different layout algorithms as needed
func (tr *TextRender) SetHTML(str String, font *FontStyle, ctxt *units.Context, clr color.Color) {
	sz := len(str)
	if sz == 0 {
		return
	}
	tr.Spans = make([]SpanRender, 1)
	curSp := &(tr.Spans[0])
	initsz := kit.MinInt(sz, 1020)
	curSp.Init(initsz)

	reader := bytes.NewReader([]byte(str))
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
			nm := se.Name.Local
			switch {
			case nm == "b":
				curf := fstack[len(fstack)-1]
				fs := *curf
				fs.Weight = WeightBold
				fs.LoadFont(ctxt, "")
				fstack = append(fstack, &fs)
			case nm == "i":
				curf := fstack[len(fstack)-1]
				fs := *curf
				fs.Style = FontItalic
				fs.LoadFont(ctxt, "")
				fstack = append(fstack, &fs)
			}
		case xml.EndElement:
			if len(fstack) >= 1 {
				fstack = fstack[:len(fstack)-1]
			}
			switch se.Name.Local {
			case "p":
				tr.Spans = append(tr.Spans, SpanRender{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
				nextIsParaStart = true
				// todo: could allocate..
			case "br":
				tr.Spans = append(tr.Spans, SpanRender{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
				// todo: could allocate..
			}
		case xml.CharData:
			curf := fstack[len(fstack)-1]
			atStart := len(curSp.Text) == 0
			curSp.AppendString(string(se), curf.Face, clr, curf.Decoration)
			if nextIsParaStart && atStart {
				bitflag.Set(&(curSp.Render[0].Deco), int(DecoParaStart))
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
	Align      Align       `xml:"text-align" inherit:"true" desc:"how to align text"`
	AlignV     Align       `xml:"-" json:"-" desc:"vertical alignment of text -- copied from layout style AlignV"`
	LineHeight float32     `xml:"line-height" inherit:"true" desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (note: specific values such as pixels are not supported)"`
	Indent     units.Value `xml:"text-indent" inherit:"true" desc:"how much to indent the first line in a paragraph"`
	TabSize    units.Value `xml:"tab-size" inherit:"true" desc:"tab size"`
	WordWrap   bool        `xml:"word-wrap" inherit:"true" desc:"wrap text within a given size"`
	// todo:
	// page-break options
	// text-decoration-line -- underline, overline, line-through, -style, -color inherit
	// text-justify  inherit:"true" -- how to justify text
	// text-overflow -- clip, ellipsis, string..
	// text-shadow  inherit:"true"
	// text-transform --  inherit:"true" uppercase, lowercase, capitalize
	// user-select -- can user select text?
	// white-space -- what to do with white-space  inherit:"true"
	// word-break  inherit:"true"
}

func (p *TextStyle) Defaults() {
	p.WordWrap = false
	p.Align = AlignLeft
	p.AlignV = AlignBaseline
}

// SetStylePost applies any updates after generic xml-tag property setting
func (p *TextStyle) SetStylePost() {
}

// EffLineHeight returns the effective line height (taking into account 0 value)
func (p *TextStyle) EffLineHeight() float32 {
	if p.LineHeight == 0 {
		return 1.0
	}
	return p.LineHeight
}

// AlignFactors gets basic text alignment factors
func (p *TextStyle) AlignFactors() (ax, ay float32) {
	ax = 0.0
	ay = 0.0
	hal := p.Align
	switch {
	case IsAlignMiddle(hal):
		ax = 0.5 // todo: determine if font is horiz or vert..
	case IsAlignEnd(hal):
		ax = 1.0
	}
	val := p.AlignV
	switch {
	case val == AlignSub:
		ay = -0.1 // todo: fixme -- need actual font metrics
	case val == AlignSuper:
		ay = 0.8 // todo: fixme
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

// Layout does basic standard layout of text (assigning of relative positions
// of spans and runes) according to given styles, and given size overall box
// (nonzero values used to constrain) -- returns total resulting size box for
// text.  font face in FontStyle is used for determining line spacing here --
// other versions can do more expensive calculations of variable line spacing
// as needed.
func (tr *TextRender) LayoutStd(txtSty *TextStyle, fontSty *FontStyle, ctxt *units.Context, size Vec2D) Vec2D {
	if len(tr.Spans) == 0 {
		return Vec2DZero
	}

	fontSty.LoadFont(ctxt, "")
	fht := fontSty.Height * txtSty.EffLineHeight()

	maxw := float32(0)

	// first pass gets positions and wraps text
	si := 0
	for {
		if si >= len(tr.Spans) {
			break
		}
		sr := tr.Spans[si]
		if sr.IsValid() != nil {
			continue
		}
		// todo: determine basic layout direction, call appropriate fun
		sr.SetRunePosLR(fontSty.LetterSpacing.Dots, fontSty.WordSpacing.Dots)
		ssz := sr.SizeHV()
		if size.X > 0 && ssz.X > size.X && txtSty.WordWrap {
			for {
				wp := sr.FindWrapPosLR(size.X, ssz.X)
				if wp > 0 {
					nsr := sr.SplitAt(wp)
					if si < len(tr.Spans)-1 {
						tr.Spans = append(tr.Spans[:si+1], nsr, tr.Spans[si+1:])
					} else {
						tr.Spans = append(tr.Spans, nsr)
					}
					maxw = math32.Max(maxw, sr.SizeHV().X)
					si++
					sr = tr.Spans[si] // keep going
					ssz = sr.SizeHV()
					if ssz.X <= size.X {
						maxw = math32.Max(maxw, ssz.X)
						break
					}
				} else {
					break
				}
			}
		} else {
			maxw = math32.Max(maxw, ssz.X)
		}
	}
	// have maxw, can do alignment cases..
}

// DrawString according to current settings -- width is needed for alignment
// -- if non-zero, then x position is for the left edge of the width box, and
// alignment is WRT that width -- otherwise x position is as in
// DrawStringAnchored
func (pc *Paint) DrawString(rs *RenderState, s string, x, y, width float32) {
	ax, ay := pc.TextStyle.AlignFactors()
	if width > 0.0 {
		x += ax * width // re-offset for width
	}
	if pc.TextStyle.WordWrap {
		pc.DrawStringWrapped(rs, s, x, y, ax, ay, width, pc.TextStyle.EffLineHeight())
	} else {
		pc.DrawStringAnchored(rs, s, x, y, ax, ay, width)
	}
}

func (pc *Paint) DrawStringLines(rs *RenderState, lines []string, x, y, width, height float32) {
	ax, ay := pc.TextStyle.AlignFactors()
	pc.DrawStringLinesAnchored(rs, lines, x, y, ax, ay, width, height, pc.TextStyle.EffLineHeight())
}

// DrawStringAnchored draws the specified text at the specified anchor point.
// The anchor point is x - w * ax, y - h * ay, where w, h is the size of the
// text. Use ax=0.5, ay=0.5 to center the text at the specified point.
func (pc *Paint) DrawStringAnchored(rs *RenderState, s string, x, y, ax, ay, width float32) {
	tx, ty := rs.XForm.TransformPoint(x, y)
	w, h := pc.MeasureString(s)
	tx -= ax * w
	ty += ay * h
	// fmt.Printf("ds bounds: %v point x,y %v, %v\n", rs.Bounds, x, y)
	if rs.Mask == nil {
		pc.drawString(rs, rs.Image, rs.Bounds, s, tx, ty)
	} else {
		im := image.NewRGBA(rs.Image.Bounds())
		pc.drawString(rs, im, rs.Bounds, s, tx, ty)
		draw.DrawMask(rs.Image, rs.Image.Bounds(), im, image.ZP, rs.Mask, image.ZP, draw.Over)
	}
}

// DrawStringWrapped word-wraps the specified string to the given max width
// and then draws it at the specified anchor point using the given line
// spacing and text alignment.
func (pc *Paint) DrawStringWrapped(rs *RenderState, s string, x, y, ax, ay, width, lineHeight float32) {
	lines, h := pc.MeasureStringWrapped(s, width, lineHeight)
	pc.DrawStringLinesAnchored(rs, lines, x, y, ax, ay, width, h, lineHeight)
}

func (pc *Paint) DrawStringLinesAnchored(rs *RenderState, lines []string, x, y, ax, ay, width, h, lineHeight float32) {
	x -= ax * width
	y -= ay * h
	ax, ay = pc.TextStyle.AlignFactors()
	// ay = 1
	for _, line := range lines {
		pc.DrawStringAnchored(rs, line, x, y, ax, ay, width)
		y += pc.FontStyle.Height * lineHeight
	}
}

// todo: all of these measurements are failing to take into account transforms -- maybe that's ok -- keep the font non-scaled?  maybe add an option for that actually..

// MeasureString returns the rendered width and height of the specified text
// given the current font face.
func (pc *Paint) MeasureString(s string) (w, h float32) {
	pr := prof.Start("Paint.MeasureString")
	if pc.FontStyle.Face == nil {
		pc.FontStyle.LoadFont(&pc.UnContext, "")
	}
	d := &font.Drawer{
		Face: pc.FontStyle.Face,
	}
	a := d.MeasureString(s)
	pr.End()
	return math32.Ceil(FixedToFloat32(a)), pc.FontStyle.Height
}

// MeasureChars measures the rendered character (rune) positions of the given text in
// the current font
func (pc *Paint) MeasureChars(s []rune) []float32 {
	pr := prof.Start("Paint.MeasureChars")
	if pc.FontStyle.Face == nil {
		pc.FontStyle.LoadFont(&pc.UnContext, "")
	}
	chrs := MeasureChars(pc.FontStyle.Face, s) // in text.go
	pr.End()
	return chrs
}

// FontHeight -- returns the height of the current font
func (pc *Paint) FontHeight() float32 {
	if pc.FontStyle.Face == nil {
		pc.FontStyle.LoadFont(&pc.UnContext, "")
	}
	return pc.FontStyle.Height
}

func (pc *Paint) MeasureStringWrapped(s string, width, lineHeight float32) ([]string, float32) {
	lines := pc.WordWrap(s, width)
	h := float32(len(lines)) * pc.FontStyle.Height * lineHeight
	h -= (lineHeight - 1) * pc.FontStyle.Height
	return lines, h
}

// WordWrap wraps the specified string to the given max width and current
// font face.
func (pc *Paint) WordWrap(s string, w float32) []string {
	return wordWrap(pc, s, w)
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

// MeasureChars returns inter-character points for each rune, in float32
func MeasureChars(f font.Face, s []rune) []float32 {
	chrs := make([]float32, len(s))
	prevC := rune(-1)
	var advance fixed.Int26_6
	for i, c := range s {
		if prevC >= 0 {
			advance += f.Kern(prevC, c)
		}
		a, ok := f.GlyphAdvance(c)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		advance += a
		chrs[i] = FixedToFloat32(advance)
		prevC = c
	}
	return chrs
}

type measureStringer interface {
	MeasureString(s string) (w, h float32)
}

func splitOnSpace(x string) []string {
	var result []string
	pi := 0
	ps := false
	for i, c := range x {
		s := unicode.IsSpace(c)
		if s != ps && i > 0 {
			result = append(result, x[pi:i])
			pi = i
		}
		ps = s
	}
	result = append(result, x[pi:])
	return result
}

func wordWrap(m measureStringer, s string, width float32) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		fields := splitOnSpace(line)
		if len(fields)%2 == 1 {
			fields = append(fields, "")
		}
		x := ""
		for i := 0; i < len(fields); i += 2 {
			w, _ := m.MeasureString(x + fields[i])
			if w > width {
				if x == "" {
					result = append(result, fields[i])
					x = ""
					continue
				} else {
					result = append(result, x)
					x = ""
				}
			}
			x += fields[i] + fields[i+1]
		}
		if x != "" {
			result = append(result, x)
		}
	}
	for i, line := range result {
		result[i] = strings.TrimSpace(line)
	}
	return result
}
