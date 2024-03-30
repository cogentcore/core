// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"bytes"
	"encoding/xml"
	"html"
	"image"
	"io"
	"math"
	"strings"
	"unicode/utf8"

	"unicode"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
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

// todo: TB, RL cases -- layout is complicated.. with unicode-bidi, direction,
// writing-mode styles all interacting: https://www.w3.org/TR/SVG11/text.html#TextLayout

// Text contains one or more Span elements, typically with each
// representing a separate line of text (but they can be anything).
type Text struct {
	Spans []Span

	// last size of overall rendered text
	Size mat32.Vec2

	// fontheight computed in last Layout
	FontHeight float32

	// lineheight computed in last Layout
	LineHeight float32

	// whether has had overflow in rendering
	HasOverflow bool

	// where relevant, this is the (default, dominant) text direction for the span
	Dir styles.TextDirections

	// hyperlinks within rendered text
	Links []TextLink
}

// InsertSpan inserts a new span at given index
func (tr *Text) InsertSpan(at int, ns *Span) {
	sz := len(tr.Spans)
	tr.Spans = append(tr.Spans, Span{})
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
// use paint rendering for stroking.
func (tr *Text) Render(pc *Context, pos mat32.Vec2) {
	// pr := prof.Start("RenderText")
	// defer pr.End()

	var ppaint styles.Paint
	ppaint.CopyStyleFrom(pc.Paint)

	pc.PushTransform(mat32.Identity2()) // needed for SVG
	defer pc.PopTransform()
	pc.CurrentTransform = mat32.Identity2()

	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()

	elipses := '…'
	hadOverflow := false
	rendOverflow := false
	overBoxSet := false
	var overStart mat32.Vec2
	var overBox mat32.Box2
	var overFace font.Face
	var overColor image.Image

	for _, sr := range tr.Spans {
		if sr.IsValid() != nil {
			continue
		}

		curFace := sr.Render[0].Face
		curColor := sr.Render[0].Color
		if g, ok := curColor.(gradient.Gradient); ok {
			g.Update(pc.FontStyle.Opacity, mat32.B2FromRect(pc.LastRenderBBox), pc.CurrentTransform)
		} else {
			curColor = gradient.ApplyOpacityImage(curColor, pc.FontStyle.Opacity)
		}
		tpos := pos.Add(sr.RelPos)

		if !overBoxSet {
			overWd, _ := curFace.GlyphAdvance(elipses)
			overWd32 := mat32.FromFixed(overWd)
			overEnd := mat32.V2FromPoint(pc.Bounds.Max)
			overStart = overEnd.Sub(mat32.V2(overWd32, 0.1*tr.FontHeight))
			overBox = mat32.Box2{Min: mat32.V2(overStart.X, overEnd.Y-tr.FontHeight), Max: overEnd}
			overFace = curFace
			overColor = curColor
			overBoxSet = true
		}

		d := &font.Drawer{
			Dst:  pc.Image,
			Src:  curColor,
			Face: curFace,
		}

		// todo: cache flags if these are actually needed
		if sr.HasDeco.HasFlag(styles.DecoBackgroundColor) {
			// fmt.Println("rendering background color for span", rs)
			sr.RenderBg(pc, tpos)
		}
		if sr.HasDeco.HasFlag(styles.Underline) || sr.HasDeco.HasFlag(styles.DecoDottedUnderline) {
			sr.RenderUnderline(pc, tpos)
		}
		if sr.HasDeco.HasFlag(styles.Overline) {
			sr.RenderLine(pc, tpos, styles.Overline, 1.1)
		}

		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			if rr.Color != nil {
				curColor := rr.Color
				curColor = gradient.ApplyOpacityImage(curColor, pc.FontStyle.Opacity)
				d.Src = curColor
			}
			curFace = rr.CurFace(curFace)
			if !unicode.IsPrint(r) {
				continue
			}
			dsc32 := mat32.FromFixed(curFace.Metrics().Descent)
			rp := tpos.Add(rr.RelPos)
			scx := float32(1)
			if rr.ScaleX != 0 {
				scx = rr.ScaleX
			}
			tx := mat32.Scale2D(scx, 1).Rotate(rr.RotRad)
			ll := rp.Add(tx.MulVec2AsVec(mat32.V2(0, dsc32)))
			ur := ll.Add(tx.MulVec2AsVec(mat32.V2(rr.Size.X, -rr.Size.Y)))

			if int(mat32.Ceil(ur.X)) < pc.Bounds.Min.X || int(mat32.Ceil(ll.Y)) < pc.Bounds.Min.Y {
				continue
			}

			doingOverflow := false
			if tr.HasOverflow {
				cmid := ll.Add(mat32.V2(0.5*rr.Size.X, -0.5*rr.Size.Y))
				if overBox.ContainsPoint(cmid) {
					doingOverflow = true
					r = elipses
				}
			}

			if int(mat32.Floor(ll.X)) > pc.Bounds.Max.X+1 || int(mat32.Floor(ur.Y)) > pc.Bounds.Max.Y+1 {
				hadOverflow = true
				if !doingOverflow {
					continue
				}
			}

			if rendOverflow { // once you've rendered, no more rendering
				continue
			}

			d.Face = curFace
			d.Dot = rp.Fixed()
			dr, mask, maskp, _, ok := d.Face.Glyph(d.Dot, r)
			if !ok {
				// fmt.Printf("not ok rendering rune: %v\n", string(r))
				continue
			}
			if rr.RotRad == 0 && (rr.ScaleX == 0 || rr.ScaleX == 1) {
				idr := dr.Intersect(pc.Bounds)
				soff := image.Point{}
				if dr.Min.X < pc.Bounds.Min.X {
					soff.X = pc.Bounds.Min.X - dr.Min.X
					maskp.X += pc.Bounds.Min.X - dr.Min.X
				}
				if dr.Min.Y < pc.Bounds.Min.Y {
					soff.Y = pc.Bounds.Min.Y - dr.Min.Y
					maskp.Y += pc.Bounds.Min.Y - dr.Min.Y
				}
				draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
			} else {
				srect := dr.Sub(dr.Min)
				dbase := mat32.V2(rp.X-float32(dr.Min.X), rp.Y-float32(dr.Min.Y))

				transformer := draw.BiLinear
				fx, fy := float32(dr.Min.X), float32(dr.Min.Y)
				m := mat32.Translate2D(fx+dbase.X, fy+dbase.Y).Scale(scx, 1).Rotate(rr.RotRad).Translate(-dbase.X, -dbase.Y)
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, srect, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
			if doingOverflow {
				rendOverflow = true
			}
		}
		if sr.HasDeco.HasFlag(styles.LineThrough) {
			sr.RenderLine(pc, tpos, styles.LineThrough, 0.25)
		}
	}
	tr.HasOverflow = hadOverflow

	if hadOverflow && !rendOverflow && overBoxSet {
		d := &font.Drawer{
			Dst:  pc.Image,
			Src:  overColor,
			Face: overFace,
			Dot:  overStart.Fixed(),
		}
		dr, mask, maskp, _, _ := d.Face.Glyph(d.Dot, elipses)
		idr := dr.Intersect(pc.Bounds)
		soff := image.Point{}
		draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
	}

	pc.Paint.CopyStyleFrom(&ppaint)
}

// RenderTopPos renders at given top position -- uses first font info to
// compute baseline offset and calls overall Render -- convenience for simple
// widget rendering without layouts
func (tr *Text) RenderTopPos(pc *Context, tpos mat32.Vec2) {
	if len(tr.Spans) == 0 {
		return
	}
	sr := &(tr.Spans[0])
	if sr.IsValid() != nil {
		return
	}
	curFace := sr.Render[0].Face
	pos := tpos
	pos.Y += mat32.FromFixed(curFace.Metrics().Ascent)
	tr.Render(pc, pos)
}

// SetString is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after.  Be sure that OpenFont has been run so a
// valid Face is available.  noBG ignores any BackgroundColor in font style, and never
// renders background color
func (tr *Text) SetString(str string, fontSty *styles.FontRender, ctxt *units.Context, txtSty *styles.Text, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]Span, 1)
	}
	tr.Links = nil
	sr := &(tr.Spans[0])
	sr.SetString(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.Size = mat32.V2(ssz.X, mat32.FromFixed(vht))

}

