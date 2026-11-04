// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	_ "embed"
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"golang.org/x/net/html"
)

// browserStyles are the user agent styles for [Context.BrowserStyle],
// which follow those of web browsers.
//
//go:embed browser.css
var browserStyles string

// lineHeightNormal is the line height of the "normal" value, which
// is given by the metrics of the font in browsers; this is the value
// for the default Noto Sans font.
const lineHeightNormal = 1.362

// blockState is the state of a widget for laying out blocks
// in [Context.BrowserStyle], mainly for collapsing margins.
type blockState struct {
	// display is the CSS display of the element.
	display string

	// block is whether the widget is block-level.
	block bool

	// alignItems is the CSS align-items of a flex container.
	alignItems string

	// topBase is the total margin directly above the top border
	// edge of the widget (including its own top margin), which the
	// top margin of its first child collapses with, if openTop.
	topBase float32

	// bottomBase is the total margin directly below the bottom border
	// edge of the widget (including its own bottom margin), which the
	// bottom margin of its last child collapses with, if openBottom.
	bottomBase float32

	// borderCollapse is whether a table has collapsed borders.
	borderCollapse bool

	// legacyCenter is whether the element is centered by an align attribute.
	legacyCenter bool

	// markers is whether a list has markers, which are in the
	// space of its left padding.
	markers bool

	// openTop and openBottom are whether the margins of the first
	// and last children collapse with those of the widget, when it
	// has no padding or border there.
	openTop, openBottom bool
}

// cssState has the values of a widget's declarations that are
// resolved after all of them have been read.
type cssState struct {
	display    string
	width      *units.Value
	height     *units.Value
	minWidth   *units.Value
	minHeight  *units.Value
	maxWidth   *units.Value
	maxHeight  *units.Value
	margin     [4]*units.Value // top, right, bottom, left; nil for auto
	marginAuto [4]bool
	lineHeight string
	borderBox  bool
	flexGrow   float32
	hasGrow    bool
	alignSelf  string
	alignItems string
	background image.Image
	hasBg      bool

	borderCollapse bool
	legacyCenter   bool
	verticalAlign  string
}

// browserStyler adds the styler for the CSS declarations of the given node
// to the given widget, for [Context.BrowserStyle].
func (c *Context) browserStyler(node *html.Node, w core.Widget, decls []*cssDecl) {
	wb := w.AsWidget()
	tx, isText := w.(*core.Text)
	_, isFrame := w.(*core.Frame)
	if isText {
		// text has its one-line width until it needs to wrap,
		// for the shrink-to-fit widths of CSS
		tx.SetFitWidth(true)
	}
	plain := isText || isFrame // widgets whose look is all from CSS
	tag := ""
	if node != nil {
		tag = node.Data
	}
	if c.blocks == nil {
		c.blocks = map[core.Widget]*blockState{}
	}
	bs := &blockState{}
	c.blocks[w] = bs
	if tag == "ul" || tag == "ol" || tag == "menu" || tag == "dir" {
		ls := c.declaration(node, "list-style-type") + " " + c.declaration(node, "list-style")
		bs.markers = !strings.Contains(ls, "none")
	}
	var trBg image.Image // background of a table row, for its cells
	if (tag == "td" || tag == "th") && node.Parent != nil && node.Parent.Data == "tr" {
		if bg := c.declaration(node.Parent, "background-color"); bg != "" {
			trBg = parseColor(bg)
		}
	}
	wb.Styler(func(s *styles.Style) {
		pw, _ := wb.Parent.(core.Widget)
		var ps *styles.Style
		var pbs *blockState
		if pw != nil {
			ps = &pw.AsWidget().Styles
			pbs = c.blocks[pw]
		}
		fsz := core.AppearanceSettings.FontSize / 100
		if plain {
			// Only the inherited and CSS styles apply,
			// not the Cogent Core styles of the widget.
			if ps != nil {
				s.Text.LineHeight = ps.Text.LineHeight
				s.Font.Size = ps.Font.Size
				s.Font.Size.Value /= fsz
				s.Font.Weight = ps.Font.Weight
				s.Font.Slant = ps.Font.Slant
				s.Font.Family = ps.Font.Family
				s.Font.CustomFont = ps.Font.CustomFont
			}
			s.Gap.Zero()
			s.Padding.Zero()
			s.Margin.Zero()
			s.Grow.Set(0, 0)
			s.Overflow.Set(styles.OverflowVisible)
			if isText {
				s.Text.WhiteSpace = text.WrapAsNeeded
			}
		}
		if tag == "" && isText {
			s.Text.WhiteSpace = text.WrapAsNeeded
			if ps != nil { // white-space is inherited in CSS
				s.Text.WhiteSpace = ps.Text.WhiteSpace
			}
		}
		if tag != "" && isText { // texts for elements (as pre) fill their blocks
			s.GrowWrap = false
		}
		st := &cssState{display: c.declaration(node, "display")}
		if node == nil { // anonymous text or block
			st.display = bs.display
			if st.display == "" && isFrame {
				st.display = "block"
			}
		}
		if trBg != nil {
			st.background, st.hasBg = trBg, true
		}
		for _, d := range decls {
			c.setCSS(st, s, ps, d.Property, d.Value)
		}
		c.resolveCSS(wb, st, s, ps, bs, pbs)
		if tag == "body" && st.hasBg && pw != nil { // the canvas has the background of the body
			pw.AsWidget().Styles.Background = st.background
		}
	})
}

