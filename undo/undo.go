// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package undo

import (
	"log"
	"sync"

	"github.com/goki/gi/giv/textbuf"
)

// DefaultRawInterval is interval for saving raw state -- need to do this
// at some interval to prevent having it take too long to compute patches
// from all the diffs.
var DefaultRawInterval = 50

// Rec is one undo record, associated with one action that changed state from one to next.
type Rec struct {
	Action string        `desc:"description of this action, for user to see"`
	Data   string        `desc:"action data, encoded however you want -- some undo records can just be about this action data that can be interpeted to Undo / Redo a non-state-changing action"`
	Raw    []string      `desc:"if present, then direct save of full state -- do this at intervals to speed up computing prior states"`
	Patch  textbuf.Patch `desc:"patch to get from previous record to this one"`
}

// Mgr is the undo manager, managing the undo / redo process
type Mgr struct {
	Idx         int        `desc:"current index in the undo records -- this is the record that will be undone if user hits undo"`
	Recs        []*Rec     `desc:"the list of saved state / action records"`
	RawInterval int        `desc:"interval for saving raw data -- need to do this at some interval to prevent having it take too long to compute patches from all the diffs."`
	Mu          sync.Mutex `desc:"mutex that protects updates -- we do diff computation as a separate goroutine so it is instant from perspective of UI"`
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
		um.Idx = 0
		nr := &Rec{Action: action, Data: data, Raw: state}
		um.Recs[0] = nr
		um.Mu.Unlock()
		return
	}
	// recs will be [old..., Idx] after this
	um.Idx++
	var nr *Rec
	if len(um.Recs) > um.Idx {
		um.Recs = um.Recs[:um.Idx+1]
		nr = um.Recs[um.Idx]
	} else if len(um.Recs) == um.Idx {
		nr = &Rec{}
		um.Recs = append(um.Recs, nr)
	} else {
		log.Printf("undo.Mgr error: index: %d > len(um.Recs): %d\n", um.Idx, len(um.Recs))
		um.Idx = len(um.Recs)
		nr = &Rec{}
		um.Recs = append(um.Recs, nr)
	}
	nr.Action = action
	nr.Data = data
	nr.Patch = nil // could be re-using record
	nr.Raw = nil
	if state == nil {
		um.Mu.Unlock()
		return
	}
	go um.SaveState(nr, um.Idx, state) // fork off save -- it will unlock when done
	// now we return straight away, with lock still held
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

// IsUndoAvail returns true if there is at least one undo record available.
// This does NOT get the lock -- may rarely be inaccurate
func (um *Mgr) IsUndoAvail() bool {
	return um.Idx >= 0
}

// IsRedoAvail returns true if there is at least one redo record available.
// This does NOT get the lock -- may rarely be inaccurate.
func (um *Mgr) IsRedoAvail() bool {
	return um.Idx < len(um.Recs)-1
}

// Undo returns the action, action data, and state at the current index
// and decrements the index to point to the previous record.
// If already at the start (Idx = -1) then returns empty everything
func (um *Mgr) Undo() (action, data string, state []string) {
	um.Mu.Lock()
	if um.Idx < 0 {
		um.Mu.Unlock()
		return
	}
	rec := um.Recs[um.Idx]
	action = rec.Action
	data = rec.Data
	state = um.RecState(um.Idx)
	um.Idx--
	um.Mu.Unlock()
	return
}

// Redo returns the action, action data, and state at the next index,
// returning nil if already at end of saved records.
func (um *Mgr) Redo() (action, data string, state []string) {
	um.Mu.Lock()
	if um.Idx >= len(um.Recs)-1 {
		um.Mu.Unlock()
		return
	}
	um.Idx++
	rec := um.Recs[um.Idx]
	action = rec.Action
	data = rec.Data
	state = um.RecState(um.Idx)
	um.Mu.Unlock()
	return
}
