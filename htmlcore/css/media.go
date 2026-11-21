// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	douceur "github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// CalcPx returns the value in px of the given length, which can be a
// calc() expression of lengths in px, em, and rem (of 16px), and numbers,
// with +, -, *, / and parentheses, as in media queries.
func CalcPx(v string) (float32, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if inner, ok := strings.CutPrefix(v, "calc("); ok {
		v = "(" + inner
	}
	p := &calcParser{s: v}
	f, ok := p.sum()
	p.space()
	return float32(f), ok && p.i == len(p.s)
}

// calcParser parses calc expressions.
type calcParser struct {
	s string
	i int
}

func (p *calcParser) space() {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
}

func (p *calcParser) sum() (float64, bool) {
	f, ok := p.product()
	for ok {
		p.space()
		if p.i >= len(p.s) || (p.s[p.i] != '+' && p.s[p.i] != '-') {
			break
		}
		op := p.s[p.i]
		p.i++
		g, gok := p.product()
		if !gok {
			return 0, false
		}
		if op == '+' {
			f += g
		} else {
			f -= g
		}
	}
	return f, ok
}

func (p *calcParser) product() (float64, bool) {
	f, ok := p.factor()
	for ok {
		p.space()
		if p.i >= len(p.s) || (p.s[p.i] != '*' && p.s[p.i] != '/') {
			break
		}
		op := p.s[p.i]
		p.i++
		g, gok := p.factor()
		if !gok || (op == '/' && g == 0) {
			return 0, false
		}
		if op == '*' {
			f *= g
		} else {
			f /= g
		}
	}
	return f, ok
}

func (p *calcParser) factor() (float64, bool) {
	p.space()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		f, ok := p.sum()
		p.space()
		if !ok || p.i >= len(p.s) || p.s[p.i] != ')' {
			return 0, false
		}
		p.i++
		return f, true
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.' || (p.i == start && p.s[p.i] == '-')) {
		p.i++
	}
	n, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return 0, false
	}
	switch {
	case strings.HasPrefix(p.s[p.i:], "px"):
		p.i += 2
	case strings.HasPrefix(p.s[p.i:], "rem"):
		p.i += 3
		n *= 16
	case strings.HasPrefix(p.s[p.i:], "em"):
		p.i += 2
		n *= 16
	}
	return n, true
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

// mediaRangeOp matches the comparison operators of the range syntax
// of media features, as in (width >= 50rem) or (30rem < width <= 60rem).
var mediaRangeOp = regexp.MustCompile(`<=|>=|<|>|=`)

// MediaRangeMatches returns whether the given media feature in the range
// syntax matches the given viewport width and height.
func MediaRangeMatches(f string, width, height float32) bool {
	ops := mediaRangeOp.FindAllString(f, -1)
	parts := mediaRangeOp.Split(f, -1)
	if len(parts) != len(ops)+1 || len(ops) == 0 || len(ops) > 2 {
		return false
	}
	// value returns the value of an operand: the feature or a length.
	value := func(s string) (float32, bool, bool) { // value, ok, isFeature
		switch strings.TrimSpace(s) {
		case "width":
			return width, true, true
		case "height":
			return height, true, true
		case "aspect-ratio":
			return width / height, true, true
		}
		v, ok := CalcPx(strings.TrimSpace(s))
		return v, ok, false
	}
	feature := false
	for i, op := range ops {
		a, aok, af := value(parts[i])
		b, bok, bf := value(parts[i+1])
		if !aok || !bok {
			return false // as browsers do for a query that is not valid
		}
		feature = feature || af || bf
		var r bool
		switch op {
		case "<=":
			r = a <= b
		case ">=":
			r = a >= b
		case "<":
			r = a < b
		case ">":
			r = a > b
		case "=":
			r = a == b
		}
		if !r {
			return false
		}
	}
	return feature
}

// SupportsMatches returns whether the given condition of an @supports rule
// is true in Chrome, which this follows: its features are supported,
// except the properties and values of other browsers (-moz-, -ms-, -o-) and
// of iOS, so that the fallbacks in @supports not (...) do not apply (as
// the widths of the sections of theguardian.com for browsers without
// display: grid, which made them narrow). The preludes of the other
// grouping rules that are parsed as @supports (as @layer name) are true.
func SupportsMatches(cond string) bool {
	p := supportsParser{s: cond}
	return p.condition()
}

// supportsParser parses and evaluates an @supports condition.
type supportsParser struct {
	s string
	i int
}

// skipSpace skips the white space at the current position.
func (p *supportsParser) skipSpace() {
	for p.i < len(p.s) && strings.ContainsRune(" \t\n\r\f", rune(p.s[p.i])) {
		p.i++
	}
}

// keyword consumes the given keyword (as and) if it is next.
func (p *supportsParser) keyword(kw string) bool {
	p.skipSpace()
	rest := p.s[p.i:]
	if len(rest) < len(kw) || !strings.EqualFold(rest[:len(kw)], kw) {
		return false
	}
	if len(rest) > len(kw) && !strings.ContainsRune(" \t\n\r\f(", rune(rest[len(kw)])) {
		return false
	}
	p.i += len(kw)
	return true
}

// condition evaluates the terms joined by and or or from the current
// position.
func (p *supportsParser) condition() bool {
	v := p.term()
	for {
		switch {
		case p.keyword("and"):
			v = p.term() && v
		case p.keyword("or"):
			v = p.term() || v
		default:
			return v
		}
	}
}

// term evaluates a not, a condition or declaration in parentheses, a
// function (as selector(...)), or a name, which are true.
func (p *supportsParser) term() bool {
	if p.keyword("not") {
		return !p.term()
	}
	p.skipSpace()
	start := p.i
	for p.i < len(p.s) && p.s[p.i] != '(' && !strings.ContainsRune(" \t\n\r\f)", rune(p.s[p.i])) {
		p.i++ // the name of a function or of a layer
	}
	if p.i >= len(p.s) || p.s[p.i] != '(' {
		return true
	}
	// the text in the parentheses
	depth, open := 0, p.i
	for ; p.i < len(p.s); p.i++ {
		if p.s[p.i] == '(' {
			depth++
		} else if p.s[p.i] == ')' {
			depth--
			if depth == 0 {
				break
			}
		}
	}
	inner := strings.TrimSpace(p.s[open+1 : min(p.i, len(p.s))])
	p.i++
	if open > start {
		return true // a function, as selector(:has(a))
	}
	if prop, value, ok := CutTopLevel(inner, ':'); ok && !strings.ContainsAny(prop, "( ") {
		return supportedDeclaration(strings.ToLower(prop), strings.ToLower(value))
	}
	sub := supportsParser{s: inner}
	return sub.condition()
}

// supportedDeclaration returns whether the given property and value of an
// @supports condition are supported in Chrome (see [supportsMatches]).
func supportedDeclaration(prop, value string) bool {
	for _, other := range []string{"-moz-", "-ms-", "-o-"} {
		if strings.HasPrefix(prop, other) || strings.Contains(value, other) {
			return false
		}
	}
	switch prop {
	case "-webkit-touch-callout", "-webkit-overflow-scrolling":
		return false
	}
	return true
}
