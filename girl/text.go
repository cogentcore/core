// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"bytes"
	"encoding/xml"
	"html"
	"image"
	"io"
	"math"
	"strings"

	"unicode"
	"unicode/utf8"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
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
	Size  mat32.Vec2          `desc:"last size of overall rendered text"`
	Dir   gist.TextDirections `desc:"where relevant, this is the (default, dominant) text direction for the span"`
	Links []TextLink          `desc:"hyperlinks within rendered text"`
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
// use paint rendering for stroking
func (tr *Text) Render(rs *State, pos mat32.Vec2) {
	// pr := prof.Start("RenderText")
	// defer pr.End()

	rs.BackupPaint()
	defer rs.RestorePaint()

	rs.PushXForm(mat32.Identity2D()) // needed for SVG
	defer rs.PopXForm()
	rs.XForm = mat32.Identity2D()

	TextFontRenderMu.Lock()
	defer TextFontRenderMu.Unlock()

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
		if bitflag.Has32(int32(sr.HasDeco), int(gist.DecoBgColor)) {
			sr.RenderBg(rs, tpos)
		}
		if bitflag.HasAny32(int32(sr.HasDeco), int(gist.DecoUnderline), int(gist.DecoDottedUnderline)) {
			sr.RenderUnderline(rs, tpos)
		}
		if bitflag.Has32(int32(sr.HasDeco), int(gist.DecoOverline)) {
			sr.RenderLine(rs, tpos, gist.DecoOverline, 1.1)
		}

		for i, r := range sr.Text {
			rr := &(sr.Render[i])
			if rr.Color != nil {
				curColor = rr.Color
				d.Src = image.NewUniform(curColor)
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
			ll := rp.Add(tx.MulVec2AsVec(mat32.Vec2{0, dsc32}))
			ur := ll.Add(tx.MulVec2AsVec(mat32.Vec2{rr.Size.X, -rr.Size.Y}))
			if int(mat32.Floor(ll.X)) > rs.Bounds.Max.X || int(mat32.Floor(ur.Y)) > rs.Bounds.Max.Y ||
				int(mat32.Ceil(ur.X)) < rs.Bounds.Min.X || int(mat32.Ceil(ll.Y)) < rs.Bounds.Min.Y {
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
				idr := dr.Intersect(rs.Bounds)
				soff := image.ZP
				if dr.Min.X < rs.Bounds.Min.X {
					soff.X = rs.Bounds.Min.X - dr.Min.X
					maskp.X += rs.Bounds.Min.X - dr.Min.X
				}
				if dr.Min.Y < rs.Bounds.Min.Y {
					soff.Y = rs.Bounds.Min.Y - dr.Min.Y
					maskp.Y += rs.Bounds.Min.Y - dr.Min.Y
				}
				draw.DrawMask(d.Dst, idr, d.Src, soff, mask, maskp, draw.Over)
			} else {
				srect := dr.Sub(dr.Min)
				dbase := mat32.Vec2{rp.X - float32(dr.Min.X), rp.Y - float32(dr.Min.Y)}

				transformer := draw.BiLinear
				fx, fy := float32(dr.Min.X), float32(dr.Min.Y)
				m := mat32.Translate2D(fx+dbase.X, fy+dbase.Y).Scale(scx, 1).Rotate(rr.RotRad).Translate(-dbase.X, -dbase.Y)
				s2d := f64.Aff3{float64(m.XX), float64(m.XY), float64(m.X0), float64(m.YX), float64(m.YY), float64(m.Y0)}
				transformer.Transform(d.Dst, s2d, d.Src, srect, draw.Over, &draw.Options{
					SrcMask:  mask,
					SrcMaskP: maskp,
				})
			}
		}
		if bitflag.Has32(int32(sr.HasDeco), int(gist.DecoLineThrough)) {
			sr.RenderLine(rs, tpos, gist.DecoLineThrough, 0.25)
		}
	}
}

// RenderTopPos renders at given top position -- uses first font info to
// compute baseline offset and calls overall Render -- convenience for simple
// widget rendering without layouts
func (tr *Text) RenderTopPos(rs *State, tpos mat32.Vec2) {
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
	tr.Render(rs, pos)
}

