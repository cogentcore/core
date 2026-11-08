// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// The CSS grid template areas of a grid container (grid-template-areas,
// or the areas of the grid-template shorthand) place its children with a
// grid-area at the rows and columns of their areas, which is done by
// ordering their widgets as a core grid places them, with spans and
// empty widgets for the empty cells (see [Context.arrangeGrid]).
// The column tracks (grid-template-columns, or the columns of the
// grid-template shorthand) size the children in the columns.

// gridArea is the rectangle of the cells of a named grid area.
type gridArea struct {
	row, col, rows, cols int
}

// gridAreaStrings matches the quoted rows of grid template areas.
var gridAreaStrings = regexp.MustCompile(`"([^"]*)"|'([^']*)'`)

// gridTemplate returns the areas, the number of columns, and the column
// tracks of the grid template of the given grid container.
func (c *Context) gridTemplate(n *html.Node) (areas map[string]gridArea, ncols int, tracks []string) {
	// the shorthand or the longhand that is last in the cascade applies
	// (as a grid-template-columns of a more specific rule)
	colsDecl, areasDecl := "", ""
	if prop, v := c.lastDeclaration(n, "grid-template-columns", "grid-template"); prop == "grid-template" {
		if _, cols, ok := cutTopLevel(v, '/'); ok {
			colsDecl = cols
		}
	} else {
		colsDecl = v
	}
	if prop, v := c.lastDeclaration(n, "grid-template-areas", "grid-template"); prop == "grid-template" {
		if strings.ContainsAny(v, `"'`) {
			areasDecl = v
		}
	} else {
		areasDecl = v
	}
	tracks = gridTracks(colsDecl)
	var rows [][]string
	for _, m := range gridAreaStrings.FindAllStringSubmatch(areasDecl, -1) {
		row := strings.Fields(m[1] + m[2])
		rows = append(rows, row)
		ncols = max(ncols, len(row))
	}
	if len(rows) == 0 {
		return nil, len(tracks), tracks
	}
	areas = map[string]gridArea{}
	for r, row := range rows {
		for col, name := range row {
			if strings.Trim(name, ".") == "" {
				continue // a null cell
			}
			a, ok := areas[name]
			if !ok {
				areas[name] = gridArea{r, col, 1, 1}
				continue
			}
			a.rows = max(a.rows, r-a.row+1)
			a.cols = max(a.cols, col-a.col+1)
			areas[name] = a
		}
	}
	return areas, max(ncols, len(tracks)), tracks
}

// cutTopLevel cuts s around the first sep that is not in parentheses.
func cutTopLevel(s string, sep byte) (before, after string, found bool) {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case sep:
			if depth == 0 {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return s, "", false
}

// gridLineNames matches the line names of a track list, as [a b].
var gridLineNames = regexp.MustCompile(`\[[^\]]*\]`)

// gridTracks returns the track sizes of a grid-template-columns value,
// with repeat() of a number expanded and line names removed.
func gridTracks(v string) []string {
	var tracks []string
	v = gridLineNames.ReplaceAllString(v, " ")
	for _, f := range splitColors(v) { // fields outside of parentheses
		if strings.HasPrefix(f, "repeat(") {
			inner := strings.TrimSuffix(strings.TrimPrefix(f, "repeat("), ")")
			n, rest, _ := strings.Cut(inner, ",")
			if count, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
				for range count {
					tracks = append(tracks, gridTracks(rest)...)
				}
			}
			continue
		}
		tracks = append(tracks, f)
	}
	return tracks
}

// gridTrackStyle sets the style of a child in a grid column with the given
// track size: a length is its width, a flexible (fr) track grows by its
// factor (and minmax(0, length) grows up to the length), and the other
// tracks are as wide as their content when other tracks are flexible
// (and auto tracks share the rest of the width when none are).
func gridTrackStyle(env *lengthEnv, s *styles.Style, track string, flexible bool) {
	if strings.HasPrefix(track, "minmax(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(track, "minmax("), ")")
		_, mx, _ := strings.Cut(inner, ",")
		mx = strings.TrimSpace(mx)
		if strings.HasSuffix(mx, "fr") || mx == "auto" {
			track = mx
		} else if l, ok := env.length(mx, units.UnitPw); ok {
			s.Grow.X, s.Shrink.X = 1, 1
			s.Max.X = l
			s.BoxSizing = styles.BoxBorderBox
			return
		}
	}
	if fr, ok := strings.CutSuffix(track, "fr"); ok {
		if f, err := strconv.ParseFloat(fr, 32); err == nil {
			s.Grow.X, s.Shrink.X = float32(f), 1
		}
		return
	}
	if l, ok := env.length(track, units.UnitPw); ok {
		s.BoxSizing = styles.BoxBorderBox
		if l.Unit == units.UnitPw { // a percentage of the grid, which it fills
			s.Max.X = l
			s.Grow.X, s.Shrink.X = 1, 1
			return
		}
		s.Min.X, s.Max.X = l, l
		s.Grow.X, s.Shrink.X = 0, 0
		return
	}
	if flexible { // auto, min-content, max-content, and fit-content()
		s.Grow.X = 0
	} else if track == "auto" { // auto tracks share the rest of the width
		s.Grow.X, s.Shrink.X = 1, 1
	}
}

