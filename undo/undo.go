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

// Rec is one undo record, associated with one action that changed state from one to next.
// The state saved in this record is the state *before* the action took place.
// The state is either saved as a Raw value or as a diff Patch to the previous state.
type Rec struct {

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
func (rc *Rec) Init(action, data string) {
	rc.Action = action
	rc.Data = data
	rc.Patch = nil
	rc.Raw = nil
	rc.UndoSave = false
}

// Mgr is the undo manager, managing the undo / redo process
type Mgr struct {

	// current index in the undo records -- this is the record that will be undone if user hits undo
	Index int

	// the list of saved state / action records
	Recs []*Rec

	// interval for saving raw data -- need to do this at some interval to prevent having it take too long to compute patches from all the diffs.
	RawInterval int

	// mutex that protects updates -- we do diff computation as a separate goroutine so it is instant from perspective of UI
	Mu sync.Mutex
}

// RecState returns the state for given index, reconstructing from diffs
// as needed.  Must be called under lock.
func (um *Mgr) RecState(idx int) []string {
	stidx := 0
	var cdt []string
	for i := idx; i >= 0; i-- {
		r := um.Recs[i]
		if r.Raw != nil {
			stidx = i
			cdt = r.Raw
			break
		}
	}
	for i := stidx + 1; i <= idx; i++ {
		r := um.Recs[i]
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
func (um *Mgr) Save(action, data string, state []string) {
	um.Mu.Lock() // we start lock
	if um.Recs == nil {
		if um.RawInterval == 0 {
			um.RawInterval = DefaultRawInterval
		}
		um.Recs = make([]*Rec, 1)
		um.Index = 0
		nr := &Rec{Action: action, Data: data, Raw: state}
		um.Recs[0] = nr
		um.Mu.Unlock()
		return
	}
	// recs will be [old..., Index] after this
	um.Index++
	var nr *Rec
	if len(um.Recs) > um.Index {
		um.Recs = um.Recs[:um.Index+1]
		nr = um.Recs[um.Index]
	} else if len(um.Recs) == um.Index {
		nr = &Rec{}
		um.Recs = append(um.Recs, nr)
	} else {
		log.Printf("undo.Mgr error: index: %d > len(um.Recs): %d\n", um.Index, len(um.Recs))
		um.Index = len(um.Recs)
		nr = &Rec{}
		um.Recs = append(um.Recs, nr)
	}
	nr.Init(action, data)
	if state == nil {
		um.Mu.Unlock()
		return
	}
	go um.SaveState(nr, um.Index, state) // fork off save -- it will unlock when done
	// now we return straight away, with lock still held
}

// MustSaveUndoStart returns true if the current state must be saved as the start of
// the first Undo action when at the end of the stack.  If this returns true, then
// call SaveUndoStart.  It sets a special flag on the record.
func (um *Mgr) MustSaveUndoStart() bool {
	return um.Index == len(um.Recs)-1
}

// SaveUndoStart saves the current state -- call if MustSaveUndoStart is true.
// Sets a special flag for this record, and action, data are empty.
// Does NOT increment the index, so next undo is still as expected.
func (um *Mgr) SaveUndoStart(state []string) {
	um.Mu.Lock()
	nr := &Rec{UndoSave: true}
	um.Recs = append(um.Recs, nr)
	um.SaveState(nr, um.Index+1, state) // do it now because we need to immediately do Undo, does unlock
}

// SaveReplace replaces the current Undo record with new state,
// instead of creating a new record.  This is useful for when
// you have a stream of the same type of manipulations
// and just want to save the last (it is better to handle that case
// up front as saving the state can be relatively expensive, but
// in some cases it is not possible).
func (um *Mgr) SaveReplace(action, data string, state []string) {
	um.Mu.Lock()
	nr := um.Recs[um.Index]
	go um.SaveState(nr, um.Index, state)
}

// SaveState saves given record of state at given index
func (um *Mgr) SaveState(nr *Rec, idx int, state []string) {
	if idx%um.RawInterval == 0 {
		nr.Raw = state
		um.Mu.Unlock()
		return
	}
	prv := um.RecState(idx - 1)
	dif := textbuf.DiffLines(prv, state)
	nr.Patch = dif.ToPatch(state)
	um.Mu.Unlock()
}

// HasUndoAvail returns true if there is at least one undo record available.
// This does NOT get the lock -- may rarely be inaccurate but is used for
// gui enabling so not such a big deal.
func (um *Mgr) HasUndoAvail() bool {
	return um.Index >= 0
}

// HasRedoAvail returns true if there is at least one redo record available.
// This does NOT get the lock -- may rarely be inaccurate but is used for
// gui enabling so not such a big deal.
func (um *Mgr) HasRedoAvail() bool {
	return um.Index < len(um.Recs)-2
}

// Undo returns the action, action data, and state at the current index
// and decrements the index to the previous record.
// This state is the state just prior to the action.
// If already at the start (Index = -1) then returns empty everything
// Before calling, first check MustSaveUndoStart() -- if false, then you need
// to call SaveUndoStart() so that the state just before Undoing can be redone!
func (um *Mgr) Undo() (action, data string, state []string) {
	um.Mu.Lock()
	if um.Index < 0 || um.Index >= len(um.Recs) {
		um.Mu.Unlock()
		return
	}
	rec := um.Recs[um.Index]
	action = rec.Action
	data = rec.Data
	state = um.RecState(um.Index)
	um.Index--
	um.Mu.Unlock()
	return
}

// UndoTo returns the action, action data, and state at the given index
// and decrements the index to the previous record.
// If idx is out of range then returns empty everything
func (um *Mgr) UndoTo(idx int) (action, data string, state []string) {
	um.Mu.Lock()
	if idx < 0 || idx >= len(um.Recs) {
		um.Mu.Unlock()
		return
	}
	rec := um.Recs[idx]
	action = rec.Action
	data = rec.Data
	state = um.RecState(idx)
	um.Index = idx - 1
	um.Mu.Unlock()
	return
}

// Redo returns the action, data at the *next* index, and the state at the
// index *after that*.
// returning nil if already at end of saved records.
func (um *Mgr) Redo() (action, data string, state []string) {
	um.Mu.Lock()
	if um.Index >= len(um.Recs)-2 {
		um.Mu.Unlock()
		return
	}
	um.Index++
	rec := um.Recs[um.Index] // action being redone is this one
	action = rec.Action
	data = rec.Data
	state = um.RecState(um.Index + 1) // state is the one *after* it
	um.Mu.Unlock()
	return
}

// RedoTo returns the action, action data, and state at the given index,
// returning nil if already at end of saved records.
func (um *Mgr) RedoTo(idx int) (action, data string, state []string) {
	um.Mu.Lock()
	if idx >= len(um.Recs)-1 || idx <= 0 {
		um.Mu.Unlock()
		return
	}
	um.Index = idx
	rec := um.Recs[idx]
	action = rec.Action
	data = rec.Data
	state = um.RecState(idx + 1)
	um.Mu.Unlock()
	return
}

// Reset resets the undo state
func (um *Mgr) Reset() {
	um.Mu.Lock()
	um.Recs = nil
	um.Index = 0
	um.Mu.Unlock()
}

// UndoList returns the list actions in order from the most recent back in time
// suitable for a menu of actions to undo.
func (um *Mgr) UndoList() []string {
	al := make([]string, um.Index)
	for i := um.Index; i >= 0; i-- {
		al[um.Index-i] = um.Recs[i].Action
	}
	return al
}

// RedoList returns the list actions in order from the current forward to end
// suitable for a menu of actions to redo
func (um *Mgr) RedoList() []string {
	nl := len(um.Recs)
	if um.Index >= nl-2 {
		return nil
	}
	st := um.Index + 1
	n := (nl - 1) - st
	al := make([]string, n)
	for i := st; i < nl-1; i++ {
		al[i-st] = um.Recs[i].Action
	}
	return al
}

// MemUsed reports the amount of memory used for record
func (rc *Rec) MemUsed() int {
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
func (um *Mgr) MemStats(details bool) string {
	sb := strings.Builder{}
	// TODO(kai): add this back once we figure out how to do views.FileSize
	/*
		sum := 0
		for i, r := range um.Recs {
			mem := r.MemUsed()
			sum += mem
			if details {
				sb.WriteString(fmt.Sprintf("%d\t%s\tmem:%s\n", i, r.Action, views.FileSize(mem).String()))
			}
		}
		sb.WriteString(fmt.Sprintf("Total: %s\n", views.FileSize(sum).String()))
	*/
	return sb.String()
}
