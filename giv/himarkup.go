// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	htmlstd "html"
	"log"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/goki/gi/filecat"
	"github.com/goki/gi/histyle"
	"github.com/goki/ki"
	"github.com/goki/ki/ints"
	"github.com/goki/pi"
	"github.com/goki/pi/lex"
)

// Overall Strategy:
// * separate syntax highlighting markup from deeper semantic analysis (full parsing)
// 		+ use chroma as a fallback for highlighting
// 		+ and the client-server-based system for fallback semantics
// * issue is that these are linked in Pi so how to also keep them separate?
//		+ just assume that hi pass has happened already I guess?

// HiMarkup manages the syntax highlighting state for TextBuf
// it uses Pi if available, otherwise falls back on chroma
type HiMarkup struct {
	Info      *FileInfo         `desc:"full info about the file including category etc"`
	Style     histyle.StyleName `desc:"syntax highlighting style"`
	Lang      string            `desc:"chroma-based language name for syntax highlighting the code"`
	Has       bool              `desc:"true if both lang and style are set"`
	TabSize   int               `desc:"tab size, in chars"`
	CSSProps  ki.Props          `json:"-" xml:"-" desc:"Commpiled CSS properties for given highlighting style"`
	PiState   *pi.FileState     `desc:"pi parser state info"`
	PiParser  *pi.Parser        `desc:"if supported, this is the pi parser"`
	HiStyle   histyle.Style     `desc:"current highlighting style"`
	lastLang  string
	lastStyle histyle.StyleName
	lexer     chroma.Lexer
	formatter *html.Formatter
}

// HasHi returns true if there are highlighting parameters set (only valid after Init)
func (hm *HiMarkup) HasHi() bool {
	return hm.Has
}

// UsingPi returns true if markup is using GoPi lexer / parser -- affects
// use of results
func (hm *HiMarkup) UsingPi() bool {
	return hm.PiParser != nil
}

// Init initializes the syntax highlighting for current params
func (hm *HiMarkup) Init(info *FileInfo, pist *pi.FileState) {
	hm.Info = info
	hm.PiState = pist

	if hm.Info.Sup != filecat.NoSupport {
		if lp, ok := pi.StdLangProps[hm.Info.Sup]; ok {
			if lp.Parser != nil {
				hm.lexer = nil
				if hm.PiParser != lp.Parser {
					hm.PiParser = lp.Parser
					hm.PiParser.InitAll(hm.PiState)
				}
			} else {
				hm.PiParser = nil
			}
		}
	}

	if hm.PiParser == nil {
		lexer := lexers.Match(hm.Info.Name)
		// if lexer == nil && len(pist.Src.Lines) > 0 {
		// 	lexer = lexers.Analyse(string(tb.Txt))
		// }
		if lexer != nil {
			hm.Lang = lexer.Config().Name
			hm.lexer = lexer
		}
	}

	if hm.Style == "" || (hm.PiParser == nil && hm.lexer == nil) {
		hm.Has = false
		return
	}
	hm.Has = true

	if hm.Style != hm.lastStyle {
		hm.HiStyle = histyle.AvailStyle(hm.Style)
		hm.CSSProps = hm.HiStyle.ToProps()
		hm.lastStyle = hm.Style
	}

	if hm.lexer != nil && hm.Lang != hm.lastLang {
		hm.lexer = chroma.Coalesce(lexers.Get(hm.Lang))
		hm.formatter = html.New(html.WithClasses(), html.TabWidth(hm.TabSize))
		hm.lastLang = hm.Lang
	}
}

// SetParser sets given parser directly
func (hm *HiMarkup) SetParser(pr *pi.Parser) {
	hm.lexer = nil
	hm.PiParser = pr
	hm.PiParser.InitAll(hm.PiState)
}

// MarkupTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsAll(txt []byte) ([]lex.Line, error) {
	if hm.PiParser != nil {
		hm.PiParser.InitAll(hm.PiState)
		hm.PiParser.LexAll(hm.PiState)
		return hm.PiState.Src.Lexs, nil
	} else if hm.lexer != nil {
		return hm.ChromaTagsAll(txt)
	}
	return nil, nil
}

// MarkupTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsLine(ln int, txt []byte) (lex.Line, error) {
	if hm.PiParser != nil {
		ll := hm.PiParser.LexLine(hm.PiState, ln)
		return ll, nil
	} else if hm.lexer != nil {
		return hm.ChromaTagsLine(txt)
	}
	return nil, nil
}

// ChromaTagsForLine generates the chroma tags for one line of chroma tokens
func (hm *HiMarkup) ChromaTagsForLine(tags *lex.Line, toks []chroma.Token) {
	cp := 0
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
			ht := histyle.TokenFromChroma(tok.Type)
			tags.AddLex(ht, cp, ep)
		}
		cp = ep
	}
}

// ChromaTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) ChromaTagsAll(txt []byte) ([]lex.Line, error) {
	txtstr := string(txt)
	iterator, err := hm.lexer.Tokenise(nil, txtstr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	lines := chroma.SplitTokensIntoLines(iterator.Tokens())
	sz := len(lines)
	tags := make([]lex.Line, sz)
	for li, lt := range lines {
		hm.ChromaTagsForLine(&tags[li], lt)
	}
	return tags, nil
}

// ChromaTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hm *HiMarkup) ChromaTagsLine(txt []byte) (lex.Line, error) {
	txtstr := string(txt) + "\n"
	iterator, err := hm.lexer.Tokenise(nil, txtstr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var tags lex.Line
	toks := iterator.Tokens()
	hm.ChromaTagsForLine(&tags, toks)
	return tags, nil
}

// MarkupLine returns the line with html class tags added for each tag
// takes both the hi tags and extra tags.  Only fully nested tags are supported --
// any dangling ends are truncated.
func (hm *HiMarkup) MarkupLine(txt []byte, hitags, tags lex.Line) []byte {
	ttags := lex.MergeLines(hitags, tags)
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
		clsnm := tr.Tok.StyleName()
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
		}
		ep = ints.MinInt(sz, ep)
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