// SetStringRot90 is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does TB rotated layout (-90 deg).
// Be sure that OpenFont has been run so a valid Face is available.
// noBG ignores any BackgroundColor in font style, and never renders background color
func (tr *Text) SetStringRot90(str string, fontSty *styles.FontRender, ctxt *units.Context, txtSty *styles.Text, noBG bool, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]Span, 1)
	}
	tr.Links = nil
	sr := &(tr.Spans[0])
	rot := float32(mat32.Pi / 2)
	sr.SetString(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosTBRot(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.Size = mat32.V2(mat32.FromFixed(vht), ssz.Y)
}

// SetRunes is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after Be sure that OpenFont has been run so a
// valid Face is available.  noBG ignores any BackgroundColor in font style, and never
// renders background color
func (tr *Text) SetRunes(str []rune, fontSty *styles.FontRender, ctxt *units.Context, txtSty *styles.Text, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]Span, 1)
	}
	tr.Links = nil
	sr := &(tr.Spans[0])
	sr.SetRunes(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.Size = mat32.V2(ssz.X, mat32.FromFixed(vht))
}

// SetHTMLSimpleTag sets the styling parameters for simple html style tags
// that only require updating the given font spec values -- returns true if handled
// https://www.w3schools.com/cssref/css_default_values.asp
func SetHTMLSimpleTag(tag string, fs *styles.FontRender, ctxt *units.Context, cssAgg map[string]any) bool {
	did := false
	switch tag {
	case "b", "strong":
		fs.Weight = styles.WeightBold
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "i", "em", "var", "cite":
		fs.Style = styles.Italic
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "ins":
		fallthrough
	case "u":
		fs.SetDecoration(styles.Underline)
		did = true
	case "s", "del", "strike":
		fs.SetDecoration(styles.LineThrough)
		did = true
	case "sup":
		fs.SetDecoration(styles.DecoSuper)
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Val))
		curpts -= 2
		fs.Size = units.Pt(float32(curpts))
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "sub":
		fs.SetDecoration(styles.DecoSub)
		fallthrough
	case "small":
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Val))
		curpts -= 2
		fs.Size = units.Pt(float32(curpts))
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "big":
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Val))
		curpts += 2
		fs.Size = units.Pt(float32(curpts))
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "xx-small", "x-small", "smallf", "medium", "large", "x-large", "xx-large":
		fs.Size = units.Pt(styles.FontSizePoints[tag])
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "mark":
		fs.Background = colors.C(colors.Scheme.Warn.Container)
		did = true
	case "abbr", "acronym":
		fs.SetDecoration(styles.DecoDottedUnderline)
		did = true
	case "tt", "kbd", "samp", "code":
		fs.Family = "monospace"
		fs.Font = OpenFont(fs, ctxt)
		fs.Background = colors.C(colors.Scheme.SurfaceContainer)
		did = true
	}
	return did
}

