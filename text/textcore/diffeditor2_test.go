// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// linesText returns the lines of the given lines of a diff editor, with
// trailing spaces trimmed (the padding of aligned lines).
func linesText(n int, line func(i int) []rune) []string {
	var s []string
	for i := range n {
		s = append(s, strings.TrimRight(string(line(i)), " "))
	}
	return s
}

// TestDiffStringsAligned tests that the lines of the two sides of a diff
// editor are aligned for each kind of difference: a deletion, an insertion,
// and replacements of more and of fewer lines, with blank lines added
// where one side has fewer lines, and applying them in both directions
// with the toolbar, and undoing them.
func TestDiffStringsAligned(t *testing.T) {
	b := core.NewBody()
	de := NewDiffEditor(b)
	tb := core.NewToolbar(b)
	tb.Maker(de.MakeToolbar)
	astr := []string{"same", "only a", "same 2", "a1", "a2", "same 3", "r1", "end"}
	bstr := []string{"same", "same 2", "b1", "same 3", "s1", "s2", "s3", "end", "only b"}
	de.DiffStrings(astr, bstr)
	core.RenderImage(b, core.Locked(b, func() {
		na, nb := de.linesA.NumLines(), de.linesB.NumLines()
		require.Equal(t, na, nb, "aligned")
		a := linesText(na, de.linesA.Line)
		bl := linesText(nb, de.linesB.Line)
		assert.Equal(t, []string{"same", "only a", "same 2", "a1", "a2", "same 3", "r1", "", "", "end", ""}, a)
		assert.Equal(t, []string{"same", "", "same 2", "b1", "", "same 3", "s1", "s2", "s3", "end", "only b"}, bl)
		for i := range de.diffs.Diffs {
			ad := de.alignD[i]
			assert.LessOrEqual(t, ad.I1, ad.I2)
			assert.LessOrEqual(t, ad.J1, ad.J2)
		}

		// apply the deletion of "only a" from B to A (line 1), and back
		tva, tvb := de.textEditors()
		tva.SetCursorShow(textpos.Pos{Line: 1})
		buttons := []*core.Button{}
		for cw := range tb.AllWidgets() {
			if bt, ok := cw.(*core.Button); ok {
				buttons = append(buttons, bt)
			}
		}
		click := func(text string, nth int) {
			t.Helper()
			for _, bt := range buttons {
				if bt.Text == text {
					if nth == 0 {
						bt.Send(events.Click)
						return
					}
					nth--
				}
			}
			t.Fatalf("no button %q", text)
		}
		click("A &lt;- B", 0) // (the text is html)
		assert.Equal(t, "", strings.TrimSpace(string(de.linesA.Line(1))), "A gets B's blank line")
		click("Undo", 0)
		assert.Equal(t, "only a", string(de.linesA.Line(1)))

		tvb.SetCursorShow(textpos.Pos{Line: 10})
		click("A -&gt; B", 0)
		assert.Equal(t, "", strings.TrimSpace(string(de.linesB.Line(10))), "B gets A's blank line")
		click("Undo", 1)
		assert.Equal(t, "only b", string(de.linesB.Line(10)))
		assert.Error(t, de.undoDiff(1), "nothing more to undo in B")

		click("Next", 0)
		click("Prev", 1)
	}))
}
