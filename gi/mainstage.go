// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log/slog"

	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/mat32/v2"
)

// NewMainStage returns a new MainStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewMainStage call.
// Use an appropriate Run call at the end to start the Stage running.
func NewMainStage(typ StageTypes, sc *Scene) *Stage {
	st := &Stage{}
	st.SetType(typ)
	st.SetScene(sc)
	st.PopupMgr = &StageMgr{}
	st.PopupMgr.Main = st
	st.Main = st
	return st
}

// NewWindow returns a new Window stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func (sc *Scene) NewWindow() *Stage {
	ms := NewMainStage(WindowStage, sc)
	ms.SetNewWindow(true)
	return ms
}

// NewWindow returns a new Window stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func (bd *Body) NewWindow() *Stage {
	return bd.Sc.NewWindow()
}

// NewDialog in dialogs.go

// NewSheet returns a new Sheet stage with given scene contents,
// in connection with given widget (which provides key context).
// for given side (e.g., Bottom or LeftSide).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSheet(sc *Scene, side StageSides) *Stage {
	return NewMainStage(SheetStage, sc).SetSide(side)
}

/////////////////////////////////////////////////////
//		Decorate

// only called when !NewWindow
func (st *Stage) AddWindowDecor() *Stage {
	return st
}

func (st *Stage) AddDialogDecor() *Stage {
	return st
}

func (st *Stage) AddSheetDecor() *Stage {
	// todo: handle based on side
	return st
}

func (st *Stage) InheritBars() {
	st.Scene.InheritBarsWidget(st.Context)
}

// FirstWinManager creates a temporary Main StageMgr for the first window
// to be able to get sizing information prior to having a RenderWin,
// based on the goosi App Screen Size. Only adds a RenderCtx.
func (st *Stage) FirstWinManager() *StageMgr {
	ms := &StageMgr{}
	ms.RenderCtx = NewRenderContext()
	return ms
}

// ConfigMainStage does main-stage configuration steps
func (st *Stage) ConfigMainStage() {
	if st.NewWindow {
		st.FullWindow = true
	}
	if goosi.TheApp.Platform().IsMobile() {
		st.NewWindow = false
	}
	sc := st.Scene
	st.AddWindowDecor() // sensitive to cases
	sc.ConfigSceneBars()
	sc.ConfigSceneWidgets()
}

// RunWindow runs a Window with current settings.
func (st *Stage) RunWindow() *Stage {
	// note: need a *temporary* MainMgr to get initial pref size
	sc := st.Scene
	if CurRenderWin == nil {
		st.SetMainMgr(st.FirstWinManager())
	} else {
		if sc.App == nil { // inherit apps
			sc.App = CurRenderWin.MainStageMgr.Top().Scene.App
		}
		st.SetMainMgr(&CurRenderWin.MainStageMgr)
	}
	st.ConfigMainStage()

	sz := st.RenderCtx.Size
	// non-new full windows must take up the whole window
	// and thus don't consider pref size
	if st.NewWindow || !st.FullWindow || CurRenderWin == nil {
		sz = sc.PrefSize(sz)
		sz = sz.Add(image.Point{20, 20})
	}
	st.MainMgr = nil // reset
	if WinRenderTrace {
		fmt.Println("MainStage.RunWindow: Window Size:", sz)
	}

	if st.NewWindow || CurRenderWin == nil {
		sc.Resize(sz)
		win := st.NewRenderWin()
		if CurRenderWin == nil {
			CurRenderWin = win
		}
		win.GoStartEventLoop()
		return st
	}
	if st.Context != nil {
		ms := st.Context.AsWidget().Sc.MainStageMgr()
		msc := ms.Top().Scene
		sc.SceneGeom.Size = sz
		sc.FitInWindow(msc.SceneGeom) // does resize
		ms.Push(st)
		st.SetMainMgr(ms)
	} else {
		ms := &CurRenderWin.MainStageMgr
		msc := ms.Top().Scene
		sc.SceneGeom.Size = sz
		sc.FitInWindow(msc.SceneGeom) // does resize
		ms.Push(st)
		st.SetMainMgr(ms)
	}
	return st
}

// RunDialog runs a Dialog with current settings.
func (st *Stage) RunDialog() *Stage {
	if st.Context == nil {
		if CurRenderWin == nil {
			slog.Error("RunDialog: Context is nil and CurRenderWin is nil, cannot Run!", "Dialog", st.Name, "Title", st.Title)
			return nil
		}
		st.Context = CurRenderWin.MainStageMgr.Top().Scene
	}
	ctx := st.Context.AsWidget()
	ms := ctx.Sc.MainStageMgr()

	// if our main stage manager is nil, we wait until our context is shown and then try again
	if ms == nil {
		slog.Error("RunDialog: CurRenderWin is nil")
		ctx.OnShow(func(e events.Event) {
			st.RunDialog()
		})
		return st
	}

	st.ConfigMainStage()
	sc := st.Scene
	sc.App = ctx.Sc.App
	sc.SceneGeom.Pos = st.Pos

	st.SetMainMgr(ms) // temporary for prefs
	winsz := ms.RenderCtx.Size

	sz := winsz
	if !st.FullWindow {
		sz = sc.PrefSize(winsz)
		sz = sz.Add(image.Point{50, 50})
		sc.EventMgr.StartFocusFirst = true // popup dialogs always need focus
	}
	if WinRenderTrace {
		slog.Info("MainStage.RunDialog", "size", sz)
	}

	if st.NewWindow {
		st.MainMgr = nil
		sc.Resize(sz)
		st.Type = WindowStage            // critical: now is its own window!
		sc.SceneGeom.Pos = image.Point{} // ignore pos
		win := st.NewRenderWin()
		DialogRenderWins.Add(win)
		win.GoStartEventLoop()
		return st
	}
	winGeom := mat32.Geom2DInt{Size: winsz}
	sc.SceneGeom.Size = sz
	// fmt.Println("dlg:", sc.SceneGeom, "win:", winGeom)
	sc.FitInWindow(winGeom) // does resize
	ms.Push(st)
	// st.SetMainMgr(ms) // already set
	return st
}

