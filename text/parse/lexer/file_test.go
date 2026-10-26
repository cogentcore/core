// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLinesEditedClipped checks that line inserts and deletes beyond the
// lines of a partial file state, e.g., only the first lines parsed for
// the initial markup, are clipped to it.
func TestLinesEditedClipped(t *testing.T) {
	fl := &File{}
	fl.SetBytes([]byte("a\nb\nc\n"))
	n := len(fl.Lexs)
	fl.LinesDeleted(n+5, n+7)
	assert.Len(t, fl.Lexs, n)
	fl.LinesDeleted(n-1, n+7)
	assert.Len(t, fl.Lexs, n-1)
	assert.Len(t, fl.EosPos, n-1)
	fl.LinesInserted(n+10, 2)
	assert.Len(t, fl.Lexs, n+1)
	assert.Len(t, fl.LastStacks, n+1)
}
