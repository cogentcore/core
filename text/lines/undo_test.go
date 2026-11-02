// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"strings"
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// undoLines returns lines with the text "a", in which "b" was typed and
// saved, and then "c" was typed, as separate undo groups.
func undoLines(emacs bool) *Lines {
	ls := NewLines()
	ls.Settings.EmacsUndo = emacs
	ls.SetString("a")
	ls.InsertText(textpos.Pos{Char: 1}, []rune("b"))
	ls.NewUndoGroup()
	ls.ClearNotSaved() // as when saved
	ls.InsertText(textpos.Pos{Char: 2}, []rune("c"))
	ls.NewUndoGroup()
	return ls
}

// state returns the text of the lines and whether it is not saved.
func state(ls *Lines) (string, bool) {
	return strings.TrimSpace(ls.String()), ls.IsNotSaved()
}

// TestUndoSaved tests that undoing and redoing edits back to the text
// as it was saved makes it saved, while the undo history before the
// save is kept.
func TestUndoSaved(t *testing.T) {
	ls := undoLines(false)
	check := func(text string, notSaved bool) {
		t.Helper()
		txt, ns := state(ls)
		assert.Equal(t, text, txt)
		assert.Equal(t, notSaved, ns, text)
	}
	check("abc", true)
	ls.Undo()
	check("ab", false) // the saved text
	ls.Undo()
	check("a", true) // before the save, which can still be undone
	ls.Undo()
	check("a", true)
	ls.Redo()
	check("ab", false)
	ls.Redo()
	check("abc", true)

	// a new edit after undoing to before the save
	ls.Undo()
	ls.Undo()
	ls.InsertText(textpos.Pos{Char: 1}, []rune("b"))
	ls.NewUndoGroup()
	check("ab", true) // the same text, but a new edit
	ls.Undo()
	check("a", true)
}

// TestUndoSavedEmacs tests that, in the emacs undo mode, undoing the undos
// past the save also makes the text saved again.
func TestUndoSavedEmacs(t *testing.T) {
	ls := undoLines(true)
	check := func(text string, notSaved bool) {
		t.Helper()
		txt, ns := state(ls)
		assert.Equal(t, text, txt)
		assert.Equal(t, notSaved, ns, text)
	}
	ls.Undo()
	check("ab", false)
	ls.Undo()
	check("a", true)
	ls.EmacsUndoSave() // the end of the undos, as on another command
	ls.Undo()          // undoes the undo of b
	check("ab", false)
	ls.Undo() // undoes the undo of c
	check("abc", true)
	ls.Undo()
	check("ab", false)
}
