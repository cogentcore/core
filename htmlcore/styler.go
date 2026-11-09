// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"cogentcore.org/core/text/shaped"
	_ "embed"
	"fmt"
	"image"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
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
	// ops are the style operations of the declarations of the element,
	// and inherit those of the inherited properties of the inline
	// elements that it is in (see [Context.inlineInheritedOps]).
	ops, inherit []styleOp

	// display is the CSS display of the element.
	display string

	// block is whether the widget is block-level.
	block bool

	// hidden is whether the element is not visible (visibility: hidden),
	// which its descendants inherit.
	hidden bool

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
	hideText   bool // a large negative text-indent, which moves the text out of view
	zeroScale  bool // a transform scales the box to nothing (as scaleX(0)), so it is not shown
	clippedOut bool // a clip (or clip-path) to nothing hides it (see zeroAreaClip)

	// bgCurrentColor is whether the background color is currentcolor,
	// the color of the element as it is styled (as the masked chevrons of
	// the menus of MDN), which was transparent.
	bgCurrentColor bool
	visibility     string // the visibility property, if any
	width          *units.Value
	heightFit      bool // a height of fit-content (or max- or min-content)
	height         *units.Value
	minWidth       *units.Value
	minHeight      *units.Value
	maxWidth       *units.Value
	maxHeight      *units.Value
	margin         [4]*units.Value // top, right, bottom, left; nil for auto
	marginAuto     [4]bool
	lineHeight     string
	borderBox      bool
	flexGrow       float32
	hasGrow        bool
	flexShrink     float32
	hasShrink      bool
	flexBasis      string // the flex-basis, or "" if it is not set
	alignSelf      string
	alignItems     string
	background     image.Image
	hasBg          bool
	mask           *cssMask
	bgImage        *cssMask // the background image, with its size, position and repeat

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
	isFrame := frameOf(w) != nil
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
	// (in the state of the block, as they are compiled again when the
	// declarations of the element change; see Context.Restyle)
	bs.ops = c.compileDecls(decls)
	bs.inherit = c.inlineInheritedOps(node, wb)
	wb.FinalStyler(func(s *styles.Style) {
		if bs.hidden { // after the abilities of the widget (as scrolling)
			s.Abilities = 0
		}
	})
	wb.Styler(func(s *styles.Style) {
		pw, _ := wb.Parent.(core.Widget)
		var ps *styles.Style
		var pbs *blockState
		if pw != nil {
			ps = &pw.AsWidget().Styles
			pbs = c.blocks[pw]
		}
		if c.Zoom > 0 {
			s.Zoom = c.Zoom // (inherited, but set on each for widgets outside of the page)
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
		if c.MediaHeight > 0 { // (of the viewport, not of the scene)
			env.vh, env.fixedVh = float64(c.MediaHeight)/100, true
		}
		env.vhUsed = &c.viewportHeightUsed
		env.rem = float64(c.rootFontSize())
		env.ch = func() float64 { // of the font of the element as it is styled
			em := env.em
			if s.Font.Size.Unit == units.UnitDp && s.Font.Size.Value > 0 {
				em = float64(s.Font.Size.Value)
			}
			var sh shaped.Shaper
			if wb.Scene != nil {
				sh = wb.Scene.TextShaper()
			}
			return chOf(sh, s, em)
		}
		env.fixedVh = c.MediaHeight > 0
		if env.fixedVh {
			env.vh = float64(c.MediaHeight / 100)
		}
		// the container query units, of the container of the element
		env.cqw, env.cqh = containerUnits(w, env.vw, env.vh)
		if tag == "body" && c.declaration(node, "font-size") == "" {
			// the font size of the html element (which has no widget)
			if rf := c.rootFontSize(); rf != 16 {
				for _, op := range c.compileCSS("font-size", fmt.Sprintf("%gpx", rf)) {
					op(st, s, ps)
				}
			}
		}
		if ps != nil { // pointer-events is inherited
			s.SetAbilities(ps.Abilities.HasFlag(abilities.NoPointerEvents), abilities.NoPointerEvents)
		}
		for _, op := range bs.inherit {
			op(st, s, ps)
		}
		for _, op := range bs.ops {
			op(st, s, ps)
		}
		if node != nil {
			st.display = c.displayOf(node)
		}
		c.resolveCSS(wb, node, st, s, ps, bs, pbs)
		if node != nil && node.Type == html.ElementNode && st.display == "inline-block" && c.isInlineBox(node) {
			if d := strings.TrimSpace(c.declaration(node, "display")); d == "" || d == "inline" {
				// an inline element that is a box (see isInlineBox) does not
				// change the height of its line by its vertical padding and
				// border, as in CSS (as the padded links of the sidebar of
				// daringfireball.net, in lines as high as their text)
				// (in dots when the units are converted)
				lessBy := func(m units.Value, by ...units.Value) units.Value {
					return units.Value{Custom: func(uc *units.Context) float32 {
						v := m.ToDots(uc)
						for _, b := range by {
							v -= b.ToDots(uc)
						}
						return v
					}}
				}
				s.Margin.Top = lessBy(s.Margin.Top, s.Padding.Top, s.Border.Width.Top)
				s.Margin.Bottom = lessBy(s.Margin.Bottom, s.Padding.Bottom, s.Border.Width.Bottom)
			}
		}
		if node != nil && isButtonNode(node) && c.declaration(node, "justify-content") == "" {
			// the content of a button is aligned by its text-align (center
			// by default), and at the start of one with a display of flex
			// or grid (as the jump button of pkg.go.dev, which is left
			// aligned with both)
			switch c.declaration(node, "display") {
			case "flex", "inline-flex", "grid", "inline-grid":
				s.Justify.Content = styles.Start
			default:
				switch s.Text.Align {
				case text.Center:
					s.Justify.Content = styles.Center
				case text.End:
					s.Justify.Content = styles.End
				default:
					s.Justify.Content = styles.Start
				}
			}
		}
		if s.ZIndex != 0 && s.Position == styles.PositionStatic && (pbs == nil || !isFlexOrGrid(pbs.display)) {
			// as in CSS, the z-index of a static element only applies to
			// flex and grid items (as the site header of pkg.go.dev, over
			// the sticky header of the page)
			s.ZIndex = 0
		}
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

// isFlexOrGrid returns whether the given display is that of a flex or grid
// container, whose items are stacked by their z-index (see
// [styles.Style.ZIndex]) even if they are not positioned.
func isFlexOrGrid(d string) bool {
	switch d {
	case "flex", "inline-flex", "grid", "inline-grid":
		return true
	}
	return false
}

// isBlockDisplay returns whether the given display is block-level.
func isBlockDisplay(d string) bool {
	switch d {
	case "block", "list-item", "flex", "grid", "table", "flow-root":
		return true
	}
	return false
}

// inlineInheritedOps returns the style operations of the inherited
// properties of the inline elements that the given element is in, between
// it and the element of its parent widget, outermost first. These have no
// widgets of their own (they are in the text of a run), but their
// inherited properties apply to the widgets of their children, as the font
// size of the edit links of Wikipedia, which are inline blocks.
func (c *Context) inlineInheritedOps(node *html.Node, wb *core.WidgetBase) []styleOp {
	if node == nil || node.Type != html.ElementNode {
		return nil
	}
	pw, _ := wb.Parent.(core.Widget)
	if pw == nil {
		return nil
	}
	pn := NodeOf(pw)
	var chain []*html.Node
	for p := node.Parent; p != nil && p != pn && p.Type == html.ElementNode; p = p.Parent {
		switch c.displayOf(p) {
		case "", "inline", "contents":
			chain = append(chain, p)
			continue
		}
		break
	}
	var ops []styleOp
	for i := len(chain) - 1; i >= 0; i-- {
		var inh []*cssDecl
		for _, d := range c.declarations(chain[i]) {
			if inheritedProperties[d.Property] {
				inh = append(inh, d)
			}
		}
		ops = append(ops, c.compileDecls(inh)...)
	}
	return ops
}

// isButtonNode returns whether the given node is a button: a button element,
// or an input of a type of submit, button or reset.
func isButtonNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.Data {
	case "button":
		return true
	case "input":
		switch strings.ToLower(GetAttr(n, "type")) {
		case "submit", "button", "reset":
			return true
		}
	}
	return false
}
