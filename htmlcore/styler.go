// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	_ "embed"
	"fmt"
	"image"
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"golang.org/x/net/html"
)

// browserStyles are the user agent styles, which follow those of web browsers.
//
//go:embed browser.css
var browserStyles string

// blockState is the state of a widget for laying out blocks,
// mainly for collapsing margins.
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

	// definiteHeight is whether the element has a definite height,
	// which percentage heights of its children are relative to.
	definiteHeight bool

	// openTop and openBottom are whether the margins of the first
	// and last children collapse with those of the widget, when it
	// has no padding or border there.
	openTop, openBottom bool

	// marginTop and marginBottom are the computed top and bottom margins
	// of the element in dots, before they collapse (as in the computed
	// style of a browser).
	marginTop, marginBottom float32
}

// cssState has the values of a widget's declarations that are
// resolved after all of them have been read.
type cssState struct {
	// env is the length environment of the style pass.
	env *lengthEnv

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
	flexShrink float32
	hasShrink  bool
	alignSelf  string
	alignItems string
	background image.Image
	hasBg      bool

	borderCollapse bool
	legacyCenter   bool
	verticalAlign  string

	// overflowHidden is whether the overflow in each dimension is hidden.
	overflowHidden [2]bool
}

// browserStyler adds the styler for the CSS declarations of the given node
// to the given widget.
func (c *Context) browserStyler(node *html.Node, w core.Widget, decls []*cssDecl) {
	wb := w.AsWidget()
	_, isText := w.(*core.Text)
	_, isFrame := w.(*core.Frame)
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
			if isFrame {
				// content that overflows a box is shown, unless the
				// overflow is hidden or scrolls
				s.NoClip = true
			}
			if isText {
				s.Text.WhiteSpace = text.WrapAsNeeded
			}
			if ps != nil { // white-space and text-align are inherited in CSS
				s.Text.WhiteSpace = ps.Text.WhiteSpace
				s.Text.Align = ps.Text.Align
			}
		}
		if tag == "" && isText {
			s.Text.WhiteSpace = text.WrapAsNeeded
			if ps != nil { // white-space and text-align are inherited in CSS
				s.Text.WhiteSpace = ps.Text.WhiteSpace
				s.Text.Align = ps.Text.Align
			}
		}
		if tag != "" && isText { // texts for elements (as pre) fill their blocks
			s.GrowWrap = false
		}
		st := &cssState{display: c.displayOf(node), env: defaultLengthEnv()}
		env := st.env
		if node == nil { // anonymous text or block
			st.display = bs.display
			if st.display == "" && isFrame {
				st.display = "block"
			}
		}
		if trBg != nil {
			st.background, st.hasBg = trBg, true
		}
		if ps != nil { // for calc()
			uc := &ps.UnitContext
			if dp := uc.Dots(units.UnitDp); dp > 0 {
				env.em = float64(uc.FontEm / dp)
				env.vw = float64(uc.Vpw / dp / 100)
				env.vh = float64(uc.Vph / dp / 100)
			}
		}
		env.rem = float64(c.rootFontSize())
		env.fixedVh = c.MediaHeight > 0
		if env.fixedVh {
			env.vh = float64(c.MediaHeight / 100)
		}
		if tag == "body" && c.declaration(node, "font-size") == "" {
			// the font size of the html element (which has no widget)
			if rf := c.rootFontSize(); rf != 16 {
				c.setCSS(st, s, ps, "font-size", fmt.Sprintf("%gpx", rf))
			}
		}
		for _, d := range decls {
			if !d.invalid {
				c.setCSS(st, s, ps, d.Property, d.Value)
			}
		}
		if node != nil {
			st.display = c.displayOf(node)
		}
		c.resolveCSS(wb, node, st, s, ps, bs, pbs)
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
func (c *Context) resolveCSS(wb *core.WidgetBase, node *html.Node, st *cssState, s, ps *styles.Style, bs, pbs *blockState) {
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
	isTable := node != nil && node.Data == "table" // laid out as a grid of its cells
	switch st.display {
	case "block", "list-item", "table-cell", "flow-root", "inline-block", "table-caption":
		s.Display = styles.Flex
		s.Direction = styles.Column
	case "table", "inline-table", "table-row-group", "table-header-group", "table-footer-group":
		if !isTable && (node == nil || !strings.HasPrefix(node.Data, "t")) {
			// other elements displayed as tables are approximated by their
			// rows (or caption and content) in a column
			s.Display = styles.Flex
			s.Direction = styles.Column
		}
	case "flex", "inline-flex":
		s.Display = styles.Flex
	case "grid", "inline-grid":
		s.Display = styles.Grid
		if s.Columns == 0 { // one column without column tracks, as in CSS
			s.Columns = 1
		}
	case "none":
		s.Display = styles.DisplayNone
	case anonymousInline:
		s.Display = styles.Flex
		s.Direction = styles.Row
		s.Wrap = true
		s.Align.Items = styles.End // close to the baseline for images
		// as wide as its content on one line, and wraps in less width
		s.Shrink.X = 1
	}
	if size, ok := wb.Property("svg-size").(math32.Vector2); ok {
		// an img of an svg has the size of the svg, or that of its aspect,
		// within percentage max sizes of a parent with a size
		cb := node // the containing block: the nearest block it is in
		for cb != nil && cb.Parent != nil && cb.Parent.Type == html.ElementNode {
			cb = cb.Parent
			if d := c.displayOf(cb); d != "" && d != "inline" && d != "contents" {
				break
			}
		}
		if cb != nil && cb != node && cb.Type == html.ElementNode {
			scale := float32(1)
			for i, mx := range []*units.Value{st.maxWidth, st.maxHeight} {
				if mx == nil || (mx.Unit != units.UnitPw && mx.Unit != units.UnitPh) {
					continue
				}
				prop := "width"
				if i == 1 {
					prop = "height"
				}
				if px, ok := st.env.lengthPx(c.declaration(cb, prop)); ok && px > 0 {
					scale = min(scale, px*mx.Value/100/size.Dim(math32.Dims(i)))
				}
			}
			size = size.MulScalar(scale)
		}
		if st.width == nil && st.height == nil && node != nil {
			// an absolute max-width or max-height scales it down to it,
			// keeping its aspect, as its width and height are auto
			for i, prop := range []string{"max-width", "max-height"} {
				d := math32.Dims(i)
				if px, ok := st.env.lengthPx(c.declaration(node, prop)); ok && px > 0 && px < size.Dim(d) {
					size = size.MulScalar(px / size.Dim(d))
				}
			}
		}
		switch {
		case st.width != nil && st.height != nil && st.height.Unit != units.UnitPh:
			mh := *st.height
			s.Max.Y = mh // as an image is not higher than its height
		case st.width == nil && st.height == nil:
			s.Min.Set(units.Dp(size.X), units.Dp(size.Y))
			s.Max = s.Min
		// the other dimension is of the aspect of the svg, in the same unit
		case st.height == nil && st.width.Unit != units.UnitPw:
			s.Min.Y = units.Value{Value: st.width.Value * size.Y / size.X, Unit: st.width.Unit}
			s.Max.Y = s.Min.Y
		case st.width == nil && st.height.Unit != units.UnitPh && st.height.Unit != units.UnitPw:
			s.Min.X = units.Value{Value: st.height.Value * size.X / size.Y, Unit: st.height.Unit}
			s.Max.X = s.Min.X
			mh := *st.height
			s.Max.Y = mh // as an image is not higher than its height
		}
	}
	if _, isSVG := wb.This.(*core.SVG); isSVG && node != nil && node.Data == "svg" {
		// an svg without a size is as wide as its container, and as
		// high as the aspect ratio of its view box makes it
		if st.width == nil {
			s.Min.X, s.Max.X = units.Pw(100), units.Pw(100)
		}
		if st.height == nil {
			aspect := float32(0.5)
			vb := strings.Fields(strings.ReplaceAll(GetAttr(node, "viewBox")+GetAttr(node, "viewbox"), ",", " "))
			if len(vb) == 4 {
				w, _ := strconv.ParseFloat(vb[2], 32)
				h, _ := strconv.ParseFloat(vb[3], 32)
				if w > 0 && h > 0 {
					aspect = float32(h / w)
				}
			}
			if st.width != nil && st.width.Unit == units.UnitDp {
				s.Min.Y = units.Dp(st.width.Value * aspect)
			} else {
				s.Min.Y = units.Pw(100 * aspect)
			}
			s.Max.Y = s.Min.Y
		}
	}
	if _, isImage := wb.This.(*core.Image); isImage && pbs != nil && pbs.display == anonymousInline {
		// images are on the baseline, above the descent of the text
		if st.margin[2] == nil {
			m := units.Em(0.27)
			st.margin[2] = &m
		}
	}
	if pbs != nil && s.Float != styles.FloatNone {
		switch pbs.display {
		case "flex", "inline-flex", "grid", "inline-grid":
			s.Float = styles.FloatNone // float does not apply to flex and grid items
		}
	}
	bs.block = isBlockDisplay(st.display) || st.display == anonymousInline
	outOfFlow := s.Position == styles.PositionAbsolute || s.Position == styles.PositionFixed
	if outOfFlow || s.Float != styles.FloatNone {
		bs.block = false // not in the flow of blocks, and its width fits its content
	}
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
	} else if !bs.block {
		// an inline element is aligned in the height of its line
		switch st.verticalAlign {
		case "middle":
			s.Align.Self = styles.Center
		case "top", "text-top":
			s.Align.Self = styles.Start
		case "bottom", "text-bottom":
			s.Align.Self = styles.End
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
		} else if v, ok := st.env.length(lh, units.UnitEm); ok {
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
		// flex items shrink to fit in the main dimension (by flex-shrink,
		// which is 1 by default), and are at most as wide as a column
		s.Shrink.X = 1
		if main == math32.X && st.hasShrink {
			// a flex item that does not shrink (flex-shrink: 0) is still as
			// wide as its content (on one line, for wrapped content), which
			// core lays out for those that shrink, so it shrinks only a tiny bit
			s.Shrink.X = max(st.flexShrink, 1e-4)
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
	case parentDisplay == "grid" || parentDisplay == "inline-grid":
		// grid items are stretched to their cells by default
		align := st.alignSelf
		if align == "" || align == "auto" {
			align = pbs.alignItems
		}
		if st.width == nil {
			s.Grow.X = 1
		}
		if (align == "" || align == "stretch" || align == "normal") && st.height == nil {
			s.Grow.Y = 1
		}
		s.Shrink.X = 1
	case bs.block && st.display != "table":
		s.Grow.X = 1
	case st.display == "table-cell":
		s.Shrink.X = 1 // the columns of a table fit in its width
	case st.width == nil:
		// shrink-to-fit (tables, inline blocks, cells, and inline elements):
		// as wide as the content, up to the available width
		s.Shrink.X = 1
	}
	if _, isText := wb.This.(*core.Text); isText {
		s.Shrink.X = 1 // wraps in the available width
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
	// core subtracts the padding and border of a border box (see styles.BoxSizing)
	if st.borderBox {
		s.BoxSizing = styles.BoxBorderBox
	}
	contentBox := func(v units.Value, space float32) (min, max units.Value) {
		return v, v
	}
	if st.width != nil {
		s.Min.X, s.Max.X = contentBox(*st.width, spaceX)
		s.Grow.X = 0
		_, isImage := wb.This.(*core.Image)
		if st.width.Unit == units.UnitPw && !isImage {
			// A percentage of the width of the parent fills that much
			// of it, which as a minimum would make a parent that is as
			// wide as its content grow without end.
			s.Min.X = units.Value{}
			s.Grow.X = 1
		}
	}
	// A percentage height is only of a parent with a definite height,
	// and the viewport is definite. Content that overflows a height is
	// shown in CSS, but it is clipped in core, so the height is only a
	// minimum unless the overflow is clipped anyway.
	parentDefinite := pbs == nil || pbs.definiteHeight
	if pbs == nil && node != nil && node.Parent != nil && node.Parent.Data == "html" {
		// The html element (which has no widget) has a definite height,
		// taken to be that of the viewport, only when it has a height.
		parentDefinite = c.declaration(node.Parent, "height") != ""
	}
	for _, h := range []**units.Value{&st.height, &st.minHeight, &st.maxHeight} {
		if *h == nil || (*h).Unit != units.UnitPh {
			continue
		}
		switch {
		case !parentDefinite:
			*h = nil // auto
		case pbs == nil:
			// a percentage of the viewport (the containing block of the
			// top element), not of the scene, which is as high as the page
			v := (*h).Value
			if c.MediaHeight > 0 {
				*h = new(units.Value)
				**h = units.Dp(v * c.MediaHeight / 100)
			} else {
				*h = new(units.Value)
				**h = units.Vh(v)
			}
		}
	}
	bs.definiteHeight = st.height != nil
	if (st.overflowHidden[0] || s.Overflow.X == styles.OverflowVisible) &&
		(st.overflowHidden[1] || s.Overflow.Y == styles.OverflowVisible) && (st.overflowHidden[0] || st.overflowHidden[1]) {
		s.ScrollbarWidth.Zero() // hidden, not scrolled
	}
	// Overflow only scrolls (or clips) a box whose height is limited,
	// and that of the body is that of the viewport, which scrolls anyway.
	// A box that is out of flow with its top and bottom set is as high as
	// they make it (in its containing block), so it can scroll too.
	insetHeight := (s.Position == styles.PositionAbsolute || s.Position == styles.PositionFixed) &&
		s.InsetSet.Top && s.InsetSet.Bottom
	if tag := wb.Property("tag"); tag == "body" || tag == "html" ||
		(st.height == nil && st.maxHeight == nil && !insetHeight && s.Overflow.Y != styles.OverflowVisible) {
		s.Overflow.Y = styles.OverflowVisible
	}
	if st.height != nil {
		mn, mx := contentBox(*st.height, spaceY)
		s.Min.Y = mn
		if s.Overflow.Y != styles.OverflowVisible {
			s.Max.Y = mx
		}
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
	if tag := wb.Property("tag"); tag == "body" && st.width == nil {
		// it fills the width of its parent (the viewport, as in a browser,
		// or a panel of a page, as in content), and no more
		s.Grow.X, s.Shrink.X = 1, 1
	}
	spc := core.AppearanceSettings.Spacing / 100
	if spc <= 0 {
		spc = 1
	}
	top, bottom := dots(st.margin[0]), dots(st.margin[2])
	bs.marginTop, bs.marginBottom = top, bottom
	c.collapseMargins(wb, s, ps, bs, pbs, pad, bw, &top, &bottom)
	s.Margin.Top = units.Dot(top / spc)
	s.Margin.Bottom = units.Dot(bottom / spc)
	// The spacing of the page is that of its CSS, not scaled by the
	// Spacing setting, which applies to the spacing of Cogent Core widgets
	// (see core.WidgetBase.styleSettings, which multiplies by it).
	if _, isFrame := wb.This.(*core.Frame); isFrame || isTextWidget(wb) {
		for _, v := range []*units.Value{&s.Margin.Left, &s.Margin.Right, &s.Padding.Top, &s.Padding.Right,
			&s.Padding.Bottom, &s.Padding.Left, &s.Gap.X, &s.Gap.Y} {
			v.Value /= spc
		}
	}
}

// isTextWidget returns whether the widget is a [core.Text].
func isTextWidget(wb *core.WidgetBase) bool {
	_, ok := wb.This.(*core.Text)
	return ok
}

// sideDots returns the given sides in dots, using the given function.
func sideDots(v *sides.Values, dots func(v *units.Value) float32) sides.Floats {
	return sides.NewFloats(dots(&v.Top), dots(&v.Right), dots(&v.Bottom), dots(&v.Left))
}
