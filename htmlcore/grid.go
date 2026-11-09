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
	tracks = gridTracksAuto(colsDecl, func(track string, fit bool) int { return c.autoRepeatCount(n, track, fit) })
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
	return gridTracksAuto(v, nil)
}

// gridTracksAuto is [gridTracks] with repeat(auto-fit, ...) and
// repeat(auto-fill, ...) expanded to the number of tracks that the given
// function returns for the repeated track (and whether it is auto-fit).
func gridTracksAuto(v string, autoCount func(track string, fit bool) int) []string {
	var tracks []string
	v = gridLineNames.ReplaceAllString(v, " ")
	for _, f := range splitColors(v) { // fields outside of parentheses
		if strings.HasPrefix(f, "repeat(") {
			inner := strings.TrimSuffix(strings.TrimPrefix(f, "repeat("), ")")
			n, rest, _ := strings.Cut(inner, ",")
			n = strings.TrimSpace(n)
			count, err := strconv.Atoi(n)
			if err != nil && (n == "auto-fit" || n == "auto-fill") && autoCount != nil {
				count, err = autoCount(strings.TrimSpace(rest), n == "auto-fit"), nil
			}
			if err == nil {
				for range count {
					tracks = append(tracks, gridTracksAuto(rest, autoCount)...)
				}
			}
			continue
		}
		tracks = append(tracks, f)
	}
	return tracks
}

// autoRepeatCount returns the number of tracks of the given track of a
// repeat(auto-fit or auto-fill) of the grid of the given node: as many as
// fit in its width at their minimum size (of minmax(), or their size), with
// the column gap, as in CSS, of the width of the page when the grid has no
// width of its own to go by (which a percentage minimum does not depend
// on much), and at most as many as the grid has items for auto-fit, whose
// empty tracks collapse (as the 3 cards of repeat(auto-fit, minmax(25%,
// 1fr)) of edition.cnn.com, which were in one column).
func (c *Context) autoRepeatCount(n *html.Node, track string, fit bool) int {
	mn := track
	if inner, ok := strings.CutPrefix(track, "minmax("); ok {
		mn, _, _ = strings.Cut(strings.TrimSuffix(inner, ")"), ",")
		mn = strings.TrimSpace(mn)
	}
	env := c.readEnv()
	width := c.mediaWidth()
	if w, ok := env.lengthPx(strings.TrimSpace(c.declaration(n, "width"))); ok && w > 0 {
		width = w
	}
	gap := float32(0)
	g := strings.TrimSpace(c.declaration(n, "column-gap"))
	if g == "" {
		if _, col, ok := cutTopLevel(strings.TrimSpace(c.declaration(n, "gap")), ' '); ok {
			g = col
		} else {
			g = strings.TrimSpace(c.declaration(n, "gap"))
		}
	}
	if px, ok := env.lengthPx(g); ok {
		gap = px
	}
	size := float32(0)
	if p, ok := strings.CutSuffix(mn, "%"); ok {
		if f, err := strconv.ParseFloat(p, 32); err == nil {
			size = width * float32(f) / 100
		}
	} else if px, ok := env.lengthPx(mn); ok {
		size = px
	}
	count := 1
	if size > 0 {
		count = max(int((width+gap)/(size+gap)), 1)
	}
	if fit && n != nil {
		items := 0
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode && c.inFlexFlow(k) {
				items++
			}
		}
		count = max(min(count, items), 1)
	}
	return count
}

