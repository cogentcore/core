// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// The CSS multi-column layout of an element (columns, column-count, and
// column-width) is approximated by moving its children into frames for
// the columns, side by side, filled in order and balanced by the length
// of their text (see [Context.arrangeColumns]). The number of columns is
// from column-count, or from column-width and an estimate of the width
// that the element has (from the viewport and the widths and spaces of
// the elements that it is in), as it is set when the page is read.

// columnCount returns the number of columns of the given element, or 0
// if it does not have a multi-column layout.
func (c *Context) columnCount(n *html.Node) int {
	has := false
	for _, d := range c.declarations(n) {
		if d.Property == "columns" || d.Property == "column-count" || d.Property == "column-width" {
			has = true
			break
		}
	}
	if !has {
		return 0
	}
	defer c.lockCalcEnv()()
	count := 0
	width := float32(0)
	set := func(v string) {
		if v == "auto" {
			return
		}
		if i, err := strconv.Atoi(v); err == nil {
			count = i
		} else if px, ok := c.lengthPx(v); ok {
			width = px
		}
	}
	for _, d := range c.declarations(n) {
		v := strings.ToLower(strings.TrimSpace(d.Value))
		switch d.Property {
		case "columns":
			count, width = 0, 0
			for _, f := range strings.Fields(v) {
				set(f)
			}
		case "column-count":
			count = 0
			set(v)
		case "column-width":
			width = 0
			set(v)
		}
	}
	if width <= 0 {
		if count > 1 {
			return count
		}
		return 0
	}
	gap := float32(16) // the normal column gap is 1em
	if g, ok := c.lengthPx(c.declaration(n, "column-gap")); ok {
		gap = g
	}
	fit := int((c.availableWidth(n) + gap) / (width + gap))
	if count > 0 {
		fit = min(fit, count)
	}
	if fit < 2 {
		return 0
	}
	return fit
}

// lengthPx returns the given CSS length in px, for lengths that are
// absolute or relative to the font size (taken to be 16px for em).
func (c *Context) lengthPx(v string) (float32, bool) {
	v = strings.TrimSpace(v)
	if f, ok := strings.CutSuffix(v, "rem"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * c.rootFontSize(), true
		}
	}
	if f, ok := strings.CutSuffix(v, "em"); ok {
		if x, err := strconv.ParseFloat(f, 32); err == nil {
			return float32(x) * 16, true
		}
	}
	l, ok := parseLength(v, units.UnitPw)
	if !ok || l.Unit != units.UnitDp {
		return 0, false
	}
	return l.Value, true
}

// availableWidth returns an estimate of the width of the content of the
// given element, in px: that of the viewport, within the widths and max
// widths of it and the elements that it is in, less their spaces.
func (c *Context) availableWidth(n *html.Node) float32 {
	w := c.mediaWidth()
	var chain []*html.Node
	for p := n; p != nil && p.Type == html.ElementNode; p = p.Parent {
		chain = append(chain, p)
	}
	for i := len(chain) - 1; i >= 0; i-- { // from the top down
		p := chain[i]
		// the spaces are outside of the width (of the content box)
		for _, base := range []string{"padding", "margin"} {
			l, r := c.horizontalSides(p, base)
			w -= max(l, 0) + max(r, 0)
		}
		for _, prop := range []string{"width", "max-width"} {
			if px, ok := c.lengthPx(c.declaration(p, prop)); ok && px > 0 {
				w = min(w, px)
			}
		}
	}
	return max(w, 0)
}

// arrangeColumns moves the children of the given widget for the given
// element into frames for its columns, as above.
func (c *Context) arrangeColumns(n *html.Node, fr *core.Frame) {
	count := c.columnCount(n)
	if count < 2 {
		return
	}
	// The content of an element with only one child (as a list in a div
	// with columns, as for the references of Wikipedia) is that of the
	// child, which is split into the columns, as in CSS.
	for fr.NumChildren() == 1 {
		k, ok := fr.Child(0).(*core.Frame)
		if !ok || k.NumChildren() < 2 {
			break
		}
		fr = k
	}
	kids := append([]tree.Node(nil), fr.Children...)
	if len(kids) < 2 {
		return
	}
	count = min(count, len(kids))
	gap := units.Em(1)
	unlock := c.lockCalcEnv()
	if g, ok := parseLength(c.declaration(n, "column-gap"), units.UnitPw); ok {
		gap = g
	}
	unlock()
	fr.FinalStyler(func(s *styles.Style) {
		s.Display = styles.Flex
		s.Direction = styles.Row
		s.Wrap = false
		s.Align.Items = styles.Start
		s.Gap.X = gap
	})
	// the weight of a child is the length of its text (as its height)
	weights := make([]int, len(kids))
	total := 0
	for i, k := range kids {
		w := 40
		k.AsTree().WalkDown(func(d tree.Node) bool {
			if tx, ok := d.(*core.Text); ok {
				w += len(tx.Text)
			}
			return tree.Continue
		})
		weights[i] = w
		total += w
	}
	var cols []*core.Frame
	for range count {
		col := core.NewFrame(fr)
		col.SetName("column")
		tree.SetUniqueName(col)
		c.Node = nil
		c.browserStyler(nil, col, nil)
		c.blocks[col].display = "block"
		col.FinalStyler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.X, s.Shrink.X = 1, 1
			s.Min.X = units.Value{}
			s.Max.X = units.Pw(100 / float32(count))
		})
		cols = append(cols, col)
	}
	// each column is filled up to its share of the total weight
	ci, sum := 0, 0
	for i, k := range kids {
		if ci < count-1 && sum > 0 && sum+weights[i]/2 > total*(ci+1)/count {
			ci++
		}
		tree.MoveToParent(k, cols[ci])
		sum += weights[i]
	}
}

// horizontalSides returns the left and right lengths in px of the given
// sides property (padding or margin) of the given element, from its
// declarations in the order of the cascade (with the shorthand).
func (c *Context) horizontalSides(n *html.Node, base string) (left, right float32) {
	px := func(v string) float32 {
		l, _ := c.lengthPx(v)
		return l
	}
	for _, d := range c.declarations(n) {
		switch d.Property {
		case base:
			if vals := expandSides(splitColors(d.Value)); len(vals) == 4 {
				right, left = px(vals[1]), px(vals[3])
			}
		case base + "-left", base + "-inline-start":
			left = px(d.Value)
		case base + "-right", base + "-inline-end":
			right = px(d.Value)
		case base + "-inline":
			fs := splitColors(d.Value)
			if len(fs) > 0 {
				left, right = px(fs[0]), px(fs[len(fs)-1])
			}
		}
	}
	return
}
