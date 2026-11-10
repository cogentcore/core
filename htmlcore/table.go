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
	"cogentcore.org/core/tree"
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
	// the columns of a table with the fixed layout are as wide as its
	// columns (col) say, not as its content, which wraps (as the diffs of
	// Wikipedia, whose long lines are in columns of 48%)
	// (and columns with widths in pixels are that wide)
	fixed, fixedWeight, fixedPct, fixedPx := false, float32(0), float32(0), float32(0)
	if table := ancestorTable(ctx.Node); table != nil && strings.EqualFold(strings.TrimSpace(ctx.declaration(table, "table-layout")), "fixed") {
		fixed = true
		cols, pxs := ctx.columnWidths(table)
		start := cellColumn(ctx.Node)
		allPct, allPx := true, true
		for c := start; c < start+span; c++ {
			if c < len(cols) && cols[c] > 0 {
				fixedWeight += cols[c]
				fixedPct += cols[c]
			} else {
				fixedWeight += 1
				allPct = false
			}
			if c < len(pxs) && pxs[c] > 0 {
				fixedPx += pxs[c]
			} else {
				allPx = false
			}
		}
		if !allPct {
			fixedPct = 0
		}
		if !allPx {
			fixedPx = 0
		}
	}
	w.FinalStyler(func(s *styles.Style) {
		if pct > 0 {
			s.Min.X, s.Max.X = units.Pw(pct), units.Pw(pct)
			s.BoxSizing = styles.BoxBorderBox
		}
		if fixed {
			s.Overflow.X = styles.OverflowAuto // no min-content width (it wraps)
			s.ScrollbarWidth.Zero()
			s.Grow.X, s.Shrink.X = fixedWeight, 1
			if fixedPct > 0 { // the percentage of the table of its columns
				s.Min.X, s.Max.X = units.Pw(fixedPct), units.Pw(fixedPct)
				s.BoxSizing = styles.BoxBorderBox
			}
			if fixedPx > 0 { // the width of its columns
				s.Min.X, s.Max.X = units.Dp(fixedPx), units.Dp(fixedPx)
				s.BoxSizing = styles.BoxBorderBox
				s.Grow.X = 0
			}
		}
		if span > 1 {
			s.ColSpan = span
		}
		if rspan > 1 {
			s.RowSpan = rspan
		}
		if s.Grow.X > 0 && !fixed {
			s.Grow.X = weight / float32(span)
		}
	})
	return w
}

// columnWidths returns the widths of the columns of the given table from
// its col elements (with span): as percentages, or 0 for those without
// one (or with a width that is not a percentage), and in pixels (as
// width="52" of the line numbers of the diffs of GitHub), or 0.
func (c *Context) columnWidths(table *html.Node) ([]float32, []float32) {
	var ws, pxs []float32
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type != html.ElementNode {
				continue
			}
			switch k.Data {
			case "colgroup":
				walk(k)
			case "col":
				span, err := strconv.Atoi(GetAttr(k, "span"))
				if err != nil || span < 1 {
					span = 1
				}
				w := strings.TrimSpace(c.declaration(k, "width"))
				if w == "" {
					w = GetAttr(k, "width")
				}
				pct, px := float32(0), float32(0)
				if v, ok := strings.CutSuffix(w, "%"); ok {
					if f, err := strconv.ParseFloat(v, 32); err == nil {
						pct = float32(f)
					}
				} else if f, err := strconv.ParseFloat(strings.TrimSuffix(w, "px"), 32); err == nil && f > 0 {
					px = float32(f)
				}
				for range span {
					ws = append(ws, pct)
					pxs = append(pxs, px)
				}
			}
		}
	}
	walk(table)
	return ws, pxs
}

// cellColumn returns the index of the first column of the given cell in
// its row, from the column spans of the cells before it.
func cellColumn(cell *html.Node) int {
	col := 0
	for k := cell.PrevSibling; k != nil; k = k.PrevSibling {
		if k.Type == html.ElementNode && (k.Data == "td" || k.Data == "th") {
			span, err := strconv.Atoi(GetAttr(k, "colspan"))
			if err != nil || span < 1 {
				span = 1
			}
			col += span
		}
	}
	return col
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

// tableRows is the state of the rows of a table as they are read, for the
// slots of the rows below that cells with row spans take (see
// [Context.endTableRow]).
type tableRows struct {
	cols  int
	spans []int // for each column, the number of rows below that it takes
}

// endTableRow fills the rest of the given row of a table (tr), after its
// cells, with empty cells, so that the cells of the next row are in the
// next row, as in HTML, and not in the slots that this row leaves empty
// (as the major debates table of Wikipedia, whose header row of parties
// has fewer cells than its columns). The slots of the row that cells
// above it take with their row spans are not filled.
func (c *Context) endTableRow(tr *html.Node) {
	table := ancestorTable(tr)
	if table == nil {
		return
	}
	if c.tableRowState == nil {
		c.tableRowState = map[*html.Node]*tableRows{}
	}
	st := c.tableRowState[table]
	if st == nil {
		cols := c.tableColumns(table)
		st = &tableRows{cols: cols, spans: make([]int, cols)}
		c.tableRowState[table] = st
	}
	taken := make([]bool, st.cols)
	for i, n := range st.spans {
		if n > 0 {
			taken[i] = true
			st.spans[i]--
		}
	}
	col := 0
	for cell := tr.FirstChild; cell != nil; cell = cell.NextSibling {
		if cell.Type != html.ElementNode || (cell.Data != "td" && cell.Data != "th") || c.displayOf(cell) == "none" {
			continue
		}
		for col < st.cols && taken[col] {
			col++
		}
		span, err := strconv.Atoi(GetAttr(cell, "colspan"))
		if err != nil || span < 1 {
			span = 1
		}
		rspan, err := strconv.Atoi(GetAttr(cell, "rowspan"))
		if err != nil || rspan < 1 {
			rspan = 1
		}
		for k := col; k < col+span && k < st.cols; k++ {
			taken[k] = true
			if rspan > 1 {
				st.spans[k] = rspan - 1
			}
		}
		col += span
	}
	empty := 0
	for _, t := range taken {
		if !t {
			empty++
		}
	}
	w := c.tableFor(tr)
	for range empty {
		filler := core.NewFrame(w)
		filler.SetName("row-filler")
		tree.SetUniqueName(filler)
		filler.Styler(func(s *styles.Style) {
			s.Padding.Zero()
			s.Margin.Zero()
			s.Grow.Set(0, 0)
		})
	}
}
