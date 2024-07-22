// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	stdhtml "html"
	"log/slog"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/lexer"
	_ "cogentcore.org/core/parse/supportedlanguages"
	"cogentcore.org/core/parse/token"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
)

// Highlighter performs syntax highlighting,
// using [parse] if available, otherwise falls back on chroma.
type Highlighter struct {

	// syntax highlighting style to use
	StyleName core.HighlightingName

	// chroma-based language name for syntax highlighting the code
	language string

	// Has is whether there are highlighting parameters set
	// (only valid after [Highlighter.init] has been called).
	Has bool

	// tab size, in chars
	TabSize int

	// Commpiled CSS properties for given highlighting style
	CSSProperties map[string]any

	// parser state info
	parseState *parse.FileStates

	// if supported, this is the [parse.Language] support for parsing
	parseLanguage parse.Language

	// current highlighting style
	style *Style

	// external toggle to turn off automatic highlighting
	off          bool
	lastLanguage string
	lastStyle    core.HighlightingName
	lexer        chroma.Lexer
	formatter    *html.Formatter
}

// UsingParse returns true if markup is using parse lexer / parser, which affects
// use of results
func (hi *Highlighter) UsingParse() bool {
	return hi.parseLanguage != nil
}

// Init initializes the syntax highlighting for current params
func (hi *Highlighter) Init(info *fileinfo.FileInfo, pist *parse.FileStates) {
	hi.parseState = pist

	if info.Known != fileinfo.Unknown {
		if lp, err := parse.LanguageSupport.Properties(info.Known); err == nil {
			if lp.Lang != nil {
				hi.lexer = nil
				hi.parseLanguage = lp.Lang
			} else {
				hi.parseLanguage = nil
			}
		}
	}

	if hi.parseLanguage == nil {
		lexer := lexers.MatchMimeType(info.Mime)
		if lexer == nil {
			lexer = lexers.Match(info.Name)
		}
		if lexer != nil {
			hi.language = lexer.Config().Name
			hi.lexer = lexer
		}
	}

	if hi.StyleName == "" || (hi.parseLanguage == nil && hi.lexer == nil) {
		hi.Has = false
		return
	}
	hi.Has = true

	if hi.StyleName != hi.lastStyle {
		hi.style = AvailableStyle(hi.StyleName)
		hi.CSSProperties = hi.style.ToProperties()
		hi.lastStyle = hi.StyleName
	}

	if hi.lexer != nil && hi.language != hi.lastLanguage {
		hi.lexer = chroma.Coalesce(lexers.Get(hi.language))
		hi.formatter = html.New(html.WithClasses(true), html.TabWidth(hi.TabSize))
		hi.lastLanguage = hi.language
	}
}

// SetStyle sets the highlighting style and updates corresponding settings
func (hi *Highlighter) SetStyle(style core.HighlightingName) {
	if style == "" {
		return
	}
	st := AvailableStyle(hi.StyleName)
	if st == nil {
		slog.Error("Highlighter Style not found:", "style", style)
		return
	}
	hi.StyleName = style
	hi.style = st
	hi.CSSProperties = hi.style.ToProperties()
	hi.lastStyle = hi.StyleName
}

// MarkupTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hi *Highlighter) MarkupTagsAll(txt []byte) ([]lexer.Line, error) {
	if hi.off {
		return nil, nil
	}
	if hi.parseLanguage != nil {
		hi.parseLanguage.ParseFile(hi.parseState, txt) // processes in Proc(), does Switch()
		return hi.parseState.Done().Src.Lexs, nil      // Done() is previous Proc() -- still has type info coming in later, but lexs are good
	} else if hi.lexer != nil {
		return hi.chromaTagsAll(txt)
	}
	return nil, nil
}

// MarkupTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hi *Highlighter) MarkupTagsLine(ln int, txt []rune) (lexer.Line, error) {
	if hi.off {
		return nil, nil
	}
	if hi.parseLanguage != nil {
		ll := hi.parseLanguage.HighlightLine(hi.parseState, ln, txt)
		return ll, nil
	} else if hi.lexer != nil {
		return hi.chromaTagsLine(txt)
	}
	return nil, nil
}