// display returns the display of the given widget.
func (c *Context) display(w core.Widget) string {
	if bs := c.blocks[w]; bs != nil {
		return bs.display
	}
	return ""
}

// isBlockDisplay returns whether the given display is block-level.
func isBlockDisplay(d string) bool {
	switch d {
	case "block", "list-item", "flex", "grid", "table", "flow-root":
		return true
	}
	return false
}

// resolveCSS resolves the declarations that depend on each other
// and on the parent, after all of them have been set.
func (c *Context) resolveCSS(wb *core.WidgetBase, st *cssState, s, ps *styles.Style, bs, pbs *blockState) {
	bs.display = st.display
	bs.alignItems = st.alignItems
	bs.borderCollapse = st.borderCollapse
	bs.legacyCenter = st.legacyCenter
	if st.legacyCenter {
		s.Align.Items = styles.Center
		s.Align.Content = styles.Center
	}
	if pbs != nil && pbs.legacyCenter && (st.display == "table" || st.display == "inline-table") {
		s.Text.Align = text.Start // tables do not inherit the legacy center
	}
	if st.hasBg {
		s.Background = st.background
	}
	switch st.display {
	case "block", "list-item", "table-cell", "flow-root", "inline-block":
		s.Display = styles.Flex
		s.Direction = styles.Column
	case "flex", "inline-flex":
		s.Display = styles.Flex
	case "grid", "inline-grid":
		s.Display = styles.Grid
	case "none":
		s.Display = styles.DisplayNone
	case anonymousInline:
		s.Display = styles.Flex
		s.Direction = styles.Row
		s.Wrap = true
		s.Align.Items = styles.End // close to the baseline for images
	}
	if _, isImage := wb.This.(*core.Image); isImage && pbs != nil && pbs.display == anonymousInline {
		// images are on the baseline, above the descent of the text
		if st.margin[2] == nil {
			m := units.Em(0.27)
			st.margin[2] = &m
		}
	}
	bs.block = isBlockDisplay(st.display) || st.display == anonymousInline
	if st.display == "table-cell" {
		s.Grow.Set(1, 1)
		switch st.verticalAlign {
		case "top", "text-top":
			s.Justify.Content = styles.Start
		case "bottom", "text-bottom":
			s.Justify.Content = styles.End
		default:
			s.Justify.Content = styles.Center // middle
		}
	}

	// units context for converting to dots
	uc := s.UnitContext
	if ps != nil {
		uc = ps.UnitContext
	}
	uc.FontEm = uc.ToDots(s.Font.Size.Value, s.Font.Size.Unit) * core.AppearanceSettings.FontSize / 100
	dots := func(v *units.Value) float32 {
		if v == nil {
			return 0
		}
		return uc.ToDots(v.Value, v.Unit)
	}

	// line height
	switch lh := st.lineHeight; {
	case lh == "":
	case lh == "normal":
		s.Text.LineHeight = lineHeightNormal
	default:
		if f, err := strconv.ParseFloat(lh, 32); err == nil {
			s.Text.LineHeight = float32(f)
		} else if v, ok := parseLength(lh, units.UnitEm); ok {
			if v.Unit == units.UnitEm {
				s.Text.LineHeight = v.Value / 100 // from %
				if !strings.HasSuffix(lh, "%") {
					s.Text.LineHeight = v.Value
				}
			} else if uc.FontEm > 0 {
				s.Text.LineHeight = dots(&v) / uc.FontEm
			}
		}
	}

	// block-level elements fill the width of a block container,
	// and flex items are stretched in the cross dimension
	parentDisplay := ""
	if ps != nil && pbs != nil {
		parentDisplay = pbs.display
	}
	switch {
	case parentDisplay == "flex" || parentDisplay == "inline-flex":
		main := ps.Direction.Dim()
		cross := main.Other()
		if st.hasGrow {
			s.Grow.SetDim(main, st.flexGrow)
		}
		align := st.alignSelf
		if align == "" || align == "auto" {
			align = pbs.alignItems
		}
		if align == "" || align == "stretch" || align == "normal" {
			if (cross == 0 && st.width == nil) || (cross == 1 && st.height == nil) {
				s.Grow.SetDim(cross, 1)
			}
		}
	case bs.block && st.display != "table":
		s.Grow.X = 1
	}

	// sizes: Min and Max are the size of the content box
	pad := sideDots(&s.Padding, dots)
	if bs.markers { // the markers of the items take the space of the padding
		mw := dots(&markerWidth)
		pl := max(pad.Left-mw, 0)
		s.Padding.Left = units.Dot(pl)
		pad.Left = pl
	}
	bw := sideDots(&s.Border.Width, dots)
	dp := max(uc.ToDots(1, units.UnitDp), 0.01)
	spaceX := (pad.Left + pad.Right + bw.Left + bw.Right) / dp
	spaceY := (pad.Top + pad.Bottom + bw.Top + bw.Bottom) / dp
	contentBox := func(v units.Value, space float32) (min, max units.Value) {
		min, max = v, v
		if v.Unit != units.UnitDp {
			return
		}
		if st.borderBox {
			min.Value -= space
			max.Value -= space
		}
		return
	}
	if st.width != nil {
		s.Min.X, s.Max.X = contentBox(*st.width, spaceX)
		s.Grow.X = 0
	}
	if st.height != nil {
		s.Min.Y, s.Max.Y = contentBox(*st.height, spaceY)
		s.Grow.Y = 0
	}
	if st.minWidth != nil {
		s.Min.X, _ = contentBox(*st.minWidth, spaceX)
	}
	if st.minHeight != nil {
		s.Min.Y, _ = contentBox(*st.minHeight, spaceY)
	}
	if st.maxWidth != nil {
		_, s.Max.X = contentBox(*st.maxWidth, spaceX)
	}
	if st.maxHeight != nil {
		_, s.Max.Y = contentBox(*st.maxHeight, spaceY)
	}

	// Text is only as wide as its lines, so it is aligned
	// within its block for text-align, as are inline rows.
	if _, isText := wb.This.(*core.Text); isText && bs.block {
		switch s.Text.Align {
		case text.Center:
			s.Align.Self = styles.Center
		case text.End:
			s.Align.Self = styles.End
		}
	}
	if st.display == anonymousInline {
		switch s.Text.Align {
		case text.Center:
			s.Justify.Content = styles.Center
		case text.End:
			s.Justify.Content = styles.End
		}
	}

	// border-collapse is approximated by the cells drawing their right
	// and bottom borders, and the table the top and left ones
	if st.borderCollapse {
		s.Gap.Zero()
	} else if st.display == "table" && s.Padding.Top.Value == 0 && s.Padding.Left.Value == 0 {
		// border-spacing is also between the cells and the table
		s.Padding.Set(s.Gap.Y, s.Gap.X)
	}
	if pbs != nil && pbs.borderCollapse && st.display == "table-cell" {
		if ps.Border.Width.Top.Value == 0 && ps.Border.Width.Left.Value == 0 {
			ps.Border.Width.Top = s.Border.Width.Top
			ps.Border.Width.Left = s.Border.Width.Left
			ps.Border.Color.Top = s.Border.Color.Top
			ps.Border.Color.Left = s.Border.Color.Left
			ps.Border.Style.Top = s.Border.Style.Top
			ps.Border.Style.Left = s.Border.Style.Left
			ps.Border.Width.ToDots(&ps.UnitContext)
		}
		s.Border.Width.Top.Value = 0
		s.Border.Width.Left.Value = 0
	}

	// margins: auto horizontal margins align a block that does not fill
	// the width, which it then does not grow to fill
	if st.marginAuto[1] || st.marginAuto[3] {
		if st.marginAuto[1] && st.marginAuto[3] {
			s.Align.Self = styles.Center
		} else if st.marginAuto[3] {
			s.Align.Self = styles.End
		}
		// A block of a fixed width grows to its width (its Max), so that
		// it is aligned within the width of its parent.
		if st.width != nil && st.width.Unit == units.UnitDp && bs.block && parentDisplay != "flex" && parentDisplay != "inline-flex" {
			s.Grow.X = 1
		}
	}
	// align-items of a flex container on one line aligns the line
	if (st.display == "flex" || st.display == "inline-flex") && !s.Wrap {
		switch s.Align.Items {
		case styles.Center, styles.End:
			s.Align.Content = s.Align.Items
		}
	}
	if st.margin[1] != nil {
		s.Margin.Right = *st.margin[1]
	}
	if st.margin[3] != nil {
		s.Margin.Left = *st.margin[3]
	}
	spc := core.AppearanceSettings.Spacing / 100
	if spc <= 0 {
		spc = 1
	}
	top, bottom := dots(st.margin[0]), dots(st.margin[2])
	c.collapseMargins(wb, s, ps, bs, pbs, pad, bw, &top, &bottom)
	s.Margin.Top = units.Dot(top / spc)
	s.Margin.Bottom = units.Dot(bottom / spc)
}

