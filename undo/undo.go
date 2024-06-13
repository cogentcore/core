// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package undo package provides a generic undo / redo functionality based on `[]string`
representations of any kind of state representation (typically JSON dump of the 'document'
state). It stores the compact diffs from one state change to the next, with raw copies saved
at infrequent intervals to tradeoff cost of computing diffs.

In addition state (which is optional on any given step), a description of the action
and arbitrary string-encoded data can be saved with each record.  Thus, for cases
where the state doesn't change, you can just save some data about the action sufficient
to undo / redo it.

A new record must be saved of the state just *before* an action takes place
and the nature of the action taken.

Thus, undoing the action restores the state to that prior state.

Redoing the action means restoring the state *after* the action.

This means that the first Undo action must save the current state
before doing the undo.

The Index is always on the last state saved, which will then be the one
that would be undone for an undo action.
*/
package undo

import (
	"log"
	"strings"
	"sync"

	"cogentcore.org/core/texteditor/textbuf"
)

// DefaultRawInterval is interval for saving raw state -- need to do this
// at some interval to prevent having it take too long to compute patches
// from all the diffs.
var DefaultRawInterval = 50

// Record is one undo record, associated with one action that changed state from one to next.
// The state saved in this record is the state *before* the action took place.
// The state is either saved as a Raw value or as a diff Patch to the previous state.
type Record struct {

	// description of this action, for user to see
	Action string

	// action data, encoded however you want -- some undo records can just be about this action data that can be interpreted to Undo / Redo a non-state-changing action
	Data string

	// if present, then direct save of full state -- do this at intervals to speed up computing prior states
	Raw []string

	// patch to get from previous record to this one
	Patch textbuf.Patch

	// this record is an UndoSave, when Undo first called from end of stack
	UndoSave bool
}

// Init sets the action and data in a record -- overwriting any prior values
func (rc *Record) Init(action, data string) {
	rc.Action = action
	rc.Data = data
	rc.Patch = nil
	rc.Raw = nil
	rc.UndoSave = false
}

// Stack is the undo stack manager that manages the undo and redo process.
type Stack struct {

	// current index in the undo records -- this is the record that will be undone if user hits undo
	Index int

	// the list of saved state / action records
	Records []*Record

	// interval for saving raw data -- need to do this at some interval to prevent having it take too long to compute patches from all the diffs.
	RawInterval int

	// mutex that protects updates -- we do diff computation as a separate goroutine so it is instant from perspective of UI
	Mu sync.Mutex
}

// RecState returns the state for given index, reconstructing from diffs
// as needed.  Must be called under lock.
func (us *Stack) RecState(idx int) []string {
	stidx := 0
	var cdt []string
	for i := idx; i >= 0; i-- {
		r := us.Records[i]
		if r.Raw != nil {
			stidx = i
			cdt = r.Raw
			break
		}
	}
	for i := stidx + 1; i <= idx; i++ {
		r := us.Records[i]
		if r.Patch != nil {
			cdt = r.Patch.Apply(cdt)
		}
	}
	return cdt
}

// Save saves a new action as next action to be undone, with given action
// data and current full state of the system (either of which are optional).
// The state must be available for saving -- we do not copy in case we save the
// full raw copy.
func (us *Stack) Save(action, data string, state []string) {
	us.Mu.Lock() // we start lock
	if us.Records == nil {
		if us.RawInterval == 0 {
			us.RawInterval = DefaultRawInterval
		}
		us.Records = make([]*Record, 1)
		us.Index = 0
		nr := &Record{Action: action, Data: data, Raw: state}
		us.Records[0] = nr
		us.Mu.Unlock()
		return
	}
	// recs will be [old..., Index] after this
	us.Index++
	var nr *Record
	if len(us.Records) > us.Index {
		us.Records = us.Records[:us.Index+1]
		nr = us.Records[us.Index]
	} else if len(us.Records) == us.Index {
		nr = &Record{}
		us.Records = append(us.Records, nr)
	} else {
		log.Printf("undo.Stack error: index: %d > len(um.Recs): %d\n", us.Index, len(us.Records))
		us.Index = len(us.Records)
		nr = &Record{}
		us.Records = append(us.Records, nr)
	}
	nr.Init(action, data)
	if state == nil {
		us.Mu.Unlock()
		return
	}
	go us.SaveState(nr, us.Index, state) // fork off save -- it will unlock when done
	// now we return straight away, with lock still held
}

// MustSaveUndoStart returns true if the current state must be saved as the start of
// the first Undo action when at the end of the stack.  If this returns true, then
// call SaveUndoStart.  It sets a special flag on the record.
func (us *Stack) MustSaveUndoStart() bool {
	return us.Index == len(us.Records)-1
}

// SaveUndoStart saves the current state -- call if MustSaveUndoStart is true.
// Sets a special flag for this record, and action, data are empty.
// Does NOT increment the index, so next undo is still as expected.
func (us *Stack) SaveUndoStart(state []string) {
	us.Mu.Lock()
	nr := &Record{UndoSave: true}
	us.Records = append(us.Records, nr)
	us.SaveState(nr, us.Index+1, state) // do it now because we need to immediately do Undo, does unlock
}

