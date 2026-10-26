// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"errors"
	"fmt"
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// testCodeError is an error at a position in code.
type testCodeError struct {
	line, col int
	msg       string
}

func (e *testCodeError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.line, e.col, e.msg)
}
func (e *testCodeError) CodePos() (line, col int) { return e.line, e.col }
func (e *testCodeError) CodeMessage() string      { return e.msg }

func TestShowCodeErrors(t *testing.T) {
	b := core.NewBody()
	ed := textcore.NewEditor(b)
	ed.Lines.SetString("x := 1\ny := x + zz\nfmt.Println(y)\n")
	parent := core.NewFrame(b)
	err := errors.Join(&testCodeError{2, 10, "undefined: zz"}, errors.New("a command failed"))

	// clearing the errors clears the tooltips
	ShowCodeErrors(ed, parent, err)
	ShowCodeErrors(ed, parent, nil)
	assert.Len(t, ed.CodeErrors(), 0)
	ed.Style()
	assert.False(t, ed.Styles.Abilities.HasFlag(abilities.LongHoverable))
	parent.DeleteChildren()

	ShowCodeErrors(ed, parent, err)
	errs := ed.CodeErrors()
	if assert.Len(t, errs, 1) {
		assert.Equal(t, textpos.Pos{Line: 1, Char: 9}, errs[0].Region.Start)
		assert.Equal(t, textpos.Pos{Line: 1, Char: 11}, errs[0].Region.End)
		assert.Equal(t, "undefined: zz", errs[0].Msg)
	}
	var msgs []string
	parent.WalkDown(func(n tree.Node) bool {
		if tx, ok := n.(*core.Text); ok {
			msgs = append(msgs, tx.Text)
		}
		return true
	})
	assert.Equal(t, []string{"2:10: undefined: zz", "a command failed"}, msgs)
	// the tooltips show on hover
	ed.Style()
	assert.True(t, ed.Styles.Abilities.HasFlag(abilities.LongHoverable))

	b.AssertRender(t, "code-errors", func() {
		// the tooltip at the error, and none elsewhere
		reg := errs[0].Region
		pos := ed.CharStartPos(reg.Start).ToPoint().Add(image.Pt(2, 2))
		tip, _ := ed.WidgetTooltip(pos)
		assert.Equal(t, "undefined: zz", tip)
		pos = ed.CharStartPos(textpos.Pos{Line: 0, Char: 1}).ToPoint().Add(image.Pt(2, 2))
		tip, _ = ed.WidgetTooltip(pos)
		assert.Equal(t, "", tip)
	})
}