// SetHTML sets text by decoding all standard inline HTML text style
// formatting tags in the string and sets the per-character font information
// appropriately, using given font style info.  <P> and <BR> tags create new
// spans, with <P> marking start of subsequent span with DecoParaStart.
// Critically, it does NOT deal at all with layout (positioning) except in
// breaking lines into different spans, but not with word wrapping -- only
// sets font, color, and decoration info, and strips out the tags it processes
// -- result can then be processed by different layout algorithms as needed.
// cssAgg, if non-nil, should contain CSSAgg properties -- will be tested for
// special css styling of each element.
func (tr *Text) SetHTML(str string, font *styles.FontRender, txtSty *styles.Text, ctxt *units.Context, cssAgg map[string]any) {
	if txtSty.HasPre() {
		tr.SetHTMLPre([]byte(str), font, txtSty, ctxt, cssAgg)
	} else {
		tr.SetHTMLNoPre([]byte(str), font, txtSty, ctxt, cssAgg)
	}
}

// SetHTMLBytes does SetHTML with bytes as input -- more efficient -- use this
// if already in bytes
func (tr *Text) SetHTMLBytes(str []byte, font *styles.FontRender, txtSty *styles.Text, ctxt *units.Context, cssAgg map[string]any) {
	if txtSty.HasPre() {
		tr.SetHTMLPre(str, font, txtSty, ctxt, cssAgg)
	} else {
		tr.SetHTMLNoPre(str, font, txtSty, ctxt, cssAgg)
	}
}

