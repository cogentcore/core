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

	"github.com/goki/gi/units"
	"github.com/goki/ki"
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
	Face    font.Face       `desc:"fully-specified font rendering info, includes fully computed font size -- this is exactly what will be drawn -- no further transforms"`
	Color   color.Color     `desc:"color to draw characters in"`
	BgColor color.Color     `desc:"background color to fill background of color -- for highlighting, <mark> tag, etc"`
	Deco    TextDecorations `desc:"additional decoration to apply -- underline, strike-through, etc -- also used for encoding a few special layout hints to pass info from styling tags to separate layout algorithms (e.g., <P> vs <BR>)"`
	RelPos  fixed.Point26_6 `desc:"relative position from start of TextRender for the lower-left baseline rendering position of the font character"`
	Size    fixed.Point26_6 `desc:"size of the rune itself, exclusive of spacing that might surround it"`
	RotRad  float32         `desc:"rotation in radians for this character, relative to its lower-left baseline rendering position"`
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

// RelPosAfterLR returns the relative position after given rune for LR order: RelPos.X + Size.X
func (rr *RuneRender) RelPosAfterLR() fixed.Int26_6 {
	return rr.RelPos.X + rr.Size.X
}

// RelPosAfterRL returns the relative position after given rune for RL order: RelPos.X - Size.X
func (rr *RuneRender) RelPosAfterRL() fixed.Int26_6 {
	return rr.RelPos.X - rr.Size.X
}