// sideDots returns the given sides in dots, using the given function.
func sideDots(v *sides.Values, dots func(v *units.Value) float32) sides.Floats {
	return sides.NewFloats(dots(&v.Top), dots(&v.Right), dots(&v.Bottom), dots(&v.Left))
}

// collapseMargins collapses the given top and bottom margins (in dots) of
// the given block widget with the adjoining margins of its previous
// sibling and its parent, as browsers do. The margins of the widgets stay
// within their parents, but the space between them is that of the
// collapsed margins.
func (c *Context) collapseMargins(wb *core.WidgetBase, s, ps *styles.Style, bs, pbs *blockState, pad, bw sides.Floats, top, bottom *float32) {
	flow := s.Display == styles.Flex && s.Direction == styles.Column
	bs.openTop = flow && pad.Top == 0 && bw.Top == 0
	bs.openBottom = flow && pad.Bottom == 0 && bw.Bottom == 0 && s.Min.Y.Value == 0
	bs.topBase, bs.bottomBase = *top, *bottom
	if pbs == nil || !bs.block || !(pbs.display == "block" || pbs.display == "list-item" || pbs.display == "table-cell" || pbs.display == "inline-block") {
		return
	}
	pw := wb.Parent.(core.Widget)
	prev, isLast := c.blockSiblings(pw, wb.This.(core.Widget))

	adj := float32(0)
	switch {
	case prev != nil:
		adj = c.chainBelow(prev)
	case pbs.openTop:
		adj = pbs.topBase
	}
	if *top >= 0 && adj > 0 {
		*top = max(0, *top-adj)
	}
	bs.topBase = adj + *top

	if isLast && pbs.openBottom {
		base := pbs.bottomBase
		if *bottom >= 0 && base > 0 {
			*bottom = max(0, *bottom-base)
		}
		bs.bottomBase = base + *bottom
	}
}

