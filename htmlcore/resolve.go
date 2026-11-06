// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
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

// resolver resolves the declarations of a widget that depend on each other
// and on its parent, after all of them have been applied (see
// [Context.resolveCSS]), in steps that each have a method.
type resolver struct {
	c       *Context
	wb      *core.WidgetBase
	node    *html.Node
	st      *cssState
	s, ps   *styles.Style
	bs, pbs *blockState

	// uc is the unit context for converting the lengths of the widget to dots.
	uc units.Context

	// parentDisplay is the display of the parent, or "" if there is none.
	parentDisplay string

	// pad and bw are the padding and border widths of the widget, in dots.
	pad, bw sides.Floats
}

// resolveCSS resolves the declarations that depend on each other
// and on the parent, after all of them have been set.
func (c *Context) resolveCSS(wb *core.WidgetBase, node *html.Node, st *cssState, s, ps *styles.Style, bs, pbs *blockState) {
	r := &resolver{c: c, wb: wb, node: node, st: st, s: s, ps: ps, bs: bs, pbs: pbs}
	r.blockState()
	r.layoutDisplay()
	r.replacedSize()
	r.flow()
	r.lineHeight()
	r.itemGrowth()
	r.sizes()
	r.textAlign()
	r.tableBorders()
	r.margins()
}

// dots returns the given length in dots, or 0 if it is nil.
func (r *resolver) dots(v *units.Value) float32 {
	if v == nil {
		return 0
	}
	return r.uc.ToDots(v.Value, v.Unit)
}

// blockState copies the state that the children need into the block state,
// and applies the legacy center and the background.
func (r *resolver) blockState() {
	st, s, bs, pbs := r.st, r.s, r.bs, r.pbs
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
}

// layoutDisplay sets the core layout for the display of the element.
func (r *resolver) layoutDisplay() {
	node, st, s := r.node, r.st, r.s
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
}

// replacedSize sets the size of an img of an svg and of an svg element, from
// the intrinsic size of the svg and the sizes of their containers.
func (r *resolver) replacedSize() {
	c, wb, node, st, s := r.c, r.wb, r.node, r.st, r.s
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
}

// flow sets whether the widget is block-level, in the flow of blocks,
// and the alignment of an inline element in its line (or of the content
// of a table cell).
func (r *resolver) flow() {
	wb, st, s, bs, pbs := r.wb, r.st, r.s, r.bs, r.pbs
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
	if s.Position != styles.PositionStatic {
		// positioned elements are painted after the others, which the
		// background of the parent in their margins would cover (margins
		// are transparent in CSS)
		s.FillMargin = false
	}
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
}

// lineHeight sets the unit context for converting the lengths of the widget
// to dots, and the line height.
func (r *resolver) lineHeight() {
	st, s, ps := r.st, r.s, r.ps
	uc, dots := &r.uc, r.dots
	// units context for converting to dots
	*uc = s.UnitContext
	if ps != nil {
		*uc = ps.UnitContext
	}
	uc.FontEm = uc.ToDots(s.Font.Size.Value, s.Font.Size.Unit) * core.AppearanceSettings.FontSize / 100

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
}

// itemGrowth sets how the widget grows and shrinks in the formatting context
// of its parent: block, flex, grid or table, or shrink-to-fit.
func (r *resolver) itemGrowth() {
	wb, st, s, ps, bs, pbs := r.wb, r.st, r.s, r.ps, r.bs, r.pbs
	// block-level elements fill the width of a block container,
	// and flex items are stretched in the cross dimension
	if ps != nil && pbs != nil {
		r.parentDisplay = pbs.display
	}
	parentDisplay := r.parentDisplay
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
}

// sizes sets the sizes (Min and Max, of the content box), with percentage
// heights only of a parent with a definite height, and the overflow.
func (r *resolver) sizes() {
	c, wb, node, st, s, bs, pbs := r.c, r.wb, r.node, r.st, r.s, r.bs, r.pbs
	dots := r.dots
	// sizes: Min and Max are the size of the content box (or of the
	// border box, for box-sizing: border-box, of which core subtracts
	// the padding and border; see styles.BoxSizing)
	pad := sideDots(&s.Padding, dots)
	if bs.markers { // the markers of the items take the space of the padding
		mw := dots(&markerWidth)
		pl := max(pad.Left-mw, 0)
		s.Padding.Left = units.Dot(pl)
		pad.Left = pl
	}
	r.pad = pad
	r.bw = sideDots(&s.Border.Width, dots)
	if st.borderBox {
		s.BoxSizing = styles.BoxBorderBox
	}
	if st.width != nil {
		intrinsic := s.Min.X
		s.Min.X, s.Max.X = *st.width, *st.width
		s.Grow.X = 0
		_, isImage := wb.This.(*core.Image)
		if st.width.Unit == units.UnitPw && !isImage {
			// A percentage of the width of the parent fills that much
			// of it, which as a minimum would make a parent that is as
			// wide as its content grow without end.
			s.Min.X = units.Value{}
			s.Grow.X = 1
			if _, isField := wb.This.(*core.TextField); isField && intrinsic.Value > 0 {
				// A text field (input) is as wide as its intrinsic width
				// in a parent that is as wide as its content, and can
				// shrink, as in CSS (as the email field of github.com).
				s.Min.X = intrinsic
				s.Shrink.X = 1
			}
		}
		nowrap := s.Text.WhiteSpace == text.WrapNever || s.Text.WhiteSpace == text.WhiteSpacePre
		if st.display == "table-cell" && st.width.Unit == units.UnitPw && nowrap {
			// The column of a table cell is at least as wide as its content,
			// as in CSS, which for text that does not wrap is its whole
			// width: then a percentage width is not a limit (as for a cell
			// with a width of 1% to be as narrow as its text). Text that
			// wraps fits the percentage, which is kept (as for halves).
			s.Max.X = units.Value{}
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
		mn, mx := *st.height, *st.height
		s.Min.Y = mn
		if s.Overflow.Y != styles.OverflowVisible {
			s.Max.Y = mx
		}
		s.Grow.Y = 0
	}
	if st.minWidth != nil {
		_, isField := r.wb.This.(*core.TextField)
		if !(isField && st.minWidth.Value == 0 && s.Shrink.X > 0) {
			// (a text field that can shrink keeps its intrinsic width)
			s.Min.X = *st.minWidth
		}
	}
	if st.minHeight != nil {
		s.Min.Y = *st.minHeight
	}
	if st.maxWidth != nil {
		s.Max.X = *st.maxWidth
	}
	if st.maxHeight != nil {
		s.Max.Y = *st.maxHeight
	}
}

// textAlign aligns text and inline rows within their block for text-align.
func (r *resolver) textAlign() {
	wb, st, s, bs := r.wb, r.st, r.s, r.bs
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
}

// tableBorders approximates border-collapse and border-spacing of tables.
func (r *resolver) tableBorders() {
	st, s, ps, pbs := r.st, r.s, r.ps, r.pbs
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
}

// margins sets the margins: auto margins align a block, and the vertical
// margins of blocks collapse.
func (r *resolver) margins() {
	c, wb, st, s, ps, bs, pbs := r.c, r.wb, r.st, r.s, r.ps, r.bs, r.pbs
	dots, parentDisplay, pad, bw := r.dots, r.parentDisplay, r.pad, r.bw
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