// SaveReplace replaces the current Undo record with new state,
// instead of creating a new record.  This is useful for when
// you have a stream of the same type of manipulations
// and just want to save the last (it is better to handle that case
// up front as saving the state can be relatively expensive, but
// in some cases it is not possible).
func (us *Stack) SaveReplace(action, data string, state []string) {
	us.Mu.Lock()
	nr := us.Records[us.Index]
	go us.SaveState(nr, us.Index, state)
}

// SaveState saves given record of state at given index
func (us *Stack) SaveState(nr *Record, idx int, state []string) {
	if idx%us.RawInterval == 0 {
		nr.Raw = state
		us.Mu.Unlock()
		return
	}
	prv := us.RecState(idx - 1)
	dif := textbuf.DiffLines(prv, state)
	nr.Patch = dif.ToPatch(state)
	us.Mu.Unlock()
}

// HasUndoAvail returns true if there is at least one undo record available.
// This does NOT get the lock -- may rarely be inaccurate but is used for
// gui enabling so not such a big deal.
func (us *Stack) HasUndoAvail() bool {
	return us.Index >= 0
}

// HasRedoAvail returns true if there is at least one redo record available.
// This does NOT get the lock -- may rarely be inaccurate but is used for
// GUI enabling so not such a big deal.
func (us *Stack) HasRedoAvail() bool {
	return us.Index < len(us.Records)-2
}

// Undo returns the action, action data, and state at the current index
// and decrements the index to the previous record.
// This state is the state just prior to the action.
// If already at the start (Index = -1) then returns empty everything
// Before calling, first check MustSaveUndoStart() -- if false, then you need
// to call SaveUndoStart() so that the state just before Undoing can be redone!
func (us *Stack) Undo() (action, data string, state []string) {
	us.Mu.Lock()
	if us.Index < 0 || us.Index >= len(us.Records) {
		us.Mu.Unlock()
		return
	}
	rec := us.Records[us.Index]
	action = rec.Action
	data = rec.Data
	state = us.RecState(us.Index)
	us.Index--
	us.Mu.Unlock()
	return
}

// UndoTo returns the action, action data, and state at the given index
// and decrements the index to the previous record.
// If idx is out of range then returns empty everything
func (us *Stack) UndoTo(idx int) (action, data string, state []string) {
	us.Mu.Lock()
	if idx < 0 || idx >= len(us.Records) {
		us.Mu.Unlock()
		return
	}
	rec := us.Records[idx]
	action = rec.Action
	data = rec.Data
	state = us.RecState(idx)
	us.Index = idx - 1
	us.Mu.Unlock()
	return
}

// Redo returns the action, data at the *next* index, and the state at the
// index *after that*.
// returning nil if already at end of saved records.
func (us *Stack) Redo() (action, data string, state []string) {
	us.Mu.Lock()
	if us.Index >= len(us.Records)-2 {
		us.Mu.Unlock()
		return
	}
	us.Index++
	rec := us.Records[us.Index] // action being redone is this one
	action = rec.Action
	data = rec.Data
	state = us.RecState(us.Index + 1) // state is the one *after* it
	us.Mu.Unlock()
	return
}

// RedoTo returns the action, action data, and state at the given index,
// returning nil if already at end of saved records.
func (us *Stack) RedoTo(idx int) (action, data string, state []string) {
	us.Mu.Lock()
	if idx >= len(us.Records)-1 || idx <= 0 {
		us.Mu.Unlock()
		return
	}
	us.Index = idx
	rec := us.Records[idx]
	action = rec.Action
	data = rec.Data
	state = us.RecState(idx + 1)
	us.Mu.Unlock()
	return
}

// Reset resets the undo state
func (us *Stack) Reset() {
	us.Mu.Lock()
	us.Records = nil
	us.Index = 0
	us.Mu.Unlock()
}

// UndoList returns the list actions in order from the most recent back in time
// suitable for a menu of actions to undo.
func (us *Stack) UndoList() []string {
	al := make([]string, us.Index)
	for i := us.Index; i >= 0; i-- {
		al[us.Index-i] = us.Records[i].Action
	}
	return al
}

// RedoList returns the list actions in order from the current forward to end
// suitable for a menu of actions to redo
func (us *Stack) RedoList() []string {
	nl := len(us.Records)
	if us.Index >= nl-2 {
		return nil
	}
	st := us.Index + 1
	n := (nl - 1) - st
	al := make([]string, n)
	for i := st; i < nl-1; i++ {
		al[i-st] = us.Records[i].Action
	}
	return al
}

// MemUsed reports the amount of memory used for record
func (rc *Record) MemUsed() int {
	mem := 0
	if rc.Raw != nil {
		for _, s := range rc.Raw {
			mem += len(s)
		}
	} else {
		for _, pr := range rc.Patch {
			for _, s := range pr.Blines {
				mem += len(s)
			}
		}
	}
	return mem
}

// MemStats reports the memory usage statistics.
// if details is true, each record is reported.
func (us *Stack) MemStats(details bool) string {
	sb := strings.Builder{}
	// TODO(kai): add this back once we figure out how to do core.FileSize
	/*
		sum := 0
		for i, r := range um.Recs {
			mem := r.MemUsed()
			sum += mem
			if details {
				sb.WriteString(fmt.Sprintf("%d\t%s\tmem:%s\n", i, r.Action, core.FileSize(mem).String()))
			}
		}
		sb.WriteString(fmt.Sprintf("Total: %s\n", core.FileSize(sum).String()))
	*/
	return sb.String()
}
