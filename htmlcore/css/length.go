// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// LengthEnv has the sizes that relative lengths are resolved with, in px:
// the em (font size of the element), the rem (font size of the html
// element), and 1% of the viewport width and height. fixedVh is whether
// Vh is from htmlcore.Context.MediaHeight, to which all vh lengths are then
// relative (as dp); otherwise vh stays a unit of core, except in calc().
// Each style pass of a widget has its own (see htmlcore.Context.browserStyler),
// and reading a page uses that of the page (see htmlcore.Context.readEnv).
type LengthEnv struct {
	Em, Vw, Vh float64
	FixedVh    bool
	Rem        float64

	// VhUsed, if set, is set when a length uses vh (or vmin and vmax) of
	// the viewport (see htmlcore.Context.ViewportHeightUsed).
	VhUsed *atomic.Bool

	// Cqw and Cqh are the container query units (1% of the width and the
	// height of the container of the element; see container.go), which
	// are those of the viewport (vw, vh) if they are not set.
	Cqw, Cqh float64

	// Ch, if set, returns the size of the ch unit of the element in px:
	// the advance of the 0 glyph of its font, as in CSS.
	Ch func() float64
}

// DefaultLengthEnv returns the length environment of a 16px font
// and an 800x600 viewport, for lengths that are absolute.
func DefaultLengthEnv() *LengthEnv {
	return &LengthEnv{Em: 16, Vw: 8, Vh: 6, Rem: 16}
}

// Length parses the given CSS length, with px as dp (which is
// what px is in browsers), % as the given unit, and lengths without
// units, as in the attributes of HTML elements, as dp.
func (e *LengthEnv) Length(v string, pct units.Units) (units.Value, bool) {
	v = strings.TrimSpace(v)
	switch v {
	case "", "auto", "none", "normal", "inherit", "initial":
		return units.Value{}, false
	}
	if strings.HasSuffix(v, ")") {
		return e.math(v, pct)
	}
	if r, ok := strings.CutSuffix(v, "ch"); ok && e.Ch != nil {
		if f, err := strconv.ParseFloat(r, 32); err == nil {
			if ch := e.Ch(); ch > 0 {
				return units.Dp(float32(f * ch)), true
			}
		}
	}
	if l, ok := e.containerLength(v); ok {
		return l, true
	}
	if r, ok := strings.CutSuffix(v, "rem"); ok { // of the font size of the html element
		if f, err := strconv.ParseFloat(r, 32); err == nil {
			return units.Dp(float32(f) * float32(e.Rem)), true
		}
	}
	if before, ok := strings.CutSuffix(v, "%"); ok {
		f, err := strconv.ParseFloat(before, 32)
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
	switch l.Unit {
	case units.UnitVh, units.UnitVmin, units.UnitVmax:
		e.useVh()
	}
	if l.Unit == units.UnitVh && e.FixedVh {
		return units.Dp(l.Value * float32(e.Vh)), true
	}
	return l, true
}

// useVh records that a length uses the height of the viewport.
func (e *LengthEnv) useVh() {
	if e.VhUsed != nil {
		e.VhUsed.Store(true)
	}
}

// containerUnits returns the size in px of 1% of the width and height of
// the container of the element (cqw, cqh), or of the viewport if none.
func (e *LengthEnv) containerUnits() (cqw, cqh float64) {
	cqw, cqh = e.Cqw, e.Cqh
	if cqw <= 0 {
		cqw = e.Vw
	}
	if cqh <= 0 {
		cqh = e.Vh
	}
	return
}

// cqUnits are the container query units of CSS, by how they are of the
// width (cqw, cqi) and height (cqh, cqb) of the container, or the smaller
// (cqmin) or larger (cqmax) of them.
var cqUnits = []string{"cqmin", "cqmax", "cqw", "cqh", "cqi", "cqb"}

// cqUnit returns the size in px of the given container query unit.
func (e *LengthEnv) cqUnit(u string) float64 {
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
func (e *LengthEnv) containerLength(v string) (units.Value, bool) {
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

// LengthPtr returns the given length, or nil if it is not one (as auto).
func (e *LengthEnv) LengthPtr(v string, pct units.Units) *units.Value {
	l, ok := e.Length(v, pct)
	if !ok {
		return nil
	}
	return &l
}

// BorderWidth returns the given CSS border width.
func (e *LengthEnv) BorderWidth(v string) units.Value {
	switch v {
	case "thin":
		return units.Dp(1)
	case "medium":
		return units.Dp(3)
	case "thick":
		return units.Dp(5)
	}
	l, _ := e.Length(v, units.UnitDp)
	return l
}

// Linear is a length that is a percentage plus an absolute length in px,
// the value of a calc() expression of CSS.
type Linear struct {
	Pct, Px float64
}

// linear evaluates the given calc() expression (or length) as a
// linear combination of a percentage and px. It returns false if it
// is not valid (as with a product of two lengths).
func (e *LengthEnv) linear(v string) (Linear, bool) {
	v = strings.TrimSpace(strings.ToLower(v))
	if inner, ok := strings.CutPrefix(v, "calc("); ok {
		v = "(" + inner
	}
	p := &linearParser{s: v, em: e.Em, vw: e.Vw, vh: e.Vh, rem: e.Rem, env: e}
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
	env        *LengthEnv // for the container query units

	// lengths is the number of lengths parsed.
	lengths int
}

func (p *linearParser) space() {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
}

func (p *linearParser) sum() (Linear, bool) {
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
			return Linear{}, false
		}
		l.Pct += sign * r.Pct
		l.Px += sign * r.Px
	}
	return l, ok
}

// product parses a product, in which at most one factor is a length.
func (p *linearParser) product() (Linear, bool) {
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
			return Linear{}, false
		}
		var n float64 // the number factor
		if rLen {
			n = l.Px // l is a number, as px
			l = r
			isLen = true
		} else {
			n = r.Px
		}
		if op == '/' {
			if n == 0 {
				return Linear{}, false
			}
			n = 1 / n
		}
		l.Pct *= n
		l.Px *= n
	}
	return l, ok
}