// gridTrackStyle sets the style of a child in a grid column with the given
// track size: a length is its width, a flexible (fr) track grows by its
// factor (and minmax(0, length) grows up to the length), and the other
// tracks are as wide as their content when other tracks are flexible
// (and auto tracks share the rest of the width when none are).
func gridTrackStyle(env *lengthEnv, s *styles.Style, track string, flexible bool) {
	if strings.HasPrefix(track, "minmax(") {
		inner := strings.TrimSuffix(strings.TrimPrefix(track, "minmax("), ")")
		mn, mx, _ := strings.Cut(inner, ",")
		mx = strings.TrimSpace(mx)
		if strings.HasSuffix(mx, "fr") || mx == "auto" {
			if mn = strings.TrimSpace(mn); (mn == "0" || mn == "0px") && strings.HasSuffix(mx, "fr") && s.Contain == styles.ContainNone {
				// the content of a cell does not size its track, so the
				// tracks share the width by their fr (as the cells of the
				// repository grid of github.com, whose hidden panels
				// widened their columns), and it overflows them
				s.Contain = styles.ContainInlineSize
			}
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
		// a length or a percentage of the grid is the width of the track,
		// which its content does not narrow (as the 21.5% navigation of
		// pkg.go.dev, beside documentation with wide lines of code)
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

// stackGrid makes the given grid container widget a stack, showing its
// last child, if all of its children (two or more) are in the same cell
// of it by number (as grid-area: 1 / 1), over each other (as the counters
// of github.com, with a hidden copy of the value under it, for its width),
// and returns whether it did. (Core grids do not overlap their cells.)
func (c *Context) stackGrid(w core.Widget) bool {
	fr := core.AsFrame(w)
	if fr == nil || fr.NumChildren() < 2 {
		return false
	}
	cell := ""
	for _, k := range fr.Children {
		kw, ok := k.(core.Widget)
		if !ok {
			return false
		}
		kn, _ := kw.AsWidget().Property("node").(*html.Node)
		if kn == nil {
			return false
		}
		row, col, _ := cutTopLevel(c.declaration(kn, "grid-area"), '/')
		row, col = strings.TrimSpace(row), strings.TrimSpace(col)
		if row == "" {
			// (lines with an end, as 1 / span 2, are not one cell)
			row, col = c.gridLines(kn, "row"), c.gridLines(kn, "column")
		}
		if _, err := strconv.Atoi(row); err != nil {
			return false
		}
		if col == "" {
			col = "1"
		}
		if _, err := strconv.Atoi(strings.TrimSpace(col)); err != nil {
			return false
		}
		if cell != "" && cell != row+"/"+col {
			return false
		}
		cell = row + "/" + col
	}
	// the one on top that is shown: the last that is not transparent or
	// hidden (as the name of a repository of github.com, before a copy of
	// it for the accessibility tree, of opacity 0, which hid it)
	top := fr.NumChildren() - 1
	for i := top; i >= 0; i-- {
		kn, _ := core.AsWidget(fr.Child(i)).Property("node").(*html.Node)
		op := strings.TrimSpace(c.declaration(kn, "opacity"))
		vis := strings.TrimSpace(c.declaration(kn, "visibility"))
		if op != "0" && vis != "hidden" {
			top = i
			break
		}
	}
	fr.FinalStyler(func(s *styles.Style) { s.Display = styles.Stacked })
	fr.StackTop = top
	return true
}

// gridFiller returns a new empty widget of the given grid, for a cell in
// a column of the given track that no child is in, which the caller moves
// into its place in the children.
func (c *Context) gridFiller(w core.Widget, env *lengthEnv, track string, flexible bool) *core.Frame {
	filler := core.NewFrame(w)
	filler.SetName("grid-filler")
	tree.SetUniqueName(filler) // (names of children must be unique, as for the inspector)
	filler.Styler(func(s *styles.Style) {
		s.Padding.Zero()
		s.Margin.Zero()
		s.Gap.Zero()
		s.Background = nil
		gridTrackStyle(env, s, track, flexible)
	})
	return filler
}

// gridLines returns the lines of the given element in the given axis of
// its grid (column or row) as "start / end", from its grid-column (or
// grid-row) shorthand and its -start and -end longhands, in the order of
// the cascade (as the 12 column grid of nytimes.com/wirecutter, of
// grid-column-end: span 8 and grid-column-start: 10), or "" if it has none.
func (c *Context) gridLines(n *html.Node, axis string) string {
	if n == nil {
		return ""
	}
	short := "grid-" + axis
	start, end := "", ""
	for _, d := range c.declarations(n) {
		if d.invalid {
			continue
		}
		v := strings.TrimSpace(d.Value)
		switch d.Property {
		case short:
			st, en, _ := cutTopLevel(v, '/')
			start, end = strings.TrimSpace(st), strings.TrimSpace(en)
		case short + "-start":
			start = v
		case short + "-end":
			end = v
		}
	}
	if start == "" && end == "" {
		return ""
	}
	if start == "" {
		start = "auto"
	}
	if end == "" {
		return start
	}
	return start + " / " + end
}

// gridPlacement returns the area of the given child of a grid with the
// given number of columns from the numbers of its start lines in its
// grid-row and grid-column (or grid-area), if it has both (as the cells
// of the repository grid of github.com, as grid-column: 1; grid-row: 1).
func (c *Context) gridPlacement(n *html.Node, ncols, nrows int) (gridArea, bool) {
	if n == nil {
		return gridArea{}, false
	}
	row, col := c.gridLines(n, "row"), c.gridLines(n, "column")
	if ga := c.declaration(n, "grid-area"); ga != "" && row == "" && col == "" {
		parts := strings.Split(ga, "/")
		if len(parts) >= 2 {
			row, col = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			if len(parts) >= 3 {
				row += " / " + strings.TrimSpace(parts[2])
			}
			if len(parts) >= 4 {
				col += " / " + strings.TrimSpace(parts[3])
			}
		}
	}
	start := func(v string) (int, bool) {
		s, _, _ := cutTopLevel(v, '/')
		k, err := strconv.Atoi(strings.TrimSpace(s))
		return k, err == nil && k > 0
	}
	r, okr := start(row)
	cl, okc := start(col)
	if !okc || cl > ncols {
		return gridArea{}, false
	}
	a := gridArea{row: r - 1, col: cl - 1, rows: gridSpan(row, nrows), cols: min(gridSpan(col, ncols), ncols-cl+1)}
	if !okr {
		a.row = -1 // auto: in the first row where its columns are free
	}
	return a, true
}

// placeAutoRows places the given grid items that have a column and an auto
// row (-1) in the first rows where their columns are free of the others,
// in order, as the auto-placement of CSS (as the middle column of the top
// stories of bbc.com, which was in a row of its own below the others).
func placeAutoRows[T any](items []T, area func(*T) *gridArea) {
	covered := map[[2]int]bool{}
	cover := func(a *gridArea) {
		for r := a.row; r < a.row+a.rows; r++ {
			for c := a.col; c < a.col+a.cols; c++ {
				covered[[2]int{r, c}] = true
			}
		}
	}
	for i := range items {
		if a := area(&items[i]); a.row >= 0 {
			cover(a)
		}
	}
	for i := range items {
		a := area(&items[i])
		if a.row >= 0 {
			continue
		}
		for r := 0; ; r++ {
			free := true
			for rr := r; rr < r+a.rows && free; rr++ {
				for c := a.col; c < a.col+a.cols; c++ {
					if covered[[2]int{rr, c}] {
						free = false
						break
					}
				}
			}
			if free {
				a.row = r
				break
			}
		}
		cover(a)
	}
}

// gridSpan returns the number of tracks that the given value of
// grid-column or grid-row spans: span N (with auto or a start line before
// or after it), or from a start line to an end line (with -1 the last line
// of the given number of tracks, if known), or 1.
func gridSpan(v string, ntracks int) int {
	start, end, slash := cutTopLevel(strings.ToLower(v), '/')
	start, end = strings.TrimSpace(start), strings.TrimSpace(end)
	for _, p := range []string{start, end} {
		if n, ok := strings.CutPrefix(p, "span"); ok {
			if k, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && k > 0 {
				return k
			}
		}
	}
	if !slash {
		return 1
	}
	a, errA := strconv.Atoi(start)
	b, errB := strconv.Atoi(end)
	if errA != nil || errB != nil {
		return 1
	}
	lines := ntracks + 1
	if a < 0 && ntracks > 0 {
		a += lines + 1
	}
	if b < 0 && ntracks > 0 {
		b += lines + 1
	}
	if b <= a || a <= 0 {
		return 1
	}
	return b - a
}

// arrangeGrid places the children of the given grid container widget for
// its node by its grid template (see the comment at the top of grid.go).
func (c *Context) arrangeGrid(n *html.Node, w core.Widget) {
	if c.stackGrid(w) {
		return
	}
	areas, ncols, tracks := c.gridTemplate(n)
	env := c.readEnv() // for the lengths of the tracks
	if ncols == 0 {
		return
	}
	flexible := slices.ContainsFunc(tracks, func(t string) bool { return strings.HasSuffix(t, "fr") || strings.HasSuffix(t, "fr)") })
	if fr := frameOf(w); fr != nil && len(tracks) > 0 && !slices.ContainsFunc(tracks, func(t string) bool { return !strings.HasSuffix(t, "fr") && !strings.HasSuffix(t, "fr)") }) {
		fr.FrColumns = true // all of its tracks are fr (see core.Frame.FrColumns)
	}
	// the row tracks: of grid-template-rows, or before the / of grid-template
	rowsDecl := ""
	if prop, v := c.lastDeclaration(n, "grid-template-rows", "grid-template"); prop == "grid-template" {
		if r, _, ok := cutTopLevel(v, '/'); ok && !strings.ContainsAny(r, `"'`) {
			rowsDecl = r
		}
	} else {
		rowsDecl = v
	}
	rowTracks := gridTracks(rowsDecl)
	if fr := frameOf(w); fr != nil {
		// the last row is flexible, and the others are auto (see core.Frame.FrLastRow)
		fr.FrLastRow = len(rowTracks) > 1 && strings.HasSuffix(rowTracks[len(rowTracks)-1], "fr") &&
			!slices.ContainsFunc(rowTracks[:len(rowTracks)-1], func(t string) bool { return t != "auto" && t != "min-content" && t != "max-content" })
	}
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
	// Children with a column and an auto row are placed only with others
	// placed in rows (or areas), as otherwise they are in the order of the
	// children, with the others.
	explicitRow := false
	for _, k := range w.AsTree().Children {
		if kw, ok := k.(core.Widget); ok {
			kn, _ := kw.AsWidget().Property("node").(*html.Node)
			name, _, _ := cutTopLevel(c.declaration(kn, "grid-area"), '/')
			if a, ok := c.gridPlacement(kn, ncols, len(rowTracks)); (ok && a.row >= 0) || (name != "" && areas[name] != (gridArea{})) {
				explicitRow = true
				break
			}
		}
	}
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
		} else if a, ok := c.gridPlacement(kn, ncols, len(rowTracks)); ok && (a.row >= 0 || explicitRow) {
			items = append(items, placed{kw, a})
		} else {
			auto = append(auto, k)
		}
	}
	placeAutoRows(items, func(p *placed) *gridArea { return &p.area })
	w.AsWidget().FinalStyler(func(s *styles.Style) { s.Columns = ncols })
	if len(items) == 0 {
		// The children in order, with the columns and rows that they span
		// (as the 12 columns of the grids of github.com, of which its
		// features span 6 or 4), and the tracks of the columns that they
		// are in.
		// An item with a column start (and an auto row) is placed at it,
		// in the next row if the cells before it in this one are taken
		// (as the sidebar of nytimes.com/wirecutter at column 10, after
		// the text in columns 1 to 8), with empty cells before it.
		col := 0
		var kids []tree.Node
		filled := false
		fill := func(n int) {
			for range n {
				kids = append(kids, c.gridFiller(w, env, trackOf(len(kids)%ncols), flexible))
			}
			filled = filled || n > 0
		}
		for _, k := range auto {
			kw, ok := k.(core.Widget)
			if !ok {
				kids = append(kids, k)
				continue
			}
			cs, rs := 1, 1
			if kn, _ := kw.AsWidget().Property("node").(*html.Node); kn != nil {
				// children out of the flow are in no cell (as a visually
				// hidden heading, which took the first column of the
				// testimonials of github.com, so the others were in the
				// tracks after theirs)
				pos := strings.TrimSpace(c.declaration(kn, "position"))
				if pos == "absolute" || pos == "fixed" || c.displayOf(kn) == "none" {
					continue
				}
				cols := c.gridLines(kn, "column")
				cs = gridSpan(cols, ncols)
				rs = gridSpan(c.gridLines(kn, "row"), 0)
				st, _, _ := cutTopLevel(cols, '/')
				if start, err := strconv.Atoi(strings.TrimSpace(st)); err == nil && start > 0 && start <= ncols {
					if start-1 < col { // in the next row
						fill(ncols - col)
						col = 0
					}
					fill(start - 1 - col)
					col = start - 1
					cs = min(cs, ncols-col)
				}
			}
			cs = min(cs, ncols)
			if col > 0 && col+cs > ncols { // it starts the next row
				col = 0
			}
			kids = append(kids, k)
			i := col
			kw.AsWidget().FinalStyler(func(s *styles.Style) {
				if cs > 1 {
					s.ColSpan = cs
				} else if len(tracks) > 0 {
					gridTrackStyle(env, s, trackOf(i), flexible)
				}
				if rs > 1 {
					s.RowSpan = rs
				}
			})
			col = (col + cs) % ncols
		}
		if filled && col > 0 {
			// the flexible tracks after the last item take their share
			for ; col < ncols && strings.HasSuffix(trackOf(col), "fr"); col++ {
				fill(1)
			}
		}
		if filled {
			w.AsTree().Children = kids
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
			if r == last.row && col > last.col && !strings.HasSuffix(trackOf(col), "fr") {
				// the cells after the last item are empty, but those of
				// flexible tracks take their share of the width (as the
				// third column of 1fr 1024px 1fr of nytimes.com/wirecutter,
				// whose space went to the first)
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
			kids = append(kids, c.gridFiller(w, env, trackOf(col), flexible)) // moved into its place below
		}
	}
	for ; next < len(items); next++ { // items that share a start with another
		auto = append(auto, items[next].w)
	}
	w.AsTree().Children = append(kids, auto...)
}
