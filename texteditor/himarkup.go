// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	stdhtml "html"
	"log/slog"
	"strings"

	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi"
	"cogentcore.org/core/pi/lex"
	_ "cogentcore.org/core/pi/suplangs"
	"cogentcore.org/core/pi/token"
	"cogentcore.org/core/texteditor/histyle"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
)

// HiMarkup manages the syntax highlighting state for Buf.
// It uses Pi if available, otherwise falls back on chroma
type HiMarkup struct {

	// full info about the file including category etc
	Info *fi.FileInfo

	// syntax highlighting style
	Style gi.HiStyleName

	// chroma-based language name for syntax highlighting the code
	Lang string

	// true if both lang and style are set
	Has bool

	// tab size, in chars
	TabSize int

	// Commpiled CSS properties for given highlighting style
	CSSProps *ki.Props `json:"-" xml:"-"`

	// pi parser state info
	PiState *pi.FileStates

	// if supported, this is the pi Lang support for parsing
	PiLang pi.Lang

	// current highlighting style
	HiStyle *histyle.Style

	// external toggle to turn off automatic highlighting
	Off       bool
	lastLang  string
	lastStyle gi.HiStyleName
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
	return hm.PiLang != nil
}

// Init initializes the syntax highlighting for current params
func (hm *HiMarkup) Init(info *fi.FileInfo, pist *pi.FileStates) {
	hm.Info = info
	hm.PiState = pist

	if hm.Info.Known != fi.Unknown {
		if lp, err := pi.LangSupport.Props(hm.Info.Known); err == nil {
			if lp.Lang != nil {
				hm.lexer = nil
				hm.PiLang = lp.Lang
			} else {
				hm.PiLang = nil
			}
		}
	}

	if hm.PiLang == nil {
		lexer := lexers.MatchMimeType(hm.Info.Mime)
		if lexer == nil {
			lexer = lexers.Match(hm.Info.Name)
		}
		if lexer != nil {
			hm.Lang = lexer.Config().Name
			hm.lexer = lexer
		}
	}

	if hm.Style == "" || (hm.PiLang == nil && hm.lexer == nil) {
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
		hm.formatter = html.New(html.WithClasses(true), html.TabWidth(hm.TabSize))
		hm.lastLang = hm.Lang
	}
}

// SetHiStyle sets the highlighting style and updates corresponding settings
func (hm *HiMarkup) SetHiStyle(style gi.HiStyleName) {
	if style == "" {
		return
	}
	st := histyle.AvailStyle(hm.Style)
	if st == nil {
		slog.Error("Highlighting Style not found:", "style", style)
		return
	}
	hm.Style = style
	hm.HiStyle = st
	hm.CSSProps = hm.HiStyle.ToProps()
	hm.lastStyle = hm.Style
}

// MarkupTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsAll(txt []byte) ([]lex.Line, error) {
	if hm.Off {
		return nil, nil
	}
	if hm.PiLang != nil {
		hm.PiLang.ParseFile(hm.PiState, txt)   // processes in Proc(), does Switch()
		return hm.PiState.Done().Src.Lexs, nil // Done() is previous Proc() -- still has type info coming in later, but lexs are good
	} else if hm.lexer != nil {
		return hm.ChromaTagsAll(txt)
	}
	return nil, nil
}

// MarkupTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupTagsLine(ln int, txt []rune) (lex.Line, error) {
	if hm.Off {
		return nil, nil
	}
	if hm.PiLang != nil {
		ll := hm.PiLang.HiLine(hm.PiState, ln, txt)
		return ll, nil
	} else if hm.lexer != nil {
		return hm.ChromaTagsLine(txt)
	}
	return nil, nil
}

// ChromaTagsForLine generates the chroma tags for one line of chroma tokens
func ChromaTagsForLine(tags *lex.Line, toks []chroma.Token) {
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
			ht := histyle.TokenFromChroma(tok.Type)
			tags.AddLex(token.KeyToken{Tok: ht}, cp, ep)
		}
		cp = ep
	}
}

