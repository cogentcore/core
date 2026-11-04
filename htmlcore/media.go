// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"math"
	"strconv"
	"strings"

	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// parseCSS parses the given style sheet. If it does not parse as a whole,
// as with constructs that the parser does not support, each of its top-level
// rules is parsed on its own, so that only the rules that do not parse are
// left out.
func parseCSS(style string) *css.Stylesheet {
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
	if system.TheApp != nil && system.TheApp.NScreens() > 0 {
		sc := system.TheApp.Screen(0)
		if sc.LogicalDPI > 0 && sc.PixelSize.X > 0 {
			return float32(sc.PixelSize.X) * 160 / sc.LogicalDPI
		}
	}
	return 1024
}

// mediaMatches returns whether the given media query list matches.
func (c *Context) mediaMatches(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
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
	case "prefers-reduced-motion", "prefers-contrast", "forced-colors":
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

// linear is a length that is a percentage plus an absolute length in px,
// the value of a calc() expression of CSS.
type linear struct {
	pct, px float64
}

// calcLinear evaluates the given calc() expression (or length) as a
// linear combination of a percentage and px, with px per em and px per
// percent of the viewport width and height (vw, vh) as given. It returns
// false if it is not valid (as with a product of two lengths).
func calcLinear(v string, em, vw, vh float64) (linear, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if inner, ok := strings.CutPrefix(v, "calc("); ok {
		v = "(" + inner
	}
	p := &linearParser{s: v, em: em, vw: vw, vh: vh}
	l, ok := p.sum()
	p.space()
	return l, ok && p.i == len(p.s)
}

// linearParser parses calc expressions into linear lengths.
type linearParser struct {
	s          string
	i          int
	em, vw, vh float64

	// lengths is the number of lengths parsed.
	lengths int
}

func (p *linearParser) space() {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
}

func (p *linearParser) sum() (linear, bool) {
	l, ok := p.product()
	for ok {
		p.space()
		if p.i >= len(p.s) || (p.s[p.i] != '+' && p.s[p.i] != '-') {
			break
		}
		sign := 1.0
		if p.s[p.i] == '-' {
			sign = -1
		}
		p.i++
		r, rok := p.product()
		if !rok {
			return linear{}, false
		}
		l.pct += sign * r.pct
		l.px += sign * r.px
	}
	return l, ok
}

// product parses a product, in which at most one factor is a length.
func (p *linearParser) product() (linear, bool) {
	l, isLen, ok := p.factor()
	for ok {
		p.space()
		if p.i >= len(p.s) || (p.s[p.i] != '*' && p.s[p.i] != '/') {
			break
		}
		op := p.s[p.i]
		p.i++
		r, rLen, rok := p.factor()
		if !rok || (isLen && rLen) || (op == '/' && rLen) {
			return linear{}, false
		}
		var n float64 // the number factor
		if rLen {
			n = l.px // l is a number, as px
			l = r
			isLen = true
		} else {
			n = r.px
		}
		if op == '/' {
			if n == 0 {
				return linear{}, false
			}
			n = 1 / n
		}
		l.pct *= n
		l.px *= n
	}
	return l, ok
}

// factor parses a number or length, or an expression in parentheses,
// returning whether it is a length (as opposed to a number, in px).
func (p *linearParser) factor() (linear, bool, bool) {
	p.space()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		before := p.lengths
		l, ok := p.sum()
		p.space()
		if !ok || p.i >= len(p.s) || p.s[p.i] != ')' {
			return linear{}, false, false
		}
		p.i++
		return l, p.lengths > before, true
	}
	if strings.HasPrefix(p.s[p.i:], "calc(") {
		p.i += 4
		return p.factor()
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.' || (p.i == start && (p.s[p.i] == '-' || p.s[p.i] == '+'))) {
		p.i++
	}
	n, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return linear{}, false, false
	}
	rest := p.s[p.i:]
	unit := func(u string, scale float64) (linear, bool, bool) {
		p.i += len(u)
		p.lengths++
		return linear{px: n * scale}, true, true
	}
	switch {
	case strings.HasPrefix(rest, "%"):
		p.i++
		p.lengths++
		return linear{pct: n}, true, true
	case strings.HasPrefix(rest, "px"):
		return unit("px", 1)
	case strings.HasPrefix(rest, "rem"):
		return unit("rem", 16)
	case strings.HasPrefix(rest, "em"):
		return unit("em", p.em)
	case strings.HasPrefix(rest, "vw"):
		return unit("vw", p.vw)
	case strings.HasPrefix(rest, "vh"):
		return unit("vh", p.vh)
	case strings.HasPrefix(rest, "vmin"):
		return unit("vmin", math.Min(p.vw, p.vh))
	case strings.HasPrefix(rest, "vmax"):
		return unit("vmax", math.Max(p.vw, p.vh))
	case strings.HasPrefix(rest, "pt"):
		return unit("pt", 96.0/72)
	}
	return linear{px: n}, false, true // a number
}

// calcEnv has the sizes that calc() lengths are relative to, in px, for
// the widget being styled: the em (font size) and 1% of the viewport
// width and height. It is set by [Context.browserStyler], as styling
// is done in one goroutine at a time.
var calcEnv = struct{ em, vw, vh float64 }{16, 8, 6}

// mathLength returns the length of the given CSS math function (calc(),
// min(), max(), clamp()), as a percentage of the given unit, or in dp.
// A length with both a percentage and an absolute part is approximated
// by its percentage, as units have one unit.
func mathLength(v string, pct units.Units) (units.Value, bool) {
	name, args, ok := strings.Cut(v, "(")
	if !ok || !strings.HasSuffix(args, ")") {
		return units.Value{}, false
	}
	args = strings.TrimSuffix(args, ")")
	eval := func(e string) (linear, bool) {
		e = strings.TrimSpace(e)
		if l, ok := mathLength(e, pct); ok && strings.Contains(e, "(") && !strings.HasPrefix(e, "(") {
			if l.Unit == pct {
				return linear{pct: float64(l.Value)}, true
			}
			return linear{px: float64(l.Value)}, true
		}
		return calcLinear(e, calcEnv.em, calcEnv.vw, calcEnv.vh)
	}
	var l linear
	switch name {
	case "calc":
		l, ok = eval(args)
	case "min", "max", "clamp":
		parts := splitSelectors(args) // at the top-level commas
		var vals []linear
		for _, p := range parts {
			pl, pok := eval(p)
			if !pok {
				return units.Value{}, false
			}
			vals = append(vals, pl)
		}
		if len(vals) == 0 || (name == "clamp" && len(vals) != 3) {
			return units.Value{}, false
		}
		// Percentages cannot be compared with absolute lengths here, so the
		// absolute ones (with no percentage) decide, if there are any.
		pick := func(a, b linear, less bool) linear {
			if a.pct != 0 || b.pct != 0 {
				if a.pct == 0 {
					return a
				}
				return b
			}
			if (a.px < b.px) == less {
				return a
			}
			return b
		}
		switch name {
		case "min":
			l = vals[0]
			for _, v := range vals[1:] {
				l = pick(l, v, true)
			}
		case "max":
			l = vals[0]
			for _, v := range vals[1:] {
				l = pick(l, v, false)
			}
		case "clamp":
			l = pick(pick(vals[1], vals[2], true), vals[0], false)
		}
		ok = true
	default:
		return units.Value{}, false
	}
	if !ok {
		return units.Value{}, false
	}
	if l.pct != 0 {
		return units.Value{Value: float32(l.pct), Unit: pct}, true
	}
	return units.Dp(float32(l.px)), true
}
