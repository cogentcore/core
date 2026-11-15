// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package undo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// state returns a state of lines from the given text.
func state(s string) []string {
	return strings.Split(s, "\n")
}

// TestUndoRedo tests saving states, undoing back to the start, and
// redoing forward again, with states stored as raw copies and diffs.
func TestUndoRedo(t *testing.T) {
	for _, interval := range []int{1, 2, 50} {
		us := &Stack{RawInterval: interval}
		states := []string{"a\nb", "a\nb\nc", "x\nb\nc", "x\nc"}
		for i, s := range states {
			us.Save("act"+string(rune('0'+i)), "d", state(s))
		}
		assert.True(t, us.HasUndoAvail())
		assert.False(t, us.HasRedoAvail())
		assert.Equal(t, []string{"act3", "act2", "act1", "act0"}, us.UndoList())
		assert.Nil(t, us.RedoList())

		// the current state is saved to be able to redo the last action
		assert.True(t, us.MustSaveUndoStart())
		us.SaveUndoStart(state("x"))
		act, data, st := us.Undo()
		assert.Equal(t, "act3", act)
		assert.Equal(t, "d", data)
		assert.Equal(t, state("x\nc"), st)
		assert.False(t, us.MustSaveUndoStart())
		for i := 2; i >= 0; i-- {
			act, _, st = us.Undo()
			assert.Equal(t, "act"+string(rune('0'+i)), act, interval)
			assert.Equal(t, state(states[i]), st, interval)
		}
		act, _, st = us.Undo()
		assert.Empty(t, act, "at the start")
		assert.Nil(t, st)
		assert.True(t, us.HasRedoAvail())
		assert.Equal(t, []string{"act0", "act1", "act2", "act3"}, us.RedoList())

		for i := 0; i < 3; i++ {
			act, _, st = us.Redo()
			assert.Equal(t, "act"+string(rune('0'+i)), act)
			assert.Equal(t, state(states[i+1]), st, "the state after the action")
		}
		act, _, st = us.Redo()
		assert.Equal(t, "act3", act)
		assert.Equal(t, state("x"), st, "the saved undo start")
		act, _, _ = us.Redo()
		assert.Empty(t, act, "at the end")
	}
}

// TestUndoToRedoTo tests jumping to records, saving after undoing, which
// drops the records after, and resetting.
func TestUndoToRedoTo(t *testing.T) {
	us := &Stack{}
	for i := range 4 {
		us.Save("act"+string(rune('0'+i)), "", state(strings.Repeat("l\n", i+1)))
	}
	us.SaveUndoStart(state("end"))
	act, _, st := us.UndoTo(1)
	assert.Equal(t, "act1", act)
	assert.Equal(t, state(strings.Repeat("l\n", 2)), st)
	assert.Equal(t, 0, us.Index)
	act, _, _ = us.UndoTo(10)
	assert.Empty(t, act)

	act, _, st = us.RedoTo(2)
	assert.Equal(t, "act2", act)
	assert.Equal(t, state(strings.Repeat("l\n", 4)), st)
	act, _, _ = us.RedoTo(0)
	assert.Empty(t, act)

	us.Save("new", "", state("n"))
	assert.Equal(t, 3, us.Index)
	assert.Len(t, us.Records, 4, "the later records are dropped")
	assert.Equal(t, []string{"new", "act2", "act1", "act0"}, us.UndoList())

	us.Reset()
	assert.Nil(t, us.Records)
	assert.Nil(t, us.UndoList())
}

// TestSaveReplaceData tests replacing the current record, saving records
// without state, and the memory of records.
func TestSaveReplaceData(t *testing.T) {
	us := &Stack{RawInterval: 2}
	us.Save("first", "", state("a"))
	us.Save("move", "1", nil)
	us.SaveReplace("move", "2", state("a\nb"))
	us.SaveUndoStart(state("a\nb\nc"))
	act, data, st := us.Undo()
	assert.Equal(t, "move", act)
	assert.Equal(t, "2", data, "replaced")
	assert.Equal(t, state("a\nb"), st)

	assert.Equal(t, 1, us.Records[0].MemUsed(), "raw")
	assert.Equal(t, 1, us.Records[1].MemUsed(), "the added line of the patch")
	assert.Positive(t, us.Records[2].MemUsed(), "a raw copy at the interval")
	assert.Empty(t, us.MemStats(true))
}
