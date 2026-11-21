// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strings"
	"sync"

	douceur "github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// ImportURL returns the URL and the media query list of the given
// prelude of an @import rule (url("x") media, or "x" media).
func ImportURL(prelude string) (url, media string) {
	p := strings.TrimSpace(prelude)
	if rest, ok := strings.CutPrefix(p, "url("); ok {
		before, after, ok := strings.Cut(rest, ")")
		if !ok {
			return "", ""
		}
		url, media = before, after
	} else if len(p) > 0 && (p[0] == '"' || p[0] == '\'') {
		end := strings.IndexByte(p[1:], p[0])
		if end < 0 {
			return "", ""
		}
		url, media = p[:end+2], p[end+2:]
	} else {
		return "", ""
	}
	url = strings.Trim(strings.TrimSpace(url), `"'`)
	media = strings.TrimSpace(media)
	// layer() and supports() conditions are not supported, and are ignored
	if strings.HasPrefix(media, "layer") || strings.HasPrefix(media, "supports(") {
		media = ""
	}
	return url, media
}

// URLPattern matches the url() of CSS values, with its URL quoted or not.
var URLPattern = regexp.MustCompile(`url\(\s*(?:"([^"]*)"|'([^']*)'|([^)'"\s]*))\s*\)`)

// ContainerCond is the condition of an @container rule: the name of the
// container that it queries ("" for the nearest one), and its query (as
// (width <= 799px)), and that of an @container rule that it is in.
type ContainerCond struct {
	Name, Query string
	Outer       *ContainerCond
}

// ParseContainerCond returns the condition of an @container rule of the
// given prelude (as "spotlight (width <= 799px)"), in the given one.
func ParseContainerCond(prelude string, outer *ContainerCond) *ContainerCond {
	prelude = strings.TrimSpace(prelude)
	cc := &ContainerCond{Query: prelude, Outer: outer}
	if i := strings.IndexAny(prelude, "( "); i > 0 && !strings.HasPrefix(strings.ToLower(prelude), "not") {
		name := strings.TrimSpace(prelude[:i])
		if name != "" && !strings.ContainsAny(name, "()") {
			cc.Name = name
			cc.Query = strings.TrimSpace(prelude[i:])
		}
	}
	return cc
}

// Parse parses the given style sheet. If it does not parse as a whole,
// as with constructs that the parser does not support, each of its top-level
// rules is parsed on its own, so that only the rules that do not parse are
// left out.
func Parse(style string) *douceur.Stylesheet {
	// The style sheets of a page are parsed again each time that it is read
	// (as for each change that its scripts make), so they are cached (as
	// they are not changed after they are parsed).
	parsedCSSMu.Lock()
	ss, ok := parsedCSS[style]
	parsedCSSMu.Unlock()
	if ok {
		return ss
	}
	ss = parseCSSText(style)
	if len(style) >= minCachedCSS {
		parsedCSSMu.Lock()
		if parsedCSSSize+len(style) > maxParsedCSS {
			parsedCSS, parsedCSSSize = map[string]*douceur.Stylesheet{}, 0
		}
		parsedCSS[style] = ss
		parsedCSSSize += len(style)
		parsedCSSMu.Unlock()
	}
	return ss
}

// parsedCSS has the style sheets that have been parsed, by their text,
// up to maxParsedCSS bytes of text (after which it is cleared). Only style
// sheets of at least minCachedCSS bytes are cached (as small ones, as of
// style attributes, are parsed quickly).
var (
	parsedCSSMu   sync.Mutex
	parsedCSS     = map[string]*douceur.Stylesheet{}
	parsedCSSSize int
)

const (
	maxParsedCSS = 16 << 20
	minCachedCSS = 1024
)

// parseCSSText parses the given style sheet (see [parseCSS]).
func parseCSSText(style string) *douceur.Stylesheet {
	style = RemoveStray(style)
	// (the condition of an @container rule is kept, after containerMark)
	style = containerRule.ReplaceAllString(style, "@supports "+ContainerMark+"$1{")
	style = unknownGroupingRule.ReplaceAllString(style, "@supports$1{")
	if ss, err := parser.Parse(style); err == nil {
		return ss
	}
	ss := douceur.NewStylesheet()
	for _, block := range topLevelBlocks(style) {
		bs, err := parser.Parse(block)
		if err != nil {
			continue
		}
		ss.Rules = append(ss.Rules, bs.Rules...)
	}
	return ss
}

// unknownGroupingRule matches the start of the block of a grouping
// at-rule that the parser does not know has rules in it (and parses as
// declarations, failing at the end of the first rule in it, as for the
// @container rules in the @layer of the style sheet of github.com), which
// are parsed as @supports rules, whose rules apply (see addRuleList).
var unknownGroupingRule = regexp.MustCompile(`@(?:layer|container|scope|starting-style)\b([^;{}]*)\{`)

