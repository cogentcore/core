// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// setCSS sets the given CSS property to the given value.
func (c *Context) setCSS(st *cssState, s, ps *styles.Style, prop, val string) {
	val = strings.TrimSpace(val)
	lval := strings.ToLower(val)
	if lval == "inherit" || lval == "initial" || lval == "unset" {
		s.FromProperty(ps, prop, lval, colors.BaseContext(colors.ToUniform(s.Color)))
		return
	}
	if p, ok := logicalProperties[prop]; ok {
		prop = p
	}
	switch prop {
	case "margin-inline", "margin-block", "padding-inline", "padding-block":
		fs := splitColors(val)
		if len(fs) == 0 {
			return
		}
		start, end := fs[0], fs[0]
		if len(fs) > 1 {
			end = fs[1]
		}
		base, dir, _ := strings.Cut(prop, "-")
		if dir == "inline" {
			c.setCSS(st, s, ps, base+"-left", start)
			c.setCSS(st, s, ps, base+"-right", end)
		} else {
			c.setCSS(st, s, ps, base+"-top", start)
			c.setCSS(st, s, ps, base+"-bottom", end)
		}
		return
	case "display":
		st.display = lval
	case "margin":
		vals := expandSides(splitColors(lval))
		for i, v := range vals {
			setMargin(st, i, v)
		}
	case "margin-top":
		setMargin(st, 0, lval)
	case "margin-right":
		setMargin(st, 1, lval)
	case "margin-bottom":
		setMargin(st, 2, lval)
	case "margin-left":
		setMargin(st, 3, lval)
	case "padding":
		vals := expandSides(splitColors(lval))
		for i, v := range vals {
			if l, ok := st.env.length(v, units.UnitPw); ok {
				setValueSide(&s.Padding, i, l)
			}
		}
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		if l, ok := st.env.length(lval, units.UnitPw); ok {
			setValueSide(&s.Padding, sideIndex(prop), l)
		}
	case "border":
		setBorder(st.env, s, -1, val)
	case "border-top", "border-right", "border-bottom", "border-left":
		setBorder(st.env, s, sideIndex(prop), val)
	case "border-width":
		for i, v := range expandSides(splitColors(lval)) {
			setValueSide(&s.Border.Width, i, st.env.borderWidth(v))
		}
	case "border-style":
		for i, v := range expandSides(splitColors(lval)) {
			setBorderStyle(s, i, v)
		}
	case "border-color":
		for i, v := range expandSides(splitColors(val)) {
			setSide(&s.Border.Color, i, parseColor(v))
		}
	case "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		setValueSide(&s.Border.Width, sideIndex(prop), st.env.borderWidth(lval))
	case "border-top-style", "border-right-style", "border-bottom-style", "border-left-style":
		setBorderStyle(s, sideIndex(prop), lval)
	case "border-top-color", "border-right-color", "border-bottom-color", "border-left-color":
		setSide(&s.Border.Color, sideIndex(prop), parseColor(val))
	case "border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius":
		if l, ok := st.env.length(strings.Fields(lval + " 0")[0], units.UnitEw); ok {
			i := map[string]int{"border-top-left-radius": 0, "border-top-right-radius": 1,
				"border-bottom-right-radius": 2, "border-bottom-left-radius": 3}[prop]
			setValueSide(&s.Border.Radius, i, l)
		}
	case "border-radius":
		vals := strings.Fields(strings.Split(lval, "/")[0])
		for i, v := range expandSides(vals) {
			if l, ok := st.env.length(v, units.UnitEw); ok {
				setValueSide(&s.Border.Radius, i, l)
			}
		}
	case "background", "background-color":
		if bg := backgroundColor(val); bg != "" {
			st.background, st.hasBg = parseColor(bg), true
			if bg == "transparent" || bg == "none" {
				st.background = nil
			}
		}
	case "color":
		if cl := parseColor(val); cl != nil {
			s.Color = cl
		}
	case "width":
		st.width = st.env.lengthPtr(lval, units.UnitPw)
	case "height":
		st.height = st.env.lengthPtr(lval, units.UnitPh)
	case "min-width":
		st.minWidth = st.env.lengthPtr(lval, units.UnitPw)
	case "min-height":
		st.minHeight = st.env.lengthPtr(lval, units.UnitPh)
	case "max-width":
		st.maxWidth = st.env.lengthPtr(lval, units.UnitPw)
	case "max-height":
		st.maxHeight = st.env.lengthPtr(lval, units.UnitPh)
	case "position":
		switch lval {
		case "relative", "sticky": // sticky is not supported, and is like relative
			s.Position = styles.PositionRelative
		case "absolute":
			s.Position = styles.PositionAbsolute
		case "fixed":
			s.Position = styles.PositionFixed
		default:
			s.Position = styles.PositionStatic
		}
	case "top", "right", "bottom", "left":
		setInset(st.env, s, sideIndex("-"+prop), lval)
	case "inset":
		for i, v := range expandSides(splitColors(lval)) {
			setInset(st.env, s, i, v)
		}
	case "inset-inline", "inset-block":
		fs := splitColors(lval)
		if len(fs) == 0 {
			return
		}
		start, end := fs[0], fs[0]
		if len(fs) > 1 {
			end = fs[1]
		}
		if prop == "inset-inline" {
			setInset(st.env, s, 3, start)
			setInset(st.env, s, 1, end)
		} else {
			setInset(st.env, s, 0, start)
			setInset(st.env, s, 2, end)
		}
	case "float":
		switch lval {
		case "left", "inline-start":
			s.Float = styles.FloatLeft
		case "right", "inline-end":
			s.Float = styles.FloatRight
		default:
			s.Float = styles.FloatNone
		}
	case "clear":
		switch lval {
		case "left", "inline-start":
			s.Clear = styles.ClearLeft
		case "right", "inline-end":
			s.Clear = styles.ClearRight
		case "both":
			s.Clear = styles.ClearBoth
		default:
			s.Clear = styles.ClearNone
		}
	case "z-index":
		if z, err := strconv.Atoi(lval); err == nil {
			s.ZIndex = z
		}
	case "vertical-align":
		st.verticalAlign = lval
	case "border-collapse":
		st.borderCollapse = lval == "collapse"
	case "box-sizing":
		st.borderBox = lval == "border-box"
	case "font-size":
		setFontSize(st.env, s, ps, lval)
	case "font-weight":
		setFontWeight(s, ps, lval)
	case "font-style":
		if lval == "italic" || lval == "oblique" {
			s.Font.Slant = rich.Italic
		} else {
			s.Font.Slant = rich.SlantNormal
		}
	case "font-family":
		s.Font.Family = fontFamily(val)
	case "font":
		setFont(st, s, ps, val)
	case "line-height":
		st.lineHeight = lval
	case "text-align":
		switch lval {
		case "left", "start":
			s.Text.Align = text.Start
		case "right", "end":
			s.Text.Align = text.End
		case "center":
			s.Text.Align = text.Center
		case "-webkit-center":
			// the legacy center of the align attribute,
			// which also centers the blocks in the element
			s.Text.Align = text.Center
			st.legacyCenter = true
		case "justify":
			s.Text.Align = text.Justify
		}
	case "white-space":
		switch lval {
		case "nowrap":
			s.Text.WhiteSpace = text.WrapNever
		case "pre":
			s.Text.WhiteSpace = text.WhiteSpacePre
		case "pre-wrap", "pre-line", "break-spaces":
			s.Text.WhiteSpace = text.WhiteSpacePreWrap
		default:
			s.Text.WhiteSpace = text.WrapAsNeeded
		}
	case "text-indent":
		if l, ok := st.env.length(lval, units.UnitPw); ok {
			s.Text.Indent = l
		}
	case "text-decoration", "text-decoration-line":
		s.Font.Decoration = 0
		for _, f := range strings.Fields(lval) {
			switch f {
			case "underline":
				s.Font.Decoration.SetFlag(true, rich.Underline)
			case "overline":
				s.Font.Decoration.SetFlag(true, rich.Overline)
			case "line-through":
				s.Font.Decoration.SetFlag(true, rich.LineThrough)
			}
		}
	case "flex-direction":
		if strings.HasPrefix(lval, "column") {
			s.Direction = styles.Column
		} else {
			s.Direction = styles.Row
		}
	case "flex-wrap":
		s.Wrap = lval == "wrap" || lval == "wrap-reverse"
	case "flex-flow":
		for _, f := range strings.Fields(lval) {
			c.setCSS(st, s, ps, "flex-direction", f)
			if strings.Contains(f, "wrap") {
				c.setCSS(st, s, ps, "flex-wrap", f)
			}
		}
	case "flex-grow":
		if f, err := strconv.ParseFloat(lval, 32); err == nil {
			st.flexGrow, st.hasGrow = float32(f), true
		}
	case "flex-shrink":
		if f, err := strconv.ParseFloat(lval, 32); err == nil {
			st.flexShrink, st.hasShrink = float32(f), true
		}
	case "flex":
		switch lval {
		case "none":
			st.flexGrow, st.hasGrow = 0, true
			st.flexShrink, st.hasShrink = 0, true
		case "auto":
			st.flexGrow, st.hasGrow = 1, true
			st.flexShrink, st.hasShrink = 1, true
		default:
			if fs := strings.Fields(lval); len(fs) > 0 {
				if f, err := strconv.ParseFloat(fs[0], 32); err == nil {
					st.flexGrow, st.hasGrow = float32(f), true
				}
				st.flexShrink, st.hasShrink = 1, true
				if len(fs) > 1 {
					if f, err := strconv.ParseFloat(fs[1], 32); err == nil {
						st.flexShrink = float32(f)
					}
				}
			}
		}
	case "gap", "grid-gap":
		fs := splitColors(lval)
		if len(fs) > 0 {
			if l, ok := st.env.length(fs[0], units.UnitPh); ok {
				s.Gap.Y = l
				s.Gap.X = l
			}
		}
		if len(fs) > 1 {
			if l, ok := st.env.length(fs[1], units.UnitPw); ok {
				s.Gap.X = l
			}
		}
	case "row-gap":
		if l, ok := st.env.length(lval, units.UnitPh); ok {
			s.Gap.Y = l
		}
	case "column-gap":
		if l, ok := st.env.length(lval, units.UnitPw); ok {
			s.Gap.X = l
		}
	case "border-spacing":
		fs := splitColors(lval)
		if len(fs) > 0 {
			if l, ok := st.env.length(fs[0], units.UnitPw); ok {
				s.Gap.Set(l)
			}
		}
		if len(fs) > 1 {
			if l, ok := st.env.length(fs[1], units.UnitPh); ok {
				s.Gap.Y = l
			}
		}
	case "justify-content":
		s.Justify.Content = alignValue(lval)
	case "justify-items":
		s.Justify.Items = alignValue(lval)
	case "justify-self":
		s.Justify.Self = alignValue(lval)
	case "align-items":
		st.alignItems = lval
		s.Align.Items = alignValue(lval)
	case "align-self":
		st.alignSelf = lval
		s.Align.Self = alignValue(lval)
	case "align-content":
		s.Align.Content = alignValue(lval)
	case "overflow", "overflow-x", "overflow-y":
		// Overflow that is hidden is clipped to the size of the box, which
		// does not grow to fit it, as with a scrolling overflow in core, but
		// without scrollbars (OverflowHidden in core does not clip).
		vals := strings.Fields(lval)
		if len(vals) == 0 {
			return
		}
		if len(vals) == 1 {
			vals = append(vals, vals[0])
		}
		for d, v := range vals[:2] {
			if prop == "overflow-x" && d == 1 || prop == "overflow-y" && d == 0 {
				continue
			}
			ov := styles.OverflowVisible
			hd := d
			if prop == "overflow-y" {
				hd = 1
			}
			st.overflowHidden[hd] = false
			switch v {
			case "hidden", "clip":
				ov = styles.OverflowAuto
				st.overflowHidden[hd] = true
			case "auto", "scroll":
				ov = styles.OverflowAuto
			}
			if prop == "overflow-y" {
				d = 1
			}
			s.Overflow.SetDim(math32.Dims(d), ov)
		}
	case "grid-template-columns":
		s.Columns = gridColumns(lval)
	case "opacity":
		if f, err := strconv.ParseFloat(lval, 32); err == nil {
			s.Opacity = float32(f)
		}
	case "visibility":
		if lval == "hidden" || lval == "collapse" {
			s.Opacity = 0
		}
	}
}

