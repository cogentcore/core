// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "image"

// These are for the tests of package core_test, which can use other
// packages that use package core, such as htmlcore.

// TextRunePoint returns the window position of the middle of the rune of
// the given index of the given text.
func TextRunePoint(tx *Text, i int) image.Point { return runePoint(tx, i) }

// TextSelected returns the text that is selected, in the texts of the
// selection of the scene of the given text (see [Text.selectedText]).
func TextSelected(tx *Text) string { return tx.selectedText() }

// TextOwnSelection returns the text that is selected in the given text.
func TextOwnSelection(tx *Text) string {
	return string(tx.richText.Join()[tx.selectRange.Start:tx.selectRange.End])
}
