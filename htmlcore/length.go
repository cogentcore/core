// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// lengthEnv has the sizes that relative lengths are resolved with, in px:
// the em (font size of the element), the rem (font size of the html
// element), and 1% of the viewport width and height. fixedVh is whether
// vh is from [Context.MediaHeight], to which all vh lengths are then
// relative (as dp); otherwise vh stays a unit of core, except in calc().
// Each style pass of a widget has its own (see [Context.browserStyler]),
// and reading a page uses that of the page (see [Context.readEnv]).
type lengthEnv struct {
	em, vw, vh float64
	fixedVh    bool
	rem        float64

	// cqw and cqh are the container query units (1% of the width and the
	// height of the container of the element; see container.go), which
	// are those of the viewport (vw, vh) if they are not set.
	cqw, cqh float64

	// ch, if set, returns the size of the ch unit of the element in px:
	// the advance of the 0 glyph of its font, as in CSS.
	ch func() float64
}

// defaultLengthEnv returns the length environment of a 16px font
// and an 800x600 viewport, for lengths that are absolute.
func defaultLengthEnv() *lengthEnv {
	return &lengthEnv{em: 16, vw: 8, vh: 6, rem: 16}
}

// readEnv returns the length environment for parsing lengths when a page
// is read (outside of styling): an em of 16px (as the font size of the
// element is not known yet), the rem of the page, and its viewport.
func (c *Context) readEnv() *lengthEnv {
	e := defaultLengthEnv()
	e.rem = float64(c.rootFontSize())
	e.vw = float64(c.mediaWidth()) / 100
	e.fixedVh = c.MediaHeight > 0
	if e.fixedVh {
		e.vh = float64(c.MediaHeight) / 100
	}
	return e
}

// length parses the given CSS length, with px as dp (which is
// what px is in browsers), % as the given unit, and lengths without
// units, as in the attributes of HTML elements, as dp.
func (e *lengthEnv) length(v string, pct units.Units) (units.Value, bool) {
	v = strings.TrimSpace(v)
	switch v {
	case "", "auto", "none", "normal", "inherit", "initial":
		return units.Value{}, false
	}
	if strings.HasSuffix(v, ")") {
		return e.math(v, pct)
	}
	if r, ok := strings.CutSuffix(v, "ch"); ok && e.ch != nil {
		if f, err := strconv.ParseFloat(r, 32); err == nil {
			if ch := e.ch(); ch > 0 {
				return units.Dp(float32(f * ch)), true
			}
		}
	}
	if l, ok := e.containerLength(v); ok {
		return l, true
	}
	if r, ok := strings.CutSuffix(v, "rem"); ok { // of the font size of the html element
		if f, err := strconv.ParseFloat(r, 32); err == nil {
			return units.Dp(float32(f) * float32(e.rem)), true
		}
	}
	if strings.HasSuffix(v, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 32)
		if err != nil {
			return units.Value{}, false
		}
		return units.Value{Value: float32(f), Unit: pct}, true
	}
	if f, err := strconv.ParseFloat(v, 32); err == nil {
		return units.Dp(float32(f)), true
	}
	// absolute units are relative to px in CSS (96px per inch)
	for _, u := range cssAbsoluteUnits {
		if num, ok := strings.CutSuffix(v, u.suffix); ok {
			f, err := strconv.ParseFloat(num, 32)
			if err != nil {
				return units.Value{}, false
			}
			return units.Dp(float32(f) * u.px), true
		}
	}
	var l units.Value
	if err := l.SetString(v); err != nil {
		return units.Value{}, false
	}
	if l.Unit == units.UnitVh && e.fixedVh {
		return units.Dp(l.Value * float32(e.vh)), true
	}
	return l, true
}

// containerUnits returns the size in px of 1% of the width and height of
// the container of the element (cqw, cqh), or of the viewport if none.
func (e *lengthEnv) containerUnits() (cqw, cqh float64) {
	cqw, cqh = e.cqw, e.cqh
	if cqw <= 0 {
		cqw = e.vw
	}
	if cqh <= 0 {
		cqh = e.vh
	}
	return
}

// cqUnits are the container query units of CSS, by how they are of the
// width (cqw, cqi) and height (cqh, cqb) of the container, or the smaller
// (cqmin) or larger (cqmax) of them.
var cqUnits = []string{"cqmin", "cqmax", "cqw", "cqh", "cqi", "cqb"}

// cqUnit returns the size in px of the given container query unit.
func (e *lengthEnv) cqUnit(u string) float64 {
	cqw, cqh := e.containerUnits()
	switch u {
	case "cqw", "cqi":
		return cqw
	case "cqh", "cqb":
		return cqh
	case "cqmin":
		return math.Min(cqw, cqh)
	}
	return math.Max(cqw, cqh)
}

// containerLength returns the given length if it is in a container
// query unit (as 50cqw), in dp.
func (e *lengthEnv) containerLength(v string) (units.Value, bool) {
	for _, u := range cqUnits {
		if num, ok := strings.CutSuffix(v, u); ok {
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return units.Value{}, false
			}
			return units.Dp(float32(f * e.cqUnit(u))), true
		}
	}
	return units.Value{}, false
}

