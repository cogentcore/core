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
	"cogentcore.org/core/tree"
)

// newMainStage returns a new MainStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the newMainStage call.
// Use an appropriate Run call at the end to start the Stage running.
func newMainStage(typ StageTypes, sc *Scene) *Stage {
	st := &Stage{}
	st.setType(typ)
	st.setScene(sc)
	st.popups = &stages{}
	st.popups.main = st
	st.Main = st
	return st
}

// RunMainWindow creates a new main window from the body,
// runs it, starts the app's main loop, and waits for all windows
// to close. It should typically be called once by every app at
// the end of their main function. It can not be called more than
// once for one app. For secondary windows, see [Body.RunWindow].
func (bd *Body) RunMainWindow() {
	if ExternalParent != nil {
		bd.handleExternalParent()
		return
	}
	bd.RunWindow()
	Wait()
}

// ExternalParent is a parent widget external to this program.
// If it is set, calls to [Body.RunWindow] before [Wait] and
// calls to [Body.RunMainWindow] will add the [Body] to this
// parent instead of creating a new window. It should typically not be
// used by end users; it is used in yaegicore and for pre-rendering apps
// as HTML that can be used as a preview and for SEO purposes.
var ExternalParent Widget

// waitCalled is whether [Wait] has been called. It is used for
// [ExternalParent] logic in [Body.RunWindow].
var waitCalled bool

// RunWindow returns and runs a new [WindowStage] that is placed in
// a new system window on multi-window platforms.
// See [Body.NewWindow] to make a window without running it.
// For the first window of your app, you should typically call
// [Body.RunMainWindow] instead.
func (bd *Body) RunWindow() *Stage {
	if ExternalParent != nil && !waitCalled {
		bd.handleExternalParent()
		return nil
	}
	return bd.NewWindow().Run()
}

// handleExternalParent handles [ExternalParent] logic for
// [Body.RunWindow] and [Body.RunMainWindow].
func (bd *Body) handleExternalParent() {
	ExternalParent.AsWidget().AddChild(bd)
	// we must set the correct scene for each node
	bd.WalkDown(func(n tree.Node) bool {
		n.(Widget).AsWidget().Scene = bd.Scene
		return tree.Continue
	})
	// we must not get additional scrollbars here
	bd.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowVisible)
	})
}

// NewWindow returns a new [WindowStage] that is placed in
// a new system window on multi-window platforms.
// You must call [Stage.Run] to run the window; see [Body.RunWindow]
// for a version that automatically runs it.
func (bd *Body) NewWindow() *Stage {
	ms := newMainStage(WindowStage, bd.Scene)
	ms.SetNewWindow(true)
	return ms
}

func (st *Stage) addDialogParts() *Stage {
	if st.FullWindow {
		return st
	}
	sc := st.Scene
	parts := sc.newParts()
	parts.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(0, 1)
		s.Gap.Zero()
	})
	mv := NewHandle(parts)
	mv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	mv.FinalStyler(func(s *styles.Style) {
		s.Cursor = cursors.Move
	})
	mv.SetName("move")
	mv.OnChange(func(e events.Event) {
		e.SetHandled()
		pd := e.PrevDelta()
		np := sc.sceneGeom.Pos.Add(pd)
		np.X = max(np.X, 0)
		np.Y = max(np.Y, 0)
		rw := sc.RenderWindow()
		sz := rw.SystemWindow.Size()
		mx := sz.X - int(sc.sceneGeom.Size.X)
		my := sz.Y - int(sc.sceneGeom.Size.Y)
		np.X = min(np.X, mx)
		np.Y = min(np.Y, my)
		sc.sceneGeom.Pos = np
		sc.NeedsRender()
	})
	rsz := NewHandle(parts)
	rsz.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.FillMargin = false
	})
	rsz.FinalStyler(func(s *styles.Style) {
		s.Cursor = cursors.ResizeNWSE
		s.Min.Set(units.Em(1))
	})
	rsz.SetName("resize")
	rsz.OnChange(func(e events.Event) {
		e.SetHandled()
		pd := e.PrevDelta()
		np := sc.sceneGeom.Size.Add(pd)
		minsz := 100
		np.X = max(np.X, minsz)
		np.Y = max(np.Y, minsz)
		ng := sc.sceneGeom
		ng.Size = np
		sc.resize(ng)
	})
	return st
}

