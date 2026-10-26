// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/token"
	"github.com/alecthomas/chroma/v3"
	"github.com/alecthomas/chroma/v3/formatters/html"
)

// UseParse is whether syntax highlighting, completion, lookup and
// indentation use the [parse] lexer and parser for the languages that it
// supports, instead of chroma. It is off by default, which keeps the parse
// languages out of programs that only need highlighting, making them smaller
// and faster to start. Programs that want the parse languages, such as code
// editors, set it to true and import
// cogentcore.org/core/text/parse/supportedlanguages to register them.
var UseParse = false

// ParseFilter, if set, further restricts [UseParse] to the languages for
// which it returns true, so that a program can use [parse] for some
// languages and other means (such as a language server) for the rest.
// It can be called from any goroutine. Buffers pick up a change the next
// time their file info is set (lines.Lines.SetFileInfo).
var ParseFilter func(known fileinfo.Known) bool

// ParseLanguage returns the [parse] language properties for the given
// known file type if [UseParse] is on, [ParseFilter] allows it, and the
// language is registered, and nil otherwise.
func ParseLanguage(known fileinfo.Known) *parse.LanguageProperties {
	if !UseParse || known == fileinfo.Unknown {
		return nil
	}
	if ParseFilter != nil && !ParseFilter(known) {
		return nil
	}
	lp, err := parse.LanguageSupport.Properties(known)
	if err != nil || lp.Lang == nil {
		return nil
	}
	return lp
}

// Highlighter performs syntax highlighting, using [parse] if [UseParse]
// is on and the language is supported, and otherwise chroma.
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

	if lp := ParseLanguage(info.Known); lp != nil {
		hi.lexer = nil
		hi.parseLanguage = lp.Lang
	} else {
		hi.parseLanguage = nil
	}

	if hi.parseLanguage == nil {
		lexer := Lexers().MatchMimeType(info.Mime)
		if lexer == nil {
			lexer = Lexers().Match(info.Name)
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

	if hi.lexer != nil {
		// always coalesce the newly matched lexer, which can also follow
		// a switch from parse back to chroma for the same language
		hi.lexer = chroma.Coalesce(GetLexer(hi.language))
		if hi.language != hi.lastLanguage || hi.formatter == nil {
			hi.formatter = html.New(html.WithClasses(true), html.TabWidth(hi.TabSize))
			hi.lastLanguage = hi.language
		}
	}
}

// SetStyle sets the highlighting style and updates corresponding settings
func (hi *Highlighter) SetStyle(style HighlightingName) {
	if style == "" {
		return
	}
	st := AvailableStyle(style)
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
	toks, err := collectTokens(iterator)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	lines := chroma.SplitTokensIntoLines(toks)
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
	toks, err := collectTokens(iterator)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	chromaTagsForLine(&tags, toks)
	return tags, nil
}

// collectTokens collects the tokens of a chroma lexer's iterator. The
// iterator cannot return an error, so chroma panics when lexing fails,
// for example when a rule's regular expression exceeds its match timeout,
// and leaves it to the caller to recover; this returns that as an error.
func collectTokens(iterator iter.Seq[chroma.Token]) (toks []chroma.Token, err error) {
	defer func() {
		if r := recover(); r != nil {
			toks = nil
			if e, ok := r.(error); ok {
				err = fmt.Errorf("highlighting: lexing failed: %w", e)
			} else {
				err = fmt.Errorf("highlighting: lexing failed: %v", r)
			}
		}
	}()
	return slices.Collect(iterator), nil
}
