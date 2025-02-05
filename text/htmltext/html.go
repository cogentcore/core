// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"unicode"

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/text/rich"
	"golang.org/x/net/html/charset"
)

// HTMLToRich translates HTML-formatted rich text into a [rich.Text],
// using given initial text styling parameters and css properties.
// This uses the golang XML decoder system, which strips all whitespace
// and therefore does not capture any preformatted text. See HTMLPre.
// cssProps are a list of css key-value pairs that are used to set styling
// properties for the text, and can include class names with a value of
// another property map that is applied to elements of that class,
// including standard elements like a for links, etc.
func HTMLToRich(str []byte, sty *rich.Style, cssProps map[string]any) (rich.Text, error) {
	sz := len(str)
	if sz == 0 {
		return nil, nil
	}
	var errs []error

	spcstr := bytes.Join(bytes.Fields(str), []byte(" "))

	reader := bytes.NewReader(spcstr)
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charset.NewReaderLabel

	// set when a </p> is encountered
	nextIsParaStart := false

	// stack of font styles
	fstack := make(stack.Stack[*rich.Style], 0)
	fstack.Push(sty)

	// stack of rich text spans that are later joined for final result
	spstack := make(stack.Stack[rich.Text], 0)
	curSp := rich.NewText(sty, nil)
	spstack.Push(curSp)

	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			errs = append(errs, err)
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			fs := rich.NewStyle() // new style for new element
			*fs = *fstack.Peek()
			nm := strings.ToLower(se.Name.Local)
			insertText := []rune{}
			if !fs.SetFromHTMLTag(nm) {
				switch nm {
				case "a":
					fs.SetFillColor(colors.ToUniform(colors.Scheme.Primary.Base))
					fs.Decoration.SetFlag(true, rich.Underline)
					for _, attr := range se.Attr {
						if attr.Name.Local == "href" {
							fs.SetLink(attr.Value)
						}
					}
				case "span":
					// just uses properties
				case "q":
					atStart := curSp.Len() == 0
					if nextIsParaStart && atStart {
						fs.Decoration.SetFlag(true, rich.ParagraphStart)
					}
					nextIsParaStart = false
					insertText = []rune{'“'}
				case "dfn":
					// no default styling
				case "bdo":
					// bidirectional override..
				case "p":
					fs.Decoration.SetFlag(true, rich.ParagraphStart)
					nextIsParaStart = true // todo: redundant?
				case "br":
					insertText = []rune{'\n'}
					nextIsParaStart = false
				default:
					err := fmt.Errorf("%q tag not recognized", nm)
					errs = append(errs, err)
				}
			}
			if len(se.Attr) > 0 {
				sprop := make(map[string]any, len(se.Attr))
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "style":
						styleprops.FromXMLString(attr.Value, sprop)
					case "class":
						if cssProps != nil {
							clnm := "." + attr.Value
							if aggp, ok := SubProperties(clnm, cssProps); ok {
								fs.StyleFromProperties(nil, aggp, nil)
							}
						}
					default:
						sprop[attr.Name.Local] = attr.Value
					}
				}
				fs.StyleFromProperties(nil, sprop, nil)
			}
			if cssProps != nil {
				FontStyleCSS(fs, nm, cssProps)
			}
			fstack.Push(fs)
			if curSp.Len() == 0 && len(spstack) > 0 { // we started something but added nothing to it.
				spstack.Pop()
			}
			curSp = rich.NewText(fs, insertText)
			spstack.Push(curSp)
		case xml.EndElement:
			switch se.Name.Local {
			case "p":
				nextIsParaStart = true
			case "br":
				curSp.AddRunes([]rune{'\n'}) // todo: different char?
				nextIsParaStart = false
			case "q":
				curSp.AddRunes([]rune{'”'})
			}

			if len(fstack) > 0 {
				fstack.Pop()
				fs := fstack.Peek()
				curSp = rich.NewText(fs, nil)
				spstack.Push(curSp) // start a new span with previous style
			} else {
				err := fmt.Errorf("imbalanced start / end tags: %q", se.Name.Local)
				errs = append(errs, err)
			}
		case xml.CharData:
			atStart := curSp.Len() == 0
			sstr := html.UnescapeString(string(se))
			if nextIsParaStart && atStart {
				sstr = strings.TrimLeftFunc(sstr, func(r rune) bool {
					return unicode.IsSpace(r)
				})
			}
			curSp.AddRunes([]rune(sstr))
		}
	}
	return rich.Join(spstack...), errors.Join(errs...)
}

/*

// SetHTMLPre sets preformatted HTML-styled text by decoding all standard
// inline HTML text style formatting tags in the string and sets the
// per-character font information appropriately, using given font style info.
// Only basic styling tags, including <span> elements with style parameters
// (including class names) are decoded.  Whitespace is decoded as-is,
// including LF \n etc, except in WhiteSpacePreLine case which only preserves LF's
func (tr *Text) SetHTMLPre(str []byte, font *rich.FontRender, txtSty *rich.Text, ctxt *units.Context, cssAgg map[string]any) {
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

	fstack := make([]*rich.FontRender, 1, 10)
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
						fs.SetDecoration(rich.Underline)
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
							rich.SetStylePropertiesXML(vl, &sprop)
						case "class":
							if cssAgg != nil {
								clnm := "." + vl
								if aggp, ok := rich.SubProperties(cssAgg, clnm); ok {
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

*/

// SubProperties returns a properties map[string]any from given key tag
// of given properties map, if the key exists and the value is a sub props map.
// Otherwise returns nil, false
func SubProperties(tag string, cssProps map[string]any) (map[string]any, bool) {
	tp, ok := cssProps[tag]
	if !ok {
		return nil, false
	}
	pmap, ok := tp.(map[string]any)
	if !ok {
		return nil, false
	}
	return pmap, true
}

// FontStyleCSS looks for "tag" name properties in cssProps properties, and applies those to
// style if found, and returns true -- false if no such tag found
func FontStyleCSS(fs *rich.Style, tag string, cssProps map[string]any) bool {
	if cssProps == nil {
		return false
	}
	pmap, ok := SubProperties(tag, cssProps)
	if !ok {
		return false
	}
	fs.StyleFromProperties(nil, pmap, nil)
	return true
}
