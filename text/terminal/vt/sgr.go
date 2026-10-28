// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vt

import "github.com/charmbracelet/x/ansi"

// handleSGR handles the Select Graphic Rendition sequence,
// which sets the style of the following text.
func (em *Emulator) handleSGR(params ansi.Params) {
	st := &em.Cursor.Style
	if len(params) == 0 {
		st.Fg, st.Bg, st.Attrs = ColorDefault, ColorDefault, 0
		return
	}
	for i := 0; i < len(params); i++ {
		p, more, _ := params.Param(i, 0)
		switch {
		case p == 0:
			st.Fg, st.Bg, st.Attrs = ColorDefault, ColorDefault, 0
		case p == 1:
			st.Attrs |= Bold
		case p == 2:
			st.Attrs |= Faint
		case p == 3:
			st.Attrs |= Italic
		case p == 4:
			st.Attrs &^= Underline | DoubleUnderline
			if more { // 4:n underline style
				i++
				if u, _, _ := params.Param(i, 1); u == 0 {
					break
				} else if u == 2 {
					st.Attrs |= DoubleUnderline
					break
				}
			}
			st.Attrs |= Underline
		case p == 5 || p == 6:
			st.Attrs |= Blink
		case p == 7:
			st.Attrs |= Reverse
		case p == 8:
			st.Attrs |= Invisible
		case p == 9:
			st.Attrs |= Strikethrough
		case p == 21:
			st.Attrs |= DoubleUnderline
		case p == 22:
			st.Attrs &^= Bold | Faint
		case p == 23:
			st.Attrs &^= Italic
		case p == 24:
			st.Attrs &^= Underline | DoubleUnderline
		case p == 25:
			st.Attrs &^= Blink
		case p == 27:
			st.Attrs &^= Reverse
		case p == 28:
			st.Attrs &^= Invisible
		case p == 29:
			st.Attrs &^= Strikethrough
		case p >= 30 && p <= 37:
			st.Fg = IndexedColor(uint8(p - 30))
		case p == 38:
			c, n, ok := extendedColor(params, i)
			if ok {
				st.Fg = c
			}
			i += n
		case p == 39:
			st.Fg = ColorDefault
		case p >= 40 && p <= 47:
			st.Bg = IndexedColor(uint8(p - 40))
		case p == 48:
			c, n, ok := extendedColor(params, i)
			if ok {
				st.Bg = c
			}
			i += n
		case p == 49:
			st.Bg = ColorDefault
		case p == 53:
			st.Attrs |= Overline
		case p == 55:
			st.Attrs &^= Overline
		case p == 58: // underline color: not supported, skip its parameters
			_, n, _ := extendedColor(params, i)
			i += n
		case p >= 90 && p <= 97:
			st.Fg = IndexedColor(uint8(p - 90 + 8))
		case p >= 100 && p <= 107:
			st.Bg = IndexedColor(uint8(p - 100 + 8))
		}
	}
}

// extendedColor parses the extended color starting at parameter i
// (38, 48, or 58), in either the 38;5;n / 38;2;r;g;b form or the
// 38:5:n / 38:2::r:g:b form. It returns the color, the number of
// following parameters that it used, and whether it is valid.
func extendedColor(params ansi.Params, i int) (Color, int, bool) {
	if i+1 >= len(params) {
		return ColorDefault, 0, false
	}
	colon := params[i].HasMore()
	kind := params[i+1].Param(0)
	var vals []int
	if colon {
		for j := i + 2; j < len(params) && params[j-1].HasMore(); j++ {
			vals = append(vals, max(0, min(params[j].Param(0), 255)))
		}
	} else {
		nv := 3
		if kind == 5 {
			nv = 1
		}
		for j := i + 2; j < len(params) && len(vals) < nv; j++ {
			vals = append(vals, max(0, min(params[j].Param(0), 255)))
		}
	}
	n := 1 + len(vals)
	switch {
	case kind == 5 && len(vals) >= 1:
		return IndexedColor(uint8(vals[0])), n, true
	case kind == 2 && len(vals) >= 4 && colon: // with the color space
		return RGBColor(uint8(vals[1]), uint8(vals[2]), uint8(vals[3])), n, true
	case kind == 2 && len(vals) >= 3:
		return RGBColor(uint8(vals[0]), uint8(vals[1]), uint8(vals[2])), n, true
	}
	return ColorDefault, n, false
}