// This is the No-Pre parser that uses the golang XML decoder system, which
// strips all whitespace and is thus unsuitable for any Pre case
func (tr *Text) SetHTMLNoPre(str []byte, font *styles.FontRender, txtSty *styles.Text, ctxt *units.Context, cssAgg map[string]any) {
	//	errstr := "gi.Text SetHTML"
	sz := len(str)
	if sz == 0 {
		return
	}
	tr.Spans = make([]Span, 1)
	tr.Links = nil
	curSp := &(tr.Spans[0])
	initsz := min(sz, 1020)
	curSp.Init(initsz)

	spcstr := bytes.Join(bytes.Fields(str), []byte(" "))

	reader := bytes.NewReader(spcstr)
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel

	font.Font = OpenFont(font, ctxt)

	// set when a </p> is encountered
	nextIsParaStart := false
	curLinkIdx := -1 // if currently processing an <a> link element

	fstack := make([]*styles.FontRender, 1, 10)
	fstack[0] = font
	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			// log.Printf("%v parsing error: %v for string\n%v\n", errstr, err, string(str))
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			curf := fstack[len(fstack)-1]
			fs := *curf
			nm := strings.ToLower(se.Name.Local)
			curLinkIdx = -1
			if !SetHTMLSimpleTag(nm, &fs, ctxt, cssAgg) {
				switch nm {
				case "a":
					fs.Color = colors.C(colors.Scheme.Primary.Base)
					fs.SetDecoration(styles.Underline)
					curLinkIdx = len(tr.Links)
					tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIdx: len(curSp.Text)}
					sprop := make(map[string]any, len(se.Attr))
					tl.Props = sprop
					for _, attr := range se.Attr {
						if attr.Name.Local == "href" {
							tl.URL = attr.Value
						}
						sprop[attr.Name.Local] = attr.Value
					}
					tr.Links = append(tr.Links, *tl)
				case "span":
					// just uses props
				case "q":
					curf := fstack[len(fstack)-1]
					atStart := len(curSp.Text) == 0
					curSp.AppendRune('“', curf.Face.Face, curf.Color, curf.Background, curf.Decoration)
					if nextIsParaStart && atStart {
						curSp.SetNewPara()
					}
					nextIsParaStart = false
				case "dfn":
					// no default styling
				case "bdo":
					// bidirectional override..
				case "p":
					if len(curSp.Text) > 0 {
						// fmt.Printf("para start: '%v'\n", string(curSp.Text))
						tr.Spans = append(tr.Spans, Span{})
						curSp = &(tr.Spans[len(tr.Spans)-1])
					}
					nextIsParaStart = true
				case "br":
				default:
					// log.Printf("%v tag not recognized: %v for string\n%v\n", errstr, nm, string(str))
				}
			}
			if len(se.Attr) > 0 {
				sprop := make(map[string]any, len(se.Attr))
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "style":
						styles.SetStylePropsXML(attr.Value, &sprop)
					case "class":
						if cssAgg != nil {
							clnm := "." + attr.Value
							if aggp, ok := styles.SubProps(cssAgg, clnm); ok {
								fs.SetStyleProps(nil, aggp, nil)
								fs.Font = OpenFont(&fs, ctxt)
							}
						}
					default:
						sprop[attr.Name.Local] = attr.Value
					}
				}
				fs.SetStyleProps(nil, sprop, nil)
				fs.Font = OpenFont(&fs, ctxt)
			}
			if cssAgg != nil {
				FontStyleCSS(&fs, nm, cssAgg, ctxt, nil)
			}
			fstack = append(fstack, &fs)
		case xml.EndElement:
			switch se.Name.Local {
			case "p":
				tr.Spans = append(tr.Spans, Span{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
				nextIsParaStart = true
			case "br":
				tr.Spans = append(tr.Spans, Span{})
				curSp = &(tr.Spans[len(tr.Spans)-1])
			case "q":
				curf := fstack[len(fstack)-1]
				curSp.AppendRune('”', curf.Face.Face, curf.Color, curf.Background, curf.Decoration)
			case "a":
				if curLinkIdx >= 0 {
					tl := &tr.Links[curLinkIdx]
					tl.EndSpan = len(tr.Spans) - 1
					tl.EndIdx = len(curSp.Text)
					curLinkIdx = -1
				}
			}
			if len(fstack) > 1 {
				fstack = fstack[:len(fstack)-1]
			}
		case xml.CharData:
			curf := fstack[len(fstack)-1]
			atStart := len(curSp.Text) == 0
			sstr := html.UnescapeString(string(se))
			if nextIsParaStart && atStart {
				sstr = strings.TrimLeftFunc(sstr, func(r rune) bool {
					return unicode.IsSpace(r)
				})
			}
			curSp.AppendString(sstr, curf.Face.Face, curf.Color, curf.Background, curf.Decoration, font, ctxt)
			if nextIsParaStart && atStart {
				curSp.SetNewPara()
			}
			nextIsParaStart = false
			if curLinkIdx >= 0 {
				tl := &tr.Links[curLinkIdx]
				tl.Label = sstr
			}
		}
	}
}

// note: adding print / log statements to following when inside gide will cause
// an infinite loop because the console redirection uses this very same code!

