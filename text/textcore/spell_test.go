// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/spell"
	"cogentcore.org/core/text/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasSpellErr returns whether the given line has a spelling error tag.
func hasSpellErr(ed *Editor, ln int) bool {
	for _, t := range ed.Lines.AdjustedTags(ln) {
		if t.Token.Token == token.TextSpellErr {
			return true
		}
	}
	return false
}

// spellTest makes an editor of a markdown document with the given text,
// in which spelling is checked, and calls the given function with it.
func spellTest(t *testing.T, text string, fun func(ed *Editor)) {
	t.Helper()
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetLanguage(fileinfo.Markdown)
	ed.Lines.SetString(text)
	ed.Styler(func(s *styles.Style) {
		s.Min.X.Em(30)
		s.Min.Y.Em(10)
	})
	core.RenderImage(b, core.Locked(b, func() {
		ed.SetFocus()
		require.NotNil(t, ed.spell)
		if spell.Spell == nil {
			t.Skip("no spell checker on this platform")
		}
		fun(ed)
	}))
}

// TestSpellTyping tests that typing after a misspelled word marks it, in
// a document, and that a known word is not marked.
func TestSpellTyping(t *testing.T) {
	spellTest(t, "this is a tset\nthe end\n", func(ed *Editor) {
		assert.True(t, ed.isSpellEnabled(pos(0, 0)))
		ed.SetCursorShow(pos(0, 14))
		core.SendKey(ed, ' ', key.CodeSpacebar)
		assert.True(t, hasSpellErr(ed, 0), "the misspelled word is marked")
		assert.False(t, hasSpellErr(ed, 1))

		// the other keys that check the word before them
		ed.SetCursorShow(pos(1, 3))
		core.SendKeyFunction(ed, keymap.MoveRight)
		core.SendKeyFunction(ed, keymap.MoveUp)
		core.SendKeyFunction(ed, keymap.MoveDown)
		ed.SetCursorShow(pos(1, 7))
		core.SendKeyFunction(ed, keymap.Enter)
		core.SendKeyFunction(ed, keymap.Backspace)
		core.SendKeyFunction(ed, keymap.MoveRight)
		assert.False(t, hasSpellErr(ed, 1), "known words are not marked")

		errs := ed.SpellCheckLineErrors(0)
		assert.NotEmpty(t, errs, "the line has a misspelled word")
		assert.Nil(t, ed.SpellCheckLineErrors(99))
	})
}

// TestSpellCorrect tests offering corrections for a misspelled word,
// correcting it, and learning, unlearning, and ignoring words.
func TestSpellCorrect(t *testing.T) {
	spellTest(t, "a tset here\n", func(ed *Editor) {
		ed.SetCursorShow(pos(0, 3))
		require.True(t, ed.OfferCorrect(), "a misspelled word has corrections")
		assert.NotNil(t, ed.spell.stage, "shown in a popup")
		assert.Equal(t, "tset", ed.spell.word)
		ed.spell.spell("test")
		assert.Equal(t, "a test here", string(ed.Lines.Line(0)))
		assert.Nil(t, ed.spell.stage, "closed")

		ed.SetCursorShow(pos(0, 3))
		assert.False(t, ed.OfferCorrect(), "a known word has none")

		sp := ed.spell
		sp.setWord("zqxwv", nil, 0, 0)
		sp.learnWord()
		assert.True(t, sp.isLastLearned("ZQXWV"))
		_, known := sp.checkWord("zqxwv")
		assert.True(t, known, "a learned word is known")
		sp.unLearnLast()
		assert.False(t, sp.isLastLearned("zqxwv"))
		sp.unLearnLast() // nothing to unlearn: logged
		sp.setWord("qqzzyy", nil, 0, 0)
		sp.ignoreWord()
		_, known = sp.checkWord("qqzzyy")
		assert.True(t, known, "an ignored word is known")

		sp.show("tset", ed.Scene, ed.Geom.ContentBBox.Min)
		assert.NotNil(t, sp.stage)
		ed.cancelCorrect()
		assert.Nil(t, sp.stage)
		assert.False(t, sp.cancel(), "nothing to cancel")
	})
}

// TestSpellCode tests that spelling is checked only in the comments and
// strings of code.
func TestSpellCode(t *testing.T) {
	b := core.NewBody()
	ed := NewEditor(b)
	ed.Lines.SetLanguage(fileinfo.Go)
	ed.Lines.SetString("package a\n\n// a tset comment\nvar x = 1\n")
	core.RenderImage(b, core.Locked(b, func() {
		assert.True(t, ed.isSpellEnabled(pos(2, 6)), "in a comment")
		assert.False(t, ed.isSpellEnabled(pos(3, 1)), "in code")
		// a data file, which is not spell checked
		ed.Lines.SetFileInfo(&fileinfo.FileInfo{Cat: fileinfo.Data})
		assert.False(t, ed.isSpellEnabled(pos(2, 6)), "not for other files")
		ed.Lines.Settings.SpellCorrect = false
		assert.False(t, ed.OfferCorrect())
	}))
}
