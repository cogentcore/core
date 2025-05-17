// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
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
			fs := fstack.Peek().Clone() // new style for new element
			atStart := curSp.Len() == 0
			if nextIsParaStart && atStart {
				fs.Decoration.SetFlag(true, rich.ParagraphStart)
			}
			nextIsParaStart = false
			nm := strings.ToLower(se.Name.Local)
			insertText := []rune{}
			special := rich.Nothing
			linkURL := ""
			if !fs.SetFromHTMLTag(nm) {
				switch nm {
				case "a":
					special = rich.Link
					fs.SetLinkStyle()
					for _, attr := range se.Attr {
						if attr.Name.Local == "href" {
							linkURL = attr.Value
						}
					}
				case "span": // todo: , "pre"
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
				case "p":
					// todo: detect <p> at end of paragraph only
					fs.Decoration.SetFlag(true, rich.ParagraphStart)
				case "br":
					// handled in end: standalone <br> is in both start and end
				case "err":
					// custom; used to mark errors
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
						if attr.Value == "math inline" {
							special = rich.MathInline
						}
						if attr.Value == "math display" {
							special = rich.MathDisplay
						}
						if cssProps != nil {
							clnm := "." + attr.Value
							if aggp, ok := SubProperties(clnm, cssProps); ok {
								fs.FromProperties(nil, aggp, nil)
							}
						}
					default:
						sprop[attr.Name.Local] = attr.Value
					}
				}
				fs.FromProperties(nil, sprop, nil)
			}
			if cssProps != nil {
				FontStyleCSS(fs, nm, cssProps)
			}
			fstack.Push(fs)
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
		case xml.EndElement:
			switch se.Name.Local {
			case "p":
				curSp.AddRunes([]rune{'\n'})
				nextIsParaStart = true
			case "br":
				curSp.AddRunes([]rune{'\n'})
				nextIsParaStart = false
			case "a", "q", "math", "sub", "sup": // important: any special must be ended!
				nsp := rich.Text{}
				nsp.EndSpecial()
				spstack.Push(nsp)
			case "span":
				sty, stx := curSp.Span(0)
				if sty.Special != rich.Nothing {
					if sty.IsMath() {
						stx = runes.TrimPrefix(stx, []rune("\\("))
						stx = runes.TrimSuffix(stx, []rune("\\)"))
						stx = runes.TrimPrefix(stx, []rune("\\["))
						stx = runes.TrimSuffix(stx, []rune("\\]"))
						// fmt.Println("math:", string(stx))
						curSp.SetSpanRunes(0, stx)
					}
					nsp := rich.Text{}
					nsp.EndSpecial()
					spstack.Push(nsp)
				}
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
	fs.FromProperties(nil, pmap, nil)
	return true
}