// SetString is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after.  Be sure that OpenFont has been run so a
// valid Face is available.  noBG ignores any BgColor in font style, and never
// renders background color
func (tr *Text) SetString(str string, fontSty *gist.Font, ctxt *units.Context, txtSty *gist.Text, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]Span, 1)
	}
	tr.Links = nil
	sr := &(tr.Spans[0])
	sr.SetString(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.Size = mat32.Vec2{ssz.X, mat32.FromFixed(vht)}

}

// SetStringRot90 is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does TB rotated layout (-90 deg).
// Be sure that OpenFont has been run so a valid Face is available.
// noBG ignores any BgColor in font style, and never renders background color
func (tr *Text) SetStringRot90(str string, fontSty *gist.Font, ctxt *units.Context, txtSty *gist.Text, noBG bool, scalex float32) {
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
	tr.Size = mat32.Vec2{mat32.FromFixed(vht), ssz.Y}
}

// SetRunes is for basic text rendering with a single style of text (see
// SetHTML for tag-formatted text) -- configures a single Span with the
// entire string, and does standard layout (LR currently).  rot and scalex are
// general rotation and x-scaling to apply to all chars -- alternatively can
// apply these per character after Be sure that OpenFont has been run so a
// valid Face is available.  noBG ignores any BgColor in font style, and never
// renders background color
func (tr *Text) SetRunes(str []rune, fontSty *gist.Font, ctxt *units.Context, txtSty *gist.Text, noBG bool, rot, scalex float32) {
	if len(tr.Spans) != 1 {
		tr.Spans = make([]Span, 1)
	}
	tr.Links = nil
	sr := &(tr.Spans[0])
	sr.SetRunes(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.Size = mat32.Vec2{ssz.X, mat32.FromFixed(vht)}
}

// SetHTMLSimpleTag sets the styling parameters for simple html style tags
// that only require updating the given font spec values -- returns true if handled
// https://www.w3schools.com/cssref/css_default_values.asp
func SetHTMLSimpleTag(tag string, fs *gist.Font, ctxt *units.Context, cssAgg ki.Props) bool {
	did := false
	switch tag {
	case "b", "strong":
		fs.Weight = gist.WeightBold
		OpenFont(fs, ctxt)
		did = true
	case "i", "em", "var", "cite":
		fs.Style = gist.FontItalic
		OpenFont(fs, ctxt)
		did = true
	case "ins":
		fallthrough
	case "u":
		fs.SetDeco(gist.DecoUnderline)
		did = true
	case "s", "del", "strike":
		fs.SetDeco(gist.DecoLineThrough)
		did = true
	case "sup":
		fs.SetDeco(gist.DecoSuper)
		curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
		curpts -= 2
		fs.Size = units.NewPt(float32(curpts))
		fs.Size.ToDots(ctxt)
		OpenFont(fs, ctxt)
		did = true
	case "sub":
		fs.SetDeco(gist.DecoSub)
		fallthrough
	case "small":
		curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
		curpts -= 2
		fs.Size = units.NewPt(float32(curpts))
		fs.Size.ToDots(ctxt)
		OpenFont(fs, ctxt)
		did = true
	case "big":
		curpts := math.Round(float64(fs.Size.Convert(units.Pt, ctxt).Val))
		curpts += 2
		fs.Size = units.NewPt(float32(curpts))
		fs.Size.ToDots(ctxt)
		OpenFont(fs, ctxt)
		did = true
	case "xx-small", "x-small", "smallf", "medium", "large", "x-large", "xx-large":
		fs.Size = units.NewPt(gist.FontSizePoints[tag])
		fs.Size.ToDots(ctxt)
		OpenFont(fs, ctxt)
		did = true
	case "mark":
		fs.BgColor.SetColor(gist.ThePrefs.PrefColor("highlight"))
		did = true
	case "abbr", "acronym":
		fs.SetDeco(gist.DecoDottedUnderline)
		did = true
	case "tt", "kbd", "samp", "code":
		fs.Family = "monospace"
		OpenFont(fs, ctxt)
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
func (tr *Text) SetHTML(str string, font *gist.Font, txtSty *gist.Text, ctxt *units.Context, cssAgg ki.Props) {
	if txtSty.HasPre() {
		tr.SetHTMLPre([]byte(str), font, txtSty, ctxt, cssAgg)
	} else {
		tr.SetHTMLNoPre([]byte(str), font, txtSty, ctxt, cssAgg)
	}
}

// SetHTMLBytes does SetHTML with bytes as input -- more efficient -- use this
// if already in bytes
func (tr *Text) SetHTMLBytes(str []byte, font *gist.Font, txtSty *gist.Text, ctxt *units.Context, cssAgg ki.Props) {
	if txtSty.HasPre() {
		tr.SetHTMLPre(str, font, txtSty, ctxt, cssAgg)
	} else {
		tr.SetHTMLNoPre(str, font, txtSty, ctxt, cssAgg)
	}
}

// This is the No-Pre parser that uses the golang XML decoder system, which
// strips all whitespace and is thus unsuitable for any Pre case
func (tr *Text) SetHTMLNoPre(str []byte, font *gist.Font, txtSty *gist.Text, ctxt *units.Context, cssAgg ki.Props) {
	//	errstr := "gi.Text SetHTML"
	sz := len(str)
	if sz == 0 {
		return
	}
	tr.Spans = make([]Span, 1)
	tr.Links = nil
	curSp := &(tr.Spans[0])
	initsz := ints.MinInt(sz, 1020)
	curSp.Init(initsz)

	spcstr := bytes.Join(bytes.Fields(str), []byte(" "))

	reader := bytes.NewReader(spcstr)
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel

	OpenFont(font, ctxt)

	// set when a </p> is encountered
	nextIsParaStart := false
	curLinkIdx := -1 // if currently processing an <a> link element

	fstack := make([]*gist.Font, 1, 10)
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
					fs.Color.SetColor(gist.ThePrefs.PrefColor("link"))
					fs.SetDeco(gist.DecoUnderline)
					curLinkIdx = len(tr.Links)
					tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIdx: len(curSp.Text)}
					sprop := make(ki.Props, len(se.Attr))
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
					curSp.AppendRune('“', curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
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
				sprop := make(ki.Props, len(se.Attr))
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "style":
						gist.SetStylePropsXML(attr.Value, &sprop)
					case "class":
						if cssAgg != nil {
							clnm := "." + attr.Value
							if aggp, ok := ki.SubProps(cssAgg, clnm); ok {
								fs.SetStyleProps(nil, aggp, nil)
								OpenFont(&fs, ctxt)
							}
						}
					default:
						sprop[attr.Name.Local] = attr.Value
					}
				}
				fs.SetStyleProps(nil, sprop, nil)
				OpenFont(&fs, ctxt)
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
				curSp.AppendRune('”', curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
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
			curSp.AppendString(sstr, curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco, font, ctxt)
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
func (tr *Text) SetHTMLPre(str []byte, font *gist.Font, txtSty *gist.Text, ctxt *units.Context, cssAgg ki.Props) {
	// errstr := "gi.Text SetHTMLPre"

	sz := len(str)
	tr.Spans = make([]Span, 1)
	tr.Links = nil
	if sz == 0 {
		return
	}
	curSp := &(tr.Spans[0])
	initsz := ints.MinInt(sz, 1020)
	curSp.Init(initsz)

	OpenFont(font, ctxt)

	nextIsParaStart := false
	curLinkIdx := -1 // if currently processing an <a> link element

	fstack := make([]*gist.Font, 1, 10)
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
				curSp.AppendString(string(str[bidx:bidx+1]), curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco, font, ctxt)
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
					curSp.AppendRune('”', curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
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
						fs.Color.SetColor(gist.ThePrefs.PrefColor("link"))
						fs.SetDeco(gist.DecoUnderline)
						curLinkIdx = len(tr.Links)
						tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIdx: len(curSp.Text)}
						if nattr > 0 {
							sprop := make(ki.Props, len(parts)-1)
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
						curSp.AppendRune('“', curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco)
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
					sprop := make(ki.Props, nattr)
					for ai := 0; ai < nattr; ai++ {
						nm := strings.TrimSpace(attr[ai*2])
						vl := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(attr[ai*2+1]), `"`), `"`)
						// fmt.Printf("nm: %v  val: %v\n", nm, vl)
						switch nm {
						case "style":
							gist.SetStylePropsXML(vl, &sprop)
						case "class":
							if cssAgg != nil {
								clnm := "." + vl
								if aggp, ok := ki.SubProps(cssAgg, clnm); ok {
									fs.SetStyleProps(nil, aggp, nil)
									OpenFont(&fs, ctxt)
								}
							}
						default:
							sprop[nm] = vl
						}
					}
					fs.SetStyleProps(nil, sprop, nil)
					OpenFont(&fs, ctxt)
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
					curSp.AppendString(unestr, curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco, font, ctxt)
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
				curSp.AppendString(unestr, curf.Face.Face, curf.Color, curf.BgColor.ColorOrNil(), curf.Deco, font, ctxt)
				if curLinkIdx >= 0 {
					tl := &tr.Links[curLinkIdx]
					tl.Label = unestr
				}
			}
		}
	}
}

