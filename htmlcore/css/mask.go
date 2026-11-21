// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strconv"
	"strings"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
)

// SplitTopLevelSpaces splits the given value at the spaces that are not in
// parentheses (as "calc(1rem + 4px) auto").
func SplitTopLevelSpaces(v string) []string {
	var parts []string
	depth, start := 0, -1
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ' ' && depth == 0:
			if start >= 0 {
				parts = append(parts, v[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, v[start:])
	}
	return parts
}

// MaskSize returns the size of the mask image in the given box, for the
// given mask-size (as "20px auto", "contain", "cover", or "50%"), and the
// intrinsic size of the image, in pixels, with the given dots per px.
func MaskSize(v string, box, intrinsic math32.Vector2, dp float32) math32.Vector2 {
	aspect := intrinsic.Y / intrinsic.X
	switch strings.TrimSpace(v) {
	case "contain", "cover":
		sx, sy := box.X/intrinsic.X, box.Y/intrinsic.Y
		s := min(sx, sy)
		if v == "cover" {
			s = max(sx, sy)
		}
		return intrinsic.MulScalar(s)
	case "", "auto", "auto auto":
		return intrinsic
	}
	fs := strings.Fields(v)
	length := func(f string, ref float32) (float32, bool) {
		if pct, ok := strings.CutSuffix(f, "%"); ok {
			n, err := strconv.ParseFloat(pct, 32)
			return float32(n) / 100 * ref, err == nil
		}
		if f == "auto" {
			return 0, false
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(f, "px"), 32)
		return float32(n) * dp, err == nil
	}
	w, wok := length(fs[0], box.X)
	h, hok := float32(0), false
	if len(fs) > 1 {
		h, hok = length(fs[1], box.Y)
	}
	switch {
	case wok && hok:
		return math32.Vec2(w, h)
	case wok:
		return math32.Vec2(w, w*aspect)
	case hok:
		return math32.Vec2(h/aspect, h)
	}
	return intrinsic
}

// MaskPosition returns the fractions of the extra space of the box on each
// axis that are before the mask image, for the given mask-position (as
// "50% 50%", "center", or "left top").
func MaskPosition(v string) (x, y float32) {
	fs := strings.Fields(strings.ToLower(v))
	if len(fs) == 0 {
		return 0, 0
	}
	if len(fs) == 1 {
		fs = append(fs, "center")
	}
	frac := func(f string) float32 {
		switch f {
		case "left", "top":
			return 0
		case "center":
			return 0.5
		case "right", "bottom":
			return 1
		}
		if pct, ok := strings.CutSuffix(f, "%"); ok {
			n, _ := strconv.ParseFloat(pct, 32)
			return float32(n) / 100
		}
		return 0
	}
	if fs[0] == "top" || fs[0] == "bottom" || fs[1] == "left" || fs[1] == "right" {
		fs[0], fs[1] = fs[1], fs[0] // as "top left"
	}
	return frac(fs[0]), frac(fs[1])
}

// MaskOffset returns the offsets in px of the given mask-position (or
// background-position) that are lengths (as the positions of the images
// in a sprite, as "0 -261px"), which are from the left and top.
func MaskOffset(v string) (x, y float32) {
	fs := strings.Fields(strings.ToLower(v))
	if len(fs) == 0 {
		return 0, 0
	}
	if len(fs) == 1 {
		fs = append(fs, "center")
	}
	if fs[0] == "top" || fs[0] == "bottom" || fs[1] == "left" || fs[1] == "right" {
		fs[0], fs[1] = fs[1], fs[0]
	}
	px := func(f string) float32 {
		if n, ok := strings.CutSuffix(f, "px"); ok {
			v, _ := strconv.ParseFloat(n, 32)
			return float32(v)
		}
		return 0
	}
	return px(fs[0]), px(fs[1])
}

// MaskURL returns the URL of the url() of the given value of mask-image
// (or the mask shorthand), or "" if it has none (as none).
func MaskURL(v string) string {
	i := strings.Index(strings.ToLower(v), "url(")
	if i < 0 {
		return ""
	}
	rest := v[i+4:]
	before, _, ok := strings.Cut(rest, ")")
	if t := strings.TrimSpace(rest); strings.HasPrefix(t, `"`) || strings.HasPrefix(t, `'`) {
		// a CSS string, with backslash escapes (as the quotes of the
		// attributes of an SVG in a data URL)
		q := t[0]
		var sb strings.Builder
		for i := 1; i < len(t); i++ {
			switch c := t[i]; {
			case c == '\\' && i+1 < len(t):
				i++
				sb.WriteByte(t[i])
			case c == q:
				return sb.String()
			default:
				sb.WriteByte(c)
			}
		}
		return ""
	}
	if !ok {
		return ""
	}
	return strings.TrimSpace(before)
}

// ResolveMaskSize returns the given mask-size with its lengths (as em and
// calc()) resolved to px with the given length environment, as maskSize
// reads px, percentages, auto, contain, and cover.
func ResolveMaskSize(env *LengthEnv, v string) string {
	parts := SplitTopLevelSpaces(v)
	for i, p := range parts {
		switch {
		case p == "auto" || p == "contain" || p == "cover" || strings.HasSuffix(p, "%"):
			continue
		}
		if l, ok := env.Length(p, units.UnitPw); ok && l.Unit == units.UnitDp { // (% are of the box)
			parts[i] = strconv.FormatFloat(float64(l.Value), 'f', -1, 32) + "px"
		}
	}
	return strings.Join(parts, " ")
}

// IsPositionWord returns whether the given word is one of a position (a
// keyword or a length), not a color.
func IsPositionWord(f string) bool {
	switch f {
	case "left", "right", "top", "bottom", "center":
		return true
	}
	return f != "" && (f[0] >= '0' && f[0] <= '9' || f[0] == '-' || f[0] == '.')
}
