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

// FontProperties are the properties of fonts, which are compiled first
// (with line-height, which the font shorthand sets).
var FontProperties = map[string]bool{
	"font": true, "font-family": true, "font-size": true, "font-weight": true,
	"font-style": true, "line-height": true,
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

// WebkitAliases are the properties whose -webkit- prefixed names are
// aliases of them in browsers, which pages may set only with the prefix.
var WebkitAliases = map[string]bool{
	"transform": true, "transform-origin": true, "box-sizing": true, "box-shadow": true,
	"border-radius": true, "flex": true, "flex-direction": true, "flex-wrap": true, "flex-flow": true,
	"flex-grow": true, "flex-shrink": true, "flex-basis": true, "order": true, "align-items": true,
	"align-self": true, "justify-content": true, "user-select": true, "appearance": true,
}

// KeywordLonghands are the longhands of the shorthands, which their
// CSS-wide keywords (inherit, initial) apply to (the others are applied
// to the style whole, see styles.Style.FromProperty).
var KeywordLonghands = map[string][]string{
	"font":   {"font-style", "font-weight", "font-size", "line-height", "font-family"},
	"margin": {"margin-top", "margin-right", "margin-bottom", "margin-left"},
}

// InheritedProperties are the properties that are inherited in CSS, for
// which unset is inherit (and initial for the others).
var InheritedProperties = map[string]bool{
	"color": true, "font": true, "font-family": true, "font-size": true, "font-style": true,
	"font-weight": true, "font-variant": true, "line-height": true, "letter-spacing": true,
	"word-spacing": true, "text-align": true, "text-indent": true, "text-transform": true,
	"text-shadow": true, "white-space": true, "visibility": true, "cursor": true, "direction": true, "pointer-events": true,
	"list-style": true, "list-style-type": true, "list-style-position": true, "quotes": true,
	"word-break": true, "overflow-wrap": true, "hyphens": true, "tab-size": true,
	"border-collapse": true, "border-spacing": true, "caption-side": true, "empty-cells": true,
}

// LogicalProperties are the physical properties for the logical
// properties of CSS, for horizontal, left to right text.
var LogicalProperties = map[string]string{
	"inline-size": "width", "block-size": "height",
	"min-inline-size": "min-width", "max-inline-size": "max-width",
	"min-block-size": "min-height", "max-block-size": "max-height",
	"margin-inline-start": "margin-left", "margin-inline-end": "margin-right",
	"margin-block-start": "margin-top", "margin-block-end": "margin-bottom",
	"padding-inline-start": "padding-left", "padding-inline-end": "padding-right",
	"padding-block-start": "padding-top", "padding-block-end": "padding-bottom",
	"border-inline-start": "border-left", "border-inline-end": "border-right",
	"border-block-start": "border-top", "border-block-end": "border-bottom",
	"border-inline-start-width": "border-left-width", "border-inline-end-width": "border-right-width",
	"border-block-start-width": "border-top-width", "border-block-end-width": "border-bottom-width",
	"border-inline-start-color": "border-left-color", "border-inline-end-color": "border-right-color",
	"border-block-start-color": "border-top-color", "border-block-end-color": "border-bottom-color",
	"inset-inline-start": "left", "inset-inline-end": "right",
	"inset-block-start": "top", "inset-block-end": "bottom",
	"border-start-start-radius": "border-top-left-radius", "border-start-end-radius": "border-top-right-radius",
	"border-end-start-radius": "border-bottom-left-radius", "border-end-end-radius": "border-bottom-right-radius",
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

var BorderStyleNames = map[string]bool{"none": true, "hidden": true, "solid": true, "dotted": true, "dashed": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true}

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
