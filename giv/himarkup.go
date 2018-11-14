// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	htmlstd "html"
	"log"
	"sort"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/goki/gi/histyle"
	"github.com/goki/ki"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/nptime"
)

// TagRegion defines a region of a line of text that has a given markup tag
// region is only defined in terms of character positions -- line is implicit
type TagRegion struct {
	Tag  histyle.HiTags `desc:"tag for this region of text"`
	St   int            `desc:"starting character position"`
	Ed   int            `desc:"ending character position -- exclusive (after last char)"`
	Time nptime.Time    `desc:"time when region was set -- needed for updating locations in the text based on time stamp (using efficient non-pointer time)"`
}

// OverlapsReg returns true if the two regions overlap
func (tr *TagRegion) OverlapsReg(or TagRegion) bool {
	// start overlaps
	if (tr.St >= or.St && tr.St < or.Ed) || (or.St >= tr.St && or.St < tr.Ed) {
		return true
	}
	// end overlaps
	if (tr.Ed > or.St && tr.Ed <= or.Ed) || (or.Ed > tr.St && or.Ed <= tr.Ed) {
		return true
	}
	return false
}

// ContainsPos returns true if the region contains the given point
func (tr *TagRegion) ContainsPos(pos int) bool {
	return pos >= tr.St && pos < tr.Ed
}

// TagRegionsMerge merges the two tag regions into a combined list
// properly ordered by sequence of tags within the line.
func TagRegionsMerge(t1, t2 []TagRegion) []TagRegion {
	if len(t1) == 0 {
		return t2
	}
	if len(t2) == 0 {
		return t1
	}
	sz1 := len(t1)
	sz2 := len(t2)
	tsz := sz1 + sz2
	tl := make([]TagRegion, 0, tsz)
	for i := 0; i < sz1; i++ {
		tl = append(tl, t1[i])
	}
	for i := 0; i < sz2; i++ {
		TagRegionsAdd(&tl, t2[i])
	}
	return tl
}

// TagRegionsAdd adds a new tag region in sorted order to list
func TagRegionsAdd(tl *[]TagRegion, tr TagRegion) {
	for i, t := range *tl {
		if t.St < tr.St {
			continue
		}
		*tl = append(*tl, tr)
		copy((*tl)[i+1:], (*tl)[i:])
		(*tl)[i] = tr
		return
	}
	*tl = append(*tl, tr)
}

// TagRegionsDeOverlap removes any overlapping regions in tag regions
func TagRegionsDeOverlap(tl *[]TagRegion) {
	sz := len(*tl)
	if sz <= 1 {
		return
	}
	for i := sz - 1; i > 0; i-- {
		ct := (*tl)[i]
		pt := (*tl)[i-1]
		if ct.OverlapsReg(pt) {
			*tl = append((*tl)[:i], (*tl)[i+1:]...)
		}
	}
}

// TagRegionsSort sorts the tags by starting pos
func TagRegionsSort(tags []TagRegion) {
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].St < tags[j].St
	})
}

// HiMarkup manages the syntax highlighting state for TextBuf
type HiMarkup struct {
	Lang      string            `desc:"language for syntax highlighting the code"`
	Style     histyle.StyleName `desc:"syntax highlighting style"`
	Has       bool              `desc:"true if both lang and style are set"`
	TabSize   int               `desc:"tab size, in chars"`
	CSSProps  ki.Props          `json:"-" xml:"-" desc:"Commpiled CSS properties for given highlighting style"`
	lastLang  string
	lastStyle histyle.StyleName
	lexer     chroma.Lexer
	formatter *html.Formatter
	style     histyle.Style
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
	hm.style = histyle.AvailStyle(hm.Style)

	hm.CSSProps = hm.style.ToProps()

	hm.lastLang = hm.Lang
	hm.lastStyle = hm.Style
}