// RunSheet runs a Sheet with current settings.
// Sheet MUST have context set.
func (st *Stage) RunSheet() *Stage {
	ctx := st.Context.AsWidget()
	ms := ctx.Sc.MainStageMgr()

	st.ConfigMainStage() // should set pos and size for side
	ms.Push(st)
	st.SetMainMgr(ms)
	return st
}

func (st *Stage) NewRenderWin() *RenderWin {
	if st.Scene == nil {
		slog.Error("MainStage.NewRenderWin: Scene is nil")
	}
	name := st.Name
	title := st.Title
	opts := &goosi.NewWindowOptions{
		Title: title, Size: st.Scene.SceneGeom.Size, StdPixels: false,
	}
	wgp := WinGeomMgr.Pref(title, nil)
	if goosi.TheApp.Platform() != goosi.Offscreen && wgp != nil {
		WinGeomMgr.SettingStart()
		opts.Size = wgp.Size()
		opts.Pos = wgp.Pos()
		opts.StdPixels = false
		// fmt.Printf("got prefs for %v: size: %v pos: %v\n", name, opts.Size, opts.Pos)
		if _, found := AllRenderWins.FindName(name); found { // offset from existing
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
		if wgp.Fullscreen {
			opts.SetFullscreen()
		}
	}
	win := NewRenderWin(name, title, opts)
	WinGeomMgr.SettingEnd()
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.SetFlag(true, WinHasGeomPrefs)
	}
	AllRenderWins.Add(win)
	MainRenderWins.Add(win)
	WinNewCloseStamp()
	// initialize MainStageMgr
	win.MainStageMgr.RenderWin = win
	win.MainStageMgr.RenderCtx = NewRenderContext() // sets defaults according to Screen
	// note: win is not yet created by the OS and we don't yet know its actual size
	// or dpi.
	win.MainStageMgr.Push(st)
	st.SetMainMgr(&win.MainStageMgr)
	return win
}

// MainHandleEvent handles main stage events
func (st *Stage) MainHandleEvent(e events.Event) {
	if st.Scene == nil {
		return
	}
	st.PopupMgr.PopupHandleEvent(e)
	if e.IsHandled() || st.PopupMgr.TopIsModal() {
		if EventTrace && e.Type() != events.MouseMove {
			fmt.Println("Event handled by popup:", e)
		}
		return
	}
	e.SetLocalOff(st.Scene.SceneGeom.Pos)
	st.Scene.EventMgr.HandleEvent(e)
}

// MainHandleEvent calls MainHandleEvent on relevant stages in reverse order.
func (sm *StageMgr) MainHandleEvent(e events.Event) {
	n := sm.Stack.Len()
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i)
		st.MainHandleEvent(e)
		if e.IsHandled() || st.Modal || st.Type == WindowStage || st.FullWindow {
			break
		}
	}
}

/*
todo: main menu on full win

// ConfigVLay creates and configures the vertical layout as first child of
// Scene, and installs MainMenu as first element of layout.
func (w *RenderWin) ConfigVLay() {
	sc := w.Scene
	updt := sc.UpdateStart()
	defer sc.UpdateEnd(updt)
	if !sc.HasChildren() {
		sc.NewChild(LayoutType, "main-vlay")
	}
	w.Scene.Frame = sc.Child(0).Embed(LayoutType).(*Layout)
	if !w.Scene.Frame.HasChildren() {
		w.Scene.Frame.NewChild(TypeMenuBar, "main-menu")
	}
	w.MainMenu = w.Scene.Frame.Child(0).(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
}

// AddMainMenu installs MainMenu as first element of main layout
// used for dialogs that don't always have a main menu -- returns
// menubar -- safe to call even if there is a menubar
func (w *RenderWin) AddMainMenu() *MenuBar {
	sc := w.Scene
	updt := sc.UpdateStart()
	defer sc.UpdateEnd(updt)
	if !sc.HasChildren() {
		sc.NewChild(LayoutType, "main-vlay")
	}
	w.Scene.Frame = sc.Child(0).Embed(LayoutType).(*Layout)
	if !w.Scene.Frame.HasChildren() {
		w.MainMenu = w.Scene.Frame.NewChild(TypeMenuBar, "main-menu").(*MenuBar)
	} else {
		mmi := w.Scene.Frame.ChildByName("main-menu", 0)
		if mmi != nil {
			mm := mmi.(*MenuBar)
			w.MainMenu = mm
			return mm
		}
	}
	w.MainMenu = w.Scene.Frame.InsertNewChild(TypeMenuBar, 0, "main-menu").(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
	return w.MainMenu
}

*/
