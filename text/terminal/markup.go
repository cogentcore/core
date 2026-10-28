// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/terminal/vt"
)

// wideFiller is the character used for the cell after a wide character,
// so that each cell has one character, which keeps the columns aligned.
// It is a zero width space, which is removed from copied text.
const wideFiller = '​'

// Palette is the set of 16 standard terminal colors: black, red, green,
// yellow, blue, magenta, cyan, white, and their bright versions.
type Palette [16]color.RGBA

var (
	// DarkPalette is the palette used with a dark color scheme.
	DarkPalette = Palette{
		{0x00, 0x00, 0x00, 0xff}, {0xcd, 0x31, 0x31, 0xff}, {0x0d, 0xbc, 0x79, 0xff}, {0xe5, 0xe5, 0x10, 0xff},
		{0x24, 0x72, 0xc8, 0xff}, {0xbc, 0x3f, 0xbc, 0xff}, {0x11, 0xa8, 0xcd, 0xff}, {0xe5, 0xe5, 0xe5, 0xff},
		{0x66, 0x66, 0x66, 0xff}, {0xf1, 0x4c, 0x4c, 0xff}, {0x23, 0xd1, 0x8b, 0xff}, {0xf5, 0xf5, 0x43, 0xff},
		{0x3b, 0x8e, 0xea, 0xff}, {0xd6, 0x70, 0xd6, 0xff}, {0x29, 0xb8, 0xdb, 0xff}, {0xff, 0xff, 0xff, 0xff},
	}

	// LightPalette is the palette used with a light color scheme.
	LightPalette = Palette{
		{0x00, 0x00, 0x00, 0xff}, {0xcd, 0x31, 0x31, 0xff}, {0x00, 0xa0, 0x00, 0xff}, {0x94, 0x98, 0x00, 0xff},
		{0x04, 0x51, 0xa5, 0xff}, {0xbc, 0x05, 0xbc, 0xff}, {0x05, 0x98, 0xbc, 0xff}, {0x80, 0x80, 0x80, 0xff},
		{0x66, 0x66, 0x66, 0xff}, {0xe0, 0x30, 0x30, 0xff}, {0x14, 0xb8, 0x14, 0xff}, {0xb5, 0xba, 0x00, 0xff},
		{0x04, 0x51, 0xa5, 0xff}, {0xbc, 0x05, 0xbc, 0xff}, {0x05, 0x98, 0xbc, 0xff}, {0xa5, 0xa5, 0xa5, 0xff},
	}
)

// colorMap converts terminal colors and styles to rich text styles.
type colorMap struct {
	// palette is the palette of the 16 standard colors.
	palette *Palette

	// fg and bg are the default foreground and background colors.
	fg, bg color.RGBA

	// base is the base text style.
	base *rich.Style

	// links are the hyperlink URLs.
	links []string

	// cursor is the color of the cursor, which is blended with
	// the background of the cell that it is on.
	cursor color.RGBA

	// cursorFocused is whether the view with the cursor is focused,
	// in which case the cursor is blended, and otherwise it is
	// [colorMap.cursor] as is.
	cursorFocused bool
}

// newColorMap returns a new color map for the current color scheme.
func newColorMap(base *rich.Style, links []string) *colorMap {
	cm := &colorMap{base: base, links: links}
	cm.palette = &LightPalette
	if matcolor.SchemeIsDark {
		cm.palette = &DarkPalette
	}
	cm.fg = colors.ToUniform(colors.Scheme.OnSurface)
	cm.bg = colors.ToUniform(colors.Scheme.Surface)
	return cm
}

// color returns the RGB color for the given terminal color,
// which must not be the default.
func (cm *colorMap) color(c vt.Color) color.RGBA {
	if r, g, b, ok := c.RGB(); ok {
		return color.RGBA{r, g, b, 0xff}
	}
	i, _ := c.Index()
	switch {
	case i < 16:
		return cm.palette[i]
	case i < 232: // 6x6x6 cube
		i -= 16
		lev := func(v uint8) uint8 {
			if v == 0 {
				return 0
			}
			return 55 + v*40
		}
		return color.RGBA{lev(i / 36), lev((i / 6) % 6), lev(i % 6), 0xff}
	default: // grayscale
		v := 8 + (i-232)*10
		return color.RGBA{v, v, v, 0xff}
	}
}