// TagsForLine adds the tags for one line
func (hm *HiMarkup) TagsForLine(tags *[]TagRegion, toks []chroma.Token) {
	cp := 0
	sz := len(toks)
	for i, tok := range toks {
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
			ht := histyle.HiTagFromChroma(tok.Type)
			nt := TagRegion{Tag: ht, St: cp, Ed: ep}
			if ht == histyle.GenericHeading || ht == histyle.GenericSubheading {
				// extend heading for full line
				if i == 0 && sz == 2 {
					st2 := strings.TrimSuffix(toks[1].Value, "\n")
					nt.Ed = cp + slen + len(st2)
				} else if i == 1 && sz == 3 {
					st2 := strings.TrimSuffix(toks[2].Value, "\n")
					nt.Ed = cp + slen + len(st2)
					// } else {
					// 	fmt.Printf("generic heading: sz: %v i: %v\n", sz, i)
				}
			}
			*tags = append(*tags, nt)
		}
		cp = ep
	}
}

// MarkupTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsAll(txt []byte) ([][]TagRegion, error) {
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
		hm.TagsForLine(&tags[li], lt)
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
	toks := iterator.Tokens()
	hm.TagsForLine(&tags, toks)
	return tags, nil
}

// MarkupLine returns the line with html class tags added for each tag
// takes both the hi tags and extra tags.  Only fully nested tags are supported --
// any dangling ends are truncated.
func (hm *HiMarkup) MarkupLine(txt []byte, hitags, tags []TagRegion) []byte {
	ttags := TagRegionsMerge(hitags, tags)
	nt := len(ttags)
	if nt == 0 {
		return txt
	}
	sz := len(txt)
	if sz == 0 {
		return txt
	}
	sps := []byte(`<span class="`)
	sps2 := []byte(`">`)
	spe := []byte(`</span>`)
	taglen := len(sps) + len(sps2) + len(spe) + 2

	musz := sz + len(ttags)*taglen
	mu := make([]byte, 0, musz)

	cp := 0
	var tstack []int // stack of tags indexes that remain to be completed, sorted soonest at end
	for i, tr := range ttags {
		if cp >= sz {
			break
		}
		for si := len(tstack) - 1; si >= 0; si-- {
			ts := ttags[tstack[si]]
			if ts.Ed <= tr.St {
				ep := ints.MinInt(sz, ts.Ed)
				if cp < ep {
					mu = append(mu, []byte(htmlstd.EscapeString(string(txt[cp:ep])))...)
					cp = ep
				}
				mu = append(mu, spe...)
				tstack = append(tstack[:si], tstack[si+1:]...)
			}
		}
		if cp >= sz || tr.St >= sz {
			break
		}
		if tr.St > cp {
			mu = append(mu, []byte(htmlstd.EscapeString(string(txt[cp:tr.St])))...)
		}
		mu = append(mu, sps...)
		clsnm := tr.Tag.StyleName()
		mu = append(mu, []byte(clsnm)...)
		mu = append(mu, sps2...)
		ep := tr.Ed
		addEnd := true
		if i < nt-1 {
			if ttags[i+1].St < tr.Ed { // next one starts before we end, add to stack
				addEnd = false
				ep = ttags[i+1].St
				if len(tstack) == 0 {
					tstack = append(tstack, i)
				} else {
					for si := len(tstack) - 1; si >= 0; si-- {
						ts := ttags[tstack[si]]
						if tr.Ed <= ts.Ed {
							ni := si + 1 // new index in stack -- right after current
							tstack = append(tstack, i)
							copy(tstack[ni+1:], tstack[ni:])
							tstack[ni] = i
						}
					}
				}
			}
		} else {
			ep = ints.MinInt(sz, ep)
		}
		mu = append(mu, []byte(htmlstd.EscapeString(string(txt[tr.St:ep])))...)
		if addEnd {
			mu = append(mu, spe...)
		}
		cp = ep
	}
	if sz > cp {
		mu = append(mu, []byte(htmlstd.EscapeString(string(txt[cp:sz])))...)
	}
	// pop any left on stack..
	for si := len(tstack) - 1; si >= 0; si-- {
		mu = append(mu, spe...)
	}
	return mu
}
