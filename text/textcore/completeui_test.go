// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goSrc is a small Go file for completion.
const goSrc = `package a

// Hello says hello.
func Hello() {}

func use() {
	He
}
`

// TestEditorCompleteParse tests the completer of a Go file, which
// completes and looks up the symbols of the file.
func TestEditorCompleteParse(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(fn, []byte(goSrc), 0o644))
	b := core.NewBody()
	ed := NewEditor(b)
	require.NoError(t, ed.Lines.Open(fn))
	core.RenderImage(b, core.Locked(b, func() {
		ed.UpdateNewFile()
		require.NotNil(t, ed.Complete, "a completer for a known language")
		_, ps := ed.Lines.ParseState()
		ed.Lines.ReMarkup()
		md := completeParse(ps, "\tHe", 6, 3)
		var texts []string
		for _, c := range md.Matches {
			texts = append(texts, c.Text)
		}
		if len(texts) > 0 {
			assert.Contains(t, texts, "Hello()")
			// the completions are shown, as for typing, and one is chosen
			ed.SetCursorShow(pos(6, 3))
			ed.offerComplete()
			ed.completeText("Hello()")
			assert.Equal(t, "\tHello()", string(ed.Lines.Line(6)))
		}
		lookupParse(ps, "\tHello", 6, 6)
		assert.Empty(t, completeParse((*parse.FileStates)(nil), "", 0, 0).Matches)
		assert.Equal(t, complete.Lookup{}, lookupParse((*parse.FileStates)(nil), "", 0, 0))
		assert.Equal(t, complete.Edit{}, completeEditParse((*parse.FileStates)(nil), "", 0, complete.Completion{}, ""))

		ed.SetCursorShow(pos(6, 3))
		ed.offerComplete()
		ed.CancelComplete()
		ed.SetCursorShow(pos(2, 5))
		ed.offerComplete() // in a comment: nothing
		ed.Lookup()
		ed.SelectRegion.Start = pos(3, 5)
		ed.SelectRegion.End = pos(3, 10)
		ed.Lookup()
		ed.SelectRegion.End = pos(4, 1)
		ed.Lookup() // no multi-line lookup

		// the same data keeps the completer, and other data replaces it
		c := ed.Complete
		ed.setCompleter(ps, completeParse, completeEditParse, lookupParse)
		assert.Same(t, c, ed.Complete)
		ed.setCompleter(nil, completeParse, completeEditParse, lookupParse)
		assert.NotSame(t, c, ed.Complete)
		ed.deleteCompleter()
		assert.Nil(t, ed.Complete)
		ed.offerComplete()
		ed.Lookup()
		ed.CancelComplete()
	}))
}
