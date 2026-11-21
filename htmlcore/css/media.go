// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strings"
)

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
// @supports condition are supported in Chrome (see [SupportsMatches]).
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