// blockSiblings returns the previous block sibling of the given widget,
// and whether it is the last child of the parent.
func (c *Context) blockSiblings(pw, w core.Widget) (prev core.Widget, isLast bool) {
	kids := pw.AsTree().Children
	idx := -1
	for i, k := range kids {
		if k == w {
			idx = i
			break
		}
	}
	for i := idx - 1; i >= 0; i-- {
		kw, ok := kids[i].(core.Widget)
		if !ok {
			continue
		}
		if kbs := c.blocks[kw]; kbs != nil && kbs.block && kw.AsWidget().Styles.Display != styles.DisplayNone {
			prev = kw
			break
		}
	}
	return prev, idx == len(kids)-1
}

// chainBelow returns the total of the collapsed margins below the given
// block widget, through its last descendants.
func (c *Context) chainBelow(w core.Widget) float32 {
	bs := c.blocks[w]
	total := bs.bottomBase
	for bs != nil && bs.openBottom {
		kids := w.AsTree().Children
		if len(kids) == 0 {
			break
		}
		kw, ok := kids[len(kids)-1].(core.Widget)
		if !ok {
			break
		}
		kbs := c.blocks[kw]
		if kbs == nil || !kbs.block {
			break
		}
		w, bs = kw, kbs
		total = bs.bottomBase
	}
	return total
}