// ChromaTagsAll returns all the markup tags according to current
// syntax highlighting settings
func (hm *HiMarkup) ChromaTagsAll(txt []byte) ([]lex.Line, error) {
	txtstr := string(txt) // expensive!
	iterator, err := hm.lexer.Tokenise(nil, txtstr)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	lines := chroma.SplitTokensIntoLines(iterator.Tokens())
	sz := len(lines)
	tags := make([]lex.Line, sz)
	for li, lt := range lines {
		ChromaTagsForLine(&tags[li], lt)
	}
	return tags, nil
}

// ChromaTagsLine returns tags for one line according to current
// syntax highlighting settings
func (hm *HiMarkup) ChromaTagsLine(txt []rune) (lex.Line, error) {
	return ChromaTagsLine(hm.lexer, txt)
}

// ChromaTagsLine returns tags for one line according to current
// syntax highlighting settings
func ChromaTagsLine(lexer chroma.Lexer, txt []rune) (lex.Line, error) {
	txtstr := string(txt) + "\n"
	iterator, err := lexer.Tokenise(nil, txtstr)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	var tags lex.Line
	toks := iterator.Tokens()
	ChromaTagsForLine(&tags, toks)
	return tags, nil
}

// MaxLineLen prevents overflow in allocating line length
const (
	MaxLineLen = 64 * 1024 * 1024
	MaxNTags   = 1024
)

// MarkupLine returns the line with html class tags added for each tag
// takes both the hi tags and extra tags.  Only fully nested tags are supported --
// any dangling ends are truncated.
func (hm *HiMarkup) MarkupLine(txt []rune, hitags, tags lex.Line) []byte {
	if len(txt) > MaxLineLen { // avoid overflow
		return nil
	}
	sz := len(txt)
	if sz == 0 {
		return nil
	}
	ttags := lex.MergeLines(hitags, tags) // ensures that inner-tags are *after* outer tags
	nt := len(ttags)
	if nt == 0 || nt > MaxNTags {
		return HTMLEscapeRunes(txt)
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
					mu = append(mu, HTMLEscapeRunes(txt[cp:ep])...)
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
			mu = append(mu, HTMLEscapeRunes(txt[cp:tr.St])...)
		}
		mu = append(mu, sps...)
		clsnm := tr.Tok.Tok.StyleName()
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
			mu = append(mu, HTMLEscapeRunes(txt[tr.St:ep])...)
		}
		if addEnd {
			mu = append(mu, spe...)
		}
		cp = ep
	}
	if sz > cp {
		mu = append(mu, HTMLEscapeRunes(txt[cp:sz])...)
	}
	// pop any left on stack..
	for si := len(tstack) - 1; si >= 0; si-- {
		mu = append(mu, spe...)
	}
	return mu
}

///////////////////////////////////////////////////////////////////////////
// HTMLEscapeBytes

// var htmlEscaper = bytereplacer.New(
// 	`&`, "&amp;",
// 	`'`, "&#39;", // "&#39;" is shorter than "&apos;" and apos was not in HTML until HTML5.
// 	`<`, "&lt;",
// 	`>`, "&gt;",
// 	`"`, "&#34;", // "&#34;" is shorter than "&quot;".
// )
//

// HTMLEscapeBytes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HTMLEscapeBytes(b []byte) []byte {
	// note: because we absolutely need to make copies, it is not clear
	// that this is any faster, and requiring the extra dependency likely
	// not worth it..
	// bc := make([]byte, len(b), len(b)+10)
	// copy(bc, b)
	// return htmlEscaper.Replace(bc)
	return []byte(stdhtml.EscapeString(string(b)))
}

// HTMLEscapeRunes escapes special characters like "<" to become "&lt;". It
// escapes only five such characters: <, >, &, ' and ".
// It operates on a *copy* of the byte string and does not modify the input!
// otherwise it causes major problems..
func HTMLEscapeRunes(r []rune) []byte {
	return []byte(stdhtml.EscapeString(string(r)))
}
