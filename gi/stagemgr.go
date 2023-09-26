// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"sync"

	"goki.dev/girl/gist"
	"goki.dev/ordmap"
)

// StageMgrBase provides base impl for stage management,
// extended by PopupMgr and StageMgr.
// Manages a stack of Stage elements.
type StageMgrBase struct {
	// position and size within the parent Render context.
	// Position is absolute offset relative to top left corner of render context.
	Geom gist.Geom2DInt

	// stack of stages
	Stack ordmap.Map[string, *Stage] `desc:"stack of stages"`

	// rendering context for the Stages here
	RenderCtx *RenderContext `desc:"rendering context for the Stages here"`

	// [view: -] mutex protecting reading / updating of the Stack -- destructive stack updating gets a Write lock, else Read
	Mu sync.RWMutex `view:"-" desc:"mutex protecting reading / updating of the Stack -- destructive stack updating gets a Write lock, else Read"`
}

// Top returns the top-most Stage in the Stack
func (sm *StageMgrBase) Top() *Stage {
	sm.Mu.RLock()
	defer sm.Mu.RUnlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}
	return sm.Stack.ValByIdx(sz - 1)
}

// Push pushes a new Stage to top
func (sm *StageMgrBase) Push(st *Stage) {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	st.StageMgr = sm
	st.PopupMgr.Geom = sm.Geom
	sm.Stack.Add(st.Name, st)

	// if pfoc != nil {
	// 	sm.EventMgr.PushFocus(pfoc)
	// } else {
	// 	sm.EventMgr.PushFocus(st)
	// }
}

// Pop pops current Stage off the stack, returning it or nil if none
func (sm *StageMgrBase) Pop() *Stage {
	sm.Mu.Lock()
	defer sm.Mu.Unlock()

	sz := sm.Stack.Len()
	if sz == 0 {
		return nil
	}
	st := sm.Stack.ValByIdx(sz - 1)
	sm.Stack.DeleteIdx(sz-1, sz)
	return st
}

// PopDelete pops current Stage off the stack and calls Delete on it.
func (sm *StageMgrBase) PopDelete() {
	st := sm.Pop()
	if st != nil {
		st.Delete
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
		return nil
	}
	for i := sz - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i)
		st.Delete()
		sm.Stack.DeleteIdx(i, i+1)
	}
}

func (sm *StageMgrBase) NewRenderCtx(dpi float32) *RenderContext {
	ctx := &RenderContext{LogicalDPI: dpi, Visible: false}
	sm.RenderCtx = ctx
}