// style returns the rich text style for the given cell style.
func (cm *colorMap) style(st vt.Style) *rich.Style {
	rs := *cm.base
	at := st.Attrs
	if at&vt.Bold != 0 {
		rs.Weight = rich.Bold
	}
	if at&vt.Italic != 0 {
		rs.Slant = rich.Italic
	}
	if at&(vt.Underline|vt.DoubleUnderline) != 0 {
		rs.Decoration.SetFlag(true, rich.Underline)
	}
	if at&vt.Strikethrough != 0 {
		rs.Decoration.SetFlag(true, rich.LineThrough)
	}
	if at&vt.Overline != 0 {
		rs.Decoration.SetFlag(true, rich.Overline)
	}
	if at&(vt.Reverse|vt.Faint|vt.Invisible) == 0 && st.Fg.IsDefault() && st.Bg.IsDefault() {
		return cm.link(&rs, st)
	}
	fg, bg := cm.fg, cm.bg
	if !st.Fg.IsDefault() {
		fg = cm.color(st.Fg)
	}
	if !st.Bg.IsDefault() {
		bg = cm.color(st.Bg)
	}
	if at&vt.Reverse != 0 {
		fg, bg = bg, fg
	}
	if at&vt.Faint != 0 {
		fg = colors.Blend(colors.RGB, 50, fg, bg)
	}
	if at&vt.Invisible != 0 {
		fg = bg
	}
	if fg != cm.fg {
		rs.SetFillColor(fg)
	}
	if bg != cm.bg || at&vt.Reverse != 0 {
		rs.SetBackground(bg)
	}
	return cm.link(&rs, st)
}

// link sets the link of the style, if the cell has one.
func (cm *colorMap) link(rs *rich.Style, st vt.Style) *rich.Style {
	if st.Link == 0 || int(st.Link) > len(cm.links) {
		return rs
	}
	rs.URL = cm.links[st.Link-1]
	rs.Special = rich.Link
	return rs
}

// lineMarkup returns the text and markup for the given cells, which
// has one character for each cell, without the trailing blank cells,
// but with at least minLen characters (for the cursor). The cursor is
// shown on the cell at the given index, unless it is negative: it is
// drawn as the background of the cell, which fills the cell exactly.
func (cm *colorMap) lineMarkup(cells []vt.Cell, minLen, cursor int) ([]rune, rich.Text) {
	n := len(cells)
	for n > 0 && cells[n-1].IsBlank() {
		n--
	}
	n = max(n, min(minLen, len(cells)))
	rs := make([]rune, n, max(n, minLen))
	for i := range n {
		c := &cells[i]
		switch {
		case c.Width == 0:
			rs[i] = wideFiller
		case c.Rune == 0:
			rs[i] = ' '
		default:
			rs[i] = c.Rune
		}
	}
	for len(rs) < minLen {
		rs = append(rs, ' ')
	}
	var tx rich.Text
	addSpan := func(sty *rich.Style, start, end int) {
		if end <= start {
			return
		}
		if sty.Special == rich.Link {
			tx.StartSpecial(sty, rich.Link, rs[start:end])
			tx.EndSpecial()
		} else {
			tx.AddSpan(sty, rs[start:end])
		}
	}
	// the cells, with the cursor cell as its own span
	cellStyle := func(i int) vt.Style {
		if i < n {
			return cells[i].Style
		}
		return vt.Style{}
	}
	start := 0
	for i := 1; i <= len(rs); i++ {
		if i < len(rs) && i != cursor && i-1 != cursor && cellStyle(i) == cellStyle(start) {
			continue
		}
		sty := cm.style(cellStyle(start))
		if start == cursor {
			sty = cm.cursorStyle(sty)
		}
		addSpan(sty, start, i)
		start = i
	}
	if len(tx) == 0 {
		tx = rich.NewText(cm.base, rs)
	}
	return rs, tx
}

// cursorStyle returns the given style with the cursor background.
func (cm *colorMap) cursorStyle(sty *rich.Style) *rich.Style {
	cs := *sty
	clr := cm.cursor
	if cm.cursorFocused {
		bg := cm.bg
		if c := sty.Background(); c != nil {
			bg = colors.AsRGBA(c)
		}
		clr = colors.Blend(colors.RGB, 50, cm.cursor, bg)
	}
	cs.SetBackground(clr)
	return &cs
}
