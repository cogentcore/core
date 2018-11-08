// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	htmlstd "html"
	"log"
	"sort"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/goki/gi"
	"github.com/goki/ki"
)

// TagRegion defines a region of a line of text that has a given markup tag
// region is only defined in terms of character positions -- line is implicit
type TagRegion struct {
	Tag chroma.TokenType `desc:"tag for this region of text"`
	St  int              `desc:"starting character position"`
	Ed  int              `desc:"ending character position -- exclusive (after last char)"`
}

// TagRegionsMerge merges the two tag regions into a combined list
// properly ordered by sequence of tags within the line.
// returns true if there are no conflicts between the two sets of tags
// and false if there are conflicts
func TagRegionsMerge(t1, t2 []TagRegion) ([]TagRegion, bool) {
	if len(t1) == 0 {
		return t2, true
	}
	if len(t2) == 0 {
		return t1, true
	}
	sz1 := len(t1)
	sz2 := len(t2)
	tsz := sz1 + sz2
	tl := make([]TagRegion, 0, tsz)
	i1 := 0
	i2 := 0
	ok := true
	for i := 0; i < tsz; i++ {
		c1 := t1[i1]
		c2 := t2[i2]
		if c1.St < c2.St && c1.Ed < c2.St {
			tl = append(tl, c1)
			i1++
			if i1 >= sz1 {
				t1 = append(tl, t2[i2:sz2]...)
				break
			}
		} else if c2.St < c1.St && c2.Ed < c1.St {
			tl = append(tl, c2)
			i2++
			if i2 >= sz2 {
				t1 = append(tl, t1[i1:sz1]...)
				break
			}
		} else {
			ok = false
			fmt.Printf("conflicting tags: %v  vs  %v\n", c1, c2)
			i1++
			i2++
			if i1 >= sz1 {
				t1 = append(tl, t2[i2:sz2]...)
				break
			}
			if i2 >= sz2 {
				t1 = append(tl, t1[i1:sz1]...)
				break
			}
		}
	}
	return tl, ok
}

// TagRegionsAdd adds a new tag region in sorted order to list
func TagRegionsAdd(tl *[]TagRegion, tr TagRegion) {
	for i, t := range *tl {
		if t.St < tr.St {
			continue
		}
		if t == tr { // dupe
			return
		}
		*tl = append(*tl, tr)
		copy((*tl)[i+1:], (*tl)[i:])
		(*tl)[i] = tr
		return
	}
	*tl = append(*tl, tr)
}

// TagRegionsSort sorts the tags by starting pos
func TagRegionsSort(tags []TagRegion) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].St < tags[j].St
	})
}

// CustomGeneric tokens -- this is where we put anything not in chroma
const (
	// spelling error
	GenericSpellErr chroma.TokenType = 7900 + iota
)

// HiStyleName is a highlighting style name
type HiStyleName string

// HiStyleDefault is the default highlighting style name -- can set this to whatever you want
var HiStyleDefault = HiStyleName("emacs")

