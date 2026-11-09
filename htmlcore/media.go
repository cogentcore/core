// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/system"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// parseCSS parses the given style sheet. If it does not parse as a whole,
// as with constructs that the parser does not support, each of its top-level
// rules is parsed on its own, so that only the rules that do not parse are
// left out.
func parseCSS(style string) *css.Stylesheet {
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
			parsedCSS, parsedCSSSize = map[string]*css.Stylesheet{}, 0
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
	parsedCSS     = map[string]*css.Stylesheet{}
	parsedCSSSize int
)

const (
	maxParsedCSS = 16 << 20
	minCachedCSS = 1024
)

// parseCSSText parses the given style sheet (see [parseCSS]).
func parseCSSText(style string) *css.Stylesheet {
	style = removeStrayCSS(style)
	// (the condition of an @container rule is kept, after containerMark)
	style = containerRule.ReplaceAllString(style, "@supports "+containerMark+"$1{")
	style = unknownGroupingRule.ReplaceAllString(style, "@supports$1{")
	if ss, err := parser.Parse(style); err == nil {
		return ss
	}
	ss := css.NewStylesheet()
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

// containerMark marks the prelude of an @supports rule that is an
// @container rule (see [containerRule]).
const containerMark = "--container:"

// groupingRules are the at-rules whose blocks have rules (not declarations).
var groupingRules = []string{"@media", "@supports", "@layer", "@container", "@document", "@-moz-document", "@scope", "@starting-style"}

// removeStrayCSS returns the given style sheet without what cannot be a
// rule where rules are expected (at the top level and in the blocks of
// grouping at-rules such as @media): statements that end with a semicolon
// but are not at-rules (as "a;b{...}"), and blocks without a selector
// (as "{a:b}", which CSS-in-JS libraries can make). The parser does not
// end on these (it does not skip a semicolon in a selector).
func removeStrayCSS(style string) string {
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
			if ch == '\\' {
				i++
			} else if ch == quote {
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
			if ch == '\\' {
				i++
			} else if ch == quote {
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

// MediaWidth is the width of the viewport in CSS px (dp) that @media rules
// are evaluated for, when it is set for the context; otherwise it is the
// width of the screen.
func (c *Context) mediaWidth() float32 {
	if c.MediaWidth > 0 {
		return c.MediaWidth
	}
	return ScreenMediaWidth()
}

// ScreenMediaWidth returns the width of the screen in CSS px (dp), for the
// @media rules of style sheets when [Context.MediaWidth] is not set, or
// 1024 if there is no screen. An app that reads HTML in another goroutine
// should set MediaWidth (as to this) in its main goroutine first, as the
// screens can change in it (as when a window is made).
func ScreenMediaWidth() float32 {
	if system.TheApp != nil && system.TheApp.NScreens() > 0 {
		sc := system.TheApp.Screen(0)
		if sc.LogicalDPI > 0 && sc.PixelSize.X > 0 {
			return float32(sc.PixelSize.X) * 160 / sc.LogicalDPI
		}
	}
	return 1024
}

// MediaMatches returns whether the given media query list (as of an
// @media rule) matches the viewport of the context, as for matchMedia
// in the scripts of a page.
func (c *Context) MediaMatches(query string) bool {
	return c.mediaMatches(query)
}

// mediaMatches returns whether the given media query list matches,
// recording its result for [Context.MediaChanged].
func (c *Context) mediaMatches(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	m := c.mediaListMatches(query)
	if !c.checkingMedia {
		if c.mediaUses == nil {
			c.mediaUses = map[string]bool{}
		}
		c.mediaUses[query] = m
	}
	return m
}

// MediaChanged returns whether the HTML read with the context would be
// read differently for a viewport of the given width in CSS px (dp):
// whether a media query of its style sheets (as of an @media rule) or a
// number of columns estimated from the width (for column-width) has
// another result for it. An app that shows pages in a viewport whose
// width can change, as when its window is resized, can then read the
// page again with a new context (see [Context.NewPage]) with that
// [Context.MediaWidth], as browsers apply the rules of the new width.
// It must be called in the goroutine that styles the widgets of the page,
// once the page has been read.
func (c *Context) MediaChanged(width float32) bool {
	if width <= 0 {
		return false
	}
	saved := c.MediaWidth
	c.MediaWidth, c.checkingMedia = width, true
	defer func() { c.MediaWidth, c.checkingMedia = saved, false }()
	for q, m := range c.mediaUses {
		if c.mediaListMatches(q) != m {
			return true
		}
	}
	for n, count := range c.columnUses {
		if c.columnCount(n) != count {
			return true
		}
	}
	return false
}

// mediaListMatches returns whether the given (non-empty, lowercase)
// media query list matches.
func (c *Context) mediaListMatches(query string) bool {
	for _, q := range strings.Split(query, ",") {
		if c.mediaQueryMatches(strings.TrimSpace(q)) {
			return true
		}
	}
	return false
}

// mediaQueryMatches returns whether the given media query matches.
func (c *Context) mediaQueryMatches(q string) bool {
	not := false
	if rest, ok := strings.CutPrefix(q, "not "); ok {
		not, q = true, rest
	}
	q = strings.TrimPrefix(q, "only ")
	match := true
	for _, part := range strings.Split(q, " and ") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.HasPrefix(part, "(") { // media type
			switch part {
			case "all", "screen":
			default:
				match = false
			}
			continue
		}
		if !c.mediaFeatureMatches(strings.Trim(part, "()")) {
			match = false
		}
	}
	return match != not
}

// mediaFeatureMatches returns whether the given media feature matches,
// treating features that are not supported as matching.
func (c *Context) mediaFeatureMatches(f string) bool {
	name, value, _ := strings.Cut(f, ":")
	name, value = strings.TrimSpace(name), strings.TrimSpace(value)
	length := func() (float32, bool) {
		return calcPx(value)
	}
	width := c.mediaWidth()
	height := width * 3 / 4
	if c.MediaHeight > 0 {
		height = c.MediaHeight
	}
	if !strings.Contains(f, ":") && strings.ContainsAny(f, "<>=") {
		return mediaRangeMatches(f, width, height)
	}
	// the device is taken to be the size of the viewport, as for the
	// screen of a phone (and the window of headless Chrome)
	name = strings.Replace(name, "device-", "", 1)
	switch name {
	case "min-width", "max-width", "min-height", "max-height":
		l, ok := length()
		if !ok {
			return false // as browsers do for a query that is not valid
		}
		v := width
		if strings.HasSuffix(name, "height") {
			v = height
		}
		if strings.HasPrefix(name, "min") {
			return v >= l
		}
		return v <= l
	case "prefers-color-scheme":
		return value == "light"
	case "prefers-reduced-motion":
		// CSS animations and transitions are not run, so pages show the
		// final states of what they animate (as the headings of github.com
		// that fade in, which were hidden)
		return value == "reduce" || value == ""
	case "prefers-contrast", "forced-colors":
		return value == "no-preference" || value == "none"
	case "orientation":
		return value == "landscape"
	case "hover", "any-hover":
		return value == "hover"
	case "pointer", "any-pointer":
		return value == "fine"
	}
	return true
}

// calcPx returns the value in px of the given length, which can be a
// calc() expression of lengths in px, em, and rem (of 16px), and numbers,
// with +, -, *, / and parentheses, as in media queries.
func calcPx(v string) (float32, bool) {
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

// mediaRangeOp matches the comparison operators of the range syntax
// of media features, as in (width >= 50rem) or (30rem < width <= 60rem).
var mediaRangeOp = regexp.MustCompile(`<=|>=|<|>|=`)

// mediaRangeMatches returns whether the given media feature in the range
// syntax matches the given viewport width and height.
func mediaRangeMatches(f string, width, height float32) bool {
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
		v, ok := calcPx(strings.TrimSpace(s))
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
