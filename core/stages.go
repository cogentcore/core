// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"sync"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/math32"
)

// stages manages a stack of [Stage]s.
type stages struct {

	// stack is the stack of stages managed by this stage manager.
	stack ordmap.Map[string, *Stage]

	// modified is set to true whenever the stack has been modified.
	// This is cleared by the RenderWindow each render cycle.
	modified bool

	// rendering context provides key rendering information and locking
	// for the RenderWindow in which the stages are running.
	renderContext *renderContext

	// render window to which we are rendering.
	// rely on the RenderContext wherever possible.
	renderWindow *renderWindow

	// main is the main stage that owns this [Stages].
	// This is only set for popup stages.
	main *Stage

	// mutex protecting reading / updating of the Stack.
	// Destructive stack updating gets a Write lock, else Read.
	mu sync.RWMutex
}

// top returns the top-most Stage in the Stack, under Read Lock
func (sm *stages) top() *Stage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sz := sm.stack.Len()
	if sz == 0 {
		return nil
	}
	return sm.stack.ValueByIndex(sz - 1)
}

// uniqueName returns unique name for given item
func (sm *stages) uniqueName(nm string) string {
	ctr := 0
	for _, kv := range sm.stack.Order {
		if kv.Key == nm {
			ctr++
		}
	}
	if ctr > 0 {
		return fmt.Sprintf("%s-%d", nm, len(sm.stack.Order))
	}
	return nm
}

// push pushes a new Stage to top, under Write lock
func (sm *stages) push(st *Stage) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.modified = true
	sm.stack.Add(sm.uniqueName(st.Name), st)
}

// deleteStage deletes given stage (removing from stack, calling Delete
// on Stage), returning true if found.
// It runs under Write lock.
func (sm *stages) deleteStage(st *Stage) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	l := sm.stack.Len()
	for i := l - 1; i >= 0; i-- {
		s := sm.stack.ValueByIndex(i)
		if st == s {
			sm.modified = true
			sm.stack.DeleteIndex(i, i+1)
			st.delete()
			return true
		}
	}
	return false
}

// deleteStageAndBelow deletes given stage (removing from stack,
// calling Delete on Stage), returning true if found.
// And also deletes all stages of the same type immediately below it.
// It runs under Write lock.
func (sm *stages) deleteStageAndBelow(st *Stage) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	styp := st.Type

	l := sm.stack.Len()
	got := false
	for i := l - 1; i >= 0; i-- {
		s := sm.stack.ValueByIndex(i)
		if !got {
			if st == s {
				sm.modified = true
				sm.stack.DeleteIndex(i, i+1)
				st.delete()
				got = true
			}
		} else {
			if s.Type == styp {
				sm.stack.DeleteIndex(i, i+1)
				st.delete()
			}
		}
	}
	return got
}

// moveToTop moves the given stage to the top of the stack,
// returning true if found. It runs under Write lock.
func (sm *stages) moveToTop(st *Stage) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	l := sm.stack.Len()
	for i := l - 1; i >= 0; i-- {
		s := sm.stack.ValueByIndex(i)
		if st == s {
			k := sm.stack.KeyByIndex(i)
			sm.modified = true
			sm.stack.DeleteIndex(i, i+1)
			sm.stack.InsertAtIndex(sm.stack.Len(), k, s)
			return true
		}
	}
	return false
}

// popType pops the top-most Stage of the given type of the stack,
// returning it or nil if none. It runs under Write lock.
func (sm *stages) popType(typ StageTypes) *Stage {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	l := sm.stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		if st.Type == typ {
			sm.modified = true
			sm.stack.DeleteIndex(i, i+1)
			return st
		}
	}
	return nil
}

// popDeleteType pops the top-most Stage of the given type off the stack
// and calls Delete on it.
func (sm *stages) popDeleteType(typ StageTypes) {
	st := sm.popType(typ)
	if st != nil {
		st.delete()
	}
}

// deleteAll deletes all of the stages.
// For when Stage with Popups is Deleted, or when a RenderWindow is closed.
// requires outer RenderContext mutex!
func (sm *stages) deleteAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sz := sm.stack.Len()
	if sz == 0 {
		return
	}
	sm.modified = true
	for i := sz - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		st.delete()
		sm.stack.DeleteIndex(i, i+1)
	}
}

// resize calls resize on all stages within based on the given window render geom.
// if nothing actually needed to be resized, it returns false.
func (sm *stages) resize(rg math32.Geom2DInt) bool {
	resized := false
	for _, kv := range sm.stack.Order {
		st := kv.Value
		if st.FullWindow {
			did := st.Scene.resize(rg)
			if did {
				st.Sprites.reset()
				resized = true
			}
		} else {
			did := st.Scene.fitInWindow(rg)
			if did {
				resized = true
			}
		}
	}
	return resized
}

// updateAll iterates through all Stages and calls DoUpdate on them.
// returns stageMods = true if any Stages have been modified (Main or Popup),
// and sceneMods = true if any Scenes have been modified.
// Stage calls DoUpdate on its Scene, ensuring everything is updated at the
// Widget level.  If nothing is needed, nothing is done.
// This is called only during RenderWindow.RenderWindow,
// under the global RenderContext.Mu Write lock so nothing else can happen.
func (sm *stages) updateAll() (stageMods, sceneMods bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	stageMods = sm.modified
	sm.modified = false

	sz := sm.stack.Len()
	if sz == 0 {
		return
	}
	for _, kv := range sm.stack.Order {
		st := kv.Value
		stMod, scMod := st.doUpdate()
		stageMods = stageMods || stMod
		sceneMods = sceneMods || scMod
	}
	return
}

// windowStage returns the highest level WindowStage (i.e., full window)
func (sm *stages) windowStage() *Stage {
	n := sm.stack.Len()
	for i := n - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		if st.Type == WindowStage {
			return st
		}
	}
	return nil
}

func (sm *stages) runDeferred() {
	for _, kv := range sm.stack.Order {
		st := kv.Value
		if st.Scene == nil {
			continue
		}
		sc := st.Scene
		if sc.hasFlag(sceneContentSizing) {
			continue
		}
		if sc.hasFlag(sceneHasDeferred) {
			sc.setFlag(false, sceneHasDeferred)
			sc.runDeferred()
		}

		if sc.showIter == sceneShowIters+1 {
			sc.showIter++
			if !sc.hasFlag(sceneHasShown) {
				sc.setFlag(true, sceneHasShown)
				sc.Shown()
			}
		}

		// If we own popups, we also need to runDeferred on them.
		if st.Main == st && st.popups.stack.Len() > 0 {
			st.popups.runDeferred()
		}
	}
}
