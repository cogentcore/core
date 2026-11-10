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
	"cogentcore.org/core/styles/abilities"
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
	if p, ok := strings.CutPrefix(prop, "-webkit-"); ok && webkitAliases[p] {
		// the old prefixed property is the standard one (as the
		// -webkit-transform with which the slider of the photos of the
		// postings of craigslist.org moves its slides); the last of them
		// in the cascade applies, as in browsers
		prop = p
	}
	val = strings.TrimSpace(val)
	lval := strings.ToLower(val)
	op := func(f styleOp) []styleOp { return []styleOp{f} }
	if lval == "unset" { // inherit for the inherited properties, initial otherwise
		lval = "initial"
		if inheritedProperties[prop] {
			lval = "inherit"
		}
	}
	if prop == "clip" || prop == "clip-path" {
		// a clip to nothing hides the element (as the visually hidden
		// skip links of go.dev, of clip: rect(0 0 0 0), unset on focus);
		// other clips are not applied
		zero := zeroAreaClip(prop, lval)
		return op(func(st *cssState, s, ps *styles.Style) { st.clippedOut = zero })
	}
	if prop == "visibility" && (lval == "inherit" || lval == "initial") {
		// that of the parent, or visible (as the repository panels of
		// github.com, which a script shows with visibility: inherit over
		// the hidden of their class, which was kept)
		v := "visible"
		if lval == "inherit" {
			v = "" // (see resolve: that of the parent)
		}
		return op(func(st *cssState, s, ps *styles.Style) { st.visibility = v })
	}
	if lval == "initial" && (prop == "background" || prop == "background-color") {
		// transparent, over the background of a control (as the menu
		// buttons of MDN, which were gray)
		return op(func(st *cssState, s, ps *styles.Style) {
			st.background, st.hasBg, st.bgCurrentColor = nil, true, false
		})
	}
	if lval == "inherit" && (prop == "background" || prop == "background-color") {
		// that of the parent, over a background of an earlier rule (as
		// the code of the Python docs, pre with background-color: inherit
		// of the gray of its highlight block, over a green for pre)
		return op(func(st *cssState, s, ps *styles.Style) {
			var bg image.Image
			if ps != nil {
				bg = ps.Background
			}
			st.background, st.hasBg, st.bgCurrentColor = bg, true, false
		})
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
		d := legacyDisplay(lval)
		return op(func(st *cssState, s, ps *styles.Style) { st.display = d })
	case "-webkit-line-clamp", "line-clamp":
		n, err := strconv.Atoi(lval)
		if lval == "none" || lval == "initial" {
			n, err = 0, nil
		}
		if err != nil || n < 0 {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Text.LineClamp = n })
	case "text-overflow":
		e := lval == "ellipsis"
		return op(func(st *cssState, s, ps *styles.Style) { s.Text.Ellipsis = e })
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
	case "background-image", "background-size", "background-position", "background-repeat":
		if m := compileBackgroundImage(prop, val); m != nil {
			return op(m)
		}
		return nil
	case "background", "background-color":
		var imageOp []styleOp
		if prop == "background" {
			imageOp = op(compileBackgroundImage(prop, val))
		}
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
			return imageOp
		}
		none := bg == "transparent" || bg == "none"
		current := strings.EqualFold(bg, "currentcolor")
		cf := colorFunc(bg)
		return append(imageOp, op(func(st *cssState, s, ps *styles.Style) {
			st.background, st.hasBg, st.bgCurrentColor = cf(), true, current
			if none || current {
				st.background = nil
			}
		})...)
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
			if prop == "width" {
				// as wide as its content on one line (see styles.MaxContentWidth)
				s.MaxContentWidth = lval == "max-content"
			}
			if prop == "height" { // the height of its content (as for a modal dialog)
				st.heightFit = strings.Contains(lval, "fit-content") || lval == "max-content" || lval == "min-content"
			}
			st.setSize(prop, st.env.lengthPtr(lval, pct))
		})
	case "position":
		pos := styles.PositionStatic
		switch lval {
		case "sticky":
			pos = styles.PositionSticky
		case "relative":
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
		zero := prop == "transform" && cssZeroScale(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			s.Translate.X, _ = st.env.length(x, units.UnitEw)
			s.Translate.Y, _ = st.env.length(y, units.UnitEh)
			if prop == "transform" {
				s.Rotate = rot
			}
			st.zeroScale = zero // (see resolve)
		})
	case "container-type", "container", "contain":
		c := styles.ContainNone
		v := lval
		if prop == "container" { // name / type
			_, v, _ = strings.Cut(lval, "/")
		}
		for _, f := range strings.Fields(v) {
			switch f {
			case "size", "strict":
				c = styles.ContainSize
			case "inline-size":
				if c == styles.ContainNone {
					c = styles.ContainInlineSize
				}
			}
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Contain = c })
	case "contain-intrinsic-width", "contain-intrinsic-height", "contain-intrinsic-size":
		// the lengths of the values, without auto (the size that a box
		// last had, which is the length until it is laid out); none is no size
		var vals []string
		for _, f := range strings.Fields(lval) {
			if f != "auto" {
				vals = append(vals, f)
			}
		}
		if len(vals) == 0 {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			length := func(v string) *units.Value {
				if v == "none" {
					return nil
				}
				return st.env.lengthPtr(v, units.UnitPw)
			}
			if prop != "contain-intrinsic-height" {
				st.intrinsicWidth = length(vals[0])
			}
			if prop != "contain-intrinsic-width" {
				st.intrinsicHeight = length(vals[len(vals)-1])
			}
		})
	case "aspect-ratio":
		r := cssAspectRatio(lval)
		return op(func(st *cssState, s, ps *styles.Style) { s.AspectRatio = r })
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
				// the text is moved out of view (as the labels of icons,
				// which have it), as an indent is not otherwise supported
				st.hideText = (l.Unit == units.UnitDp || l.Unit == units.UnitPx) && l.Value <= -500
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
		grow, shrink, basis, ok := parseFlex(lval)
		if !ok {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			st.flexGrow, st.hasGrow = grow, true
			st.flexShrink, st.hasShrink = shrink, true
			st.flexBasis = basis
		})
	case "flex-basis":
		return op(func(st *cssState, s, ps *styles.Style) { st.flexBasis = lval })
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
	case "box-shadow":
		if lval == "none" {
			return op(func(st *cssState, s, ps *styles.Style) { s.BoxShadow = nil })
		}
		items := splitTopLevel(val, ',')
		return op(func(st *cssState, s, ps *styles.Style) {
			s.BoxShadow = nil
			for _, item := range items {
				if sh, ok := boxShadow(st.env, item); ok {
					s.BoxShadow = append(s.BoxShadow, sh)
				}
			}
			s.BoxShadowOutside = true // they take no space, as in CSS
		})
	case "grid-template-columns":
		cols := gridColumns(lval)
		return op(func(st *cssState, s, ps *styles.Style) { s.Columns = cols })
	case "object-fit":
		fit, ok := map[string]styles.ObjectFits{"fill": styles.FitFill, "contain": styles.FitContain, "cover": styles.FitCover,
			"none": styles.FitNone, "scale-down": styles.FitScaleDown}[lval]
		if !ok {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.ObjectFit = fit })
	case "opacity":
		f, err := strconv.ParseFloat(lval, 32)
		if err != nil {
			return nil
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Opacity = float32(f) })
	case "pointer-events":
		none := lval == "none"
		return op(func(st *cssState, s, ps *styles.Style) { s.SetAbilities(none, abilities.NoPointerEvents) })
	case "visibility":
		// (applied with the inherited visibility; see resolveCSS)
		return op(func(st *cssState, s, ps *styles.Style) { st.visibility = lval })
	}
	return nil
}