// factor parses a number or length, or an expression in parentheses,
// returning whether it is a length (as opposed to a number, in px).
func (p *linearParser) factor() (Linear, bool, bool) {
	p.space()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		before := p.lengths
		l, ok := p.sum()
		p.space()
		if !ok || p.i >= len(p.s) || p.s[p.i] != ')' {
			return Linear{}, false, false
		}
		p.i++
		return l, p.lengths > before, true
	}
	if strings.HasPrefix(p.s[p.i:], "calc(") {
		p.i += 4
		return p.factor()
	}
	if p.env != nil {
		for _, fn := range []string{"min(", "max(", "clamp("} {
			if !strings.HasPrefix(p.s[p.i:], fn) {
				continue
			}
			// a math function in a calc() (as the gutters of github.com, in
			// calc(100% - 2 * clamp(1.25rem, calc((100vw - 768px) / 4), 8rem))),
			// of lengths that are not percentages
			depth, end := 0, -1
			for j := p.i + len(fn) - 1; j < len(p.s); j++ {
				if p.s[j] == '(' {
					depth++
				} else if p.s[j] == ')' {
					depth--
					if depth == 0 {
						end = j
						break
					}
				}
			}
			if end < 0 {
				return Linear{}, false, false
			}
			v, ok := p.env.math(p.s[p.i:end+1], units.UnitPw)
			if !ok || v.Custom != nil || v.Unit != units.UnitDp {
				return Linear{}, false, false
			}
			p.i = end + 1
			p.lengths++
			return Linear{Px: float64(v.Value)}, true, true
		}
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.' || (p.i == start && (p.s[p.i] == '-' || p.s[p.i] == '+'))) {
		p.i++
	}
	n, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return Linear{}, false, false
	}
	rest := p.s[p.i:]
	unit := func(u string, scale float64) (Linear, bool, bool) {
		p.i += len(u)
		p.lengths++
		return Linear{Px: n * scale}, true, true
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
		return Linear{Pct: n}, true, true
	case strings.HasPrefix(rest, "px"):
		return unit("px", 1)
	case strings.HasPrefix(rest, "rem"):
		return unit("rem", p.rem)
	case strings.HasPrefix(rest, "em"):
		return unit("em", p.em)
	case strings.HasPrefix(rest, "vw"):
		return unit("vw", p.vw)
	case strings.HasPrefix(rest, "vh"):
		p.env.useVh()
		return unit("vh", p.vh)
	case strings.HasPrefix(rest, "vmin"):
		p.env.useVh()
		return unit("vmin", math.Min(p.vw, p.vh))
	case strings.HasPrefix(rest, "vmax"):
		p.env.useVh()
		return unit("vmax", math.Max(p.vw, p.vh))
	case strings.HasPrefix(rest, "pt"):
		return unit("pt", 96.0/72)
	}
	return Linear{Px: n}, false, true // a number
}