// chromaTagsForLine generates the chroma tags for one line of chroma tokens
func chromaTagsForLine(tags *lexer.Line, toks []chroma.Token) {
	cp := 0
	for _, tok := range toks {
		str := []rune(strings.TrimSuffix(tok.Value, "\n"))
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
			ht := TokenFromChroma(tok.Type)
			tags.AddLex(token.KeyToken{Token: ht}, cp, ep)
		}
		cp = ep
	}
}

// chromaTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hi *Highlighter) chromaTagsAll(txt []byte) ([]lexer.Line, error) {
	txtstr := string(txt) // expensive!
	iterator, err := hi.lexer.Tokenise(nil, txtstr)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	lines := chroma.SplitTokensIntoLines(iterator.Tokens())
	sz := len(lines)
	tags := make([]lexer.Line, sz)
	for li, lt := range lines {
		chromaTagsForLine(&tags[li], lt)
	}
	return tags, nil
}

// chromaTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hi *Highlighter) chromaTagsLine(txt []rune) (lexer.Line, error) {
	return chromaTagsLine(hi.lexer, txt)
}

// chromaTagsLine returns tags for one line according to current
// syntax highlighting settings
func chromaTagsLine(clex chroma.Lexer, txt []rune) (lexer.Line, error) {
	txtstr := string(txt) + "\n"
	iterator, err := clex.Tokenise(nil, txtstr)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	var tags lexer.Line
	toks := iterator.Tokens()
	chromaTagsForLine(&tags, toks)
	return tags, nil
}

// maxLineLen prevents overflow in allocating line length
const (
	maxLineLen = 64 * 1024 * 1024
	maxNumTags = 1024
)

// MarkupLine returns the line with html class tags added for each tag
// takes both the hi tags and extra tags.  Only fully nested tags are supported,
// with any dangling ends truncated. only operates on given inputs, does not
// require any locking in terms of internal state.
func (hi *Highlighter) MarkupLine(txt []rune, hitags, tags lexer.Line) []byte {
	if len(txt) > maxLineLen { // avoid overflow
		return nil
	}
	sz := len(txt)
	if sz == 0 {
		return nil
	}
	ttags := lexer.MergeLines(hitags, tags) // ensures that inner-tags are *after* outer tags
	nt := len(ttags)
	if nt == 0 || nt > maxNumTags {
		return HtmlEscapeRunes(txt)
	}
	sps := []byte(`<span class="`)
	sps2 := []byte(`">`)
	spe := []byte(`</span>`)
	taglen := len(sps) + len(sps2) + len(spe) + 2

	musz := sz + nt*taglen
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
				ep := min(sz, ts.Ed)
				if cp < ep {
					mu = append(mu, HtmlEscapeRunes(txt[cp:ep])...)
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
			mu = append(mu, HtmlEscapeRunes(txt[cp:tr.St])...)
		}
		mu = append(mu, sps...)
		clsnm := tr.Token.Token.StyleName()
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
							ni := si // + 1 // new index in stack -- right *before* current
							tstack = append(tstack, i)
							copy(tstack[ni+1:], tstack[ni:])
							tstack[ni] = i
						}
					}
				}
			}
		}
		ep = min(len(txt), ep)
		if tr.St < ep {
			mu = append(mu, HtmlEscapeRunes(txt[tr.St:ep])...)
		}
		if addEnd {
			mu = append(mu, spe...)
		}
		cp = ep
	}
	if sz > cp {
		mu = append(mu, HtmlEscapeRunes(txt[cp:sz])...)
	}
	// pop any left on stack..
	for si := len(tstack) - 1; si >= 0; si-- {
		mu = append(mu, spe...)
	}
	return mu
}

// HtmlEscapeBytes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HtmlEscapeBytes(b []byte) []byte {
	return []byte(stdhtml.EscapeString(string(b)))
}

// HtmlEscapeRunes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HtmlEscapeRunes(r []rune) []byte {
	return []byte(stdhtml.EscapeString(string(r)))
}
