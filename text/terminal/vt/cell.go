// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import "fmt"

// Color is a terminal color: the default color, an index into the
// 256 color palette, or a 24-bit RGB value.
type Color uint32

const (
	// ColorDefault is the default foreground or background color.
	ColorDefault Color = 0

	colorIndexed Color = 1 << 24
	colorRGB     Color = 2 << 24
	colorKind    Color = 0xff << 24
)

// IndexedColor returns the color for the given palette index:
// 0-7 are the standard colors, 8-15 the bright ones, 16-231 a
// 6x6x6 color cube, and 232-255 a grayscale ramp.
func IndexedColor(i uint8) Color {
	return colorIndexed | Color(i)
}

// RGBColor returns the 24-bit color with the given components.
func RGBColor(r, g, b uint8) Color {
	return colorRGB | Color(r)<<16 | Color(g)<<8 | Color(b)
}

// IsDefault returns whether this is the default color.
func (c Color) IsDefault() bool {
	return c == ColorDefault
}

// Index returns the palette index of an indexed color, and whether it is one.
func (c Color) Index() (uint8, bool) {
	return uint8(c & 0xff), c&colorKind == colorIndexed
}

// RGB returns the components of a 24-bit color, and whether it is one.
func (c Color) RGB() (r, g, b uint8, ok bool) {
	return uint8(c >> 16), uint8(c >> 8), uint8(c), c&colorKind == colorRGB
}

func (c Color) String() string {
	if i, ok := c.Index(); ok {
		return fmt.Sprintf("%d", i)
	}
	if r, g, b, ok := c.RGB(); ok {
		return fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}
	return "default"
}

// Attrs are the text attributes of a cell, as bit flags.
type Attrs uint16

const (
	Bold Attrs = 1 << iota
	Faint
	Italic
	Underline
	DoubleUnderline
	Blink
	Reverse
	Invisible
	Strikethrough
	Overline
)

// Style is the style of a cell.
type Style struct {
	// Fg is the foreground (text) color.
	Fg Color

	// Bg is the background color.
	Bg Color

	// Attrs are the text attributes.
	Attrs Attrs

	// Link is the 1-based index of the hyperlink of the cell
	// in [Emulator.Links], or 0 for none.
	Link uint32
}

// IsDefault returns whether this is the default style.
func (s Style) IsDefault() bool {
	return s == Style{}
}

// Cell is one character cell in the terminal grid.
type Cell struct {
	// Rune is the character in the cell, or 0 for an empty cell,
	// which is shown as a space. A wide character, such as a CJK
	// ideograph, takes two cells: the second has Rune 0 and Width 0.
	Rune rune

	// Width is the number of cells that the character takes:
	// 1 normally, 2 for a wide character, and 0 for the cell
	// after a wide character.
	Width uint8

	// Style is the style of the cell.
	Style Style
}

// blankCell returns an empty cell with the given background color,
// as used for erasing.
func blankCell(bg Color) Cell {
	return Cell{Width: 1, Style: Style{Bg: bg}}
}

// IsBlank returns whether the cell is empty with the default style.
func (c *Cell) IsBlank() bool {
	return (c.Rune == 0 || c.Rune == ' ') && c.Width == 1 && c.Style.IsDefault()
}

// Line is one row of cells in the terminal grid.
type Line struct {
	// Cells are the cells in the line.
	Cells []Cell

	// Wrapped is whether the text of this line continues on the next
	// line, because it was wrapped at the end of the line (a soft wrap),
	// rather than ending with a line feed.
	Wrapped bool
}

// newLine returns a new blank line with the given number of cells
// and background color.
func newLine(cols int, bg Color) *Line {
	ln := &Line{Cells: make([]Cell, cols)}
	ln.clear(0, cols, bg)
	return ln
}

// clear clears the cells from st up to ed with the given background color.
func (ln *Line) clear(st, ed int, bg Color) {
	bc := blankCell(bg)
	for i := st; i < ed; i++ {
		ln.Cells[i] = bc
	}
}

// Len returns the length of the line without trailing blank cells.
func (ln *Line) Len() int {
	n := len(ln.Cells)
	for n > 0 && ln.Cells[n-1].IsBlank() {
		n--
	}
	return n
}

// String returns the text of the line, without the trailing blank cells.
func (ln *Line) String() string {
	return string(ln.Runes(ln.Len()))
}

// Runes returns the runes of the first n cells of the line,
// with a space for each empty cell, and nothing for the cell after
// a wide character.
func (ln *Line) Runes(n int) []rune {
	rs := make([]rune, 0, n)
	for i := range min(n, len(ln.Cells)) {
		c := &ln.Cells[i]
		switch {
		case c.Width == 0:
		case c.Rune == 0:
			rs = append(rs, ' ')
		default:
			rs = append(rs, c.Rune)
		}
	}
	return rs
}

// clone returns a copy of the line.
func (ln *Line) clone() *Line {
	return &Line{Cells: append([]Cell(nil), ln.Cells...), Wrapped: ln.Wrapped}
}

// resize changes the number of cells in the line to n,
// adding blank cells as needed.
func (ln *Line) resize(n int) {
	cur := len(ln.Cells)
	if n <= cur {
		ln.Cells = ln.Cells[:n]
		if n > 0 && ln.Cells[n-1].Width == 2 { // half of a wide character
			ln.Cells[n-1] = blankCell(ln.Cells[n-1].Style.Bg)
		}
		return
	}
	ln.Cells = append(ln.Cells, make([]Cell, n-cur)...)
	ln.clear(cur, n, ColorDefault)
}