// firstWindowStages creates a temporary [stages] for the first window
// to be able to get sizing information prior to having a RenderWindow,
// based on the system App Screen Size. Only adds a RenderContext.
func (st *Stage) firstWindowStages() *stages {
	ms := &stages{}
	ms.renderContext = newRenderContext()
	return ms
}

// configMainStage does main-stage configuration steps
func (st *Stage) configMainStage() {
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
	sc.makeSceneBars()
	sc.updateScene()
}

// runWindow runs a Window with current settings.
func (st *Stage) runWindow() *Stage {
	sc := st.Scene
	if currentRenderWindow == nil {
		// If we have no current render window, we need to be in a new window,
		// and we need a *temporary* Mains to get initial pref size
		st.setMains(st.firstWindowStages())
	} else {
		st.setMains(&currentRenderWindow.mains)
	}
	st.configMainStage()

	sz := st.renderContext.geom.Size
	// offscreen windows always consider pref size because
	// they must be unbounded by any previous window sizes
	// non-offscreen mobile windows must take up the whole window
	// and thus don't consider pref size
	// desktop new windows and non-full windows can pref size
	if TheApp.Platform() == system.Offscreen ||
		(!TheApp.Platform().IsMobile() &&
			(st.NewWindow || !st.FullWindow || currentRenderWindow == nil)) {
		sz = sc.prefSize(sz)
		// on offscreen, we don't want any extra space, as we want the smallest
		// possible representation of the content
		// also, on offscreen, if the new size is bigger than the current size,
		// we need to resize the window
		if TheApp.Platform() == system.Offscreen {
			if currentRenderWindow != nil {
				csz := currentRenderWindow.SystemWindow.Size()
				nsz := csz
				if sz.X > csz.X {
					nsz.X = sz.X
				}
				if sz.Y > csz.Y {
					nsz.Y = sz.Y
				}
				if nsz != csz {
					currentRenderWindow.SystemWindow.SetSize(nsz)
					system.TheApp.GetScreens()
				}
			}
		} else {
			// on other platforms, we want extra space and a minimum window size
			sz = sz.Add(image.Pt(20, 20))
			if st.NewWindow {
				// we require windows to be at least 60% and no more than 80% of the
				// screen size by default
				scsz := system.TheApp.Screen(0).PixSize // TODO(kai): is there a better screen to get here?
				sz = image.Pt(max(sz.X, scsz.X*6/10), max(sz.Y, scsz.Y*6/10))
				sz = image.Pt(min(sz.X, scsz.X*8/10), min(sz.Y, scsz.Y*8/10))
			}
		}
	}
	st.Mains = nil // reset
	if DebugSettings.WinRenderTrace {
		fmt.Println("MainStage.RunWindow: Window Size:", sz)
	}

	if st.NewWindow || currentRenderWindow == nil {
		sc.resize(math32.Geom2DInt{st.renderContext.geom.Pos, sz})
		win := st.newRenderWindow()
		mainRenderWindows.add(win)
		currentRenderWindow = win
		win.goStartEventLoop()
		return st
	}
	if st.Context != nil {
		ms := st.Context.AsWidget().Scene.Stage.Mains
		msc := ms.top().Scene
		sc.sceneGeom.Size = sz
		sc.fitInWindow(msc.sceneGeom) // does resize
		ms.push(st)
		st.setMains(ms)
	} else {
		ms := &currentRenderWindow.mains
		msc := ms.top().Scene
		sc.sceneGeom.Size = sz
		sc.fitInWindow(msc.sceneGeom) // does resize
		ms.push(st)
		st.setMains(ms)
	}
	return st
}

// getValidContext ensures that the Context is non-nil and has a valid
// Scene pointer, using CurrentRenderWindow if the current Context is not valid.
// If CurrentRenderWindow is nil (should not happen), then it returns false and
// the calling function must bail.
func (st *Stage) getValidContext() bool {
	if st.Context == nil || st.Context.AsTree().This == nil || st.Context.AsWidget().Scene == nil {
		if currentRenderWindow == nil {
			slog.Error("Stage.Run: Context is nil and CurrentRenderWindow is nil, so cannot Run", "Name", st.Name, "Title", st.Title)
			return false
		}
		st.Context = currentRenderWindow.mains.top().Scene
	}
	return true
}

