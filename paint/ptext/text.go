// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"bytes"
	"encoding/xml"
	"html"
	"image"
	"io"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"unicode"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
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

	// bounding box for the rendered text.  use Size() method to get the size.
	BBox math32.Box2

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

	// Context is our rendering context
	Context render.Context

	// Position for rendering
	RenderPos math32.Vector2
}

func (tr *Text) IsRenderItem() {}

// InsertSpan inserts a new span at given index
func (tr *Text) InsertSpan(at int, ns *Span) {
	tr.Spans = slices.Insert(tr.Spans, at, *ns)
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
	tr.BBox.Min.SetZero()
	tr.BBox.Max = math32.Vec2(ssz.X, math32.FromFixed(vht))
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
	rot := float32(math32.Pi / 2)
	sr.SetString(str, fontSty, ctxt, noBG, rot, scalex)
	sr.SetRunePosTBRot(txtSty.LetterSpacing.Dots, txtSty.WordSpacing.Dots, fontSty.Face.Metrics.Ch, txtSty.TabSize)
	ssz := sr.SizeHV()
	vht := fontSty.Face.Face.Metrics().Height
	tr.BBox.Min.SetZero()
	tr.BBox.Max = math32.Vec2(math32.FromFixed(vht), ssz.Y)
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
	tr.BBox.Min.SetZero()
	tr.BBox.Max = math32.Vec2(ssz.X, math32.FromFixed(vht))
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
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Value))
		curpts -= 2
		fs.Size = units.Pt(float32(curpts))
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "sub":
		fs.SetDecoration(styles.DecoSub)
		fallthrough
	case "small":
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Value))
		curpts -= 2
		fs.Size = units.Pt(float32(curpts))
		fs.Size.ToDots(ctxt)
		fs.Font = OpenFont(fs, ctxt)
		did = true
	case "big":
		curpts := math.Round(float64(fs.Size.Convert(units.UnitPt, ctxt).Value))
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
		fs.Background = colors.Scheme.Warn.Container
		did = true
	case "abbr", "acronym":
		fs.SetDecoration(styles.DecoDottedUnderline)
		did = true
	case "tt", "kbd", "samp", "code":
		fs.Family = "monospace"
		fs.Font = OpenFont(fs, ctxt)
		fs.Background = colors.Scheme.SurfaceContainer
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
	//	errstr := "core.Text SetHTML"
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
	curLinkIndex := -1 // if currently processing an <a> link element

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
			curLinkIndex = -1
			if !SetHTMLSimpleTag(nm, &fs, ctxt, cssAgg) {
				switch nm {
				case "a":
					fs.Color = colors.Scheme.Primary.Base
					fs.SetDecoration(styles.Underline)
					curLinkIndex = len(tr.Links)
					tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIndex: len(curSp.Text)}
					sprop := make(map[string]any, len(se.Attr))
					tl.Properties = sprop
					for _, attr := range se.Attr {
						if attr.Name.Local == "href" {
							tl.URL = attr.Value
						}
						sprop[attr.Name.Local] = attr.Value
					}
					tr.Links = append(tr.Links, *tl)
				case "span":
					// just uses properties
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
						styles.SetStylePropertiesXML(attr.Value, &sprop)
					case "class":
						if cssAgg != nil {
							clnm := "." + attr.Value
							if aggp, ok := styles.SubProperties(cssAgg, clnm); ok {
								fs.SetStyleProperties(nil, aggp, nil)
								fs.Font = OpenFont(&fs, ctxt)
							}
						}
					default:
						sprop[attr.Name.Local] = attr.Value
					}
				}
				fs.SetStyleProperties(nil, sprop, nil)
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
				if curLinkIndex >= 0 && curLinkIndex < len(tr.Links) {
					tl := &tr.Links[curLinkIndex]
					tl.EndSpan = len(tr.Spans) - 1
					tl.EndIndex = len(curSp.Text)
					curLinkIndex = -1
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
			if curLinkIndex >= 0 && curLinkIndex < len(tr.Links) {
				tl := &tr.Links[curLinkIndex]
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
	// errstr := "core.Text SetHTMLPre"

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
	curLinkIndex := -1 // if currently processing an <a> link element

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
					if curLinkIndex >= 0 && curLinkIndex < len(tr.Links) {
						tl := &tr.Links[curLinkIndex]
						tl.EndSpan = len(tr.Spans) - 1
						tl.EndIndex = len(curSp.Text)
						curLinkIndex = -1
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
				curLinkIndex = -1
				if !SetHTMLSimpleTag(stag, &fs, ctxt, cssAgg) {
					switch stag {
					case "a":
						fs.Color = colors.Scheme.Primary.Base
						fs.SetDecoration(styles.Underline)
						curLinkIndex = len(tr.Links)
						tl := &TextLink{StartSpan: len(tr.Spans) - 1, StartIndex: len(curSp.Text)}
						if nattr > 0 {
							sprop := make(map[string]any, len(parts)-1)
							tl.Properties = sprop
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
						// just uses properties
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
							styles.SetStylePropertiesXML(vl, &sprop)
						case "class":
							if cssAgg != nil {
								clnm := "." + vl
								if aggp, ok := styles.SubProperties(cssAgg, clnm); ok {
									fs.SetStyleProperties(nil, aggp, nil)
									fs.Font = OpenFont(&fs, ctxt)
								}
							}
						default:
							sprop[nm] = vl
						}
					}
					fs.SetStyleProperties(nil, sprop, nil)
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
				if curLinkIndex >= 0 && curLinkIndex < len(tr.Links) {
					tl := &tr.Links[curLinkIndex]
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

// UpdateColors sets the font styling colors the first rune
// based on the given font style parameters.
func (tx *Text) UpdateColors(sty *styles.FontRender) {
	for i := range tx.Spans {
		sr := &tx.Spans[i]
		sr.UpdateColors(sty)
	}
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
