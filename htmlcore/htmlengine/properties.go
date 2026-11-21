// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlengine

import (
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/htmlcore/css"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// styleOp applies a compiled CSS declaration to the given style, with the
// state of the style pass (as its length environment) and the parent style.
type styleOp func(st *cssState, s, ps *styles.Style)

// compileDecls compiles the given declarations, in the order of the cascade
// (see [Context.compileCSS]). One that is invalid, of a var() that has no
// value (invalid at computed-value time), is unset, as in CSS: inherited,
// or initial (as the headings of the rules of reddit.com, of a font of a
// variable of a theme that it did not use, which were of the font of h2,
// not of the text that they were in).
// The font properties are first, as in CSS the font size of an element
// is computed before the lengths that are relative to it (as the ch unit,
// and em in calc()).
func (c *Context) compileDecls(decls []*cssDecl) []styleOp {
	var ops []styleOp
	for _, font := range []bool{true, false} {
		for _, d := range decls {
			if css.FontProperties[d.Property] != font {
				continue
			}
			v := d.Value
			if d.invalid {
				v = "unset"
			}
			ops = append(ops, c.compileCSS(d.Property, v)...)
		}
	}
	return ops
}

// compileCSS compiles the declaration of the given CSS property with the
// given value into the operations that apply it to a style. The parts of
// the value that are the same for every element, as keywords, numbers and
// colors, are parsed once here; lengths are parsed when it is applied, as
// they depend on the length environment of the style pass (as for em, rem
// and calc()). It returns nil for a property that is not supported, or
// a value that is not valid.
func (c *Context) compileCSS(prop, val string) []styleOp {
	if p, ok := strings.CutPrefix(prop, "-webkit-"); ok && css.WebkitAliases[p] {
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
		if css.InheritedProperties[prop] {
			lval = "inherit"
		}
	}
	if ls, ok := css.KeywordLonghands[prop]; ok && (lval == "inherit" || lval == "initial") {
		// a shorthand is its longhands (as the font of a var() that has no
		// value, which is unset, inherited)
		var ops []styleOp
		for _, l := range ls {
			ops = append(ops, c.compileCSS(l, lval)...)
		}
		return ops
	}
	if prop == "clip" || prop == "clip-path" {
		// a clip to nothing hides the element (as the visually hidden
		// skip links of go.dev, of clip: rect(0 0 0 0), unset on focus);
		// other clips are not applied
		zero := css.ZeroAreaClip(prop, lval)
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
	if lval == "initial" && (prop == "margin-top" || prop == "margin-right" || prop == "margin-bottom" || prop == "margin-left") {
		// the margins of the cascade are those of the state (see setMargin)
		i := css.SideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { setMargin(st, i, "0") })
	}
	if lval == "inherit" || lval == "initial" {
		return op(func(st *cssState, s, ps *styles.Style) {
			s.FromProperty(ps, prop, lval, colors.BaseContext(colors.ToUniform(s.Color)))
		})
	}
	if p, ok := css.LogicalProperties[prop]; ok {
		prop = p
	}
	switch prop {
	case "margin-inline", "margin-block", "padding-inline", "padding-block":
		fs := css.SplitColors(val)
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
		d := css.LegacyDisplay(lval)
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
		vals := css.ExpandSides(css.SplitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setMargin(st, i, v)
			}
		})
	case "margin-top", "margin-right", "margin-bottom", "margin-left":
		i := css.SideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { setMargin(st, i, lval) })
	case "padding":
		vals := css.ExpandSides(css.SplitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				if l, ok := st.env.Length(v, units.UnitPw); ok {
					css.SetValueSide(&s.Padding, i, l)
				}
			}
		})
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		i := css.SideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.Length(lval, units.UnitPw); ok {
				css.SetValueSide(&s.Padding, i, l)
			}
		})
	case "border":
		return op(compileBorder(-1, val))
	case "border-top", "border-right", "border-bottom", "border-left":
		return op(compileBorder(css.SideIndex(prop), val))
	case "border-width":
		vals := css.ExpandSides(css.SplitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				css.SetValueSide(&s.Border.Width, i, st.env.BorderWidth(v))
			}
		})
	case "border-style":
		vals := css.ExpandSides(css.SplitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				css.SetBorderStyle(s, i, v)
			}
		})
	case "border-color":
		var cfs []func() image.Image
		for _, v := range css.ExpandSides(css.SplitColors(val)) {
			cfs = append(cfs, css.ColorFunc(v))
		}
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, cf := range cfs {
				css.SetSide(&s.Border.Color, i, cf())
			}
		})
	case "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		i := css.SideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) {
			css.SetValueSide(&s.Border.Width, i, st.env.BorderWidth(lval))
		})
	case "border-top-style", "border-right-style", "border-bottom-style", "border-left-style":
		i := css.SideIndex(prop)
		return op(func(st *cssState, s, ps *styles.Style) { css.SetBorderStyle(s, i, lval) })
	case "border-top-color", "border-right-color", "border-bottom-color", "border-left-color":
		i, cf := css.SideIndex(prop), css.ColorFunc(val)
		return op(func(st *cssState, s, ps *styles.Style) { css.SetSide(&s.Border.Color, i, cf()) })
	case "border-top-left-radius", "border-top-right-radius", "border-bottom-right-radius", "border-bottom-left-radius":
		v := strings.Fields(lval + " 0")[0]
		i := map[string]int{"border-top-left-radius": 0, "border-top-right-radius": 1,
			"border-bottom-right-radius": 2, "border-bottom-left-radius": 3}[prop]
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.Length(v, units.UnitEw); ok {
				css.SetValueSide(&s.Border.Radius, i, l)
			}
		})
	case "border-radius":
		vals := css.ExpandSides(strings.Fields(strings.Split(lval, "/")[0]))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				if l, ok := st.env.Length(v, units.UnitEw); ok {
					css.SetValueSide(&s.Border.Radius, i, l)
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
		bg := css.BackgroundColor(val)
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
		cf := css.ColorFunc(bg)
		return append(imageOp, op(func(st *cssState, s, ps *styles.Style) {
			st.background, st.hasBg, st.bgCurrentColor = cf(), true, current
			if none || current {
				st.background = nil
			}
		})...)
	case "color":
		if css.ParseColor(val) == nil {
			return nil
		}
		cf := css.ColorFunc(val)
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
			st.setSize(prop, st.env.LengthPtr(lval, pct))
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
		i := css.SideIndex("-" + prop)
		return op(func(st *cssState, s, ps *styles.Style) { setInset(st.env, s, i, lval) })
	case "inset":
		vals := css.ExpandSides(css.SplitColors(lval))
		return op(func(st *cssState, s, ps *styles.Style) {
			for i, v := range vals {
				setInset(st.env, s, i, v)
			}
		})
	case "inset-inline", "inset-block":
		fs := css.SplitColors(lval)
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
		x, y, ok := css.Translate(prop, lval)
		if !ok {
			return nil
		}
		rot := css.Rotate(prop, lval)
		zero := prop == "transform" && css.ZeroScale(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			s.Translate.X, _ = st.env.Length(x, units.UnitEw)
			s.Translate.Y, _ = st.env.Length(y, units.UnitEh)
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
		for f := range strings.FieldsSeq(v) {
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
		for f := range strings.FieldsSeq(lval) {
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
				return st.env.LengthPtr(v, units.UnitPw)
			}
			if prop != "contain-intrinsic-height" {
				st.intrinsicWidth = length(vals[0])
			}
			if prop != "contain-intrinsic-width" {
				st.intrinsicHeight = length(vals[len(vals)-1])
			}
		})
	case "aspect-ratio":
		r := css.AspectRatio(lval)
		return op(func(st *cssState, s, ps *styles.Style) { s.AspectRatio = r })
	case "rotate":
		rot := css.Angle(lval)
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
		return op(func(st *cssState, s, ps *styles.Style) { css.SetFontWeight(s, ps, lval) })
	case "font-style":
		slant := rich.SlantNormal
		if lval == "italic" || lval == "oblique" {
			slant = rich.Italic
		}
		return op(func(st *cssState, s, ps *styles.Style) { s.Font.Slant = slant })
	case "font-family":
		family := css.FontFamily(val)
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
			if l, ok := st.env.Length(lval, units.UnitPw); ok {
				s.Text.Indent = l
				// the text is moved out of view (as the labels of icons,
				// which have it), as an indent is not otherwise supported
				st.hideText = (l.Unit == units.UnitDp || l.Unit == units.UnitPx) && l.Value <= -500
			}
		})
	case "text-decoration", "text-decoration-line":
		var deco rich.Decorations
		for f := range strings.FieldsSeq(lval) {
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
		// a direction and a wrap, in any order, each initial when omitted
		// (the wrap is not a direction: a flex-flow of column nowrap, as
		// the election results of nytimes.com, is a column)
		dir, wrap := "row", "nowrap"
		for f := range strings.FieldsSeq(lval) {
			if strings.Contains(f, "wrap") {
				wrap = f
			} else {
				dir = f
			}
		}
		return append(c.compileCSS("flex-direction", dir), c.compileCSS("flex-wrap", wrap)...)
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
		grow, shrink, basis, ok := css.ParseFlex(lval)
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
		fs := css.SplitColors(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			if len(fs) > 0 {
				if l, ok := st.env.Length(fs[0], units.UnitPh); ok {
					s.Gap.Y = l
					s.Gap.X = l
				}
			}
			if len(fs) > 1 {
				if l, ok := st.env.Length(fs[1], units.UnitPw); ok {
					s.Gap.X = l
				}
			}
		})
	case "row-gap":
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.Length(lval, units.UnitPh); ok {
				s.Gap.Y = l
			}
		})
	case "column-gap":
		return op(func(st *cssState, s, ps *styles.Style) {
			if l, ok := st.env.Length(lval, units.UnitPw); ok {
				s.Gap.X = l
			}
		})
	case "border-spacing":
		fs := css.SplitColors(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			if len(fs) > 0 {
				if l, ok := st.env.Length(fs[0], units.UnitPw); ok {
					s.Gap.Set(l)
				}
			}
			if len(fs) > 1 {
				if l, ok := st.env.Length(fs[1], units.UnitPh); ok {
					s.Gap.Y = l
				}
			}
		})
	case "justify-content", "justify-items", "justify-self", "align-content":
		a := css.AlignValue(lval)
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
		a := css.AlignValue(lval)
		return op(func(st *cssState, s, ps *styles.Style) {
			st.alignItems = lval
			s.Align.Items = a
		})
	case "align-self":
		a := css.AlignValue(lval)
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
		items := css.SplitTopLevel(val, ',')
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
		cols := css.GridColumns(lval)
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
	for _, f := range css.SplitColors(val) {
		lf := strings.ToLower(f)
		switch {
		case css.BorderStyleNames[lf]:
			style = lf
		case lf == "thin" || lf == "medium" || lf == "thick" || (len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.')):
			width = lf
		default:
			if css.ParseColor(f) != nil {
				color = css.ColorFunc(f)
			}
		}
	}
	return func(st *cssState, s, ps *styles.Style) {
		w := units.Dp(3) // medium
		if width != "" {
			w = st.env.BorderWidth(width)
		}
		cl := image.Image(colors.Uniform(colors.ToUniform(s.Color)))
		if color != nil {
			cl = color()
		}
		for i := range 4 {
			if side >= 0 && i != side {
				continue
			}
			css.SetSide(&s.Border.Color, i, cl)
			css.SetBorderStyle(s, i, style)
			if style != "none" && style != "hidden" {
				css.SetValueSide(&s.Border.Width, i, w)
			}
		}
	}
}