// runDialog runs a Dialog with current settings.
func (st *Stage) runDialog() *Stage {
	if !st.getValidContext() {
		return st
	}
	ctx := st.Context.AsWidget()

	// if our main stages are nil, we wait until our context is shown and then try again
	if ctx.Scene.Stage == nil || ctx.Scene.Stage.Mains == nil {
		ctx.OnShow(func(e events.Event) {
			st.runDialog()
		})
		return st
	}

	ms := ctx.Scene.Stage.Mains

	sc := st.Scene
	st.configMainStage()
	st.addDialogParts()
	sc.sceneGeom.Pos = st.Pos

	st.setMains(ms) // temporary for prefs

	sz := ms.renderContext.geom.Size
	if !st.FullWindow || st.NewWindow {
		sz = sc.prefSize(sz)
		sz = sz.Add(image.Pt(50, 50))
		// dialogs must be at least 400dp wide by default
		minx := int(ctx.Scene.Styles.UnitContext.Dp(400))
		sz.X = max(sz.X, minx)
		sc.Events.startFocusFirst = true // popup dialogs always need focus
	}
	if DebugSettings.WinRenderTrace {
		slog.Info("MainStage.RunDialog", "size", sz)
	}

	if st.NewWindow {
		st.Mains = nil
		sc.resize(math32.Geom2DInt{st.renderContext.geom.Pos, sz})
		st.Type = WindowStage            // critical: now is its own window!
		sc.sceneGeom.Pos = image.Point{} // ignore pos
		win := st.newRenderWindow()
		dialogRenderWindows.add(win)
		currentRenderWindow = win
		win.goStartEventLoop()
		return st
	}
	sc.sceneGeom.Size = sz
	sc.fitInWindow(st.renderContext.geom) // does resize
	ms.push(st)
	// st.SetMains(ms) // already set
	return st
}

func (st *Stage) newRenderWindow() *renderWindow {
	name := st.Name
	title := st.Title
	opts := &system.NewWindowOptions{
		Title:     title,
		Icon:      appIconImages(),
		Size:      st.Scene.sceneGeom.Size,
		StdPixels: false,
	}
	wgp := theWindowGeometrySaver.pref(title, nil)
	if TheApp.Platform() != system.Offscreen && wgp != nil {
		theWindowGeometrySaver.settingStart()
		opts.Size = wgp.size()
		opts.Pos = wgp.pos()
		opts.StdPixels = false
		if w := AllRenderWindows.FindName(name); w != nil { // offset from existing
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
		if wgp.Fullscreen {
			opts.SetFullscreen()
		}
	}
	win := newRenderWindow(name, title, opts)
	theWindowGeometrySaver.settingEnd()
	if win == nil {
		return nil
	}
	AllRenderWindows.add(win)
	// initialize Mains
	win.mains.renderWindow = win
	win.mains.renderContext = newRenderContext() // sets defaults according to Screen
	// note: win is not yet created by the OS and we don't yet know its actual size
	// or dpi.
	win.mains.push(st)
	st.setMains(&win.mains)
	return win
}

// mainHandleEvent handles main stage events
func (st *Stage) mainHandleEvent(e events.Event) {
	if st.Scene == nil {
		return
	}
	st.popups.popupHandleEvent(e)
	if e.IsHandled() || (st.popups != nil && st.popups.topIsModal()) {
		if DebugSettings.EventTrace && e.Type() != events.MouseMove {
			fmt.Println("Event handled by popup:", e)
		}
		return
	}
	e.SetLocalOff(st.Scene.sceneGeom.Pos)
	st.Scene.Events.handleEvent(e)
}

// mainHandleEvent calls mainHandleEvent on relevant stages in reverse order.
func (sm *stages) mainHandleEvent(e events.Event) {
	n := sm.stack.Len()
	for i := n - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		st.mainHandleEvent(e)
		if e.IsHandled() || st.Modal || st.Type == WindowStage || st.FullWindow {
			break
		}
	}
}
