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
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// MainStage manages a Scene serving as content for a
// Window, Dialog, or Sheet, which are larger and potentially
// complex Scenes that persist until dismissed, and can have
// Decor widgets that control display.  MainStage has sprites.
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

	// sprites are named images that are rendered last overlaying everything else.
	Sprites Sprites `json:"-" xml:"-"`

	// name of sprite that is being dragged -- sprite event function is responsible for setting this.
	SpriteDragging string `json:"-" xml:"-"`
}

// AsMain returns this stage as a MainStage (for Main Window, Dialog, Sheet) types.
// returns nil for PopupStage types.
func (st *MainStage) AsMain() *MainStage {
	return st
}

func (st *MainStage) String() string {
	return "MainStage: " + st.StageBase.String()
}

func (st *MainStage) MainMgr() *MainStageMgr {
	return st.StageMgr
}

func (st *MainStage) RenderCtx() *RenderContext {
	if st.StageMgr == nil {
		slog.Error("MainStage has nil StageMgr", "stage", st.Name)
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
	st.This = st
	st.SetType(typ)
	st.SetScene(sc)
	st.CtxWidget = ctx
	st.PopupMgr.Main = st
	st.PopupMgr.This = &st.PopupMgr
	return st
}

// NewWindow returns a new Window stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewWindow(sc *Scene) *MainStage {
	return NewMainStage(WindowStage, sc, nil)
}

// NewDialog in dialogs.go

// NewSheet returns a new Sheet stage with given scene contents,
// in connection with given widget (which provides key context).
// for given side (e.g., Bottom or LeftSide).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSheet(sc *Scene, side StageSides, ctx Widget) *MainStage {
	return NewMainStage(SheetStage, sc, ctx).SetSide(side).(*MainStage)
}

/////////////////////////////////////////////////////
//		Decorate

// SetWindowInsets updates the padding on the Scene
// to the inset values provided by the RenderWin window.
func (st *MainStage) SetWindowInsets() {
	if st.StageMgr == nil {
		return
	}
	if st.StageMgr.RenderWin == nil {
		return
	}
	// insets := st.StageMgr.RenderWin.GoosiWin.Insets()
	// // fmt.Println(insets)
	// uv := func(val float32) units.Value {
	// 	return units.Custom(func(uc *units.Context) float32 {
	// 		return max(val, uc.Dp(12))
	// 	})
	// }
	// st.Scene.Style(func(s *styles.Style) {
	// 	s.Padding.Set(
	// 		uv(insets.Top),
	// 		uv(insets.Right),
	// 		uv(insets.Bottom),
	// 		uv(insets.Left),
	// 	)
	// })
}

// only called when !NewWindow
func (st *MainStage) AddWindowDecor() *MainStage {
	st.AddTopAppBar()
	return st
}

func (st *MainStage) AddDialogDecor() *MainStage {
	if st.FullWindow {
		st.AddTopAppBar()
	}
	return st
}

func (st *MainStage) AddSheetDecor() *MainStage {
	// todo: handle based on side
	return st
}

// SetDefaultTopAppbar sets the top app bar of the stage to its default value.
// This should not be called by end-user code.
func (st *StageBase) SetDefaultTopAppBar() {
	// first fall back on context widget scene top app bar if we are a dialog
	if st.Type == DialogStage {
		st.Scene.TopAppBar = st.CtxWidget.AsWidget().Sc.TopAppBar
	}
	if st.Scene.TopAppBar == nil {
		st.Scene.TopAppBar = DefaultTopAppBar
	}
}

func (st *MainStage) AddTopAppBar() *MainStage {
	if st.Scene.TopAppBar == nil {
		st.SetDefaultTopAppBar()
		// if still nil, bail
		if st.Scene.TopAppBar == nil {
			return st
		}
	}
	tb := st.Scene.InsertNewChild(TopAppBarType, 0).(*TopAppBar)
	st.Scene.TopAppBar(tb)
	return st
}

// FirstWinManager creates a MainStageMgr for the first window
// to be able to get sizing information prior to having a RenderWin,
// based on the goosi App Screen Size. Only adds a RenderCtx.
func (st *MainStage) FirstWinManager() *MainStageMgr {
	ms := &MainStageMgr{}
	ms.This = ms
	rc := &RenderContext{}
	ms.RenderCtx = rc
	scr := goosi.TheApp.Screen(0)
	rc.Size = scr.Geometry.Size()
	// fmt.Println("Screen Size:", rc.Size)
	rc.SetFlag(true, RenderVisible)
	rc.LogicalDPI = scr.LogicalDPI
	return ms
}

