// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestEditorWide tests wide chars, such as CJK chars, which take two
// cells: the chars after them, the cursor, clicks, the column for moving
// up and down, and the selection, are all in cells.
func TestEditorWide(t *testing.T) {
	b := core.NewBody()
	ed := newWideEditor(b)
	ed.Lines.SetString("ab中文cd\n中文中文\tx\n")
	b.AssertRender(t, "wide", func() {
		ed.SetCursorShow(textpos.Pos{Line: 1, Char: 2})
		ed.SelectRegion = textpos.Region{Start: textpos.Pos{Line: 0, Char: 2}, End: textpos.Pos{Line: 0, Char: 4}}
	}, func() {
		cw := ed.charSize.X
		x := func(ln, ch int) float32 {
			return ed.CharStartPos(textpos.Pos{Line: ln, Char: ch}).X
		}
		x0 := x(0, 0)
		for ch, cell := range []int{0, 1, 2, 4, 6, 7, 8} {
			assert.Equal(t, float32(cell)*cw, x(0, ch)-x0, "char %d", ch)
		}
		// the tab after the wide chars goes to the tab stop
		assert.Equal(t, 16*cw, x(1, 5)-x0)

		// a click in the first half of a wide char is before it,
		// and in its second half after it
		csp := ed.CharStartPos(textpos.Pos{Line: 0, Char: 2})
		assert.Equal(t, textpos.Pos{Line: 0, Char: 2}, ed.PixelToCursor(clickPoint(&ed.Base, csp)))
		csp.X += 1.5 * cw
		assert.Equal(t, textpos.Pos{Line: 0, Char: 3}, ed.PixelToCursor(clickPoint(&ed.Base, csp)))
		csp = ed.CharStartPos(textpos.Pos{Line: 1, Char: 5})
		assert.Equal(t, textpos.Pos{Line: 1, Char: 5}, ed.PixelToCursor(clickPoint(&ed.Base, csp)))
		csp.X -= 2 * cw // in the tab
		assert.Equal(t, textpos.Pos{Line: 1, Char: 5}, ed.PixelToCursor(clickPoint(&ed.Base, csp)))

		// moving up keeps the column in cells
		ed.setCursorColumn(ed.CursorPos)
		assert.Equal(t, 4, ed.cursorColumn)
		ed.cursorUp(1)
		assert.Equal(t, textpos.Pos{Line: 0, Char: 3}, ed.CursorPos)
		ed.cursorDown(1)
		assert.Equal(t, textpos.Pos{Line: 1, Char: 2}, ed.CursorPos)
	})
}
