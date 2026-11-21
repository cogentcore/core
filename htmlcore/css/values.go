// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"image"
	"math"
	"strconv"
	"strings"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
)

// SplitColors splits the given value into fields at spaces
// that are not inside of parentheses, as in rgb(...) and calc(...).
func SplitColors(v string) []string {
	var fs []string
	depth := 0
	start := -1
	for i, r := range v {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ' ', '\t', '\n':
			if depth == 0 {
				if start >= 0 {
					fs = append(fs, v[start:i])
				}
				start = -1
				continue
			}
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		fs = append(fs, v[start:])
	}
	return fs
}

// ZeroAreaClip returns whether the given value of clip (a rect) or of
// clip-path (an inset) clips the element to nothing, as in the visually
// hidden elements of pages (rect(0 0 0 0), rect(1px, 1px, 1px, 1px),
// inset(50%)).
func ZeroAreaClip(prop, v string) bool {
	fn := "rect("
	if prop == "clip-path" {
		if strings.HasPrefix(v, "rect(") { // as clip (as rect(0 0 0 0) of the .sr-only of GitHub)
			return ZeroAreaClip("clip", v)
		}
		fn = "inset("
	}
	args, ok := strings.CutPrefix(v, fn)
	if !ok || !strings.HasSuffix(args, ")") {
		return false
	}
	f := strings.Fields(strings.ReplaceAll(strings.TrimSuffix(args, ")"), ",", " "))
	if prop == "clip-path" { // insets of 50% or more from opposite sides
		if len(f) == 0 || len(f) > 4 || strings.Contains(args, "round") {
			return false
		}
		pct := func(x string) float64 {
			if x == "0" || x == "0px" {
				return 0
			}
			n, ok := strings.CutSuffix(x, "%")
			if !ok {
				return -1
			}
			p, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return -1
			}
			return p
		}
		// the sides of the shorthand: top, right, bottom, left
		var sides [4]float64
		for i := range sides {
			j := [4][4]int{{0, 0, 0, 0}, {0, 1, 0, 1}, {0, 1, 2, 1}, {0, 1, 2, 3}}[len(f)-1][i]
			sides[i] = pct(f[j])
		}
		top, right, bottom, left := sides[0], sides[1], sides[2], sides[3]
		return top >= 0 && bottom >= 0 && top+bottom >= 100 || right >= 0 && left >= 0 && right+left >= 100
	}
	if len(f) != 4 { // top, right, bottom, left, of the top left corner
		return false
	}
	env := DefaultLengthEnv()
	var px [4]float32
	for i, x := range f {
		if x == "auto" {
			return false
		}
		l, ok := env.LengthPx(x)
		if !ok {
			return false
		}
		px[i] = l
	}
	return px[2] <= px[0] || px[1] <= px[3]
}

// ColorFunc returns a function that returns the given CSS color (or nil if
// it is not one). A uniform color is parsed once, and is shared by the
// styles; other colors (as gradients) are parsed for each style, so that
// each has its own.
func ColorFunc(v string) func() image.Image {
	cl := ParseColor(v)
	if _, ok := cl.(*image.Uniform); ok || cl == nil {
		return func() image.Image { return cl }
	}
	return func() image.Image { return ParseColor(v) }
}

// ParseFlex returns the grow, shrink, and basis of the given value of the
// flex shorthand, as in CSS, where a number alone (as flex: 1) has a basis
// of 0, not of the width of the item.
func ParseFlex(v string) (grow, shrink float32, basis string, ok bool) {
	switch v {
	case "none":
		return 0, 0, "auto", true
	case "auto":
		return 1, 1, "auto", true
	case "initial":
		return 0, 1, "auto", true
	}
	fs := strings.Fields(v)
	if len(fs) == 0 {
		return 0, 0, "", false
	}
	shrink, basis = 1, "0"
	nums := 0
	for _, f := range fs {
		x, err := strconv.ParseFloat(f, 32)
		switch {
		case err == nil && nums == 0:
			grow = float32(x)
			nums++
		case err == nil && nums == 1:
			shrink = float32(x)
			nums++
		default:
			basis = f
			if nums == 0 { // flex: <basis> has a grow of 1
				grow = 1
			}
		}
	}
	return grow, shrink, basis, true
}

// ExpandSides expands the 1 to 4 values of a CSS shorthand for
// the sides of a box into the values for top, right, bottom, left.
func ExpandSides(vals []string) []string {
	switch len(vals) {
	case 0:
		return nil
	case 1:
		return []string{vals[0], vals[0], vals[0], vals[0]}
	case 2:
		return []string{vals[0], vals[1], vals[0], vals[1]}
	case 3:
		return []string{vals[0], vals[1], vals[2], vals[1]}
	}
	return vals[:4]
}

