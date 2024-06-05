// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"

	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
)

// NewMainStage returns a new MainStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewMainStage call.
// Use an appropriate Run call at the end to start the Stage running.
func NewMainStage(typ StageTypes, sc *Scene) *Stage {
	st := &Stage{}
	st.SetType(typ)
	st.SetScene(sc)
	st.Popups = &Stages{}
	st.Popups.Main = st
	st.Main = st
	return st
}

// RunMainWindow creates a new main window from the body,
// runs it, starts the app's main loop, and waits for all windows
// to close. It should typically be called once by every app at
// the end of their main function. It can not be called more than
// once for one app. For secondary windows, see [Body.RunWindow].
func (bd *Body) RunMainWindow() {
	bd.RunWindow().Wait()
}

// RunWindow returns and runs a new [WindowStage] that is placed in
// a new system window on multi-window platforms.
// See [Body.NewWindow] to make a window without running it.
// For the first window of your app, you should typically call
// [Body.RunMainWindow] instead.
func (bd *Body) RunWindow() *Stage {
	return bd.NewWindow().Run()
}

// NewWindow returns a new [WindowStage] that is placed in
// a new system window on multi-window platforms.
// You must call [Stage.Run] to run the window; see [Body.RunWindow]
// for a version that automatically runs it.
func (bd *Body) NewWindow() *Stage {
	ms := NewMainStage(WindowStage, bd.Scene)
	ms.SetNewWindow(true)
	return ms
}

func (st *Stage) addDialogParts() *Stage {
	if st.FullWindow {
		return st
	}
	sc := st.Scene
	parts := sc.NewParts()
	parts.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(0, 1)
		s.Gap.Zero()
	})
	mv := NewHandle(parts).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	}).StyleFinal(func(s *styles.Style) {
		s.Cursor = cursors.Move
	})
	mv.SetName("move")
	mv.OnChange(func(e events.Event) {
		e.SetHandled()
		pd := e.PrevDelta()
		np := sc.SceneGeom.Pos.Add(pd)
		np.X = max(np.X, 0)
		np.Y = max(np.Y, 0)
		rw := sc.RenderWindow()
		sz := rw.SystemWindow.Size()
		mx := sz.X - int(sc.SceneGeom.Size.X)
		my := sz.Y - int(sc.SceneGeom.Size.Y)
		np.X = min(np.X, mx)
		np.Y = min(np.Y, my)
		sc.SceneGeom.Pos = np
		sc.NeedsRender()
	})
	rsz := NewHandle(parts).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.FillMargin = false
	}).StyleFinal(func(s *styles.Style) {
		s.Cursor = cursors.ResizeNWSE
		s.Min.Set(units.Em(1))
	})
	rsz.SetName("resize")
	rsz.OnChange(func(e events.Event) {
		e.SetHandled()
		pd := e.PrevDelta()
		np := sc.SceneGeom.Size.Add(pd)
		minsz := 100
		np.X = max(np.X, minsz)
		np.Y = max(np.Y, minsz)
		ng := sc.SceneGeom
		ng.Size = np
		sc.Resize(ng)
	})
	return st
}

func (st *Stage) InheritBars() {
	st.Scene.InheritBarsWidget(st.Context)
}

// FirstWindowStages creates a temporary [Stages] for the first window
// to be able to get sizing information prior to having a RenderWindow,
// based on the system App Screen Size. Only adds a RenderContext.
func (st *Stage) FirstWindowStages() *Stages {
	ms := &Stages{}
	ms.RenderContext = NewRenderContext()
	return ms
}

// ConfigMainStage does main-stage configuration steps
func (st *Stage) ConfigMainStage() {
	if st.NewWindow {
		st.FullWindow = true
	}
	// if we are on mobile, we can never have new windows
	if TheApp.Platform().IsMobile() {
		st.NewWindow = false
	}
	if st.FullWindow || st.NewWindow {
		st.Scrim = false
	}
	sc := st.Scene
	sc.MakeSceneBars()
	sc.MakeSceneWidgets()
}

