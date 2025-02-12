// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"testing"

	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestEdit(t *testing.T) {
	src := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    tbe := ls.regionRect(st, ed)
    if tbe == nil {
        return nil
    }
`

	lns := &Lines{}
	lns.Defaults()
	lns.SetText([]byte(src))
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	st := textpos.Pos{1, 4}
	ins := []rune("var ")
	lns.NewUndoGroup()
	tbe := lns.InsertText(st, ins)

	edt := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    var tbe := ls.regionRect(st, ed)
    if tbe == nil {
        return nil
    }
`
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	assert.Equal(t, st, tbe.Region.Start)
	ed := st
	ed.Char += 4
	assert.Equal(t, ed, tbe.Region.End)
	assert.Equal(t, ins, tbe.Text[0])
	lns.Undo()
	assert.Equal(t, src+"\n", string(lns.Bytes()))
	lns.Redo()
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	lns.NewUndoGroup()
	lns.DeleteText(tbe.Region.Start, tbe.Region.End)
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	ins = []rune(` // comment
    // next line`)

	st = textpos.Pos{2, 19}
	lns.NewUndoGroup()
	tbe = lns.InsertText(st, ins)

	edt = `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    tbe := ls.regionRect(st, ed)
    if tbe == nil { // comment
    // next line
        return nil
    }
`
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	assert.Equal(t, st, tbe.Region.Start)
	ed = st
	ed.Line = 3
	ed.Char = 16
	assert.Equal(t, ed, tbe.Region.End)
	assert.Equal(t, ins[:11], tbe.Text[0])
	assert.Equal(t, ins[12:], tbe.Text[1])
	lns.Undo()
	assert.Equal(t, src+"\n", string(lns.Bytes()))
	lns.Redo()
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	lns.NewUndoGroup()
	lns.DeleteText(tbe.Region.Start, tbe.Region.End)
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	// rect insert

	tbe.Region = textpos.NewRegion(2, 4, 4, 7)
	ir := [][]rune{[]rune("abc"), []rune("def"), []rune("ghi")}
	tbe.Text = ir
	lns.NewUndoGroup()
	tbe = lns.InsertTextRect(tbe)

	edt = `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    tbe := ls.regionRect(st, ed)
    abcif tbe == nil {
    def    return nil
    ghi}
`

	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	st.Line = 2
	st.Char = 4
	assert.Equal(t, st, tbe.Region.Start)
	ed = st
	ed.Line = 4
	ed.Char = 7
	assert.Equal(t, ed, tbe.Region.End)
	// assert.Equal(t, ins[:11], tbe.Text[0])
	// assert.Equal(t, ins[12:], tbe.Text[1])
	lns.Undo()
	// fmt.Println(string(lns.Bytes()))
	assert.Equal(t, src+"\n", string(lns.Bytes()))
	lns.Redo()
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	lns.NewUndoGroup()
	lns.DeleteTextRect(tbe.Region.Start, tbe.Region.End)
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	// at end
	lns.NewUndoGroup()
	tbe.Region = textpos.NewRegion(2, 19, 4, 22)
	tbe = lns.InsertTextRect(tbe)

	edt = `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    tbe := ls.regionRect(st, ed)
    if tbe == nil {abc
        return nil def
    }              ghi
`
	// fmt.Println(string(lns.Bytes()))

	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	st.Line = 2
	st.Char = 19
	assert.Equal(t, st, tbe.Region.Start)
	ed = st
	ed.Line = 4
	ed.Char = 22
	assert.Equal(t, ed, tbe.Region.End)
	lns.Undo()

	srcsp := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
    tbe := ls.regionRect(st, ed)
    if tbe == nil {
        return nil 
    }              
`

	// fmt.Println(string(lns.Bytes()))
	assert.Equal(t, srcsp+"\n", string(lns.Bytes()))
	lns.Redo()
	assert.Equal(t, edt+"\n", string(lns.Bytes()))
	lns.NewUndoGroup()
	lns.DeleteTextRect(tbe.Region.Start, tbe.Region.End)
	assert.Equal(t, srcsp+"\n", string(lns.Bytes()))

}