// setInset sets the given side (top, right, bottom, left) of the inset
// of a positioned element; auto is not set.
func setInset(env *css.LengthEnv, s *styles.Style, i int, v string) {
	pct := units.UnitPw
	if i == 0 || i == 2 {
		pct = units.UnitPh
	}
	l, ok := env.Length(v, pct)
	css.SetSide(&s.InsetSet, i, ok)
	if ok {
		css.SetValueSide(&s.Inset, i, l)
	}
}

func setMargin(st *cssState, i int, v string) {
	if v == "auto" {
		st.marginAuto[i] = true
		st.margin[i] = nil
		return
	}
	st.marginAuto[i] = false
	if l, ok := st.env.Length(v, units.UnitPw); ok {
		st.margin[i] = &l
	}
}

// boxShadow returns the shadow of the given item of a box-shadow value:
// [inset] offset-x offset-y [blur [spread]] [color], and whether it is
// one (an inset shadow is not supported, and is not rendered).
func boxShadow(env *css.LengthEnv, item string) (styles.Shadow, bool) {
	var sh styles.Shadow
	var lens []units.Value
	clr := ""
	for _, f := range css.SplitColors(item) {
		if strings.EqualFold(f, "inset") {
			return sh, false
		}
		if l, ok := env.Length(f, units.UnitDp); ok && len(lens) < 4 && (f == "0" || strings.ContainsAny(f, "0123456789")) && css.ParseColor(f) == nil {
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
	if c := css.ParseColor(clr); c != nil {
		sh.Color = c
	}
	return sh, true
}