// RunWindow runs a Window with current settings.
func (st *Stage) RunWindow() *Stage {
	sc := st.Scene
	if CurrentRenderWindow == nil {
		// If we have no current render window, we need to be in a new window,
		// and we need a *temporary* Mains to get initial pref size
		st.SetMains(st.FirstWindowStages())
	} else {
		st.SetMains(&CurrentRenderWindow.Mains)
	}
	st.ConfigMainStage()

	sz := st.RenderContext.Geom.Size
	// offscreen windows always consider pref size because
	// they must be unbounded by any previous window sizes
	// non-offscreen mobile windows must take up the whole window
	// and thus don't consider pref size
	// desktop new windows and non-full windows can pref size
	if TheApp.Platform() == system.Offscreen ||
		(!TheApp.Platform().IsMobile() &&
			(st.NewWindow || !st.FullWindow || CurrentRenderWindow == nil)) {
		sz = sc.PrefSize(sz)
		// on offscreen, we don't want any extra space, as we want the smallest
		// possible representation of the content
		// also, on offscreen, if the new size is bigger than the current size,
		// we need to resize the window
		if TheApp.Platform() == system.Offscreen {
			if CurrentRenderWindow != nil {
				csz := CurrentRenderWindow.SystemWindow.Size()
				nsz := csz
				if sz.X > csz.X {
					nsz.X = sz.X
				}
				if sz.Y > csz.Y {
					nsz.Y = sz.Y
				}
				if nsz != csz {
					CurrentRenderWindow.SystemWindow.SetSize(nsz)
					system.TheApp.GetScreens()
				}
			}
		} else {
			// on other platforms, we want extra space and a minimum window size
			sz = sz.Add(image.Pt(20, 20))
			if st.NewWindow {
				// we require the window to be at least half of the screen size
				scsz := system.TheApp.Screen(0).PixSize // TODO(kai): is there a better screen to get here?
				sz = image.Pt(max(sz.X, scsz.X/2), max(sz.Y, scsz.Y/2))
			}
		}
	}
	st.Mains = nil // reset
	if DebugSettings.WinRenderTrace {
		fmt.Println("MainStage.RunWindow: Window Size:", sz)
	}

	if st.NewWindow || CurrentRenderWindow == nil {
		sc.Resize(math32.Geom2DInt{st.RenderContext.Geom.Pos, sz})
		win := st.NewRenderWindow()
		MainRenderWindows.Add(win)
		CurrentRenderWindow = win
		win.GoStartEventLoop()
		return st
	}
	if st.Context != nil {
		ms := st.Context.AsWidget().Scene.Stage.Mains
		msc := ms.Top().Scene
		sc.SceneGeom.Size = sz
		sc.FitInWindow(msc.SceneGeom) // does resize
		ms.Push(st)
		st.SetMains(ms)
	} else {
		ms := &CurrentRenderWindow.Mains
		msc := ms.Top().Scene
		sc.SceneGeom.Size = sz
		sc.FitInWindow(msc.SceneGeom) // does resize
		ms.Push(st)
		st.SetMains(ms)
	}
	return st
}

// GetValidContext ensures that the Context is non-nil and has a valid
// Scene pointer, using CurrentRenderWindow if the current Context is not valid.
// If CurrentRenderWindow is nil (should not happen), then it returns false and
// the calling function must bail.
func (st *Stage) GetValidContext() bool {
	if st.Context == nil || st.Context.This() == nil || st.Context.AsWidget().Scene == nil {
		if CurrentRenderWindow == nil {
			slog.Error("Stage.Run: Context is nil and CurrentRenderWindow is nil, so cannot Run", "Name", st.Name, "Title", st.Title)
			return false
		}
		st.Context = CurrentRenderWindow.Mains.Top().Scene
	}
	return true
}

