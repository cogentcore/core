// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"strings"

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/text/rich"
)

// HTMLPreToRich translates preformatted HTML-styled text into a [rich.Text]
// using given initial text styling parameters and css properties.
// This uses a custom decoder that preserves all whitespace characters,
// and decodes all standard inline HTML text style formatting tags in the string.
// Only basic styling tags, including <span> elements with style parameters
// (including class names) are decoded.  Whitespace is decoded as-is,
// including LF \n etc, except in WhiteSpacePreLine case which only preserves LF's.
func HTMLPreToRich(str []byte, sty *rich.Style, cssProps map[string]any) (rich.Text, error) {
	sz := len(str)
	if sz == 0 {
		return nil, nil
	}
	var errs []error

	// set when a </p> is encountered
	nextIsParaStart := false

	// stack of font styles
	fstack := make(stack.Stack[*rich.Style], 0)
	fstack.Push(sty)

	// stack of rich text spans that are later joined for final result
	spstack := make(stack.Stack[rich.Text], 0)
	curSp := rich.NewText(sty, nil)
	spstack.Push(curSp)

	tagstack := make(stack.Stack[string], 0)

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
				curSp.AddRunes([]rune(string(str[bidx : bidx+1])))
				bidx++
			}
		}
		if ftag != "" {
			if ftag[0] == '/' { // EndElement
				etag := strings.ToLower(ftag[1:])
				// fmt.Printf("%v  etag: %v\n", bidx, etag)
				if etag == "pre" {
					continue // ignore
				}
				if etag != curTag {
					err := fmt.Errorf("end tag: %q doesn't match current tag: %q", etag, curTag)
					errs = append(errs, err)
				}
				switch etag {
				case "p":
					curSp.AddRunes([]rune{'\n'})
					nextIsParaStart = true
				case "br":
					curSp.AddRunes([]rune{'\n'})
					nextIsParaStart = false
				case "a", "q", "math", "sub", "sup": // important: any special must be ended!
					curSp.EndSpecial()
				}
				if len(fstack) > 0 {
					fstack.Pop()
					fs := fstack.Peek()
					curSp = rich.NewText(fs, nil)
					spstack.Push(curSp) // start a new span with previous style
				} else {
					err := fmt.Errorf("imbalanced start / end tags: %q", etag)
					errs = append(errs, err)
				}
				tslen := len(tagstack)
				if tslen > 1 {
					tagstack.Pop()
					curTag = tagstack.Peek()
				} else if tslen == 1 {
					tagstack.Pop()
					curTag = ""
				} else {
					err := fmt.Errorf("imbalanced start / end tags: %q", curTag)
					errs = append(errs, err)
				}
			} else { // StartElement
				parts := strings.Split(ftag, " ")
				stag := strings.ToLower(strings.TrimSpace(parts[0]))
				// fmt.Printf("%v  stag: %v\n", bidx, stag)
				attrs := parts[1:]
				attr := strings.Split(strings.Join(attrs, " "), "=")
				nattr := len(attr) / 2
				fs := fstack.Peek().Clone() // new style for new element
				atStart := curSp.Len() == 0
				if nextIsParaStart && atStart {
					fs.Decoration.SetFlag(true, rich.ParagraphStart)
				}
				nextIsParaStart = false
				insertText := []rune{}
				special := rich.Nothing
				linkURL := ""
				if !fs.SetFromHTMLTag(stag) {
					switch stag {
					case "a":
						special = rich.Link
						fs.SetLinkStyle()
						if nattr > 0 {
							sprop := make(map[string]any, len(parts)-1)
							for ai := 0; ai < nattr; ai++ {
								nm := strings.TrimSpace(attr[ai*2])
								vl := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(attr[ai*2+1]), `"`), `"`)
								if nm == "href" {
									linkURL = vl
								}
								sprop[nm] = vl
							}
						}
					case "span":
						// just uses properties
					case "q":
						special = rich.Quote
					case "math":
						special = rich.MathInline
					case "sup":
						special = rich.Super
						fs.Size = 0.8
					case "sub":
						special = rich.Sub
						fs.Size = 0.8
					case "dfn":
						// no default styling
					case "bdo":
						// todo: bidirectional override..
					case "pre": // nop
					case "p":
						fs.Decoration.SetFlag(true, rich.ParagraphStart)
					case "br":
						curSp = rich.NewText(fs, []rune{'\n'}) // br is standalone: do it!
						spstack.Push(curSp)
						nextIsParaStart = false
					default:
						err := fmt.Errorf("%q tag not recognized", stag)
						errs = append(errs, err)
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
							styleprops.FromXMLString(vl, sprop)
						case "class":
							if vl == "math inline" {
								special = rich.MathInline
							}
							if vl == "math display" {
								special = rich.MathDisplay
							}
							if cssProps != nil {
								clnm := "." + vl
								if aggp, ok := SubProperties(clnm, cssProps); ok {
									fs.FromProperties(nil, aggp, nil)
								}
							}
						default:
							sprop[nm] = vl
						}
					}
					fs.FromProperties(nil, sprop, nil)
				}
				if cssProps != nil {
					FontStyleCSS(fs, stag, cssProps)
				}
				fstack.Push(fs)
				curTag = stag
				tagstack.Push(curTag)
				if curSp.Len() == 0 && len(spstack) > 0 { // we started something but added nothing to it.
					spstack.Pop()
				}
				if special != rich.Nothing {
					ss := fs.Clone() // key about specials: make a new one-off style so special doesn't repeat
					ss.Special = special
					if special == rich.Link {
						ss.URL = linkURL
					}
					curSp = rich.NewText(ss, insertText)
				} else {
					curSp = rich.NewText(fs, insertText)
				}
				spstack.Push(curSp)
			}
		} else { // raw chars
			// todo: deal with WhiteSpacePreLine -- trim out non-LF ws
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
					curSp.AddRunes([]rune(unestr + "\n"))
					curSp = rich.NewText(fstack.Peek(), nil)
					spstack.Push(curSp) // start a new span with previous style
					tmpbuf = tmpbuf[0:0]
					didNl = true
				default:
					didNl = false
					tmpbuf = append(tmpbuf, nb)
				}
			}
			if !didNl {
				unestr := html.UnescapeString(string(tmpbuf))
				// fmt.Printf("%v added: %v\n", bidx, unestr)
				curSp.AddRunes([]rune(unestr))
			}
		}
	}
	return rich.Join(spstack...), errors.Join(errs...)
}
