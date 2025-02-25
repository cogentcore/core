// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"fmt"
	"sync"
	"time"

	"cogentcore.org/core/text/textpos"
)

// UndoTrace; set to true to get a report of undo actions
var UndoTrace = false

// UndoGroupDelay is the amount of time above which a new group
// is started, for grouping undo events
var UndoGroupDelay = 250 * time.Millisecond

// Undo is the textview.Buf undo manager
type Undo struct {

	// if true, saving and using undos is turned off (e.g., inactive buffers)
	Off bool

	// undo stack of edits
	Stack []*textpos.Edit

	// undo stack of *undo* edits -- added to whenever an Undo is done -- for emacs-style undo
	UndoStack []*textpos.Edit

	// undo position in stack
	Pos int

	// group counter
	Group int

	// mutex protecting all updates
	Mu sync.Mutex `json:"-" xml:"-"`
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
// exceeds UndoGroupDelay since last item.
func (un *Undo) Save(tbe *textpos.Edit) {
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
		since := tbe.Region.Since(&un.Stack[len(un.Stack)-1].Region)
		if since > UndoGroupDelay {
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
func (un *Undo) UndoPop() *textpos.Edit {
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
		fmt.Printf("Undo: UndoPop of Gp: %v  pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Region, string(tbe.ToBytes()))
	}
	return tbe
}

// UndoPopIfGroup pops the top item off of the stack if it is the same as given group
func (un *Undo) UndoPopIfGroup(gp int) *textpos.Edit {
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
		fmt.Printf("Undo: UndoPopIfGroup of Gp: %v pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Region, string(tbe.ToBytes()))
	}
	return tbe
}

// SaveUndo saves given edit to UndoStack (stack of undoes that have have undone..)
// for emacs mode.
func (un *Undo) SaveUndo(tbe *textpos.Edit) {
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
	un.Stack = append(un.Stack, un.UndoStack...)
	un.Pos = len(un.Stack)
	un.UndoStack = nil
	if UndoTrace {
		fmt.Printf("Undo: undo stack saved to main stack, new pos: %v\n", un.Pos)
	}
}

// RedoNext returns the current item on Stack for Redo, and increments the position
// returns nil if at end of stack.
func (un *Undo) RedoNext() *textpos.Edit {
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
		fmt.Printf("Undo: RedoNext of Gp: %v at pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Region, string(tbe.ToBytes()))
	}
	un.Pos++
	return tbe
}

// RedoNextIfGroup returns the current item on Stack for Redo if it is same group
// and increments the position. returns nil if at end of stack.
func (un *Undo) RedoNextIfGroup(gp int) *textpos.Edit {
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
		fmt.Printf("Undo: RedoNextIfGroup of Gp: %v at pos: %v delete? %v at: %v text: %v\n", un.Group, un.Pos, tbe.Delete, tbe.Region, string(tbe.ToBytes()))
	}
	un.Pos++
	return tbe
}

// AdjustRegion adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (un *Undo) AdjustRegion(reg textpos.Region) textpos.Region {
	if un.Off {
		return reg
	}
	un.Mu.Lock()
	defer un.Mu.Unlock()
	for _, utbe := range un.Stack {
		reg = utbe.AdjustRegion(reg)
		if reg == (textpos.Region{}) {
			return reg
		}
	}
	return reg
}

////////   Lines api

// saveUndo saves given edit to undo stack.
func (ls *Lines) saveUndo(tbe *textpos.Edit) {
	if tbe == nil {
		return
	}
	ls.undos.Save(tbe)
}

// undo undoes next group of items on the undo stack
func (ls *Lines) undo() []*textpos.Edit {
	tbe := ls.undos.UndoPop()
	if tbe == nil {
		// note: could clear the changed flag on tbe == nil in parent
		return nil
	}
	stgp := tbe.Group
	var eds []*textpos.Edit
	for {
		if tbe.Rect {
			if tbe.Delete {
				utbe := ls.insertTextRectImpl(tbe)
				utbe.Group = stgp + tbe.Group
				if ls.Settings.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextRectImpl(tbe.Region.Start, tbe.Region.End)
				utbe.Group = stgp + tbe.Group
				if ls.Settings.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		} else {
			if tbe.Delete {
				utbe := ls.insertTextImpl(tbe.Region.Start, tbe.Text)
				utbe.Group = stgp + tbe.Group
				if ls.Settings.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextImpl(tbe.Region.Start, tbe.Region.End)
				utbe.Group = stgp + tbe.Group
				if ls.Settings.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		}
		tbe = ls.undos.UndoPopIfGroup(stgp)
		if tbe == nil {
			break
		}
	}
	return eds
}

// redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (ls *Lines) redo() []*textpos.Edit {
	tbe := ls.undos.RedoNext()
	if tbe == nil {
		return nil
	}
	var eds []*textpos.Edit
	stgp := tbe.Group
	for {
		if tbe.Rect {
			if tbe.Delete {
				ls.deleteTextRectImpl(tbe.Region.Start, tbe.Region.End)
			} else {
				ls.insertTextRectImpl(tbe)
			}
		} else {
			if tbe.Delete {
				ls.deleteTextImpl(tbe.Region.Start, tbe.Region.End)
			} else {
				ls.insertTextImpl(tbe.Region.Start, tbe.Text)
			}
		}
		eds = append(eds, tbe)
		tbe = ls.undos.RedoNextIfGroup(stgp)
		if tbe == nil {
			break
		}
	}
	return eds
}
