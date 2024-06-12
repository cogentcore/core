// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"sync"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// Stages manages a stack of [Stages].
type Stages struct { //types:add
	// stack of stages managed by this stage manager.
	Stack ordmap.Map[string, *Stage] `set:"-"`

	// Modified is set to true whenever the stack has been modified.
	// This is cleared by the RenderWindow each render cycle.
	Modified bool

	// rendering context provides key rendering information and locking
	// for the RenderWindow in which the stages are running.
	RenderContext *RenderContext

	// render window to which we are rendering.
	// rely on the RenderContext wherever possible.
	RenderWindow *RenderWindow

	// growing stack of viewing history of all stages.
	History []*Stage `set:"-"`

	// Main is the main stage that owns this [Stages].
	// This is only set for popup stages.
	Main *Stage

	// mutex protecting reading / updating of the Stack.
	// Destructive stack updating gets a Write lock, else Read.
	Mu sync.RWMutex `view:"-" set:"-"`
}

// Top returns the top-most Stage in the Stack, under Read Lock
func (sm *Stages) Top() *Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}
	return sm.Stack.ValueByIndex(sz - 1)
}

// TopOfType returns the top-most Stage in the Stack
// of the given type, under Read Lock
func (sm *Stages) TopOfType(typ StageTypes) *Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		if st.Type == typ {
			return st
		}
	}
	return nil
}

// TopNotType returns the top-most Stage in the Stack
// that is NOT the given type, under Read Lock
func (sm *Stages) TopNotType(typ StageTypes) *Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		if st.Type != typ {
			return st
		}
	}
	return nil
}

// UniqueName returns unique name for given item
func (sm *Stages) UniqueName(nm string) string {
	ctr := 0
	for _, kv := range sm.Stack.Order {
		if kv.Key == nm {
			ctr++
		}
	}
	if ctr > 0 {
		return fmt.Sprintf("%s-%d", nm, len(sm.Stack.Order))
	}
	return nm
}

// Push pushes a new Stage to top, under Write lock
func (sm *Stages) Push(st *Stage) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.Modified = true
	sm.Stack.Add(sm.UniqueName(st.Name), st)
}

// Pop pops current Stage off the stack, returning it or nil if none.
// It runs under Write lock.
func (sm *Stages) Pop() *Stage {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}

	sm.Modified = true
	st := sm.Stack.ValueByIndex(sz - 1)
	sm.Stack.DeleteIndex(sz-1, sz)
	return st
}

// DeleteStage deletes given stage (removing from stack, calling Delete
// on Stage), returning true if found.
// It runs under Write lock.
func (sm *Stages) DeleteStage(st *Stage) bool {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		s := sm.Stack.ValueByIndex(i)
		if st == s {
			sm.Modified = true
			sm.Stack.DeleteIndex(i, i+1)
			st.Delete()
			return true
		}
	}
	return false
}

// DeleteStageAndBelow deletes given stage (removing from stack,
// calling Delete on Stage), returning true if found.
// And also deletes all stages of the same type immediately below it.
// It runs under Write lock.
func (sm *Stages) DeleteStageAndBelow(st *Stage) bool {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	styp := st.Type

	l := sm.Stack.Len()
	got := false
	for i := l - 1; i >= 0; i-- {
		s := sm.Stack.ValueByIndex(i)
		if !got {
			if st == s {
				sm.Modified = true
				sm.Stack.DeleteIndex(i, i+1)
				st.Delete()
				got = true
			}
		} else {
			if s.Type == styp {
				sm.Stack.DeleteIndex(i, i+1)
				st.Delete()
			}
		}
	}
	return got
}

// MoveToTop moves the given stage to the top of the stack,
// returning true if found. It runs under Write lock.
func (sm *Stages) MoveToTop(st *Stage) bool {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		s := sm.Stack.ValueByIndex(i)
		if st == s {
			k := sm.Stack.KeyByIndex(i)
			sm.Modified = true
			sm.Stack.DeleteIndex(i, i+1)
			sm.Stack.InsertAtIndex(sm.Stack.Len(), k, s)
			return true
		}
	}
	return false
}

// PopType pops the top-most Stage of the given type of the stack,
// returning it or nil if none. It runs under Write lock.
func (sm *Stages) PopType(typ StageTypes) *Stage {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		if st.Type == typ {
			sm.Modified = true
			sm.Stack.DeleteIndex(i, i+1)
			return st
		}
	}
	return nil
}

// PopDelete pops current top--most Stage off the stack and calls Delete on it.
func (sm *Stages) PopDelete() {
	st := sm.Pop()
	if st != nil {
		st.Delete()
	}
}

// PopDeleteType pops the top-most Stage of the given type off the stack
// and calls Delete on it.
func (sm *Stages) PopDeleteType(typ StageTypes) {
	st := sm.PopType(typ)
	if st != nil {
		st.Delete()
	}
}

// DeleteAll deletes all of the stages.
// For when Stage with Popups is Deleted, or when a RenderWindow is closed.
// requires outer RenderContext mutex!
func (sm *Stages) DeleteAll() {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return
	}
	sm.Modified = true
	for i := sz - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		st.Delete()
		sm.Stack.DeleteIndex(i, i+1)
	}
}

// Resize calls Resize on all stages within based on the given
// window render geom.
func (sm *Stages) Resize(rg math32.Geom2DInt) {
	for _, kv := range sm.Stack.Order {
		st := kv.Value
		if st.Type == WindowStage || (st.Type == DialogStage && st.FullWindow) {
			st.Scene.Resize(rg)
		} else {
			st.Scene.FitInWindow(rg)
		}
	}
}

// UpdateAll iterates through all Stages and calls DoUpdate on them.
// returns stageMods = true if any Stages have been modified (Main or Popup),
// and sceneMods = true if any Scenes have been modified.
// Stage calls DoUpdate on its Scene, ensuring everything is updated at the
// Widget level.  If nothing is needed, nothing is done.
// This is called only during RenderWindow.RenderWindow,
// under the global RenderContext.Mu Write lock so nothing else can happen.
func (sm *Stages) UpdateAll() (stageMods, sceneMods bool) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	stageMods = sm.Modified
	sm.Modified = false

	sz := sm.Stack.Len()
	if sz == 0 {
		return
	}
	for _, kv := range sm.Stack.Order {
		st := kv.Value
		stMod, scMod := st.DoUpdate()
		stageMods = stageMods || stMod
		sceneMods = sceneMods || scMod
	}
	return
}

func (sm *Stages) SendShowEvents() {
	for _, kv := range sm.Stack.Order {
		st := kv.Value
		if st.Scene == nil {
			continue
		}
		sc := st.Scene
		if sc.ShowIter == SceneShowIters+1 {
			sc.ShowIter++
			if !sc.hasShown {
				sc.hasShown = true
				// profile.Profiling = true
				// pr := profile.Start("send show")
				sc.Events.GetShortcuts()
				sc.WidgetWalkDown(func(kwi Widget, kwb *WidgetBase) bool {
					kwi.AsWidget().Send(events.Show)
					return tree.Continue
				})
				// pr.End()
				// profile.Report(time.Millisecond)
			}
		}
	}
}
