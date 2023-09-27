// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/goosi"
)

// MainStage manages a Scene serving as content for a
// Window, Dialog, or Sheet, which are larger and potentially
// complex Scenes that persist until dismissed, and can have
// Decor widgets that control display.
// MainStages live in a StageMgr associated with a RenderWin window,
// and manage their own set of PopupStages via a PopupStageMgr,
// and handle events using an EventMgr.
type MainStage struct {
	StageBase

	// Data is item represented by this main stage -- used for recycling windows
	Data any

	// manager for the popups in this stage
	PopupMgr PopupStageMgr

	// the parent stage manager for this stage, which lives in a RenderWin
	StageMgr *MainStageMgr

	// event manager for this stage
	EventMgr EventMgr
}

// AsMain returns this stage as a MainStage (for Main Window, Dialog, Sheet) types.
// returns nil for PopupStage types.
func (st *MainStage) AsMain() *MainStage {
	return st
}

func (st *MainStage) MainMgr() *MainStageMgr {
	return st.StageMgr
}

func (st *MainStage) RenderCtx() *RenderContext {
	if st.StageMgr == nil {
		return nil
	}
	return st.StageMgr.RenderCtx
}

// NewMainStage returns a new MainStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewMainStage call.
// Use an appropriate Run call at the end to start the Stage running.
func NewMainStage(typ StageTypes, sc *Scene, ctx Widget) *MainStage {
	st := &MainStage{}
	st.Stage = st
	st.SetType(typ)
	st.SetScene(sc)
	st.CtxWidget = ctx
	st.PopupMgr.Main = st
	st.EventMgr.Main = st
	return st
}

// NewWindow returns a new Window stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewWindow(sc *Scene) *MainStage {
	return NewMainStage(Window, sc, nil)
}

// NewDialog returns a new Dialog stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewDialog(sc *Scene, ctx Widget) *MainStage {
	return NewMainStage(Dialog, sc, ctx)
}

// NewSheet returns a new Sheet stage with given scene contents,
// in connection with given widget (which provides key context).
// for given side (e.g., Bottom or LeftSide).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSheet(sc *Scene, side StageSides, ctx Widget) *MainStage {
	return NewMainStage(Sheet, sc, ctx).SetSide(side).(*MainStage)
}

/////////////////////////////////////////////////////
//		Decorate

// only called when !OwnWin
func (st *MainStage) AddWindowDecor() *MainStage {
	if st.Back {
		but := NewButton(&st.Scene.Decor, "win-back")
		_ = but
		// todo: do more button config
	}
	return st
}

func (st *MainStage) AddDialogDecor() *MainStage {
	// todo: moveable, resizable
	return st
}

func (st *MainStage) AddSheetDecor() *MainStage {
	// todo: handle based on side
	return st
}

// RunWindow runs a Window with current settings.
func (st *MainStage) RunWindow() *MainStage {
	st.AddWindowDecor() // sensitive to cases
	st.Scene.Config()
	if st.OwnWin {
		win := st.NewRenderWin()
		win.GoStartEventLoop()
		return st
	}
	if CurRenderWin == nil {
		CurRenderWin = st.NewRenderWin()
		CurRenderWin.GoStartEventLoop()
		return st
	}
	return st
}

// RunDialog runs a Dialog with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *MainStage) RunDialog() *MainStage {
	st.AddDialogDecor()
	if st.OwnWin {
		win := st.NewRenderWin()
		win.GoStartEventLoop()
		return st
	}
	if CurRenderWin == nil {
		// todo: error here -- must have main window!
		return nil
	}
	// todo: need some kind of linkage here for dialog relative to existing window
	// probably just CurRenderWin but it needs to be a stack or updated properly etc.
	CurRenderWin.StageMgr.Push(st)
	return st
}

// RunSheet runs a Sheet with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *MainStage) RunSheet() *MainStage {
	st.AddSheetDecor()
	if CurRenderWin == nil {
		// todo: error here -- must have main window!
		return nil
	}
	// todo: need some kind of linkage here for dialog relative to existing window
	// probably just CurRenderWin but it needs to be a stack or updated properly etc.
	CurRenderWin.StageMgr.Push(st)
	return st
}

func (st *MainStage) NewRenderWin() *RenderWin {
	name := st.Name
	title := st.Title
	opts := &goosi.NewWindowOptions{
		Title: title, Size: image.Point{st.Width, st.Height}, StdPixels: true,
	}
	wgp := WinGeomMgr.Pref(name, nil)
	if wgp != nil {
		WinGeomMgr.SettingStart()
		opts.Size = wgp.Size()
		opts.Pos = wgp.Pos()
		opts.StdPixels = false
		// fmt.Printf("got prefs for %v: size: %v pos: %v\n", name, opts.Size, opts.Pos)
		if _, found := AllRenderWins.FindName(name); found { // offset from existing
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
	}
	win := NewRenderWin(name, title, opts)
	WinGeomMgr.SettingEnd()
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.SetFlag(true, WinFlagHasGeomPrefs)
	}
	AllRenderWins.Add(win)
	MainRenderWins.Add(win)
	WinNewCloseStamp()
	win.StageMgr.Push(st)
	return win
}

func (st *MainStage) Delete() {
	st.PopupMgr.CloseAll()
	if st.Scene != nil {
		st.Scene.Delete()
	}
	st.Scene = nil
	st.StageMgr = nil
}

func (st *MainStage) Resize(sz image.Point) {
	if st.Scene == nil {
		return
	}
	switch st.Type {
	case Window:
		st.Scene.Resize(sz)
		// todo: other types fit in constraints
	}
}

func (st *MainStage) StageAdded(smi StageMgr) {
	st.StageMgr = smi.AsMainMgr()
	// todo: ?
	// if pfoc != nil {
	// 	sm.EventMgr.PushFocus(pfoc)
	// } else {
	// 	sm.EventMgr.PushFocus(st)
	// }
}

// HandleEvent handles all the non-Window events
func (st *MainStage) HandleEvent(evi goosi.Event) {
	if st.Scene == nil {
		return
	}
	// todo: probably want to pre-filter here and have the event manager
	// deal with all of this, not just going to the popup right away
	st.PopupMgr.HandleEvent(evi)
	if evi.IsHandled() {
		return
	}
	evi.SetLocalOff(st.Scene.Geom.Pos)
	st.EventMgr.HandleEvent(st.Scene, evi)
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
	w.Scene.Frame.Lay = LayoutVert
	w.MainMenu = w.Scene.Frame.Child(0).(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
}

// ConfigInsets updates the padding on the main layout of the window
// to the inset values provided by the RenderWin window.
func (w *RenderWin) ConfigInsets() {
	mainVlay, ok := w.Scene.ChildByName("main-vlay", 0).(*Layout)
	if ok {
		insets := w.RenderWin.Insets()
		mainVlay.AddStyler(func(w *WidgetBase, s *gist.Style) {
			mainVlay.Style.Padding.Set(
				units.Dot(insets.Top),
				units.Dot(insets.Right),
				units.Dot(insets.Bottom),
				units.Dot(insets.Left),
			)
		})
	}

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