// RuneSpanPos returns the position (span, rune index within span) within a
// sequence of spans of a given absolute rune index, starting in the first
// span -- returns false if index is out of range (and returns the last position).
func (tx *Text) RuneSpanPos(idx int) (si, ri int, ok bool) {
	if idx < 0 || len(tx.Spans) == 0 {
		return 0, 0, false
	}
	ri = idx
	for si = range tx.Spans {
		if ri < 0 {
			ri = 0
		}
		sr := &tx.Spans[si]
		if ri >= len(sr.Render) {
			ri -= len(sr.Render)
			continue
		}
		return si, ri, true
	}
	si = len(tx.Spans) - 1
	ri = len(tx.Spans[si].Render) - 1
	return si, ri, false
}

// SpanPosToRuneIdx returns the absolute rune index for a given span, rune
// index position -- i.e., the inverse of RuneSpanPos.  Returns false if given
// input position is out of range, and returns last valid index in that case.
func (tx *Text) SpanPosToRuneIdx(si, ri int) (idx int, ok bool) {
	idx = 0
	for i := range tx.Spans {
		sr := &tx.Spans[i]
		if si > i {
			idx += len(sr.Render)
			continue
		}
		if ri < len(sr.Render) {
			return idx + ri, true
		}
		return idx + (len(sr.Render) - 1), false
	}
	return 0, false
}