// RelPosAfterTB returns the relative position after given rune for TB order: RelPos.Y + Size.Y
func (rr *RuneRender) RelPosAfterTB() fixed.Int26_6 {
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
	RelPos  fixed.Point26_6 `desc:"position for start of text relative to an absolute coordinate that is provided at the time of rendering -- individual rune RelPos are added to this plus the render-time offset to get the final position"`
	LastPos fixed.Point26_6 `desc:"rune position for further edge of last rune -- for standard flat strings this is the overall length of the string -- used for size / layout computations"`
	Dir     TextDirections  `desc:"where relevant, this is the (default, dominant) text direction for the span"`
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
func (sr *SpanRender) SizeHV() fixed.Point26_6 {
	if sr.IsValid() != nil {
		return fixed.Point26_6{}
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
func (sr *SpanRender) AppendRune(r rune, face font.Face, clr color.Color, deco TextDecorations) {
	sr.Text = append(sr.Text, r)
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
	sr.Text = append(sr.Text, []rune(str)...)
	rr := RuneRender{Face: face, Color: clr, Deco: deco}
	sr.Render = append(sr.Render, rr)
	for i := 1; i < sz; i++ { // optimize by setting rest to nil for same
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
func (sr *SpanRender) SetRunePosLR(letterSpace, wordSpace fixed.Int26_6) {
	if err := sr.IsValid(); err != nil {
		log.Println(err)
		return
	}
	sr.Dir = LRTB
	sz := len(sr.Text)
	prevR := rune(-1)
	lspc := letterSpace
	wspc := wordSpace
	var fpos fixed.Int26_6
	curFace := sr.Render[0].Face
	for i, r := range sr.Text {
		curFace = sr.Render[i].CurFace(curFace)
		fht := curFace.Metrics().Ascent + curFace.Metrics().Descent
		if prevR >= 0 {
			fpos += curFace.Kern(prevR, r)
		}
		sr.Render[i].RelPos.X = fpos
		sr.Render[i].RelPos.Y = 0
		a, ok := curFace.GlyphAdvance(r)
		if !ok {
			// TODO: is falling back on the U+FFFD glyph the responsibility of
			// the Drawer or the Face?
			// TODO: set prevC = '\ufffd'?
			continue
		}
		sr.Render[i].Size = fixed.Point26_6{a, fht}
		fpos += a
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
func (sr *SpanRender) FindWrapPosLR(trgSize, curSize fixed.Int26_6) int {
	sz := len(sr.Text)
	idx := int(float32(sz) * (FixedToFloat32(trgSize) / FixedToFloat32(curSize)))
	csz := sr.Render[idx].RelPos.X
	lstgoodi := -1
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
	nsr := SpanRender{Text: sr.Text[idx:], Render: sr.Render[idx:], Dir: sr.Dir}
	nsr.LastPos.X = sr.Render[idx].RelPosAfterLR()
	sr.Text = sr.Text[:idx]
	sr.Render = sr.Render[:idx]
	sr.LastPos.X = sr.Render[idx-1].RelPosAfterLR()
	sr.TrimSpaceLR()
	nsr.TrimSpaceLR()
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
// absolute position offset -- any applicable transforms (aside from the
// char-specific rotation in Render) must be applied in advance in computing
// the relative positions of the runes, and the overall font size, etc.  todo:
// does not currently support stroking, only filling of text -- probably need
// to grab path from font and use paint rendering for stroking
func (tr *TextRender) Render(rs *RenderState, pos fixed.Point26_6) {
	pr := prof.Start("RenderText")
	defer pr.End()

	// todo: use renderstate for decoration rendering -- decide if always doing bg or not
	// always doing is probably better, except if bg is transparent or nil -- i.e., let
	// the user decide

	for _, sr := range tr.Spans {
		if sr.IsValid() != nil {
			continue
		}
		curFace := sr.Render[0].Face
		curColor := sr.Render[0].Color
		tpos := pos.Add(sr.RelPos)

		fht := curFace.Metrics().Ascent // just for bounds checking

		d := &font.Drawer{
			Dst:  rs.Image,
			Src:  image.NewUniform(curColor),
			Face: curFace,
			Dot:  tpos,
		}

		// based on Drawer.DrawString() in golang.org/x/image/font/font.go
		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			rp := tpos.Add(rr.RelPos)
			if rp.X.Floor() > rs.Bounds.Max.X || rp.Y.Ceil() < rs.Bounds.Min.Y {
				continue
			}
			d.Dot = rp
			curFace = rr.CurFace(curFace)
			d.Face = curFace
			if rr.Color != nil {
				curColor = rr.Color
				d.Src = image.NewUniform(curColor)
			}
			dr, mask, maskp, advance, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				continue
			}
			rx := d.Dot.X + advance
			ty := d.Dot.Y - fht
			if rx.Floor() < rs.Bounds.Min.X || ty.Ceil() > rs.Bounds.Max.Y {
				continue
			}
			srect := dr.Sub(dr.Min)

			// todo: decoration!
			// if rr.RotRad == 0 {
			// 	draw.Draw(d.Dst, sr, d.Src, image.ZP, draw.Over) // todo mask?
			// } else {
			transformer := draw.BiLinear
			fx, fy := float32(dr.Min.X), float32(dr.Min.Y)
			m := Rotate2D(rr.RotRad).Translate(fx, fy)
			s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
			transformer.Transform(d.Dst, s2d, d.Src, srect, draw.Over, &draw.Options{
				SrcMask:  mask,
				SrcMaskP: maskp,
			})
			// }
		}
	}
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
			case "sub":
				fs.SetDeco(DecoSub)
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
				// todo: mark requires setting background color -- need a stack, etc, or add again to fontstyle..
			case "abbr", "acronym":
				// default style is a *dotted* underline.. sheesh
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
				// quotation -- insert " " of proper unicode type
			case "dfn":
				// no default styling
			case "bdo":
				// bidirectional override..
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
			default:
				if len(fstack) > 1 {
					fstack = fstack[:len(fstack)-1]
				}
			}
		case xml.CharData:
			curf := fstack[len(fstack)-1]
			atStart := len(curSp.Text) == 0
			curSp.AppendString(string(se), curf.Face, curf.Color, curf.Deco)
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
	Align      Align       `xml:"text-align" inherit:"true" desc:"how to align text"`
	AlignV     Align       `xml:"-" json:"-" desc:"vertical alignment of text -- copied from layout style AlignV"`
	LineHeight float32     `xml:"line-height" inherit:"true" desc:"specified height of a line of text, in proportion to default font height, 0 = 1 = normal (todo: specific values such as pixels are not supported, in order to properly support percentage) -- text is centered within the overall lineheight"`
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

func (ts *TextStyle) Defaults() {
	ts.WordWrap = false
	ts.Align = AlignLeft
	ts.AlignV = AlignBaseline
}

// SetStylePost applies any updates after generic xml-tag property setting
func (ts *TextStyle) SetStylePost() {
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
	asc := fontSty.Face.Metrics().Ascent
	dsc := fontSty.Face.Metrics().Descent
	fht := asc + dsc
	lspc := Float32ToFixed(fontSty.Height * txtSty.EffLineHeight())
	lpad := (lspc - fht) / 2 // padding above / below text box for centering in line

	szf := Vec2D.Fixed(size)

	maxw := fixed.Int26_6(0)

	// first pass gets rune positions and wraps text as needed, and gets max width
	si := 0
	for si < len(tr.Spans) {
		sr := &(tr.Spans[si])
		if sr.IsValid() != nil {
			continue
		}
		if sr.LastPos.X == 0 { // don't re-do unless necessary
			sr.SetRunePosLR(Float32ToFixed(fontSty.LetterSpacing.Dots), Float32ToFixed(fontSty.WordSpacing.Dots))
		}
		ssz := sr.SizeHV()
		if szf.X > 0 && ssz.X > szf.X && txtSty.WordWrap {
			for {
				wp := sr.FindWrapPosLR(szf.X, ssz.X)
				if wp > 0 {
					nsr := sr.SplitAtLR(wp)
					tr.InsertSpan(si+1, nsr)
					ssz = sr.SizeHV()
					if ssz.X > maxw {
						maxw = ssz.X
					}
					si++
					sr = &(tr.Spans[si]) // keep going with nsr
					ssz = sr.SizeHV()
					if ssz.X <= szf.X {
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

	if maxw > szf.X {
		szf.X = maxw
	}

	// vertical alignment
	vht := lspc.Mul(Float32ToFixed(float32(len(tr.Spans))))
	if vht > szf.Y {
		szf.Y = vht
	}

	tr.Size = Vec2D{FixedToFloat32(maxw), FixedToFloat32(vht)}

	vpad := fixed.Int26_6(0) // padding at top to achieve vertical alignment
	vextra := szf.Y - vht
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
		hextra := szf.X - ssz.X
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
	return NewVec2DFmFixed(szf)
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