// setCSS sets the given CSS property to the given value.
func (c *Context) setCSS(st *cssState, s, ps *styles.Style, prop, val string) {
	val = strings.TrimSpace(val)
	lval := strings.ToLower(val)
	if lval == "inherit" || lval == "initial" || lval == "unset" {
		s.FromProperty(ps, prop, lval, colors.BaseContext(colors.ToUniform(s.Color)))
		return
	}
	switch prop {
	case "display":
		st.display = lval
	case "margin":
		vals := expandSides(strings.Fields(lval))
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
		vals := expandSides(strings.Fields(lval))
		for i, v := range vals {
			if l, ok := parseLength(v, units.UnitPw); ok {
				setValueSide(&s.Padding, i, l)
			}
		}
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		if l, ok := parseLength(lval, units.UnitPw); ok {
			setValueSide(&s.Padding, sideIndex(prop), l)
		}
	case "border":
		setBorder(s, -1, val)
	case "border-top", "border-right", "border-bottom", "border-left":
		setBorder(s, sideIndex(prop), val)
	case "border-width":
		for i, v := range expandSides(strings.Fields(lval)) {
			setValueSide(&s.Border.Width, i, borderWidth(v))
		}
	case "border-style":
		for i, v := range expandSides(strings.Fields(lval)) {
			setBorderStyle(s, i, v)
		}
	case "border-color":
		for i, v := range expandSides(splitColors(val)) {
			setSide(&s.Border.Color, i, parseColor(v))
		}
	case "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		setValueSide(&s.Border.Width, sideIndex(prop), borderWidth(lval))
	case "border-top-style", "border-right-style", "border-bottom-style", "border-left-style":
		setBorderStyle(s, sideIndex(prop), lval)
	case "border-top-color", "border-right-color", "border-bottom-color", "border-left-color":
		setSide(&s.Border.Color, sideIndex(prop), parseColor(val))
	case "border-radius":
		vals := strings.Fields(strings.Split(lval, "/")[0])
		for i, v := range expandSides(vals) {
			if l, ok := parseLength(v, units.UnitEw); ok {
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
		st.width = lengthPtr(lval, units.UnitPw)
	case "height":
		st.height = lengthPtr(lval, units.UnitPh)
	case "min-width":
		st.minWidth = lengthPtr(lval, units.UnitPw)
	case "min-height":
		st.minHeight = lengthPtr(lval, units.UnitPh)
	case "max-width":
		st.maxWidth = lengthPtr(lval, units.UnitPw)
	case "max-height":
		st.maxHeight = lengthPtr(lval, units.UnitPh)
	case "vertical-align":
		st.verticalAlign = lval
	case "border-collapse":
		st.borderCollapse = lval == "collapse"
	case "box-sizing":
		st.borderBox = lval == "border-box"
	case "font-size":
		setFontSize(s, ps, lval)
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
		if l, ok := parseLength(lval, units.UnitPw); ok {
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
	case "flex":
		switch lval {
		case "none":
			st.flexGrow, st.hasGrow = 0, true
		case "auto":
			st.flexGrow, st.hasGrow = 1, true
		default:
			if fs := strings.Fields(lval); len(fs) > 0 {
				if f, err := strconv.ParseFloat(fs[0], 32); err == nil {
					st.flexGrow, st.hasGrow = float32(f), true
				}
			}
		}
	case "gap", "grid-gap":
		fs := strings.Fields(lval)
		if len(fs) > 0 {
			if l, ok := parseLength(fs[0], units.UnitPh); ok {
				s.Gap.Y = l
				s.Gap.X = l
			}
		}
		if len(fs) > 1 {
			if l, ok := parseLength(fs[1], units.UnitPw); ok {
				s.Gap.X = l
			}
		}
	case "row-gap":
		if l, ok := parseLength(lval, units.UnitPh); ok {
			s.Gap.Y = l
		}
	case "column-gap":
		if l, ok := parseLength(lval, units.UnitPw); ok {
			s.Gap.X = l
		}
	case "border-spacing":
		fs := strings.Fields(lval)
		if len(fs) > 0 {
			if l, ok := parseLength(fs[0], units.UnitPw); ok {
				s.Gap.Set(l)
			}
		}
		if len(fs) > 1 {
			if l, ok := parseLength(fs[1], units.UnitPh); ok {
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
	case "overflow":
		switch lval {
		case "hidden", "clip":
			s.Overflow.Set(styles.OverflowHidden)
		case "auto", "scroll":
			s.Overflow.Set(styles.OverflowAuto)
		default:
			s.Overflow.Set(styles.OverflowVisible)
		}
	case "overflow-x", "overflow-y":
		ov := styles.OverflowVisible
		switch lval {
		case "hidden", "clip":
			ov = styles.OverflowHidden
		case "auto", "scroll":
			ov = styles.OverflowAuto
		}
		if prop == "overflow-x" {
			s.Overflow.X = ov
		} else {
			s.Overflow.Y = ov
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

func setMargin(st *cssState, i int, v string) {
	if v == "auto" {
		st.marginAuto[i] = true
		st.margin[i] = nil
		return
	}
	st.marginAuto[i] = false
	if l, ok := parseLength(v, units.UnitPw); ok {
		st.margin[i] = &l
	}
}

// parseLength parses the given CSS length, with px as dp (which is
// what px is in browsers), % as the given unit, and lengths without
// units, as in the attributes of HTML elements, as dp.
func parseLength(v string, pct units.Units) (units.Value, bool) {
	v = strings.TrimSpace(v)
	switch v {
	case "", "auto", "none", "normal", "inherit", "initial":
		return units.Value{}, false
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
	if strings.HasSuffix(v, "px") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32)
		if err != nil {
			return units.Value{}, false
		}
		return units.Dp(float32(f)), true
	}
	var l units.Value
	if err := l.SetString(v); err != nil {
		return units.Value{}, false
	}
	return l, true
}

func lengthPtr(v string, pct units.Units) *units.Value {
	l, ok := parseLength(v, pct)
	if !ok {
		return nil
	}
	return &l
}

// borderWidth returns the given CSS border width.
func borderWidth(v string) units.Value {
	switch v {
	case "thin":
		return units.Dp(1)
	case "medium":
		return units.Dp(3)
	case "thick":
		return units.Dp(5)
	}
	l, _ := parseLength(v, units.UnitDp)
	return l
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
func setBorder(s *styles.Style, side int, val string) {
	width := units.Dp(3) // medium
	style := "none"
	var color image.Image = colors.Uniform(colors.ToUniform(s.Color))
	for _, f := range splitColors(val) {
		lf := strings.ToLower(f)
		switch {
		case borderStyleNames[lf]:
			style = lf
		case lf == "thin" || lf == "medium" || lf == "thick" || (len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.')):
			width = borderWidth(lf)
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
// that are not inside of parentheses, as in rgb(...).
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

// backgroundColor returns the color of the given background value.
func backgroundColor(v string) string {
	for _, f := range splitColors(v) {
		lf := strings.ToLower(f)
		if strings.HasPrefix(lf, "url(") || strings.Contains(lf, "gradient(") {
			continue
		}
		if lf == "transparent" || lf == "none" {
			return lf
		}
		if parseColor(f) != nil {
			return f
		}
	}
	return ""
}

// parseColor returns the given CSS color, or nil if it is not one.
func parseColor(v string) image.Image {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if strings.EqualFold(v, "transparent") {
		return colors.Uniform(colors.Transparent)
	}
	if !strings.HasPrefix(v, "#") && !strings.Contains(v, "(") {
		// legacy colors without a #, as in bgcolor="ffffff"
		if len(v) == 6 || len(v) == 3 {
			if _, err := strconv.ParseUint(v, 16, 32); err == nil {
				v = "#" + v
			}
		}
	}
	cl, err := colors.FromString(v)
	if err != nil {
		img, err := gradient.FromString(v)
		if err != nil {
			return nil
		}
		return img
	}
	return colors.Uniform(cl)
}

// cssFontSizes are the absolute font size keywords, in px.
var cssFontSizes = map[string]float32{
	"xx-small": 9, "x-small": 10, "small": 13, "medium": 16, "large": 18,
	"x-large": 24, "xx-large": 32, "xxx-large": 48,
}

// setFontSize sets the font size, relative to the inherited one.
func setFontSize(s, ps *styles.Style, v string) {
	inherited := s.Font.Size
	if ps != nil {
		inherited = ps.Font.Size
		inherited.Value /= core.AppearanceSettings.FontSize / 100
	}
	scale := func(f float32) {
		s.Font.Size = units.Value{Value: inherited.Value * f, Unit: inherited.Unit}
	}
	if px, ok := cssFontSizes[v]; ok {
		s.Font.Size = units.Dp(px)
		return
	}
	switch v {
	case "smaller":
		scale(1 / 1.2)
		return
	case "larger":
		scale(1.2)
		return
	}
	if strings.HasSuffix(v, "%") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "%"), 32); err == nil {
			scale(float32(f) / 100)
		}
		return
	}
	if strings.HasSuffix(v, "rem") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "rem"), 32); err == nil {
			s.Font.Size = units.Dp(16 * float32(f))
		}
		return
	}
	if strings.HasSuffix(v, "em") {
		if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "em"), 32); err == nil {
			scale(float32(f))
		}
		return
	}
	if l, ok := parseLength(v, units.UnitDp); ok {
		s.Font.Size = l
	}
}

// setFontWeight sets the font weight.
func setFontWeight(s, ps *styles.Style, v string) {
	w := 400
	switch v {
	case "normal":
	case "bold":
		w = 700
	case "bolder":
		w = 700
		if ps != nil && ps.Font.Weight >= rich.Bold {
			w = 900
		}
	case "lighter":
		w = 300
	default:
		n, err := strconv.Atoi(v)
		if err != nil {
			return
		}
		w = n
	}
	s.Font.Weight = rich.Weights(min(max((w+50)/100-1, 0), 8))
}

// fontFamily returns the font family for the given CSS font-family list,
// the first generic family in it, or one for a common font name.
func fontFamily(v string) rich.Family {
	for _, f := range strings.Split(v, ",") {
		f = strings.ToLower(strings.Trim(strings.TrimSpace(f), `"'`))
		switch f {
		case "monospace", "courier", "courier new", "consolas", "menlo", "monaco", "roboto mono", "ui-monospace", "sfmono-regular":
			return rich.Monospace
		case "serif", "times", "times new roman", "georgia", "garamond", "ui-serif":
			return rich.Serif
		case "sans-serif", "arial", "helvetica", "helvetica neue", "verdana", "tahoma", "system-ui", "-apple-system", "segoe ui", "roboto", "noto sans", "ui-sans-serif":
			return rich.SansSerif
		case "cursive":
			return rich.Cursive
		case "fantasy":
			return rich.Fantasy
		}
	}
	return rich.SansSerif
}

// setFont sets the font shorthand: [style] [weight] size[/line-height] family.
func setFont(st *cssState, s, ps *styles.Style, v string) {
	fs := strings.Fields(v)
	for i, f := range fs {
		lf := strings.ToLower(f)
		switch {
		case lf == "italic" || lf == "oblique":
			s.Font.Slant = rich.Italic
		case lf == "bold" || lf == "bolder" || lf == "lighter" || (len(lf) == 3 && lf[1] == '0' && lf[2] == '0'):
			setFontWeight(s, ps, lf)
		case lf == "normal" || lf == "small-caps":
		case len(lf) > 0 && (lf[0] >= '0' && lf[0] <= '9' || lf[0] == '.') || cssFontSizes[lf] > 0:
			size, lh, _ := strings.Cut(lf, "/")
			setFontSize(s, ps, size)
			if lh != "" {
				st.lineHeight = lh
			}
			s.Font.Family = fontFamily(strings.Join(fs[i+1:], " "))
			return
		}
	}
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

// addHints adds declarations for the presentational attributes of the
// given element, such as the width of an img or the bgcolor of a td.
func (c *Context) addHints(n *html.Node) {
	for _, a := range n.Attr {
		v := strings.TrimSpace(a.Val)
		switch a.Key {
		case "width", "height":
			switch n.Data {
			case "img", "table", "td", "th", "col", "iframe", "video", "canvas", "embed", "object", "hr", "svg":
				c.addHint(n, a.Key, v)
			}
		case "bgcolor":
			c.addHint(n, "background-color", v)
		case "color":
			if n.Data == "font" {
				c.addHint(n, "color", v)
			}
		case "face":
			if n.Data == "font" {
				c.addHint(n, "font-family", v)
			}
		case "size":
			if n.Data == "font" {
				if sz, ok := fontElementSize(v); ok {
					c.addHint(n, "font-size", sz)
				}
			}
		case "valign":
			c.addHint(n, "vertical-align", strings.ToLower(v))
		case "text":
			if n.Data == "body" {
				c.addHint(n, "color", v)
			}
		case "align":
			switch strings.ToLower(v) {
			case "center", "middle":
				if n.Data == "table" {
					c.addHint(n, "margin-left", "auto")
					c.addHint(n, "margin-right", "auto")
				} else if n.Data != "img" {
					c.addHint(n, "text-align", "-webkit-center")
				}
			case "left", "right", "justify":
				if n.Data != "img" && n.Data != "table" {
					c.addHint(n, "text-align", strings.ToLower(v))
				}
			}
		case "hidden":
			c.addHint(n, "display", "none")
		case "nowrap":
			c.addHint(n, "white-space", "nowrap")
		case "border":
			if n.Data == "table" {
				if bw, err := strconv.Atoi(v); err == nil || v == "" {
					if v == "" {
						bw = 1
					}
					if bw > 0 {
						c.addHint(n, "border", strconv.Itoa(bw)+"px outset #808080")
					}
				}
			} else if n.Data == "img" {
				c.addHint(n, "border-width", v)
			}
		case "cellspacing":
			if n.Data == "table" {
				c.addHint(n, "border-spacing", v)
			}
		}
	}
	// cells get the cellpadding and border of their table
	if n.Data == "td" || n.Data == "th" {
		if t := ancestorTable(n); t != nil {
			if cp := GetAttr(t, "cellpadding"); cp != "" {
				c.addHint(n, "padding", cp)
			}
			if b := GetAttr(t, "border"); HasAttr(t, "border") && b != "0" {
				c.addHint(n, "border", "1px inset #808080")
			}
		}
	}
}

// fontElementSize returns the CSS font size for the size attribute of a font element.
func fontElementSize(v string) (string, bool) {
	sizes := []string{"x-small", "small", "medium", "large", "x-large", "xx-large", "xxx-large"}
	n, err := strconv.Atoi(strings.TrimLeft(v, "+-"))
	if err != nil {
		return "", false
	}
	switch {
	case strings.HasPrefix(v, "+"):
		n += 3
	case strings.HasPrefix(v, "-"):
		n = 3 - n
	}
	return sizes[min(max(n, 1), 7)-1], true
}

// ancestorTable returns the table element that the given cell is in.
func ancestorTable(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Data == "table" {
			return p
		}
	}
	return nil
}
