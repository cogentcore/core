// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"
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
