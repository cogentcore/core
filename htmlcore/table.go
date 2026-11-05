// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"golang.org/x/net/html"
)

// handleTable handles a table element.
func handleTable(ctx *Context) core.Widget {
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	if ctx.tables == nil {
		ctx.tables = map[*html.Node]*core.Frame{}
	}
	ctx.tables[ctx.Node] = w
	ctx.TableParent = w
	cols := ctx.tableColumns(ctx.Node)
	w.SetProperty("cols", cols)
	if ctx.declaration(ctx.Node, "width") == "" && ctx.hasPercentColumns(ctx.Node) {
		// An auto-width table with percentage columns is as wide as it needs
		// to be to give them their shares, which is approximated by filling
		// the width that it has (in which they grow up to their shares).
		w.FinalStyler(func(s *styles.Style) { s.Grow.X = 1 })
	}
	w.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = cols
	})
	return w
}

// tableFor returns the widget of the table that the given node is in.
func (c *Context) tableFor(n *html.Node) core.Widget {
	if t := ancestorTable(n); t != nil {
		if w := c.tables[t]; w != nil {
			return w
		}
	}
	if c.TableParent != nil {
		return c.TableParent
	}
	return c.Parent()
}

// hasPercentColumns returns whether the given table has cells with
// percentage widths (in its rows, not in tables in it).
func (ctx *Context) hasPercentColumns(table *html.Node) bool {
	var rows func(n *html.Node) bool
	rows = func(n *html.Node) bool {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Data {
			case "thead", "tbody", "tfoot":
				if rows(c) {
					return true
				}
			case "tr":
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if (cell.Data == "td" || cell.Data == "th") && strings.HasSuffix(strings.TrimSpace(ctx.declaration(cell, "width")), "%") {
						return true
					}
				}
			}
		}
		return false
	}
	return rows(table)
}

// tableColumns returns the number of columns of the given table, the
// largest number of cells (with their column spans) in any of its rows,
// not counting those that are not displayed.
func (ctx *Context) tableColumns(table *html.Node) int {
	cols := 0
	var rows func(n *html.Node)
	rows = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Data {
			case "thead", "tbody", "tfoot":
				rows(c)
			case "tr":
				n := 0
				for cell := c.FirstChild; cell != nil; cell = cell.NextSibling {
					if (cell.Data == "td" || cell.Data == "th") && ctx.displayOf(cell) != "none" {
						span, err := strconv.Atoi(GetAttr(cell, "colspan"))
						if err != nil || span < 1 {
							span = 1
						}
						n += span
					}
				}
				cols = max(cols, n)
			}
		}
	}
	rows(table)
	return max(cols, 1)
}

// handleCell handles a th or td element.
func handleCell(ctx *Context, tag string) core.Widget {
	ctx.BlockParent = ctx.tableFor(ctx.Node)
	ctx.inlineParent = nil
	span, err := strconv.Atoi(GetAttr(ctx.Node, "colspan"))
	if err != nil || span < 1 {
		span = 1
	}
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	// Columns grow by the size of their content, as in browsers, for
	// which the length of the text is a proxy, only when the table
	// has a width to grow into.
	weight := float32(max(len([]rune(strings.TrimSpace(nodeText(ctx.Node)))), 1))
	rspan, err := strconv.Atoi(GetAttr(ctx.Node, "rowspan"))
	if err != nil || rspan < 1 {
		rspan = 1
	}
	// a percentage width of a cell in a table that fills its width
	// (see handleTable) is the width of its column
	pct := float32(0)
	if table := ancestorTable(ctx.Node); table != nil && ctx.declaration(table, "width") == "" && ctx.hasPercentColumns(table) {
		if v, ok := strings.CutSuffix(strings.TrimSpace(ctx.declaration(ctx.Node, "width")), "%"); ok {
			if f, err := strconv.ParseFloat(v, 32); err == nil {
				pct = float32(f)
			}
		}
	}
	w.FinalStyler(func(s *styles.Style) {
		if pct > 0 {
			s.Min.X, s.Max.X = units.Pw(pct), units.Pw(pct)
			s.BoxSizing = styles.BoxBorderBox
		}
		if span > 1 {
			s.ColSpan = span
		}
		if rspan > 1 {
			s.RowSpan = rspan
		}
		if s.Grow.X > 0 {
			s.Grow.X = weight / float32(span)
		}
	})
	return w
}

// tableImageWidth returns the width attribute of the first image in the
// given element (other than in its caption), or 0 if it has none.
func tableImageWidth(n *html.Node) float32 {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || c.Data == "figcaption" || c.Data == "caption" {
			continue
		}
		if c.Data == "img" {
			w, err := strconv.ParseFloat(GetAttr(c, "width"), 32)
			if err != nil {
				return 0
			}
			return float32(w)
		}
		if w := tableImageWidth(c); w > 0 {
			return w
		}
	}
	return 0
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