// RuneRelPos returns the relative (starting) position of the given rune
// index, counting progressively through all spans present (adds Span RelPos
// and rune RelPos) -- this is typically the baseline position where rendering
// will start, not the upper left corner. If index > length, then uses
// LastPos.  Returns also the index of the span that holds that char (-1 = no
// spans at all) and the rune index within that span, and false if index is
// out of range.
func (tx *Text) RuneRelPos(idx int) (pos mat32.Vec2, si, ri int, ok bool) {
	si, ri, ok = tx.RuneSpanPos(idx)
	if ok {
		sr := &tx.Spans[si]
		return sr.RelPos.Add(sr.Render[ri].RelPos), si, ri, true
	}
	nsp := len(tx.Spans)
	if nsp > 0 {
		sr := &tx.Spans[nsp-1]
		return sr.LastPos, nsp - 1, len(sr.Render), false
	}
	return mat32.Vec2Zero, -1, -1, false
}

// RuneEndPos returns the relative ending position of the given rune index,
// counting progressively through all spans present(adds Span RelPos and rune
// RelPos + rune Size.X for LR writing). If index > length, then uses LastPos.
// Returns also the index of the span that holds that char (-1 = no spans at
// all) and the rune index within that span, and false if index is out of
// range.
func (tx *Text) RuneEndPos(idx int) (pos mat32.Vec2, si, ri int, ok bool) {
	si, ri, ok = tx.RuneSpanPos(idx)
	if ok {
		sr := &tx.Spans[si]
		spos := sr.RelPos.Add(sr.Render[ri].RelPos)
		spos.X += sr.Render[ri].Size.X
		return spos, si, ri, true
	}
	nsp := len(tx.Spans)
	if nsp > 0 {
		sr := &tx.Spans[nsp-1]
		return sr.LastPos, nsp - 1, len(sr.Render), false
	}
	return mat32.Vec2Zero, -1, -1, false
}

//////////////////////////////////////////////////////////////////////////////////
//  TextStyle-based Layout Routines

