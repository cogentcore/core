// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"log/slog"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	_ "cogentcore.org/core/text/parse/supportedlanguages"
	"cogentcore.org/core/text/token"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
)

// Highlighter performs syntax highlighting,
// using [parse] if available, otherwise falls back on chroma.
type Highlighter struct {

	// syntax highlighting style to use
	StyleName HighlightingName

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

	// Style is the current highlighting style.
	Style *Style

	// external toggle to turn off automatic highlighting
	off          bool
	lastLanguage string
	lastStyle    HighlightingName
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
	if hi.Style == nil {
		hi.SetStyle(DefaultStyle)
	}
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
		hi.Style = AvailableStyle(hi.StyleName)
		hi.CSSProperties = hi.Style.ToProperties()
		hi.lastStyle = hi.StyleName
	}

	if hi.lexer != nil && hi.language != hi.lastLanguage {
		hi.lexer = chroma.Coalesce(lexers.Get(hi.language))
		hi.formatter = html.New(html.WithClasses(true), html.TabWidth(hi.TabSize))
		hi.lastLanguage = hi.language
	}
}

// SetStyle sets the highlighting style and updates corresponding settings
func (hi *Highlighter) SetStyle(style HighlightingName) {
	if style == "" {
		return
	}
	st := AvailableStyle(hi.StyleName)
	if st == nil {
		slog.Error("Highlighter Style not found:", "style", style)
		return
	}
	hi.StyleName = style
	hi.Style = st
	hi.CSSProperties = hi.Style.ToProperties()
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
		lex := hi.parseState.Done().Src.Lexs
		return lex, nil // Done() is results of above
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
	return ChromaTagsLine(hi.lexer, string(txt))
}

// ChromaTagsLine returns tags for one line according to given chroma lexer
func ChromaTagsLine(clex chroma.Lexer, txt string) (lexer.Line, error) {
	n := len(txt)
	if n == 0 {
		return nil, nil
	}
	if txt[n-1] != '\n' {
		txt += "\n"
	}
	iterator, err := clex.Tokenise(nil, txt)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	var tags lexer.Line
	toks := iterator.Tokens()
	chromaTagsForLine(&tags, toks)
	return tags, nil
}