// math returns the length of the given CSS math function (calc(),
// min(), max(), clamp()), as a percentage of the given unit, or in dp.
// A length with both a percentage and an absolute part is approximated
// by its percentage, as units have one unit.
func (e *LengthEnv) math(v string, pct units.Units) (units.Value, bool) {
	name, args, ok := strings.Cut(v, "(")
	if !ok || !strings.HasSuffix(args, ")") {
		return units.Value{}, false
	}
	args = strings.TrimSuffix(args, ")")
	eval := func(x string) (Linear, bool) {
		x = strings.TrimSpace(x)
		if l, ok := e.math(x, pct); ok && strings.Contains(x, "(") && !strings.HasPrefix(x, "(") {
			if l.Unit == pct {
				return Linear{Pct: float64(l.Value)}, true
			}
			return Linear{Px: float64(l.Value)}, true
		}
		return e.linear(x)
	}
	var l Linear
	switch name {
	case "calc":
		l, ok = eval(args)
	case "min", "max", "clamp":
		parts := SplitSelectors(args) // at the top-level commas
		var vals []Linear
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
		pick := func(a, b Linear, less bool) Linear {
			if a.Pct != 0 || b.Pct != 0 {
				if a.Pct == 0 {
					return a
				}
				return b
			}
			if (a.Px < b.Px) == less {
				return a
			}
			return b
		}
		if mixed := slices.ContainsFunc(vals, func(l Linear) bool { return l.Pct != 0 }) &&
			slices.ContainsFunc(vals, func(l Linear) bool { return l.Px != 0 || l.Pct == 0 }); mixed {
			// compared when the size of the parent is known, in the layout
			return CustomLength(name, vals, pct), true
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
	if l.Pct != 0 && l.Px != 0 { // as the width of calc(100% - 256px)
		return CustomLength("calc", []Linear{l}, pct), true
	}
	if l.Pct != 0 {
		return units.Value{Value: float32(l.Pct), Unit: pct}, true
	}
	return units.Dp(float32(l.Px)), true
}

// CustomLength returns a custom length (see [units.Value.Custom]) of the
// given CSS math function (calc, min, max, or clamp) of the given linear
// lengths, with percentages of the given unit, which core computes with the
// size of the parent in the layout: a percentage and a length cannot be
// one unit (as the width of calc(100% - 256px) of the grids of github.com,
// which was taken to be 100%).
func CustomLength(fn string, vals []Linear, pct units.Units) units.Value {
	vals = slices.Clone(vals)
	return units.Custom(func(uc *units.Context) float32 {
		base := uc.Paw
		switch pct {
		case units.UnitPh:
			base = uc.Pah
		case units.UnitEw:
			base = uc.Elw
		case units.UnitEh:
			base = uc.Elh
		}
		dp := uc.Dots(units.UnitDp)
		dots := func(l Linear) float32 { return float32(l.Pct/100)*base + float32(l.Px)*dp }
		v := dots(vals[0])
		switch fn {
		case "min":
			for _, l := range vals[1:] {
				v = min(v, dots(l))
			}
		case "max":
			for _, l := range vals[1:] {
				v = max(v, dots(l))
			}
		case "clamp":
			if len(vals) == 3 {
				v = max(dots(vals[0]), min(dots(vals[1]), dots(vals[2])))
			}
		}
		return v
	})
}

// LengthPx returns the given CSS length in px, for lengths that are
// absolute or relative to the font size (taken to be 16px for em).
func (e *LengthEnv) LengthPx(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	if f, ok := strings.CutSuffix(v, "rem"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * float32(e.Rem), true
		}
	}
	if f, ok := strings.CutSuffix(v, "em"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * 16, true
		}
	}
	l, ok := e.Length(v, units.UnitPw)
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

// ChOf returns the size of the ch unit of CSS in px for the font of the
// given style (whose font size is the given em, in px), measured with the
// given text shaper: the advance of its 0 glyph, as in browsers. It is 0
// if it cannot be measured.
func ChOf(sh shaped.Shaper, s *styles.Style, em float64) float64 {
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