// zeroAreaClip returns whether the given value of clip (a rect) or of
// clip-path (an inset) clips the element to nothing, as in the visually
// hidden elements of pages (rect(0 0 0 0), rect(1px, 1px, 1px, 1px),
// inset(50%)).
func zeroAreaClip(prop, v string) bool {
	fn := "rect("
	if prop == "clip-path" {
		if strings.HasPrefix(v, "rect(") { // as clip (as rect(0 0 0 0) of the .sr-only of GitHub)
			return zeroAreaClip("clip", v)
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
	env := defaultLengthEnv()
	var px [4]float32
	for i, x := range f {
		if x == "auto" {
			return false
		}
		l, ok := env.lengthPx(x)
		if !ok {
			return false
		}
		px[i] = l
	}
	return px[2] <= px[0] || px[1] <= px[3]
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

// parseFlex returns the grow, shrink, and basis of the given value of the
// flex shorthand, as in CSS, where a number alone (as flex: 1) has a basis
// of 0, not of the width of the item.
func parseFlex(v string) (grow, shrink float32, basis string, ok bool) {
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

// webkitAliases are the properties whose -webkit- prefixed names are
// aliases of them in browsers, which pages may set only with the prefix.
var webkitAliases = map[string]bool{
	"transform": true, "transform-origin": true, "box-sizing": true, "box-shadow": true,
	"border-radius": true, "flex": true, "flex-direction": true, "flex-wrap": true, "flex-flow": true,
	"flex-grow": true, "flex-shrink": true, "flex-basis": true, "order": true, "align-items": true,
	"align-self": true, "justify-content": true, "user-select": true, "appearance": true,
}

// inheritedProperties are the properties that are inherited in CSS, for
// which unset is inherit (and initial for the others).
var inheritedProperties = map[string]bool{
	"color": true, "font": true, "font-family": true, "font-size": true, "font-style": true,
	"font-weight": true, "font-variant": true, "line-height": true, "letter-spacing": true,
	"word-spacing": true, "text-align": true, "text-indent": true, "text-transform": true,
	"text-shadow": true, "white-space": true, "visibility": true, "cursor": true, "direction": true, "pointer-events": true,
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
// cssAspectRatio returns the ratio of width to height of the given value
// of the aspect-ratio property (as 16 / 9, 1.5, or auto 4/3), or 0 for
// none (auto).
func cssAspectRatio(v string) float32 {
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

// cssZeroScale returns whether the given transform scales the element
// to nothing in either dimension (as scaleX(0), for a progress bar
// that is not started), which is then not shown.
func cssZeroScale(v string) bool {
	for rest := v; rest != ""; {
		name, args, found := strings.Cut(rest, "(")
		if !found {
			break
		}
		end := strings.Index(args, ")")
		if end < 0 {
			break
		}
		as := strings.Split(args[:end], ",")
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
		rest = strings.TrimSpace(args[end+1:])
	}
	return false
}

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

// boxShadow returns the shadow of the given item of a box-shadow value:
// [inset] offset-x offset-y [blur [spread]] [color], and whether it is
// one (an inset shadow is not supported, and is not rendered).
func boxShadow(env *lengthEnv, item string) (styles.Shadow, bool) {
	var sh styles.Shadow
	var lens []units.Value
	clr := ""
	for _, f := range splitColors(item) {
		if strings.EqualFold(f, "inset") {
			return sh, false
		}
		if l, ok := env.length(f, units.UnitDp); ok && len(lens) < 4 && (f == "0" || strings.IndexAny(f, "0123456789") >= 0) && parseColor(f) == nil {
			lens = append(lens, l)
			continue
		}
		clr = f
	}
	if len(lens) < 2 {
		return sh, false
	}
	sh.OffsetX, sh.OffsetY = lens[0], lens[1]
	if len(lens) > 2 {
		sh.Blur = lens[2]
	}
	if len(lens) > 3 {
		sh.Spread = lens[3]
	}
	sh.Color = colors.Uniform(colors.Black)
	if c := parseColor(clr); c != nil {
		sh.Color = c
	}
	return sh, true
}