// RunWindow runs a Window with current settings.
func (st *MainStage) RunWindow() *MainStage {
	st.AddWindowDecor() // sensitive to cases
	st.Scene.ConfigScene()

	// note: need a StageMgr to get initial pref size
	if CurRenderWin == nil {
		st.StageMgr = st.FirstWinManager()
	} else {
		st.StageMgr = &CurRenderWin.StageMgr
	}
	sz := st.Scene.PrefSize(st.RenderCtx().Size)
	if WinRenderTrace {
		fmt.Println("MainStage.RunWindow: Window Size:", sz)
	}
	st.Scene.Resize(sz)
	st.Scene.ShowLayoutIter = 0

	if st.NewWindow {
		win := st.NewRenderWin()
		if CurRenderWin == nil {
			CurRenderWin = win
		}
		st.SetWindowInsets()
		win.GoStartEventLoop()
		return st
	}
	if CurRenderWin == nil {
		CurRenderWin = st.NewRenderWin()
		st.SetWindowInsets()
		CurRenderWin.GoStartEventLoop()
		return st
	}
	if st.CtxWidget != nil {
		ms := st.CtxWidget.AsWidget().Sc.MainStageMgr()
		ms.Push(st)
	} else {
		CurRenderWin.StageMgr.Push(st)
	}
	return st
}

// RunDialog runs a Dialog with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *MainStage) RunDialog() *MainStage {
	st.AddDialogDecor()
	st.Scene.ConfigScene()

	ctx := st.CtxWidget.AsWidget()
	ms := ctx.Sc.MainStageMgr()
	if ms == nil {
		slog.Error("RunDialog: CurRenderWin is nil")
		return nil
	}
	winsz := ms.RenderCtx.Size

	sc := st.Scene
	st.StageMgr = ms // temporary
	sz := winsz
	// history-based stages always take up the whole window
	if !st.FullWindow {
		sz = sc.PrefSize(winsz)
	}
	if WinRenderTrace {
		slog.Info("MainStage.RunDialog", "size", sz)
	}

	if st.NewWindow && !goosi.TheApp.Platform().IsMobile() {
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
	return st
}

// RunSheet runs a Sheet with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *MainStage) RunSheet() *MainStage {
	st.AddSheetDecor()
	st.Scene.ConfigScene()
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
	if st.Scene == nil {
		slog.Error("MainStage.NewRenderWin: Scene is nil")
	}
	name := st.Name
	title := st.Title
	opts := &goosi.NewWindowOptions{
		Title: title, Size: st.Scene.SceneGeom.Size, StdPixels: false,
	}
	wgp := WinGeomMgr.Pref(title, nil)
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
		win.SetFlag(true, WinHasGeomPrefs)
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
		st.Scene.Delete(ki.DestroyKids)
	}
	st.Scene = nil
	st.StageMgr = nil
}

func (st *MainStage) Resize(sz image.Point) {
	if st.Scene == nil {
		return
	}
	switch st.Type {
	case WindowStage:
		st.SetWindowInsets()
		st.Scene.Resize(sz)
		// todo: other types fit in constraints
	}
}

// DoUpdate calls DoUpdate on our Scene and UpdateAll on our Popups
// returns stageMods = true if any Popup Stages have been modified
// and sceneMods = true if any Scenes have been modified.
func (st *MainStage) DoUpdate() (stageMods, sceneMods bool) {
	if st.Scene == nil {
		return
	}
	stageMods, sceneMods = st.PopupMgr.UpdateAll()
	scMod := st.Scene.DoUpdate()
	sceneMods = sceneMods || scMod
	// if scMod {
	// 	fmt.Println("main scene mod:", st.Scene.Name)
	// }
	// if stageMods {
	// 	fmt.Println("pop stage mod:", st.Name)
	// }
	return
}

func (st *MainStage) StageAdded(smi StageMgr) {
	st.StageMgr = smi.AsMainMgr()
}

// HandleEvent handles all the non-Window events
func (st *MainStage) HandleEvent(evi events.Event) {
	if st.Scene == nil {
		return
	}
	st.PopupMgr.HandleEvent(evi)
	if evi.IsHandled() || st.PopupMgr.TopIsModal() {
		if EventTrace && evi.Type() != events.MouseMove {
			fmt.Println("Event handled by popup:", evi)
		}
		return
	}
	evi.SetLocalOff(st.Scene.SceneGeom.Pos)
	st.Scene.EventMgr.HandleEvent(evi)
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