// arrangeGrid places the children of the given grid container widget for
// its node by its grid template (see the comment at the top of grid.go).
func (c *Context) arrangeGrid(n *html.Node, w core.Widget) {
	areas, ncols, tracks := c.gridTemplate(n)
	env := c.readEnv() // for the lengths of the tracks
	if ncols == 0 {
		return
	}
	flexible := slices.ContainsFunc(tracks, func(t string) bool { return strings.HasSuffix(t, "fr") || strings.HasSuffix(t, "fr)") })
	trackOf := func(col int) string {
		if col < len(tracks) {
			return tracks[col]
		}
		return "auto"
	}
	type placed struct {
		w    core.Widget
		area gridArea
	}
	var items []placed
	var auto []tree.Node
	for _, k := range w.AsTree().Children {
		kw, ok := k.(core.Widget)
		if !ok {
			auto = append(auto, k)
			continue
		}
		kn, _ := kw.AsWidget().Property("node").(*html.Node)
		name := ""
		if kn != nil {
			name, _, _ = cutTopLevel(c.declaration(kn, "grid-area"), '/')
		}
		if a, ok := areas[name]; ok && name != "" {
			items = append(items, placed{kw, a})
		} else {
			auto = append(auto, k)
		}
	}
	w.AsWidget().FinalStyler(func(s *styles.Style) { s.Columns = ncols })
	if len(items) == 0 {
		if len(tracks) > 0 {
			for _, k := range auto { // the tracks of the columns of the children in order
				if kw, ok := k.(core.Widget); ok {
					i := slices.Index(auto, k) % ncols
					kw.AsWidget().FinalStyler(func(s *styles.Style) { gridTrackStyle(env, s, trackOf(i), flexible) })
				}
			}
		}
		return
	}
	slices.SortStableFunc(items, func(a, b placed) int {
		if a.area.row != b.area.row {
			return a.area.row - b.area.row
		}
		return a.area.col - b.area.col
	})
	// The cells in row-major order, as the core grid places them: the
	// items at the start of their areas, and empty widgets in the cells
	// that are not in any area, up to the last item.
	covered := map[[2]int]bool{}
	for _, it := range items {
		for r := it.area.row; r < it.area.row+it.area.rows; r++ {
			for col := it.area.col; col < it.area.col+it.area.cols; col++ {
				covered[[2]int{r, col}] = true
			}
		}
	}
	last := items[len(items)-1].area
	var kids []tree.Node
	next := 0
	for r := 0; r <= last.row; r++ {
		for col := range ncols {
			if r == last.row && col > last.col {
				break
			}
			if next < len(items) && items[next].area.row == r && items[next].area.col == col {
				it := items[next]
				next++
				kids = append(kids, it.w)
				a := it.area
				track := trackOf(col)
				it.w.AsWidget().FinalStyler(func(s *styles.Style) {
					s.ColSpan, s.RowSpan = a.cols, a.rows
					if a.cols == 1 {
						gridTrackStyle(env, s, track, flexible)
					}
				})
				continue
			}
			if covered[[2]int{r, col}] {
				continue
			}
			filler := core.NewFrame(w) // moved into its place below
			filler.SetName("grid-filler")
			track := trackOf(col)
			filler.Styler(func(s *styles.Style) {
				s.Padding.Zero()
				s.Margin.Zero()
				s.Gap.Zero()
				s.Background = nil
				gridTrackStyle(env, s, track, flexible)
			})
			kids = append(kids, filler)
		}
	}
	for ; next < len(items); next++ { // items that share a start with another
		auto = append(auto, items[next].w)
	}
	w.AsTree().Children = append(kids, auto...)
}
