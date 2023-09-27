// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"sync"

	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
	"golang.org/x/image/draw"
)

// note: all RenderWin rendering code is in this file

const (
	// Sprites are stored as arrays of same-sized textures,
	// allocated by size in Set 2, starting at 32
	SpriteStart = vgpu.MaxTexturesPerSet * 2

	// Full set of sprite textures in set = 2
	MaxSpriteTextures = vgpu.MaxTexturesPerSet

	// Allocate 128 layers within each sprite size
	MaxSpritesPerTexture = 128
)

// RenderContext provides rendering context from outer RenderWin
// window to Stage and Scene elements to inform styling, layout
// and rendering.  It also has the master Mutex for any updates
// to the window contents: use Read lock for anything updating.
type RenderContext struct {
	// LogicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	LogicalDPI float32

	// Size of the rendering window, in actual "dot" pixels used for rendering.
	Size image.Point

	// Visible is true if window is visible and should be rendered to
	Visible bool

	// Mu is mutex for locking out rendering and any destructive updates.
	// it is Write locked during rendering to provide exclusive blocking
	// of any other updates, which are Read locked so that you don't
	// get caught in deadlocks among all the different Read locks.
	Mu sync.RWMutex
}

// WriteLock is only called by RenderWin during RenderWindow function
// when updating all widgets and rendering the screen.  All others should
// call ReadLock.  Always call corresponding Unlock in defer!
func (rc *RenderContext) WriteLock() {
	rc.Mu.Lock()
}

// WriteUnlock must be called for each WriteLock, when done.
func (rc *RenderContext) WriteUnlock() {
	rc.Mu.Unlock()
}

// ReadLock should be called whenever modifying anything in the entire
// RenderWin context.  Because it is a Read lock, it does _not_ block
// any other updates happening at the same time -- it only prevents
// the full Render from happening until these updates finish.
// Other direct resources must have their own separate Write locks to protect them.
// It is automatically called at the start of HandleEvents, so all
// standard Event-driven updates are automatically covered.
// Other update entry points, such as animations, need to call this.
// Always call corresponding Unlock in defer!
func (rc *RenderContext) ReadLock() {
	rc.Mu.RLock()
}

// ReadUnlock must be called for each ReadLock, when done.
func (rc *RenderContext) ReadUnlock() {
	rc.Mu.RUnlock()
}

//////////////////////////////////////////////////////////////////////
//  RenderScenes

// RenderScenes are a list of Scene objects, compiled in rendering order,
// whose Pixels images are composed directly to the RenderWin window.
type RenderScenes struct {

	// starting index for this set of Scenes
	StartIdx int

	// max index (exclusive) for this set of Scenes
	MaxIdx int

	// set to true to flip Y axis in drawing these images
	FlipY bool

	// ordered list of scenes -- index is Drawer image index.
	Scenes []*Scene
}

// SetIdxRange sets the index range based on starting index and n
func (rs *RenderScenes) SetIdxRange(st, n int) {
	rs.StartIdx = st
	rs.MaxIdx = st + n
}

// Reset resets the list
func (rs *RenderScenes) Reset() {
	rs.Scenes = nil
}

// Add adds a new node, returning index
func (rs *RenderScenes) Add(sc *Scene) int {
	if sc.Pixels == nil {
		return -1
	}
	idx := len(rs.Scenes)
	if idx >= rs.MaxIdx {
		fmt.Printf("gi.RenderScenes: ERROR too many Scenes to render all of them!  Max: %d\n", rs.MaxIdx)
		return -1
	}
	rs.Scenes = append(rs.Scenes, sc)
	return idx
}

// SetImages calls drw.SetGoImage on all updated Scene images
func (rs *RenderScenes) SetImages(drw *vdraw.Drawer) {
	for i, sc := range rs.Scenes {
		if sc.HasFlag(ScIsUpdating) || !sc.HasFlag(ScImageUpdated) {
			continue
		}
		drw.SetGoImage(i, 0, sc.Pixels, vgpu.NoFlipY)
		sc.SetFlag(false, ScImageUpdated)
	}
}

// DrawAll does drw.Copy drawing call for all Scenes,
// using proper TextureSet for each of vgpu.MaxTexturesPerSet Scenes.
func (rs *RenderScenes) DrawAll(drw *vdraw.Drawer) {
	nPerSet := vgpu.MaxTexturesPerSet

	for i, sc := range rs.Scenes {
		set := i / nPerSet
		if i%nPerSet == 0 && set > 0 {
			drw.UseTextureSet(set)
		}
		bb := sc.Pixels.Bounds()
		op := vdraw.Over
		if i == 0 {
			op = vdraw.Src
		}
		drw.Copy(i, 0, sc.Geom.Pos, bb, op, rs.FlipY)
	}
}

//////////////////////////////////////////////////////////////////////
//  RenderWin methods

func (w *RenderWin) RenderCtx() *RenderContext {
	return w.StageMgr.RenderCtx
}

// RenderWindow performs all rendering based on current StageMgr config.
// It sets the Write lock on RenderCtx Mutex, so nothing else can update
// during this time.  All other updates are done with a Read lock so they
// won't interfere with each other.
func (w *RenderWin) RenderWindow() {
	fmt.Printf("start render\n")
	w.RenderCtx().WriteLock()
	defer w.RenderCtx().WriteUnlock()

	stageMods, sceneMods := w.StageMgr.UpdateAll() // handles all Scene / Widget updates!
	if !stageMods && !sceneMods {                  // nothing to do!
		fmt.Printf("no mods\n")
		return
	}

	if stageMods {
		w.GatherScenes()
	}
	w.DrawScenes()
	fmt.Printf("done render\n")
}

// DrawScenes does the drawing of RenderScenes to the window.
func (w *RenderWin) DrawScenes() {
	if !w.IsVisible() || w.GoosiWin.IsMinimized() {
		if WinEventTrace {
			fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Name)
		}
		return
	}
	// pr := prof.Start("win.DrawScenes")

	drw := w.GoosiWin.Drawer()
	rs := &w.RenderScenes

	rs.SetImages(drw) // ensure all updated images copied

	drw.SyncImages()
	drw.StartDraw(0)
	drw.UseTextureSet(0)
	drw.Scale(0, 0, drw.Surf.Format.Bounds(), image.Rectangle{}, draw.Src, vgpu.NoFlipY)
	rs.DrawAll(drw)

	// todo:
	// drw.UseTextureSet(2)
	// w.DrawSprites()

	drw.EndDraw()

	// pr.End()
}

// GatherScenes finds all the Scene elements that drive rendering
// into the RenderScenes list.
func (w *RenderWin) GatherScenes() {
	rs := &w.RenderScenes
	rs.Reset()

	sm := &w.StageMgr
	sz := sm.Stack.Len()
	if sz == 0 {
		return // shouldn't happen!
	}

	// first, find the top-level window:
	winIdx := 0
	for i := sz - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i).AsMain()
		if st.Type == Window {
			rs.Add(st.Scene)
			winIdx = i
			break
		}
	}

	// then add everyone above that
	for i := winIdx; i < sz; i++ {
		st := sm.Stack.ValByIdx(i).AsMain()
		rs.Add(st.Scene)
	}

	top := sm.Stack.ValByIdx(sz - 1).AsMain()

	// then add the popups for the top main stage
	for _, kv := range top.PopupMgr.Stack.Order {
		st := kv.Val.AsBase()
		rs.Add(st.Scene)
	}

	if WinEventTrace {
		fmt.Printf("n scenes: %d\n", len(rs.Scenes))
	}
}