// cssAbsoluteUnits are the absolute units of CSS, in px.
var cssAbsoluteUnits = []struct {
	suffix string
	px     float32
}{{"px", 1}, {"pt", 96.0 / 72}, {"pc", 16}, {"in", 96}, {"cm", 96 / 2.54}, {"mm", 96 / 25.4}, {"q", 96 / 101.6}}

// lengthPtr returns the given length, or nil if it is not one (as auto).
func (e *lengthEnv) lengthPtr(v string, pct units.Units) *units.Value {
	l, ok := e.length(v, pct)
	if !ok {
		return nil
	}
	return &l
}

// borderWidth returns the given CSS border width.
func (e *lengthEnv) borderWidth(v string) units.Value {
	switch v {
	case "thin":
		return units.Dp(1)
	case "medium":
		return units.Dp(3)
	case "thick":
		return units.Dp(5)
	}
	l, _ := e.length(v, units.UnitDp)
	return l
}

// linear is a length that is a percentage plus an absolute length in px,
// the value of a calc() expression of CSS.
type linear struct {
	pct, px float64
}

// linear evaluates the given calc() expression (or length) as a
// linear combination of a percentage and px. It returns false if it
// is not valid (as with a product of two lengths).
func (e *lengthEnv) linear(v string) (linear, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if inner, ok := strings.CutPrefix(v, "calc("); ok {
		v = "(" + inner
	}
	p := &linearParser{s: v, em: e.em, vw: e.vw, vh: e.vh, rem: e.rem, env: e}
	l, ok := p.sum()
	p.space()
	return l, ok && p.i == len(p.s)
}

// linearParser parses calc expressions into linear lengths.
type linearParser struct {
	s          string
	i          int
	em, vw, vh float64
	rem        float64    // px per rem
	env        *lengthEnv // for the container query units

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
	if p.env != nil {
		for _, u := range cqUnits {
			if strings.HasPrefix(rest, u) {
				return unit(u, p.env.cqUnit(u))
			}
		}
	}
	switch {
	case strings.HasPrefix(rest, "%"):
		p.i++
		p.lengths++
		return linear{pct: n}, true, true
	case strings.HasPrefix(rest, "px"):
		return unit("px", 1)
	case strings.HasPrefix(rest, "rem"):
		return unit("rem", p.rem)
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

// math returns the length of the given CSS math function (calc(),
// min(), max(), clamp()), as a percentage of the given unit, or in dp.
// A length with both a percentage and an absolute part is approximated
// by its percentage, as units have one unit.
func (e *lengthEnv) math(v string, pct units.Units) (units.Value, bool) {
	name, args, ok := strings.Cut(v, "(")
	if !ok || !strings.HasSuffix(args, ")") {
		return units.Value{}, false
	}
	args = strings.TrimSuffix(args, ")")
	eval := func(x string) (linear, bool) {
		x = strings.TrimSpace(x)
		if l, ok := e.math(x, pct); ok && strings.Contains(x, "(") && !strings.HasPrefix(x, "(") {
			if l.Unit == pct {
				return linear{pct: float64(l.Value)}, true
			}
			return linear{px: float64(l.Value)}, true
		}
		return e.linear(x)
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

// lengthPx returns the given CSS length in px, for lengths that are
// absolute or relative to the font size (taken to be 16px for em).
func (e *lengthEnv) lengthPx(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	if f, ok := strings.CutSuffix(v, "rem"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * float32(e.rem), true
		}
	}
	if f, ok := strings.CutSuffix(v, "em"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * 16, true
		}
	}
	l, ok := e.length(v, units.UnitPw)
	if !ok || l.Unit != units.UnitDp {
		return 0, false
	}
	return l.Value, true
}

// chShaper is the text shaper that measures the ch unit when the scene
// of an element has none yet.
var (
	chShaper     shaped.Shaper
	chShaperOnce sync.Once
)

// chRatios are the advances of the 0 glyph of fonts relative to their
// size, by their family and weight, for the ch unit.
var chRatios struct {
	sync.Mutex
	ratios map[string]float64
}

// chOf returns the size of the ch unit of CSS in px for the font of the
// given style (whose font size is the given em, in px), measured with the
// given text shaper: the advance of its 0 glyph, as in browsers. It is 0
// if it cannot be measured.
func chOf(sh shaped.Shaper, s *styles.Style, em float64) float64 {
	if em <= 0 {
		return 0
	}
	key := fmt.Sprintf("%v %s %v", s.Font.Family, s.Font.CustomFont, s.Font.Weight)
	chRatios.Lock()
	r, ok := chRatios.ratios[key]
	chRatios.Unlock()
	if !ok {
		if sh == nil { // before the scene has one
			chShaperOnce.Do(func() { chShaper = shaped.NewShaper() })
			sh = chShaper
		}
		sty := rich.NewStyle()
		sty.Family, sty.Weight = s.Font.Family, s.Font.Weight
		tsty := text.NewStyle()
		tsty.CustomFont = s.Font.CustomFont
		tsty.FontSize.Dots = 100
		var adv float32
		for _, run := range sh.Shape(rich.NewText(sty, []rune("0")), tsty) {
			adv += run.Advance()
		}
		r = float64(adv) / 100
		chRatios.Lock()
		if chRatios.ratios == nil {
			chRatios.ratios = map[string]float64{}
		}
		chRatios.ratios[key] = r
		chRatios.Unlock()
	}
	return r * em
}
