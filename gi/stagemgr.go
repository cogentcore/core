// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"sync"

	"goki.dev/goosi/events"
	"goki.dev/ordmap"
)

// StageMgr is the stage manager interface
type StageMgr interface {

	// AsMainMgr returns underlying MainStageMgr or nil if a PopupStageMgr
	AsMainMgr() *MainStageMgr

	// AsPopupMgr returns underlying PopupStageMgr or nil if a MainStageMgr
	AsPopupMgr() *PopupStageMgr
}

// StageMgrBase provides base impl for stage management,
// extended by PopupStageMgr and MainStageMgr.
// Manages a stack of Stage elements.
type StageMgrBase struct {
	// This is the StageMgr as a StageMgr interface -- preserves actual identity
	// when calling interface methods in StageMgrBase.
	This StageMgr

	// stack of stages
	Stack ordmap.Map[string, Stage]

	// Modified is set to true whenever the stack has been modified.
	// This is cleared by the RenderWin each render cycle.
	Modified bool

	// [view: -] mutex protecting reading / updating of the Stack -- destructive stack updating gets a Write lock, else Read
	Mu sync.RWMutex `view:"-" desc:"mutex protecting reading / updating of the Stack -- destructive stack updating gets a Write lock, else Read"`
}

// Top returns the top-most Stage in the Stack, under Read Lock
func (sm *StageMgrBase) Top() Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}
	return sm.Stack.ValByIdx(sz - 1)
}

// TopOfType returns the top-most Stage in the Stack
// of the given type, under Read Lock
func (sm *StageMgrBase) TopOfType(typ StageTypes) Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i)
		if st.AsBase().Type == typ {
			return st
		}
	}
	return nil
}

// Push pushes a new Stage to top, under Write lock
func (sm *StageMgrBase) Push(st Stage) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sm.Modified = true
	sm.Stack.Add(st.AsBase().Name, st)
	st.StageAdded(sm.This)
}

// Pop pops current Stage off the stack, returning it or nil if none.
// It runs under Write lock.
func (sm *StageMgrBase) Pop() Stage {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}

	sm.Modified = true
	st := sm.Stack.ValByIdx(sz - 1)
	sm.Stack.DeleteIdx(sz-1, sz)
	return st
}

// PopType pops the top-most Stage of the given type of the stack,
// returning it or nil if none. It runs under Write lock.
func (sm *StageMgrBase) PopType(typ StageTypes) Stage {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	l := sm.Stack.Len()
	for i := l - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i)
		if st.AsBase().Type == typ {
			sm.Modified = true
			sm.Stack.DeleteIdx(i, i+1)
			return st
		}
	}
	return nil
}

// PopDelete pops current Stage off the stack and calls Delete on it.
func (sm *StageMgrBase) PopDelete() {
	st := sm.Pop()
	if st != nil {
		st.Delete()
	}
}

// PopDeleteType pops the top-most Stage of the given type off the stack
// and calls Delete on it.
func (sm *StageMgrBase) PopDeleteType(typ StageTypes) {
	st := sm.PopType(typ)
	if st != nil {
		st.Delete()
	}
}

// CloseAll closes all of the stages, calling Delete on each of them.
// When Stage with Popups is Deleted, or when a RenderWindow is closed.
// requires outer RenderContext mutex!
func (sm *StageMgrBase) CloseAll() {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return
	}
	sm.Modified = true
	for i := sz - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i)
		st.Delete()
		sm.Stack.DeleteIdx(i, i+1)
	}
}

// UpdateAll iterates through all Stages and calls DoUpdate on them.
// returns stageMods = true if any Stages have been modified (Main or Popup),
// and sceneMods = true if any Scenes have been modified.
// Stage calls DoUpdate on its Scene, ensuring everything is updated at the
// Widget level.  If nothing is needed, nothing is done.
//
//	This is called only during RenderWin.RenderWindow,
//
// under the global RenderCtx.Mu Write lock so nothing else can happen.
func (sm *StageMgrBase) UpdateAll() (stageMods, sceneMods bool) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	stageMods = sm.Modified
	sm.Modified = false

	sz := sm.Stack.Len()
	if sz == 0 {
		return
	}
	for _, kv := range sm.Stack.Order {
		st := kv.Val
		stMod, scMod := st.DoUpdate()
		stageMods = stageMods || stMod
		sceneMods = sceneMods || scMod
	}
	return
}

// AsMainMgr returns underlying MainStageMgr or nil if a PopupStageMgr
func (sm *StageMgrBase) AsMainMgr() *MainStageMgr {
	return nil
}

// AsPopupMgr returns underlying PopupStageMgr or nil if a MainStageMgr
func (sm *StageMgrBase) AsPopupMgr() *PopupStageMgr {
	return nil
}

//////////////////////////////////////////////////////////////
//		MainStageMgr

// MainStageMgr is the MainStage manager for MainStage types:
// Window, Dialog, Sheet.
// This lives in a RenderWin rendering window, and manages
// all the content for the window.
type MainStageMgr struct {
	StageMgrBase

	// rendering context for the Stages lives here.
	// Everyone comes back here to access it.
	RenderCtx *RenderContext

	// render window -- only set for stage manager within such a window.
	// rely on the RenderCtx wherever possible.
	RenderWin *RenderWin

	// growing stack of viewing history of all stages.
	History []*MainStage
}

// AsMainMgr returns underlying MainStageMgr or nil if a PopupStageMgr
func (sm *MainStageMgr) AsMainMgr() *MainStageMgr {
	return sm
}

// Init is called when owning RenderWin is created.
// Initializes data structures
func (sm *MainStageMgr) Init(this StageMgr, win *RenderWin) {
	sm.This = this
	sm.RenderWin = win
	sm.RenderCtx = &RenderContext{LogicalDPI: 96}
}

// resize resizes all main stages
func (sm *MainStageMgr) Resize(sz image.Point) {
	for _, kv := range sm.Stack.Order {
		st := kv.Val.AsMain()
		st.Resize(sz)
	}
}

func (sm *MainStageMgr) HandleEvent(evi events.Event) {
	n := sm.Stack.Len()
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i).AsMain()
		st.HandleEvent(evi)
		if evi.IsHandled() || st.Modal || st.Type == WindowStage {
			return
		}
	}
}
