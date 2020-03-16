// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"fmt"
	"sync"
	"time"

	"github.com/goki/pi/lex"
)

// UndoTrace -- set to true to get a report of undo actions
var UndoTrace = false

// UndoGroupDelayMSec is number of milliseconds above which a new group
// is started, for grouping undo events
var UndoGroupDelayMSec = 250

// Undo is the TextBuf undo manager
type Undo struct {
	Off       bool       `desc:"if true, saving and using undos is turned off (e.g., inactive buffers)"`
	Stack     []*Edit    `desc:"undo stack of edits"`
	UndoStack []*Edit    `desc:"undo stack of *undo* edits -- added to whenever an Undo is done -- for emacs-style undo"`
	Pos       int        `desc:"undo position in stack"`
	Group     int        `desc:"group counter"`
	Mu        sync.Mutex `json:"-" xml:"-" desc:"mutex protecting all updates"`
}

// NewGroup increments the Group counter so subsequent undos will be grouped separately
func (un *Undo) NewGroup() {
	un.Mu.Lock()
	un.Group++
	un.Mu.Unlock()
}

// Reset clears all undo records
func (un *Undo) Reset() {
	un.Pos = 0
	un.Group = 0
	un.Stack = nil
	un.UndoStack = nil
}

// Save saves given edit to undo stack, with current group marker unless timer interval
// exceeds UndoGroupDelayMSec since last item.
func (un *Undo) Save(tbe *Edit) {
	if un.Off {
		return
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if un.Pos < len(un.Stack) {
		if UndoTrace {
			fmt.Printf("Undo: resetting to pos: %v len was: %v\n", un.Pos, len(un.Stack))
		}
		un.Stack = un.Stack[:un.Pos]
	}
	if len(un.Stack) > 0 {
		since := tbe.Reg.SinceMSec(&un.Stack[len(un.Stack)-1].Reg)
		if since > UndoGroupDelayMSec {
			un.Group++
			if UndoTrace {
				fmt.Printf("Undo: incrementing group to: %v since: %v\n", un.Group, since)
			}
		}
	}
	tbe.Group = un.Group
	if UndoTrace {
		fmt.Printf("Undo: save to pos: %v: group: %v\n->\t%v\n", un.Pos, un.Group, string(tbe.ToBytes()))
	}
	un.Stack = append(un.Stack, tbe)
	un.Pos = len(un.Stack)
}

// UndoPop pops the top item off of the stack for use in Undo. returns nil if none.
func (un *Undo) UndoPop() *Edit {
	if un.Off {
		return nil
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if un.Pos == 0 {
		return nil
	}
	un.Pos--
	tbe := un.Stack[un.Pos]
	if UndoTrace {
		fmt.Printf("Undo: UndoPop of Gp: %v  pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Reg, string(tbe.ToBytes()))
	}
	return tbe
}

// UndoPopIfGroup pops the top item off of the stack if it is the same as given group
func (un *Undo) UndoPopIfGroup(gp int) *Edit {
	if un.Off {
		return nil
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if un.Pos == 0 {
		return nil
	}
	tbe := un.Stack[un.Pos-1]
	if tbe.Group != gp {
		return nil
	}
	un.Pos--
	if UndoTrace {
		fmt.Printf("Undo: UndoPopIfGroup of Gp: %v pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Reg, string(tbe.ToBytes()))
	}
	return tbe
}

// SaveUndo saves given edit to UndoStack (stack of undoes that have have undone..)
// for emacs mode.
func (un *Undo) SaveUndo(tbe *Edit) {
	un.UndoStack = append(un.UndoStack, tbe)
}

// UndoStackSave if EmacsUndo mode is active, saves the UndoStack
// to the regular Undo stack, at the end, and moves undo to the very end.
// Undo is a constant stream..
func (un *Undo) UndoStackSave() {
	if un.Off {
		return
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if len(un.UndoStack) == 0 {
		return
	}
	for _, utbe := range un.UndoStack {
		un.Stack = append(un.Stack, utbe)
	}
	un.Pos = len(un.Stack)
	un.UndoStack = nil
	if UndoTrace {
		fmt.Printf("Undo: undo stack saved to main stack, new pos: %v\n", un.Pos)
	}
}

// RedoNext returns the current item on Stack for Redo, and increments the position
// returns nil if at end of stack.
func (un *Undo) RedoNext() *Edit {
	if un.Off {
		return nil
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if un.Pos >= len(un.Stack) {
		return nil
	}
	tbe := un.Stack[un.Pos]
	if UndoTrace {
		fmt.Printf("Undo: RedoNext of Gp: %v at pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Reg, string(tbe.ToBytes()))
	}
	un.Pos++
	return tbe
}

// RedoNextIfGroup returns the current item on Stack for Redo if it is same group
// and increments the position. returns nil if at end of stack.
func (un *Undo) RedoNextIfGroup(gp int) *Edit {
	if un.Off {
		return nil
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	if un.Pos >= len(un.Stack) {
		return nil
	}
	tbe := un.Stack[un.Pos]
	if tbe.Group != gp {
		return nil
	}
	if UndoTrace {
		fmt.Printf("Undo: RedoNextIfGroup of Gp: %v at pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Reg, string(tbe.ToBytes()))
	}
	un.Pos++
	return tbe
}

// AdjustPos adjusts given text position, which was recorded at given time
// for any edits that have taken place since that time (using the Undo stack).
// del determines what to do with positions within a deleted region -- either move
// to start or end of the region, or return an error
func (un *Undo) AdjustPos(pos lex.Pos, t time.Time, del AdjustPosDel) lex.Pos {
	if un.Off {
		return pos
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	for _, utbe := range un.Stack {
		pos = utbe.AdjustPosIfAfterTime(pos, t, del)
		if pos == lex.PosErr {
			return pos
		}
	}
	return pos
}

// AdjustReg adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (un *Undo) AdjustReg(reg Region) Region {
	if un.Off {
		return reg
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	for _, utbe := range un.Stack {
		reg = utbe.AdjustReg(reg)
		if reg == RegionNil {
			return reg
		}
	}
	return reg
}