// LayoutStdLR does basic standard layout of text in LR direction, assigning
// relative positions to spans and runes according to given styles, and given
// size overall box (nonzero values used to constrain). Returns total
// resulting size box for text.  Font face in gist.Font is used for
// determining line spacing here -- other versions can do more expensive
// calculations of variable line spacing as needed.
func (tr *Text) LayoutStdLR(txtSty *gist.Text, fontSty *gist.Font, ctxt *units.Context, size mat32.Vec2) mat32.Vec2 {
	if len(tr.Spans) == 0 {
		return mat32.Vec2Zero
	}

	// pr := prof.Start("TextLayout")
	// defer pr.End()
	//
	tr.Dir = gist.LRTB
	OpenFont(fontSty, ctxt)
	fht := fontSty.Face.Metrics.Height
	dsc := mat32.FromFixed(fontSty.Face.Face.Metrics().Descent)
	lspc := fht * txtSty.EffLineHeight()
	lpad := (lspc - fht) / 2 // padding above / below text box for centering in line

	maxw := float32(0)

	// first pass gets rune positions and wraps text as needed, and gets max width
	si := 0
	for si < len(tr.Spans) {
		sr := &(tr.Spans[si])
		if err := sr.IsValid(); err != nil {
			// log.Print(err)
			si++
			continue
		}
		if sr.LastPos.X == 0 { // don't re-do unless necessary
			sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
		}
		if sr.IsNewPara() {
			sr.RelPos.X = txtSty.Indent.Dots
		} else {
			sr.RelPos.X = 0
		}
		ssz := sr.SizeHV()
		ssz.X += sr.RelPos.X
		if size.X > 0 && ssz.X > size.X && txtSty.HasWordWrap() {
			for {
				wp := sr.FindWrapPosLR(size.X, ssz.X)
				if wp > 0 && wp < len(sr.Text)-1 {
					nsr := sr.SplitAtLR(wp)
					tr.InsertSpan(si+1, nsr)
					ssz = sr.SizeHV()
					ssz.X += sr.RelPos.X
					if ssz.X > maxw {
						maxw = ssz.X
					}
					si++
					sr = &(tr.Spans[si]) // keep going with nsr
					sr.SetRunePosLR(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
					ssz = sr.SizeHV()

					// fixup links
					for li := range tr.Links {
						tl := &tr.Links[li]
						if tl.StartSpan == si-1 {
							if tl.StartIdx >= wp {
								tl.StartIdx -= wp
								tl.StartSpan++
							}
						} else if tl.StartSpan > si-1 {
							tl.StartSpan++
						}
						if tl.EndSpan == si-1 {
							if tl.EndIdx >= wp {
								tl.EndIdx -= wp
								tl.EndSpan++
							}
						} else if tl.EndSpan > si-1 {
							tl.EndSpan++
						}
					}

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

	// make sure links are still in range
	for li := range tr.Links {
		tl := &tr.Links[li]
		stsp := tr.Spans[tl.StartSpan]
		if tl.StartIdx >= len(stsp.Text) {
			tl.StartIdx = len(stsp.Text) - 1
		}
		edsp := tr.Spans[tl.EndSpan]
		if tl.EndIdx >= len(edsp.Text) {
			tl.EndIdx = len(edsp.Text) - 1
		}
	}

	if maxw > size.X {
		size.X = maxw
	}

	// vertical alignment
	nsp := len(tr.Spans)
	npara := 0
	for si := 1; si < nsp; si++ {
		sr := &(tr.Spans[si])
		if sr.IsNewPara() {
			npara++
		}
	}

	vht := lspc*float32(nsp) + float32(npara)*txtSty.ParaSpacing.Dots
	if vht > size.Y {
		size.Y = vht
	}

	tr.Size = mat32.Vec2{maxw, vht}

	vpad := float32(0) // padding at top to achieve vertical alignment
	vextra := size.Y - vht
	if vextra > 0 {
		switch {
		case gist.IsAlignMiddle(txtSty.AlignV):
			vpad = vextra / 2
		case gist.IsAlignEnd(txtSty.AlignV):
			vpad = vextra
		}
	}

	vbaseoff := lspc - lpad - dsc // offset of baseline within overall line
	vpos := vpad + vbaseoff

	for si := range tr.Spans {
		sr := &(tr.Spans[si])
		if si > 0 && sr.IsNewPara() {
			vpos += txtSty.ParaSpacing.Dots
		}
		sr.RelPos.Y = vpos
		sr.LastPos.Y = vpos
		ssz := sr.SizeHV()
		ssz.X += sr.RelPos.X
		hextra := size.X - ssz.X
		if hextra > 0 {
			switch {
			case gist.IsAlignMiddle(txtSty.Align):
				sr.RelPos.X += hextra / 2
			case gist.IsAlignEnd(txtSty.Align):
				sr.RelPos.X += hextra
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