// containerRule matches the start of the block of an @container rule, which
// is parsed as an @supports rule with its condition after containerMark.
var containerRule = regexp.MustCompile(`@container\b([^;{}]*)\{`)

// ContainerMark marks the prelude of an @supports rule that is an
// @container rule (see [containerRule]).
const ContainerMark = "--container:"

// groupingRules are the at-rules whose blocks have rules (not declarations).
var groupingRules = []string{"@media", "@supports", "@layer", "@container", "@document", "@-moz-document", "@scope", "@starting-style"}

// RemoveStray returns the given style sheet without what cannot be a
// rule where rules are expected (at the top level and in the blocks of
// grouping at-rules such as @media): statements that end with a semicolon
// but are not at-rules (as "a;b{...}"), and blocks without a selector
// (as "{a:b}", which CSS-in-JS libraries can make). The parser does not
// end on these (it does not skip a semicolon in a selector).
func RemoveStray(style string) string {
	if !strings.ContainsAny(style, ";{") {
		return style
	}
	type level struct {
		rules           bool // whether rules are expected (not declarations)
		stmtIn, stmtOut int  // the start of the current statement in style and out
	}
	out := make([]byte, 0, len(style))
	levels := []level{{rules: true}}
	var quote byte
	for i := 0; i < len(style); i++ {
		ch := style[i]
		lv := &levels[len(levels)-1]
		switch {
		case quote != 0:
			if ch == '\\' && i+1 < len(style) {
				out = append(out, ch)
				i++
				ch = style[i]
			} else if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '/' && i+1 < len(style) && style[i+1] == '*':
			end := strings.Index(style[i+2:], "*/")
			if end < 0 {
				end = len(style) - i - 2
			} else {
				end += 2
			}
			out = append(out, style[i:i+2+end]...)
			i += 1 + end
			continue
		case ch == '{':
			prelude := strings.TrimSpace(withoutComments(style[lv.stmtIn:i]))
			if lv.rules && prelude == "" { // skip the block
				i = blockEnd(style, i)
				out = out[:lv.stmtOut]
				lv.stmtIn, lv.stmtOut = i+1, len(out)
				continue
			}
			rules := false
			for _, g := range groupingRules {
				if strings.HasPrefix(strings.ToLower(prelude), g) {
					rules = true
				}
			}
			out = append(out, ch)
			levels = append(levels, level{rules: rules, stmtIn: i + 1, stmtOut: len(out)})
			continue
		case ch == '}':
			if len(levels) > 1 {
				levels = levels[:len(levels)-1]
			}
			out = append(out, ch)
			lv = &levels[len(levels)-1]
			lv.stmtIn, lv.stmtOut = i+1, len(out)
			continue
		case ch == ';':
			if lv.rules && !strings.HasPrefix(strings.TrimSpace(withoutComments(style[lv.stmtIn:i])), "@") {
				out = out[:lv.stmtOut] // a stray statement
			} else {
				out = append(out, ch)
			}
			lv.stmtIn, lv.stmtOut = i+1, len(out)
			continue
		}
		out = append(out, ch)
	}
	return string(out)
}

// withoutComments returns the given CSS without its comments.
func withoutComments(s string) string {
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			return s
		}
		end := strings.Index(s[i+2:], "*/")
		if end < 0 {
			return s[:i]
		}
		s = s[:i] + " " + s[i+2+end+2:]
	}
}

// blockEnd returns the index of the brace that closes the block that
// starts at the given index of the given CSS (or the last index).
func blockEnd(style string, start int) int {
	depth := 0
	var quote byte
	for i := start; i < len(style); i++ {
		ch := style[i]
		switch {
		case quote != 0:
			switch ch {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '/' && i+1 < len(style) && style[i+1] == '*':
			end := strings.Index(style[i+2:], "*/")
			if end < 0 {
				return len(style) - 1
			}
			i += end + 3
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(style) - 1
}

// topLevelBlocks splits the given style sheet into its top-level rules,
// at the closing braces (or semicolons of at-rules without blocks) that
// are not nested, in strings, or in comments.
func topLevelBlocks(style string) []string {
	var blocks []string
	depth := 0
	start := 0
	var quote byte
	for i := 0; i < len(style); i++ {
		ch := style[i]
		switch {
		case quote != 0:
			switch ch {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '/' && i+1 < len(style) && style[i+1] == '*':
			end := strings.Index(style[i+2:], "*/")
			if end < 0 {
				i = len(style)
			} else {
				i += end + 3
			}
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth <= 0 {
				blocks = append(blocks, style[start:i+1])
				start = i + 1
				depth = 0
			}
		case ch == ';' && depth == 0:
			blocks = append(blocks, style[start:i+1])
			start = i + 1
		}
	}
	return blocks
}