// HiMarkup manages the syntax highlighting state for TextBuf
type HiMarkup struct {
	Lang      string        `desc:"language for syntax highlighting the code"`
	Style     HiStyleName   `desc:"syntax highlighting style"`
	Has       bool          `desc:"true if both lang and style are set"`
	TabSize   int           `desc:"tab size, in chars"`
	CSSheet   gi.StyleSheet `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	CSSProps  ki.Props      `json:"-" xml:"-" desc:"Commpiled CSS properties for given highlighting style"`
	lastLang  string
	lastStyle HiStyleName
	lexer     chroma.Lexer
	formatter *html.Formatter
	style     *chroma.Style
}

// HasHi returns true if there are highighting parameters set (only valid after Init)
func (hm *HiMarkup) HasHi() bool {
	return hm.Has
}

// Init initializes the syntax highlighting for current params
func (hm *HiMarkup) Init() {
	if hm.Lang == "" || hm.Style == "" {
		hm.Has = false
		return
	}
	hm.Has = true
	if hm.Lang == hm.lastLang && hm.Style == hm.lastStyle {
		return
	}
	hm.lexer = chroma.Coalesce(lexers.Get(hm.Lang))
	hm.formatter = html.New(html.WithClasses(), html.TabWidth(hm.TabSize))
	hm.style = styles.Get(string(hm.Style))
	if hm.style == nil {
		hm.style = styles.Fallback
	}
	var cssBuf bytes.Buffer
	err := hm.formatter.WriteCSS(&cssBuf, hm.style)
	if err != nil {
		log.Println(err)
		return
	}
	csstr := cssBuf.String()
	csstr = strings.Replace(csstr, " .chroma .", " .", -1)
	hm.CSSheet.ParseString(csstr)
	hm.CSSProps = hm.CSSheet.CSSProps()

	hm.lastLang = hm.Lang
	hm.lastStyle = hm.Style
}

// MarkupTags returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTags(txt []byte) ([][]TagRegion, error) {
	txtstr := string(txt)
	iterator, err := hm.lexer.Tokenise(nil, txtstr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	lines := chroma.SplitTokensIntoLines(iterator.Tokens())
	sz := len(lines)
	tags := make([][]TagRegion, sz)
	for li, lt := range lines {
		cp := 0
		for _, tok := range lt {
			str := strings.TrimSuffix(tok.Value, "\n")
			slen := len(str)
			if slen == 0 {
				continue
			}
			ep := cp + slen
			if tok.Type < chroma.Text {
				nt := TagRegion{Tag: tok.Type, St: cp, Ed: ep}
				tags[li] = append(tags[li], nt)
			}
			cp = ep
		}
	}
	return tags, nil
}

// MarkupTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsLine(txt []byte) ([]TagRegion, error) {
	txtstr := string(txt) + "\n"
	iterator, err := hm.lexer.Tokenise(nil, txtstr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var tags []TagRegion
	cp := 0
	toks := iterator.Tokens()
	for _, tok := range toks {
		str := strings.TrimSuffix(tok.Value, "\n")
		slen := len(str)
		if slen == 0 {
			continue
		}
		if tok.Type == chroma.None { // always a parsing err AFAIK
			// fmt.Printf("type: %v  st: %v  ed: %v  txt: %v\n", tok.Type, cp, ep, str)
			continue
		}
		ep := cp + slen
		if tok.Type < chroma.Text {
			nt := TagRegion{Tag: tok.Type, St: cp, Ed: ep}
			tags = append(tags, nt)
		}
		cp = ep
	}
	return tags, nil
}

// MarkupLine returns the line with html class tags added for each tag
// takes both the hi tags and extra tags
func (hm *HiMarkup) MarkupLine(txt []byte, hitags, tags []TagRegion) []byte {
	ttags, _ := TagRegionsMerge(hitags, tags)
	if len(ttags) == 0 {
		return txt
	}
	sps := []byte(`<span class="`)
	sps2 := []byte(`">`)
	spe := []byte(`</span>`)
	taglen := len(sps) + len(sps2) + len(spe) + 2
	sz := len(txt)
	musz := sz + len(ttags)*taglen
	mu := make([]byte, 0, musz)
	cp := 0
	for _, tr := range ttags {
		if tr.St >= sz || tr.Ed > sz {
			break
		}
		if tr.St > cp {
			mu = append(mu, []byte(htmlstd.EscapeString(string(txt[cp:tr.St])))...)
		}
		mu = append(mu, sps...)
		mu = append(mu, []byte(chroma.StandardTypes[tr.Tag])...)
		mu = append(mu, sps2...)
		mu = append(mu, []byte(htmlstd.EscapeString(string(txt[tr.St:tr.Ed])))...)
		mu = append(mu, spe...)
		cp = tr.Ed
	}
	if sz > cp {
		mu = append(mu, []byte(htmlstd.EscapeString(string(txt[cp:sz])))...)
	}
	return mu
}

// todo: currently based on https://github.com/alecthomas/chroma styles, but we should
// impl our own structured style obj with a list of categories and
// corresponding colors, once we do the parsing etc

// HiStyles are all the available highlighting styles
var HiStyles = []string{
	"abap",
	"algol",
	"algol_nu",
	"api",
	"arduino",
	"autumn",
	"borland",
	"bw",
	"colorful",
	"dracula",
	"emacs",
	"friendly",
	"fruity",
	"github",
	"igor",
	"lovelace",
	"manni",
	"monokai",
	"monokailight",
	"murphy",
	"native",
	"paraiso-dark",
	"paraiso-light",
	"pastie",
	"perldoc",
	"pygments",
	"rainbow_dash",
	"rrt",
	"solarized-dark",
	"solarized-dark256",
	"solarized-light",
	"swapoff",
	"tango",
	"trac",
	"vim",
	"vs",
	"xcode",
}