// SideIndex returns the index (top, right, bottom, left) of the side in
// the name of the given property, such as "margin-left".
func SideIndex(prop string) int {
	switch {
	case strings.Contains(prop, "-top"):
		return 0
	case strings.Contains(prop, "-right"):
		return 1
	case strings.Contains(prop, "-bottom"):
		return 2
	}
	return 3
}

// SetSide sets the given side (top, right, bottom, left) of the given sides.
func SetSide[T any](s *sides.Sides[T], i int, v T) {
	setSidePtr(&s.Top, &s.Right, &s.Bottom, &s.Left, i, v)
}

// SetValueSide sets the given side (top, right, bottom, left) of the given values.
func SetValueSide(s *sides.Values, i int, v units.Value) {
	setSidePtr(&s.Top, &s.Right, &s.Bottom, &s.Left, i, v)
}

func setSidePtr[T any](top, right, bottom, left *T, i int, v T) {
	s := struct{ Top, Right, Bottom, Left *T }{top, right, bottom, left}
	switch i {
	case 0:
		*s.Top = v
	case 1:
		*s.Right = v
	case 2:
		*s.Bottom = v
	case 3:
		*s.Left = v
	}
}

// SetBorderStyle sets the style of the given side of the border.
func SetBorderStyle(s *styles.Style, i int, v string) {
	if v == "none" || v == "hidden" {
		SetValueSide(&s.Border.Width, i, units.Dp(0))
		SetSide(&s.Border.Style, i, styles.BorderNone)
		return
	}
	var bs styles.BorderStyles
	if bs.SetString(v) == nil {
		SetSide(&s.Border.Style, i, bs)
	}
}

// AlignValue returns the alignment for the given CSS value.
func AlignValue(v string) styles.Aligns {
	switch v {
	case "flex-start", "start", "left", "self-start", "normal", "stretch":
		return styles.Start
	case "flex-end", "end", "right", "self-end":
		return styles.End
	case "center":
		return styles.Center
	case "baseline", "first baseline":
		return styles.Baseline
	case "space-between":
		return styles.SpaceBetween
	case "space-around":
		return styles.SpaceAround
	case "space-evenly":
		return styles.SpaceEvenly
	}
	return styles.Auto
}

// GridColumns returns the number of columns of a grid-template-columns value.
func GridColumns(v string) int {
	if after, ok := strings.CutPrefix(v, "repeat("); ok {
		n, _, _ := strings.Cut(after, ",")
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i
		}
	}
	return len(SplitColors(v))
}

// cssTranslate returns the translation of the given value of the transform
// property (the sum of its translate functions; the other functions, as
// rotate and scale, are not supported), or of the translate property, as
// the lengths of x and y. It returns false if it is not valid.
// AspectRatio returns the ratio of width to height of the given value
// of the aspect-ratio property (as 16 / 9, 1.5, or auto 4/3), or 0 for
// none (auto).
func AspectRatio(v string) float32 {
	v = strings.TrimSpace(strings.Replace(v, "auto", "", 1))
	a, b, slash := strings.Cut(v, "/")
	w, err := strconv.ParseFloat(strings.TrimSpace(a), 32)
	if err != nil || w <= 0 {
		return 0
	}
	h := 1.0
	if slash {
		if h, err = strconv.ParseFloat(strings.TrimSpace(b), 32); err != nil || h <= 0 {
			return 0
		}
	}
	return float32(w / h)
}

// ZeroScale returns whether the given transform scales the element
// to nothing in either dimension (as scaleX(0), for a progress bar
// that is not started), which is then not shown.
func ZeroScale(v string) bool {
	for rest := v; rest != ""; {
		name, args, found := strings.Cut(rest, "(")
		if !found {
			break
		}
		before, after, ok := strings.Cut(args, ")")
		if !ok {
			break
		}
		as := strings.Split(before, ",")
		zero := func(i int) bool {
			if i >= len(as) {
				return false
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(as[i]), 64)
			return err == nil && f == 0
		}
		switch strings.TrimSpace(name) {
		case "scale", "scale3d":
			if zero(0) || (len(as) > 1 && zero(1)) {
				return true
			}
		case "scalex", "scaley":
			if zero(0) {
				return true
			}
		}
		rest = strings.TrimSpace(after)
	}
	return false
}

