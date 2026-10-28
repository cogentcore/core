// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"image/color"
	"testing"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/terminal/vt"
	"github.com/stretchr/testify/assert"
)

// spans returns the text of the spans of the given markup, with the
// cursor span in brackets.
func spans(cm *colorMap, tx rich.Text) []string {
	var ss []string
	for si := range tx.NumSpans() {
		sty, rs := tx.Span(si)
		s := string(rs)
		if sty.Background() == cm.cursor {
			s = "[" + s + "]"
		}
		ss = append(ss, s)
	}
	return ss
}

func TestLineMarkupCursor(t *testing.T) {
	cm := &colorMap{base: rich.NewStyle(), palette: &LightPalette, cursor: color.RGBA{255, 0, 0, 255}}
	cells := func(s string) []vt.Cell {
		var cs []vt.Cell
		for _, r := range s {
			cs = append(cs, vt.Cell{Rune: r, Width: 1})
		}
		for len(cs) < 10 {
			cs = append(cs, vt.Cell{Width: 1})
		}
		return cs
	}
	rs, tx := cm.lineMarkup(cells("$ ls"), 3, 2)
	assert.Equal(t, "$ ls", string(rs))
	assert.Equal(t, []string{"$ ", "[l]", "s"}, spans(cm, tx))

	rs, tx = cm.lineMarkup(cells("$ "), 3, 2) // after the text
	assert.Equal(t, "$  ", string(rs))
	assert.Equal(t, []string{"$ ", "[ ]"}, spans(cm, tx))

	rs, tx = cm.lineMarkup(cells("$ ls"), 1, 0)
	assert.Equal(t, []string{"[$]", " ls"}, spans(cm, tx))

	rs, tx = cm.lineMarkup(cells("$ ls"), 3, -1) // hidden
	assert.Equal(t, "$ ls", string(rs))
	assert.Equal(t, []string{"$ ls"}, spans(cm, tx))
}
