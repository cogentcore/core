// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"math"
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

// styleOp applies a compiled CSS declaration to the given style, with the
// state of the style pass (as its length environment) and the parent style.
type styleOp func(st *cssState, s, ps *styles.Style)

// compileDecls compiles the given declarations, in the order of the cascade,
// without those that are invalid (see [Context.compileCSS]).
// The font properties are first, as in CSS the font size of an element
// is computed before the lengths that are relative to it (as the ch unit,
// and em in calc()).
func (c *Context) compileDecls(decls []*cssDecl) []styleOp {
	var ops []styleOp
	for _, font := range []bool{true, false} {
		for _, d := range decls {
			if !d.invalid && fontProperties[d.Property] == font {
				ops = append(ops, c.compileCSS(d.Property, d.Value)...)
			}
		}
	}
	return ops
}

// fontProperties are the properties of fonts, which are compiled first
// (with line-height, which the font shorthand sets).
var fontProperties = map[string]bool{
	"font": true, "font-family": true, "font-size": true, "font-weight": true,
	"font-style": true, "line-height": true,
}

// compileCSS compiles the declaration of the given CSS property with the
// given value into the operations that apply it to a style. The parts of
// the value that are the same for every element, as keywords, numbers and
// colors, are parsed once here; lengths are parsed when it is applied, as
// they depend on the length environment of the style pass (as for em, rem
// and calc()). It returns nil for a property that is not supported, or
// a value that is not valid.
func (c *Context) compileCSS(prop, val string) []styleOp {
	val = strings.TrimSpace(val)
	lval := strings.ToLower(val)
	op := func(f styleOp) []styleOp { return []styleOp{f} }
	if lval == "unset" { // inherit for the inherited properties, initial otherwise
		lval = "initial"
		if inheritedProperties[prop] {
			lval = "inherit"
		}
	}
	if lval == "inherit" || lval == "initial" {
		return op(func(st *cssState, s, ps *styles.Style) {
			s.FromProperty(ps, prop, lval, colors.BaseContext(colors.ToUniform(s.Color)))
		})
	}
	if p, ok := logicalProperties[prop]; ok {
		prop = p
	}
	switch prop {
	case "margin-inline", "margin-block", "padding-inline", "padding-block":
		fs := splitColors(val)
		if len(fs) == 0 {
			return nil
		}
		start, end := fs[0], fs[0]
		if len(fs) > 1 {
			end = fs[1]
		}
		base, dir, _ := strings.Cut(prop, "-")
		if dir == "inline" {
			return append(c.compileCSS(base+"-left", start), c.compileCSS(base+"-right", end)...)
		}
		return append(c.compileCSS(base+"-top", start), c.compileCSS(base+"-bottom", end)...)
	case "display":
		return op(func(st *cssState, s, ps *styles.Style) { st.display = lval })
	case "margin":
		vals := expandSides(splitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setMargin(st, i, v)
			}
		})
	case "margin-top", "margin-right", "margin-bottom", "margin-left":
		i := sideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { setMargin(st, i, lval) })
	case "padding":
		vals := expandSides(splitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				if l, ok := st.env.length(v, units.UnitPw); ok {
					setValueSide(&s.Padding, i, l)
				}
			}
		})
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		i := sideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.length(lval, units.UnitPw); ok {
				setValueSide(&s.Padding, i, l)
			}
		})
	case "border":
		return op(compileBorder(-1, val))
	case "border-top", "border-right", "border-bottom", "border-left":
		return op(compileBorder(sideIndex(prop), val))
	case "border-width":
		vals := expandSides(splitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setValueSide(&s.Border.Width, i, st.env.borderWidth(v))
			}
		})
	case "border-style":
		vals := expandSides(splitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setBorderStyle(s, i, v)
			}
		})
	case "border-color":
		var cfs []func() image.Image
		for _, v := range expandSides(splitColors(val)) {
			cfs = append(cfs, colorFunc(v))
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, cf := range cfs {
				setSide(&s.Border.Color, i, cf())
			}
		})
	case "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		i := sideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { setValueSide(&s.Border.Width, i, st.env.borderWidth(lval)) })
	case "border-top-style", "border-right-style", "border-bottom-style", "border-left-style":
		i := sideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { setBorderStyle(s, i, lval) })
	case "border-top-color", "border-right-color", "border-bottom-color", "border-left-color":
		i, cf := sideIndex(prop), colorFunc(val)
		return op(func(st *cssState, s, ps *styles.Style) { setSide(&s.Border.Color, i, cf()) })
	case "border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius":
		v := strings.Fields(lval + " 0")[0]
		i := map[string]int{"border-top-left-radius": 0, "border-top-right-radius": 1,
			"border-bottom-right-radius": 2, "border-bottom-left-radius": 3}[prop]
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.length(v, units.UnitEw); ok {
				setValueSide(&s.Border.Radius, i, l)
			}
		})
	case "border-radius":
		vals := expandSides(strings.Fields(strings.Split(lval, "/")[0]))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				if l, ok := st.env.length(v, units.UnitEw); ok {
					setValueSide(&s.Border.Radius, i, l)
				}
			}
		})
	case "background", "background-color":
		bg := backgroundColor(val)
		if bg == "" && prop == "background" && !strings.Contains(lval, "url(") && !strings.Contains(lval, "gradient(") {
			// The shorthand without a color (as "0 0" or "none", as
			// github.com resets its buttons with) sets the color to
			// transparent; with an image or gradient, which are not
			// supported, the color before it (a fallback) is kept.
			switch lval {
			case "inherit", "unset", "revert", "revert-layer":
			default:
				bg = "transparent"
			}
		}
		if bg == "" {
			return nil
		}
		none := bg == "transparent" || bg == "none"
		cf := colorFunc(bg)
		return op(func(st *cssState, s, ps *styles.Style) {
			st.background, st.hasBg = cf(), true
			if none {
				st.background = nil
			}
		})
	case "color":
		if parseColor(val) == nil {
			return nil
		}
		cf := colorFunc(val)
		return op(func(st *cssState, s, ps *styles.Style) { s.Color = cf() })
	case "width", "height", "min-width", "min-height", "max-width", "max-height":
		pct := units.UnitPw
		if strings.HasSuffix(prop, "height") {
			pct = units.UnitPh
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			st.setSize(prop, st.env.lengthPtr(lval, pct))
		})
	case "position":
		pos := styles.PositionStatic
		switch lval {
		case "relative", "sticky": // sticky is not supported, and is like relative
			pos = styles.PositionRelative
		case "absolute":
			pos = styles.PositionAbsolute
		case "fixed":
			pos = styles.PositionFixed
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Position = pos })
	case "top", "right", "bottom", "left":
		i := sideIndex("-" + prop)
		return op(func(st *cssState, s, ps *styles.Style) { setInset(st.env, s, i, lval) })
	case "inset":
		vals := expandSides(splitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setInset(st.env, s, i, v)
			}
		})
	case "inset-inline", "inset-block":
		fs := splitColors(lval)
		if len(fs) == 0 {
			return nil
		}
		start, end := fs[0], fs[0]
		if len(fs) > 1 {
			end = fs[1]
		}
		first, second := 3, 1 // the sides of the start and end
		if prop == "inset-block" {
			first, second = 0, 2
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			setInset(st.env, s, first, start)
			setInset(st.env, s, second, end)
		})
	case "float":
		f := styles.FloatNone
		switch lval {
		case "left", "inline-start":
			f = styles.FloatLeft
		case "right", "inline-end":
			f = styles.FloatRight
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Float = f })
	case "clear":
		cl := styles.ClearNone
		switch lval {
		case "left", "inline-start":
			cl = styles.ClearLeft
		case "right", "inline-end":
			cl = styles.ClearRight
		case "both":
			cl = styles.ClearBoth
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Clear = cl })
	case "mask", "mask-image", "mask-size", "mask-position", "mask-repeat",
		"-webkit-mask", "-webkit-mask-image", "-webkit-mask-size", "-webkit-mask-position", "-webkit-mask-repeat":
		if m := compileMask(prop, val); m != nil {
			return op(m)
		}
		return nil
	case "transform", "translate":
		x, y, ok := cssTranslate(prop, lval)
		if !ok {
			return nil
		}
		rot := cssRotate(prop, lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			s.Translate.X, _ = st.env.length(x, units.UnitEw)
			s.Translate.Y, _ = st.env.length(y, units.UnitEh)
			if prop == "transform" {
				s.Rotate = rot
			}
		})
	case "rotate":
		rot := cssAngle(lval)
		return op(func(st *cssState, s, ps *styles.Style) { s.Rotate = rot })
	case "z-index":
		z, err := strconv.Atoi(lval)
		if err != nil {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.ZIndex = z })
	case "vertical-align":
		return op(func(st *cssState, s, ps *styles.Style) { st.verticalAlign = lval })
	case "border-collapse":
		collapse := lval == "collapse"
		return op(func(st *cssState, s, ps *styles.Style) { st.borderCollapse = collapse })
	case "box-sizing":
		borderBox := lval == "border-box"
		return op(func(st *cssState, s, ps *styles.Style) { st.borderBox = borderBox })
	case "font-size":
		return op(func(st *cssState, s, ps *styles.Style) { setFontSize(st.env, s, ps, lval) })
	case "font-weight":
		return op(func(st *cssState, s, ps *styles.Style) { setFontWeight(s, ps, lval) })
	case "font-style":
		slant := rich.SlantNormal
		if lval == "italic" || lval == "oblique" {
			slant = rich.Italic
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Font.Slant = slant })
	case "font-family":
		family := fontFamily(val)
		return op(func(st *cssState, s, ps *styles.Style) { c.setFontFamily(s, val, family) })
	case "font":
		return op(func(st *cssState, s, ps *styles.Style) {
			if family := setFont(st, s, ps, val); family != "" {
				c.setFontFamily(s, family, s.Font.Family)
			}
		})
	case "line-height":
		return op(func(st *cssState, s, ps *styles.Style) { st.lineHeight = lval })
	case "text-align":
		align, legacy := text.Start, false
		switch lval {
		case "left", "start":
		case "right", "end":
			align = text.End
		case "center":
			align = text.Center
		case "-webkit-center":
			// the legacy center of the align attribute,
			// which also centers the blocks in the element
			align, legacy = text.Center, true
		case "justify":
			align = text.Justify
		default:
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			s.Text.Align = align
			if legacy {
				st.legacyCenter = true
			}
		})
	case "white-space":
		ws := text.WrapAsNeeded
		switch lval {
		case "nowrap":
			ws = text.WrapNever
		case "pre":
			ws = text.WhiteSpacePre
		case "pre-wrap", "pre-line", "break-spaces":
			ws = text.WhiteSpacePreWrap
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Text.WhiteSpace = ws })
	case "text-indent":
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.length(lval, units.UnitPw); ok {
				s.Text.Indent = l
			}
		})
	case "text-decoration", "text-decoration-line":
		var deco rich.Decorations
		for _, f := range strings.Fields(lval) {
			switch f {
			case "underline":
				deco.SetFlag(true, rich.Underline)
			case "overline":
				deco.SetFlag(true, rich.Overline)
			case "line-through":
				deco.SetFlag(true, rich.LineThrough)
			}
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Font.Decoration = deco })
	case "flex-direction":
		dir := styles.Row
		if strings.HasPrefix(lval, "column") {
			dir = styles.Column
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Direction = dir })
	case "flex-wrap":
		wrap := lval == "wrap" || lval == "wrap-reverse"
		return op(func(st *cssState, s, ps *styles.Style) { s.Wrap = wrap })
	case "flex-flow":
		var ops []styleOp
		for _, f := range strings.Fields(lval) {
			ops = append(ops, c.compileCSS("flex-direction", f)...)
			if strings.Contains(f, "wrap") {
				ops = append(ops, c.compileCSS("flex-wrap", f)...)
			}
		}
		return ops
	case "flex-grow", "flex-shrink":
		f, err := strconv.ParseFloat(lval, 32)
		if err != nil {
			return nil
		}
		if prop == "flex-grow" {
			return op(func(st *cssState, s, ps *styles.Style) { st.flexGrow, st.hasGrow = float32(f), true })
		}
		return op(func(st *cssState, s, ps *styles.Style) { st.flexShrink, st.hasShrink = float32(f), true })
	case "flex":
		var grow, shrink float32
		hasGrow, hasShrink := false, false
		switch lval {
		case "none":
			hasGrow, hasShrink = true, true
		case "auto":
			grow, hasGrow, shrink, hasShrink = 1, true, 1, true
		default:
			fs := strings.Fields(lval)
			if len(fs) == 0 {
				return nil
			}
			if f, err := strconv.ParseFloat(fs[0], 32); err == nil {
				grow, hasGrow = float32(f), true
			}
			shrink, hasShrink = 1, true
			if len(fs) > 1 {
				if f, err := strconv.ParseFloat(fs[1], 32); err == nil {
					shrink = float32(f)
				}
			}
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			if hasGrow {
				st.flexGrow, st.hasGrow = grow, true
			}
			if hasShrink {
				st.flexShrink, st.hasShrink = shrink, true
			}
		})
	case "gap", "grid-gap":
		fs := splitColors(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
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
		})
	case "row-gap":
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.length(lval, units.UnitPh); ok {
				s.Gap.Y = l
			}
		})
	case "column-gap":
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.length(lval, units.UnitPw); ok {
				s.Gap.X = l
			}
		})
	case "border-spacing":
		fs := splitColors(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
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
		})
	case "justify-content", "justify-items", "justify-self", "align-content":
		a := alignValue(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			switch prop {
			case "justify-content":
				s.Justify.Content = a
			case "justify-items":
				s.Justify.Items = a
			case "justify-self":
				s.Justify.Self = a
			default:
				s.Align.Content = a
			}
		})
	case "align-items":
		a := alignValue(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			st.alignItems = lval
			s.Align.Items = a
		})
	case "align-self":
		a := alignValue(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			st.alignSelf = lval
			s.Align.Self = a
		})
	case "overflow", "overflow-x", "overflow-y":
		// Overflow that is hidden is clipped to the size of the box, which
		// does not grow to fit it, as with a scrolling overflow in core, but
		// without scrollbars (OverflowHidden in core does not clip).
		vals := strings.Fields(lval)
		if len(vals) == 0 {
			return nil
		}
		if len(vals) == 1 {
			vals = append(vals, vals[0])
		}
		type dimOverflow struct {
			dim    math32.Dims
			ov     styles.Overflows
			hidden bool
		}
		var dims []dimOverflow
		for d, v := range vals[:2] {
			if prop == "overflow-x" && d == 1 || prop == "overflow-y" && d == 0 {
				continue
			}
			do := dimOverflow{dim: math32.Dims(d), ov: styles.OverflowVisible}
			switch v {
			case "hidden", "clip":
				do.ov, do.hidden = styles.OverflowAuto, true
			case "auto", "scroll":
				do.ov = styles.OverflowAuto
			}
			dims = append(dims, do)
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			for _, do := range dims {
				st.overflowHidden[do.dim] = do.hidden
				s.Overflow.SetDim(do.dim, do.ov)
			}
		})
	case "grid-template-columns":
		cols := gridColumns(lval)
		return op(func(st *cssState, s, ps *styles.Style) { s.Columns = cols })
	case "opacity":
		f, err := strconv.ParseFloat(lval, 32)
		if err != nil {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Opacity = float32(f) })
	case "visibility":
		if lval == "hidden" || lval == "collapse" {
			return op(func(st *cssState, s, ps *styles.Style) { s.Opacity = 0 })
		}
	}
	return nil
}

