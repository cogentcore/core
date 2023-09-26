// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/goosi"
)

// StageMgr is the outer Stage manager for Window, Dialog, Sheet
type StageMgr struct {
	StageMgrBase

	// growing stack of viewing history of all stages.
	History []*Stage

	// sprites are named images that are rendered last overlaying everything else.
	Sprites Sprites `json:"-" xml:"-" desc:"sprites are named images that are rendered last overlaying everything else."`

	// name of sprite that is being dragged -- sprite event function is responsible for setting this.
	SpriteDragging string `json:"-" xml:"-" desc:"name of sprite that is being dragged -- sprite event function is responsible for setting this."`
}

func (st *Stage) NewRenderWin() *RenderWin {
	name := st.Name
	title := st.Title
	opts := &goosi.NewRenderWinOptions{
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

func (st *StageMgr) HandleEvent(evi goosi.Event) {

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