// logicalProperties are the physical properties for the logical
// properties of CSS, for horizontal, left to right text.
var logicalProperties = map[string]string{
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

// expandSides expands the 1 to 4 values of a CSS shorthand for
// the sides of a box into the values for top, right, bottom, left.
func expandSides(vals []string) []string {
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

// sideIndex returns the index (top, right, bottom, left) of the side in
// the name of the given property, such as "margin-left".
func sideIndex(prop string) int {
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

// setSide sets the given side (top, right, bottom, left) of the given sides.
func setSide[T any](s *sides.Sides[T], i int, v T) {
	setSidePtr(&s.Top, &s.Right, &s.Bottom, &s.Left, i, v)
}

// setValueSide sets the given side (top, right, bottom, left) of the given values.
func setValueSide(s *sides.Values, i int, v units.Value) {
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

// setInset sets the given side (top, right, bottom, left) of the inset
// of a positioned element; auto is not set.
func setInset(env *lengthEnv, s *styles.Style, i int, v string) {
	pct := units.UnitPw
	if i == 0 || i == 2 {
		pct = units.UnitPh
	}
	l, ok := env.length(v, pct)
	setSide(&s.InsetSet, i, ok)
	if ok {
		setValueSide(&s.Inset, i, l)
	}
}

func setMargin(st *cssState, i int, v string) {
	if v == "auto" {
		st.marginAuto[i] = true
		st.margin[i] = nil
		return
	}
	st.marginAuto[i] = false
	if l, ok := st.env.length(v, units.UnitPw); ok {
		st.margin[i] = &l
	}
}

var borderStyleNames = map[string]bool{"none": true, "hidden": true, "solid": true, "dotted": true, "dashed": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true}

// setBorderStyle sets the style of the given side of the border.
func setBorderStyle(s *styles.Style, i int, v string) {
	if v == "none" || v == "hidden" {
		setValueSide(&s.Border.Width, i, units.Dp(0))
		setSide(&s.Border.Style, i, styles.BorderNone)
		return
	}
	var bs styles.BorderStyles
	if bs.SetString(v) == nil {
		setSide(&s.Border.Style, i, bs)
	}
}

// setBorder sets the border shorthand for the given side, or all if -1.
func setBorder(env *lengthEnv, s *styles.Style, side int, val string) {
	width := units.Dp(3) // medium
	style := "none"
	var color image.Image = colors.Uniform(colors.ToUniform(s.Color))
	for _, f := range splitColors(val) {
		lf := strings.ToLower(f)
		switch {
		case borderStyleNames[lf]:
			style = lf
		case lf == "thin" || lf == "medium" || lf == "thick" || (len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.')):
			width = env.borderWidth(lf)
		default:
			if cl := parseColor(f); cl != nil {
				color = cl
			}
		}
	}
	for i := range 4 {
		if side >= 0 && i != side {
			continue
		}
		setSide(&s.Border.Color, i, color)
		setBorderStyle(s, i, style)
		if style != "none" && style != "hidden" {
			setValueSide(&s.Border.Width, i, width)
		}
	}
}

// splitColors splits the given value into fields at spaces
// that are not inside of parentheses, as in rgb(...) and calc(...).
func splitColors(v string) []string {
	var fs []string
	depth := 0
	start := -1
	for i, r := range v {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ' ' || r == '\t' || r == '\n':
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

// alignValue returns the alignment for the given CSS value.
func alignValue(v string) styles.Aligns {
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

// gridColumns returns the number of columns of a grid-template-columns value.
func gridColumns(v string) int {
	if strings.HasPrefix(v, "repeat(") {
		n, _, _ := strings.Cut(strings.TrimPrefix(v, "repeat("), ",")
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i
		}
	}
	return len(splitColors(v))
}