// SetHTMLPre sets preformatted HTML-styled text by decoding all standard
// inline HTML text style formatting tags in the string and sets the
// per-character font information appropriately, using given font style info.
// Only basic styling tags, including <span> elements with style parameters
// (including class names) are decoded.  Whitespace is decoded as-is,
// including LF \n etc, except in WhiteSpacePreLine case which only preserves LF's
func (tr *Text) SetHTMLPre(str []byte, font *styles.FontRender, txtSty *styles.Text, ctxt *units.Context, cssAgg map[string]any) {
	// errstr := "gi.Text SetHTMLPre"

	sz := len(str)
	tr.Spans = make([]Span, 1)
	tr.Links = nil
	if sz == 0 {
		return
	}
	curSp := &(tr.Spans[0])
	initsz := min(sz, 1020)
	curSp.Init(initsz)

	font.Font = OpenFont(font, ctxt)

	nextIsParaStart := false
	curLinkIdx := -1 // if currently processing an <a> link element

	fstack := make([]*styles.FontRender, 1, 10)
	fstack[0] = font

	tagstack := make([]string, 0, 10)

	tmpbuf := make([]byte, 0, 1020)

	bidx := 0
	curTag := ""
	for bidx < sz {
		cb := str[bidx]
		ftag := ""
		if cb == '<' && sz > bidx+1 {
			eidx := bytes.Index(str[bidx+1:], []byte(">"))
			if eidx > 0 {
				ftag = string(str[bidx+1 : bidx+1+eidx])
				bidx += eidx + 2
			} else { // get past <
				curf := fstack[len(fstack)-1]
				curSp.AppendString(string(str[bidx:bidx+1]), curf.Face.Face, curf.Color, curf.Background, curf.Decoration, font, ctxt)
				bidx++
			}
		}
		if ftag != "" {
			if ftag[0] == '/' {
				etag := strings.ToLower(ftag[1:])
				// fmt.Printf("%v  etag: %v\n", bidx, etag)
				if etag == "pre" {
					continue // ignore
				}
				if etag != curTag {
					// log.Printf("%v end tag: %v doesn't match current tag: %v for string\n%v\n", errstr, etag, curTag, string(str))
				}
				switch etag {
				// case "p":
				// 	tr.Spans = append(tr.Spans, Span{})
				// 	curSp = &(tr.Spans[len(tr.Spans)-1])
				// 	nextIsParaStart = true
				// case "br":
				// 	tr.Spans = append(tr.Spans, Span{})
				// 	curSp = &(tr.Spans[len(tr.Spans)-1])
				case "q":
					curf := fstack[len(fstack)-1]
					curSp.AppendRune('”', curf.Face.Face, curf.Color, curf.Background, curf.Decoration)
				case "a":
					if curLinkIdx >= 0 {
						tl := &tr.Links[curLinkIdx]
						tl.EndSpan = len(tr.Spans) - 1
						tl.EndIdx = len(curSp.Text)
						curLinkIdx = -1
					}
				}
				if len(fstack) > 1 { // pop at end
					fstack = fstack[:len(fstack)-1]
				}
				tslen := len(tagstack)
				if tslen > 1 {
					curTag = tagstack[tslen-2]
					tagstack = tagstack[:tslen-1]
				} else if tslen == 1 {
					curTag = ""
					tagstack = tagstack[:0]
				}
			} else { // start tag
				parts := strings.Split(ftag, " ")
				stag := strings.ToLower(strings.TrimSpace(parts[0]))
				// fmt.Printf("%v  stag: %v\n", bidx, stag)
				attrs := parts[1:]
				attr := strings.Split(strings.Join(attrs, " "), "=")
				nattr := len(attr) / 2
				curf := fstack[len(fstack)-1]
				fs := *curf
				curLinkIdx = -1
				if !SetHTMLSimpleTag(stag, &fs, ctxt, cssAgg) {
					switch stag {
					case "a":
						fs.Color = colors.C(colors.Scheme.Primary.Base)
						fs.SetDecoration(styles.Underline)
						curLinkIdx = len(tr.Links)
						tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIdx: len(curSp.Text)}
						if nattr > 0 {
							sprop := make(map[string]any, len(parts)-1)
							tl.Props = sprop
							for ai := 0; ai < nattr; ai++ {
								nm := strings.TrimSpace(attr[ai*2])
								vl := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(attr[ai*2+1]), `"`), `"`)
								if nm == "href" {
									tl.URL = vl
								}
								sprop[nm] = vl
							}
						}
						tr.Links = append(tr.Links, *tl)
					case "span":
						// just uses props
					case "q":
						curf := fstack[len(fstack)-1]
						atStart := len(curSp.Text) == 0
						curSp.AppendRune('“', curf.Face.Face, curf.Color, curf.Background, curf.Decoration)
						if nextIsParaStart && atStart {
							curSp.SetNewPara()
						}
						nextIsParaStart = false
					case "dfn":
						// no default styling
					case "bdo":
						// bidirectional override..
					// case "p":
					// 	if len(curSp.Text) > 0 {
					// 		// fmt.Printf("para start: '%v'\n", string(curSp.Text))
					// 		tr.Spans = append(tr.Spans, Span{})
					// 		curSp = &(tr.Spans[len(tr.Spans)-1])
					// 	}
					// 	nextIsParaStart = true
					// case "br":
					case "pre":
						continue // ignore
					default:
						// log.Printf("%v tag not recognized: %v for string\n%v\n", errstr, stag, string(str))
						// just ignore it and format as is, for pre case!
						// todo: need to include
					}
				}
				if nattr > 0 { // attr
					sprop := make(map[string]any, nattr)
					for ai := 0; ai < nattr; ai++ {
						nm := strings.TrimSpace(attr[ai*2])
						vl := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(attr[ai*2+1]), `"`), `"`)
						// fmt.Printf("nm: %v  val: %v\n", nm, vl)
						switch nm {
						case "style":
							styles.SetStylePropsXML(vl, &sprop)
						case "class":
							if cssAgg != nil {
								clnm := "." + vl
								if aggp, ok := styles.SubProps(cssAgg, clnm); ok {
									fs.SetStyleProps(nil, aggp, nil)
									fs.Font = OpenFont(&fs, ctxt)
								}
							}
						default:
							sprop[nm] = vl
						}
					}
					fs.SetStyleProps(nil, sprop, nil)
					fs.Font = OpenFont(&fs, ctxt)
				}
				if cssAgg != nil {
					FontStyleCSS(&fs, stag, cssAgg, ctxt, nil)
				}
				fstack = append(fstack, &fs)
				curTag = stag
				tagstack = append(tagstack, curTag)
			}
		} else { // raw chars
			// todo: deal with WhiteSpacePreLine -- trim out non-LF ws
			curf := fstack[len(fstack)-1]
			// atStart := len(curSp.Text) == 0
			tmpbuf := tmpbuf[0:0]
			didNl := false
		aggloop:
			for ; bidx < sz; bidx++ {
				nb := str[bidx] // re-gets cb so it can be processed here..
				switch nb {
				case '<':
					if (bidx > 0 && str[bidx-1] == '<') || sz == bidx+1 {
						tmpbuf = append(tmpbuf, nb)
						didNl = false
					} else {
						didNl = false
						break aggloop
					}
				case '\n': // todo absorb other line endings
					unestr := html.UnescapeString(string(tmpbuf))
					curSp.AppendString(unestr, curf.Face.Face, curf.Color, curf.Background, curf.Decoration, font, ctxt)
					tmpbuf = tmpbuf[0:0]
					tr.Spans = append(tr.Spans, Span{})
					curSp = &(tr.Spans[len(tr.Spans)-1])
					didNl = true
				default:
					didNl = false
					tmpbuf = append(tmpbuf, nb)
				}
			}
			if !didNl {
				unestr := html.UnescapeString(string(tmpbuf))
				// fmt.Printf("%v added: %v\n", bidx, unestr)
				curSp.AppendString(unestr, curf.Face.Face, curf.Color, curf.Background, curf.Decoration, font, ctxt)
				if curLinkIdx >= 0 {
					tl := &tr.Links[curLinkIdx]
					tl.Label = unestr
				}
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////
//  Utilities

func (tx *Text) String() string {
	s := ""
	for i := range tx.Spans {
		sr := &tx.Spans[i]
		s += string(sr.Text) + "\n"
	}
	return s
}

// SetBackground sets the BackgroundColor of the first Render in each Span
// to given value, if was not nil.
func (tx *Text) SetBackground(bg image.Image) {
	for i := range tx.Spans {
		sr := &tx.Spans[i]
		sr.SetBackground(bg)
	}
}

// NextRuneAt returns the next rune starting from given index -- could be at
// that index or some point thereafter -- returns utf8.RuneError if no valid
// rune could be found -- this should be a standard function!
func NextRuneAt(str string, idx int) rune {
	r, _ := utf8.DecodeRuneInString(str[idx:])
	return r
}