// RunDialog runs a Dialog with current settings.
func (st *Stage) RunDialog() *Stage {
	if !st.GetValidContext() {
		return st
	}
	ctx := st.Context.AsWidget()

	// if our main stages are nil, we wait until our context is shown and then try again
	if ctx.Scene.Stage == nil || ctx.Scene.Stage.Mains == nil {
		ctx.OnShow(func(e events.Event) {
			st.RunDialog()
		})
		return st
	}

	ms := ctx.Scene.Stage.Mains

	sc := st.Scene
	st.ConfigMainStage()
	st.addDialogParts()
	sc.SceneGeom.Pos = st.Pos

	st.SetMains(ms) // temporary for prefs

	sz := ms.RenderContext.Geom.Size
	if !st.FullWindow || st.NewWindow {
		sz = sc.PrefSize(sz)
		sz = sz.Add(image.Point{50, 50})
		sc.Events.StartFocusFirst = true // popup dialogs always need focus
	}
	if DebugSettings.WinRenderTrace {
		slog.Info("MainStage.RunDialog", "size", sz)
	}

	if st.NewWindow {
		st.Mains = nil
		sc.Resize(math32.Geom2DInt{st.RenderContext.Geom.Pos, sz})
		st.Type = WindowStage            // critical: now is its own window!
		sc.SceneGeom.Pos = image.Point{} // ignore pos
		win := st.NewRenderWindow()
		DialogRenderWindows.Add(win)
		CurrentRenderWindow = win
		win.GoStartEventLoop()
		return st
	}
	sc.SceneGeom.Size = sz
	// fmt.Println("dlg:", sc.SceneGeom, "win:", winGeom)
	sc.FitInWindow(st.RenderContext.Geom) // does resize
	ms.Push(st)
	// st.SetMains(ms) // already set
	return st
}

func (st *Stage) NewRenderWindow() *RenderWindow {
	name := st.Name
	title := st.Title
	opts := &system.NewWindowOptions{
		Title:     title,
		Icon:      AppIconImages(),
		Size:      st.Scene.SceneGeom.Size,
		StdPixels: false,
	}
	wgp := TheWindowGeometrySaver.Pref(title, nil)
	if TheApp.Platform() != system.Offscreen && wgp != nil {
		TheWindowGeometrySaver.SettingStart()
		opts.Size = wgp.Size()
		opts.Pos = wgp.Pos()
		opts.StdPixels = false
		if _, found := AllRenderWindows.FindName(name); found { // offset from existing
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
		if wgp.Fullscreen {
			opts.SetFullscreen()
		}
	}
	win := NewRenderWindow(name, title, opts)
	TheWindowGeometrySaver.SettingEnd()
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.SetFlag(true, WindowHasSavedGeom)
	}
	AllRenderWindows.Add(win)
	// initialize Mains
	win.Mains.RenderWindow = win
	win.Mains.RenderContext = NewRenderContext() // sets defaults according to Screen
	// note: win is not yet created by the OS and we don't yet know its actual size
	// or dpi.
	win.Mains.Push(st)
	st.SetMains(&win.Mains)
	return win
}

// MainHandleEvent handles main stage events
func (st *Stage) MainHandleEvent(e events.Event) {
	if st.Scene == nil {
		return
	}
	st.Popups.PopupHandleEvent(e)
	if e.IsHandled() || (st.Popups != nil && st.Popups.TopIsModal()) {
		if DebugSettings.EventTrace && e.Type() != events.MouseMove {
			fmt.Println("Event handled by popup:", e)
		}
		return
	}
	e.SetLocalOff(st.Scene.SceneGeom.Pos)
	st.Scene.Events.HandleEvent(e)
}

// MainHandleEvent calls MainHandleEvent on relevant stages in reverse order.
func (sm *Stages) MainHandleEvent(e events.Event) {
	n := sm.Stack.Len()
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		st.MainHandleEvent(e)
		if e.IsHandled() || st.Modal || st.Type == WindowStage || st.FullWindow {
			break
		}
	}
}