// colorFunc returns a function that returns the given CSS color (or nil if
// it is not one). A uniform color is parsed once, and is shared by the
// styles; other colors (as gradients) are parsed for each style, so that
// each has its own.
func colorFunc(v string) func() image.Image {
	cl := parseColor(v)
	if _, ok := cl.(*image.Uniform); ok || cl == nil {
		return func() image.Image { return cl }
	}
	return func() image.Image { return parseColor(v) }
}

// setSize sets the given size property (as max-width) to the given length.
func (st *cssState) setSize(prop string, v *units.Value) {
	switch prop {
	case "width":
		st.width = v
	case "height":
		st.height = v
	case "min-width":
		st.minWidth = v
	case "min-height":
		st.minHeight = v
	case "max-width":
		st.maxWidth = v
	default:
		st.maxHeight = v
	}
}

// compileBorder compiles the border shorthand for the given side, or all if -1.
func compileBorder(side int, val string) styleOp {
	style, width := "none", ""
	var color func() image.Image // the color of the text if none
	for _, f := range splitColors(val) {
		lf := strings.ToLower(f)
		switch {
		case borderStyleNames[lf]:
			style = lf
		case lf == "thin" || lf == "medium" || lf == "thick" || (len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.')):
			width = lf
		default:
			if parseColor(f) != nil {
				color = colorFunc(f)
			}
		}
	}
	return func(st *cssState, s, ps *styles.Style) {
		w := units.Dp(3) // medium
		if width != "" {
			w = st.env.borderWidth(width)
		}
		var cl image.Image = colors.Uniform(colors.ToUniform(s.Color))
		if color != nil {
			cl = color()
		}
		for i := range 4 {
			if side >= 0 && i != side {
				continue
			}
			setSide(&s.Border.Color, i, cl)
			setBorderStyle(s, i, style)
			if style != "none" && style != "hidden" {
				setValueSide(&s.Border.Width, i, w)
			}
		}
	}
}

// inheritedProperties are the properties that are inherited in CSS, for
// which unset is inherit (and initial for the others).
var inheritedProperties = map[string]bool{
	"color": true, "font": true, "font-family": true, "font-size": true, "font-style": true,
	"font-weight": true, "font-variant": true, "line-height": true, "letter-spacing": true,
	"word-spacing": true, "text-align": true, "text-indent": true, "text-transform": true,
	"text-shadow": true, "white-space": true, "visibility": true, "cursor": true, "direction": true,
	"list-style": true, "list-style-type": true, "list-style-position": true, "quotes": true,
	"word-break": true, "overflow-wrap": true, "hyphens": true, "tab-size": true,
	"border-collapse": true, "border-spacing": true, "caption-side": true, "empty-cells": true,
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

// cssTranslate returns the translation of the given value of the transform
// property (the sum of its translate functions; the other functions, as
// rotate and scale, are not supported), or of the translate property, as
// the lengths of x and y. It returns false if it is not valid.
func cssTranslate(prop, v string) (x, y string, ok bool) {
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
		end := strings.Index(args, ")")
		if end < 0 {
			return "", "", false
		}
		as := strings.Split(args[:end], ",")
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
		rest = strings.TrimSpace(args[end+1:])
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

// cssRotate returns the sum of the angles of the rotate functions of the
// given transform, in degrees.
func cssRotate(prop, v string) float32 {
	var deg float32
	for rest := v; rest != ""; {
		name, args, found := strings.Cut(rest, "(")
		if !found {
			break
		}
		end := strings.Index(args, ")")
		if end < 0 {
			break
		}
		if n := strings.TrimSpace(name); n == "rotate" || n == "rotatez" {
			deg += cssAngle(strings.TrimSpace(args[:end]))
		}
		rest = strings.TrimSpace(args[end+1:])
	}
	return deg
}

// cssAngle returns the given CSS angle in degrees (deg, rad, grad, turn).
func cssAngle(v string) float32 {
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
