// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

// EditorSettings contains text editor settings.
type EditorSettings struct { //types:add

	// size of a tab, in chars; also determines indent level for space indent
	TabSize int `default:"4"`

	// use spaces for indentation, otherwise tabs
	SpaceIndent bool

	// wrap lines at word boundaries; otherwise long lines scroll off the end
	WordWrap bool `default:"true"`

	// whether to show line numbers
	LineNumbers bool `default:"true"`

	// use the completion system to suggest options while typing
	Completion bool `default:"true"`

	// suggest corrections for unknown words while typing
	SpellCorrect bool `default:"true"`

	// automatically indent lines when enter, tab, }, etc pressed
	AutoIndent bool `default:"true"`

	// use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo
	EmacsUndo bool

	// colorize the background according to nesting depth
	DepthColor bool `default:"true"`
}

func (es *EditorSettings) Defaults() {
	es.TabSize = 4
	es.SpaceIndent = false
}
