// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

// TestParseEditsAfterSetText checks editing files that use the parser
// right after setting their text, when only the first lines have been
// parsed for the initial markup, and the full parse is under way.
func TestParseEditsAfterSetText(t *testing.T) {
	for _, tc := range []struct{ ext, head, line, ins string }{
		{"go", "package x\n\n", "var a = 1\n", "var b = 2\n"},
		{"md", "# Title\n\n", "Some *text* here.\n", "More `code` text.\n"},
		{"tex", "\\section{A}\n\n", "Some \\emph{text} here.\n", "More \\textbf{text}.\n"},
	} {
		t.Run(tc.ext, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "x."+tc.ext)
			src := tc.head + strings.Repeat(tc.line, 500)
			os.WriteFile(fname, []byte(src), 0644)
			ls := NewLines()
			ls.SetFilename(fname)
			ls.SetText([]byte(src))
			assert.True(t, ls.Highlighter.UsingParse())
			n := ls.NumLines()
			for i := range 400 {
				ln := ls.NumLines() - 2
				if i%2 == 0 {
					ls.ReplaceText(textpos.Pos{Line: ln}, textpos.Pos{Line: ln + 1}, textpos.Pos{Line: ln}, "", false)
				} else {
					ls.ReplaceText(textpos.Pos{Line: ln}, textpos.Pos{Line: ln}, textpos.Pos{Line: ln}, tc.ins, false)
				}
			}
			assert.Equal(t, n, ls.NumLines())
			// the full parse then covers all of the lines:
			ls.ReMarkup()
			deadline := time.Now().Add(10 * time.Second)
			synced := func() bool {
				ls.Lock()
				defer ls.Unlock()
				// the parsed text ends with a line feed, so it has an extra empty line:
				return len(ls.parseState.Done().Src.Lexs) == ls.numLines()+1 && len(ls.hiTags) == ls.numLines()
			}
			for !synced() && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			assert.True(t, synced(), "parse state not synced with the lines")
		})
	}
}