func Translate(prop, v string) (x, y string, ok bool) {
	x, y = "0", "0"
	if v == "none" || v == "" {
		return x, y, true
	}
	if prop == "translate" { // x [y [z]]
		fs := strings.Fields(v)
		if len(fs) > 0 {
			x = fs[0]
		}
		if len(fs) > 1 {
			y = fs[1]
		}
		return x, y, true
	}
	var xs, ys []string
	for rest := v; rest != ""; {
		name, args, found := strings.Cut(rest, "(")
		if !found {
			break
		}
		before, after, ok := strings.Cut(args, ")")
		if !ok {
			return "", "", false
		}
		as := strings.Split(before, ",")
		for i := range as {
			as[i] = strings.TrimSpace(as[i])
		}
		switch strings.TrimSpace(name) {
		case "translate", "translate3d":
			xs = append(xs, as[0])
			if len(as) > 1 {
				ys = append(ys, as[1])
			}
		case "translatex":
			xs = append(xs, as[0])
		case "translatey":
			ys = append(ys, as[0])
		}
		rest = strings.TrimSpace(after)
	}
	sum := func(ls []string) string {
		switch len(ls) {
		case 0:
			return "0"
		case 1:
			return ls[0]
		}
		return "calc(" + strings.Join(ls, " + ") + ")"
	}
	return sum(xs), sum(ys), true
}

// Rotate returns the sum of the angles of the rotate functions of the
// given transform, in degrees.
func Rotate(_, v string) float32 {
	var deg float32
	for rest := v; rest != ""; {
		name, args, found := strings.Cut(rest, "(")
		if !found {
			break
		}
		before, after, ok := strings.Cut(args, ")")
		if !ok {
			break
		}
		if n := strings.TrimSpace(name); n == "rotate" || n == "rotatez" {
			deg += Angle(strings.TrimSpace(before))
		}
		rest = strings.TrimSpace(after)
	}
	return deg
}

// Angle returns the given CSS angle in degrees (deg, rad, grad, turn).
func Angle(v string) float32 {
	for _, u := range []struct {
		suffix string
		deg    float64
	}{{"deg", 1}, {"grad", 0.9}, {"rad", 180 / math.Pi}, {"turn", 360}} {
		if n, ok := strings.CutSuffix(v, u.suffix); ok {
			f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
			return float32(f * u.deg)
		}
	}
	f, _ := strconv.ParseFloat(v, 64)
	return float32(f)
}

// EmLength returns the given CSS length in em, with px for 16px to the
// em (and rem as em), or 0 if it is not one of these.
func EmLength(v string) float64 {
	v = strings.TrimSpace(strings.ToLower(v))
	for _, u := range []struct {
		suffix string
		scale  float64
	}{{"rem", 1}, {"em", 1}, {"px", 1.0 / 16}} {
		if num, ok := strings.CutSuffix(v, u.suffix); ok {
			f, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return f * u.scale
		}
	}
	return 0
}

// EmSpaces returns a string of space characters (which are not collapsed
// or trimmed) about as wide as the given number of em.
func EmSpaces(em float64) string {
	var sb strings.Builder
	for _, sp := range []struct {
		r rune
		w float64
	}{{'\u2003', 1}, {'\u2002', 0.5}, {'\u2004', 1.0 / 3}, {'\u2005', 0.25}, {'\u2006', 1.0 / 6}, {'\u200A', 0.1}} {
		for em >= sp.w-0.04 && sb.Len() < 64 {
			sb.WriteRune(sp.r)
			em -= sp.w
		}
	}
	return sb.String()
}

// PercentUnit returns whether the given unit is a percentage of a size of
// the parent or the element, which is not definite for a replaced element.
func PercentUnit(u units.Units) bool {
	return u == units.UnitPw || u == units.UnitPh || u == units.UnitEw || u == units.UnitEh
}

// UnitlessCalc returns the value of the given calc() expression of numbers
// only (no lengths or percentages), and whether it is one.
func UnitlessCalc(v string) (float32, bool) {
	inner, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(v)), "calc(")
	if !ok || strings.ContainsAny(inner, "abcdefghijklmnopqrstuvwxyz%") {
		return 0, false
	}
	return CalcPx(v)
}

// ZeroLength returns whether the given CSS length is zero (as 0, 0px, 0%).
func ZeroLength(v string) bool {
	v = strings.TrimSpace(v)
	v = strings.TrimRight(v, "abcdefghijklmnopqrstuvwxyz%")
	f, err := strconv.ParseFloat(v, 32)
	return err == nil && f == 0
}

// SideDots returns the given sides in dots, using the given function.
func SideDots(v *sides.Values, dots func(v *units.Value) float32) sides.Floats {
	return sides.NewFloats(dots(&v.Top), dots(&v.Right), dots(&v.Bottom), dots(&v.Left))
}
