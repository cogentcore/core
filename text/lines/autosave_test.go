// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestAutosaveConcurrentEdits checks that the autosave, which runs in a
// goroutine after each edit, does not read the lines while later edits
// change them, as when applying the multiple edits of a code action
// (this crashed with an index out of range; run with -race).
func TestAutosaveConcurrentEdits(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "x.txt")
	src := strings.Repeat("line of text\n", 500)
	os.WriteFile(fname, []byte(src), 0644)
	ls := NewLines()
	ls.SetFilename(fname)
	ls.SetText([]byte(src))
	ls.Autosave = true
	for i := range 400 {
		ln := ls.NumLines() - 2
		if i%2 == 0 { // delete a line near the end
			ls.ReplaceText(textpos.Pos{Line: ln}, textpos.Pos{Line: ln + 1}, textpos.Pos{Line: ln}, "", false)
		} else { // and add one back
			ls.ReplaceText(textpos.Pos{Line: ln}, textpos.Pos{Line: ln}, textpos.Pos{Line: ln}, "new\n", false)
		}
	}
	assert.Equal(t, 500, ls.NumLines())
}

// TestReplaceTexts checks a batch of replacements, which is undone as one.
func TestReplaceTexts(t *testing.T) {
	ls := NewLines()
	ls.SetText([]byte("one two three\nfour five\n"))
	ls.Autosave = true // no filename: autosave is not written
	n := ls.ReplaceTexts([]Replacement{ // reverse position order
		{Region: textpos.Region{Start: textpos.Pos{Line: 1, Char: 5}, End: textpos.Pos{Line: 1, Char: 9}}, Text: "5"},
		{Region: textpos.Region{Start: textpos.Pos{Line: 0, Char: 4}, End: textpos.Pos{Line: 0, Char: 7}}, Text: "2"},
		{Region: textpos.Region{Start: textpos.Pos{Line: 0, Char: 0}, End: textpos.Pos{Line: 0, Char: 3}}, Text: "1"},
	}, ReplaceNoMatchCase)
	assert.Equal(t, 3, n)
	assert.Equal(t, "1 2 three\nfour 5\n", ls.String())
	ls.Undo()
	assert.Equal(t, "one two three\nfour five\n", ls.String())
}
