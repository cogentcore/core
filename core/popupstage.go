// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"time"

	"cogentcore.org/core/events"
)

// NewPopupStage returns a new PopupStage with given type and scene contents.
// The given context widget must be non-nil.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewPopupStage call.
// Use Run call at the end to start the Stage running.
func NewPopupStage(typ StageTypes, sc *Scene, ctx Widget) *Stage {
	ctx = nonNilContext(ctx)
	st := &Stage{}
	st.setType(typ)
	st.setScene(sc)
	st.Context = ctx
	st.Pos = ctx.ContextMenuPos(nil)
	sc.Stage = st
	// note: not setting all the connections until run
	return st
}

// runPopupAsync runs a popup-style Stage in context widget's popups.
// This version is for Asynchronous usage outside the main event loop,
// for example in a delayed callback AfterFunc etc.
func (st *Stage) runPopupAsync() *Stage {
	ctx := st.Context.AsWidget()
	if ctx.Scene.Stage == nil {
		return st.runPopup()
	}
	ms := ctx.Scene.Stage.Main
	rc := ms.renderContext
	rc.Lock()
	defer rc.Unlock()
	return st.runPopup()
}

// runPopup runs a popup-style Stage in context widget's popups.
func (st *Stage) runPopup() *Stage {
	if !st.getValidContext() { // doesn't even have a scene
		return st
	}
	ctx := st.Context.AsWidget()
	// if our context stage is nil, we wait until
	// our context is shown and then try again
	if ctx.Scene.Stage == nil {
		ctx.Defer(func() {
			st.runPopup()
		})
		return st
	}

	if st.Type == SnackbarStage {
		st.Scene.makeSceneBars()
	}
	st.Scene.updateScene()
	sc := st.Scene

	ms := ctx.Scene.Stage.Main
	msc := ms.Scene

	if st.Type == SnackbarStage {
		// only one snackbar can exist
		ms.popups.popDeleteType(SnackbarStage)
	}

	ms.popups.push(st)
	st.setPopups(ms) // sets all pointers

	maxGeom := msc.SceneGeom
	winst := ms.Mains.windowStage()
	usingWinGeom := false
	if winst != nil && winst.Scene != nil && winst.Scene != msc {
		usingWinGeom = true
		maxGeom = winst.Scene.SceneGeom // use the full window if possible
	}
	// original size and position, which is that of the context widget / location for a tooltip
	osz := sc.SceneGeom.Size
	opos := sc.SceneGeom.Pos

	sc.SceneGeom.Size = maxGeom.Size
	sc.SceneGeom.Pos = st.Pos
	sz := sc.contentSize(maxGeom.Size)
	bigPopup := false
	if usingWinGeom && 4*sz.X*sz.Y > 3*msc.SceneGeom.Size.X*msc.SceneGeom.Size.Y { // reasonable fraction
		bigPopup = true
	}
	scrollWd := int(sc.Styles.ScrollbarWidth.Dots)
	fontHt := sc.Styles.Font.FontHeight()
	if fontHt == 0 {
		fontHt = 16
	}

	switch st.Type {
	case MenuStage:
		sz.X += scrollWd * 2
		maxht := int(float32(AppearanceSettings.MenuMax) * fontHt)
		sz.Y = min(maxht, sz.Y)
	case SnackbarStage:
		b := msc.SceneGeom.Bounds()
		// Go in the middle [(max - min) / 2], and then subtract
		// half of the size because we are specifying starting point,
		// not the center. This results in us being centered.
		sc.SceneGeom.Pos.X = (b.Max.X - b.Min.X - sz.X) / 2
		// get enough space to fit plus 10 extra pixels of margin
		sc.SceneGeom.Pos.Y = b.Max.Y - sz.Y - 10
	case TooltipStage:
		sc.SceneGeom.Pos.X = opos.X
		// default to tooltip above element
		ypos := opos.Y - sz.Y - 10
		if ypos < 0 {
			ypos = 0
		}
		// however, if we are within 10 pixels of the element,
		// we put the tooltip below it instead of above it
		maxy := ypos + sz.Y
		if maxy > opos.Y-10 {
			ypos = opos.Add(osz).Y + 10
		}
		sc.SceneGeom.Pos.Y = ypos
	}

	sc.SceneGeom.Size = sz
	if bigPopup { // we have a big popup -- make it not cover the original window;
		sc.fitInWindow(maxGeom) // does resize
		// reposition to be as close to top-right of main scene as possible
		tpos := msc.SceneGeom.Pos
		tpos.X += msc.SceneGeom.Size.X
		if tpos.X+sc.SceneGeom.Size.X > maxGeom.Size.X { // favor left side instead
			tpos.X = max(msc.SceneGeom.Pos.X-sc.SceneGeom.Size.X, 0)
		}
		bpos := tpos.Add(sc.SceneGeom.Size)
		if bpos.X > maxGeom.Size.X {
			tpos.X -= bpos.X - maxGeom.Size.X
		}
		if bpos.Y > maxGeom.Size.Y {
			tpos.Y -= bpos.Y - maxGeom.Size.Y
		}
		if tpos.X < 0 {
			tpos.X = 0
		}
		if tpos.Y < 0 {
			tpos.Y = 0
		}
		sc.SceneGeom.Pos = tpos
	} else {
		sc.fitInWindow(msc.SceneGeom)
	}
	sc.showIter = 0

	if st.Timeout > 0 {
		// The timer runs outside of the event loop, so the popup is deleted
		// with the lock of the render context that it was run in.
		rc := st.renderContext
		time.AfterFunc(st.Timeout, func() {
			rc.Lock()
			defer rc.Unlock()
			if st.Main == nil {
				return
			}
			st.popups.deleteStage(st)
		})
	}

	return st
}

// closePopupAsync closes this stage as a popup.
// This version is for Asynchronous usage outside the main event loop,
// for example in a delayed callback AfterFunc etc.
func (st *Stage) closePopupAsync() {
	rc := st.Mains.renderContext
	rc.Lock()
	defer rc.Unlock()
	st.ClosePopup()
}

// ClosePopup closes this stage as a popup, returning whether it was closed.
func (st *Stage) ClosePopup() bool {
	// NOTE: this is critical for Completer to not crash due to async closing
	if st.Main == nil || st.popups == nil || st.Mains == nil {
		return false
	}
	return st.popups.deleteStage(st)
}

// closePopupAndBelow closes this stage as a popup,
// and all those immediately below it of the same type.
// It returns whether it successfully closed popups.
func (st *Stage) closePopupAndBelow() bool {
	// NOTE: this is critical for Completer to not crash due to async closing
	if st.Main == nil || st.popups == nil || st.Mains == nil {
		return false
	}
	return st.popups.deleteStageAndBelow(st)
}

func (st *Stage) popupHandleEvent(e events.Event) {
	if st.Scene == nil {
		return
	}
	if e.IsHandled() {
		return
	}
	e.SetLocalOff(st.Scene.SceneGeom.Pos)
	// fmt.Println("pos:", evi.Pos(), "local:", evi.LocalPos())
	st.Scene.Events.handleEvent(e)
}

// topIsModal returns true if there is a Top PopupStage and it is Modal.
func (pm *stages) topIsModal() bool {
	top := pm.top()
	if top == nil {
		return false
	}
	return top.Modal
}

// popupHandleEvent processes Popup events.
// requires outer RenderContext mutex.
func (pm *stages) popupHandleEvent(e events.Event) {
	top := pm.top()
	if top == nil {
		return
	}
	ts := top.Scene

	// we must get the top stage that does not ignore events
	if top.ignoreEvents {
		var ntop *Stage
		for i := pm.stack.Len() - 1; i >= 0; i-- {
			s := pm.stack.ValueByIndex(i)
			if !s.ignoreEvents {
				ntop = s
				break
			}
		}
		if ntop == nil {
			return
		}
		top = ntop
		ts = top.Scene
	}

	if e.HasPos() {
		pos := e.WindowPos()
		if top.pressOpened && pm.pressOpenedEvent(top, e) {
			e.SetHandled()
			return
		}
		// fmt.Println("pos:", pos, "top geom:", ts.SceneGeom)
		if pos.In(ts.SceneGeom.Bounds()) {
			top.popupHandleEvent(e)
			e.SetHandled()
			return
		}
		if pm.parentMenuPress(top, e) {
			e.SetHandled()
			return
		}
		if top.ClickOff && e.Type() == events.MouseUp {
			top.closePopupAndBelow()
		}
		if top.Modal { // absorb any other events!
			e.SetHandled()
			return
		}
		// otherwise not Handled, so pass on to first lower stage
		// that accepts events and is in bounds
		for i := pm.stack.Len() - 1; i >= 0; i-- {
			s := pm.stack.ValueByIndex(i)
			ss := s.Scene
			if !s.ignoreEvents && pos.In(ss.SceneGeom.Bounds()) {
				s.popupHandleEvent(e)
				e.SetHandled()
				return
			}
		}
	} else { // typically focus, so handle even if not in bounds
		top.popupHandleEvent(e) // could be set as Handled or not
	}
}

// pressOpenedEvent handles the given event for the given top menu stage,
// which was opened by pressing the mouse button on its context button,
// which is still down (see [Stage.pressOpened]), returning whether it
// handled it. Dragging over the menu hovers its items, and releasing the
// mouse button over one of them clicks it. Releasing it over the context
// button is a click on it, which leaves the menu open, and releasing it
// anywhere else closes the menu.
func (pm *stages) pressOpenedEvent(top *Stage, e events.Event) bool {
	pos := e.WindowPos()
	inMenu := pos.In(top.Scene.SceneGeom.Bounds())
	switch e.Type() {
	case events.MouseDrag:
		if inMenu {
			top.popupHandleEvent(mouseClone(e, events.MouseMove))
		}
		return true
	case events.MouseUp:
		top.pressOpened = false
		// the context button gets the release, ending its press
		// (a click on it if it is over it, which leaves the menu open)
		var ctx *WidgetBase
		if top.Context != nil {
			ctx = top.Context.AsWidget()
		}
		if ctx != nil && ctx.Scene != nil && ctx.Scene.Stage != nil {
			ctx.Scene.Stage.popupHandleEvent(mouseClone(e, events.MouseUp))
		}
		// a release where it was pressed is a click on the context button,
		// even over the menu, which can overlap it
		d := pos.Sub(e.WindowStartPos())
		dragged := max(abs(d.X), abs(d.Y)) > TimingSettings.DragStartDistance
		switch {
		case !dragged:
		case inMenu: // a click on the item
			top.popupHandleEvent(mouseClone(e, events.MouseDown))
			top.popupHandleEvent(e)
		case ctx == nil || !ctx.posInScBBox(pos.Sub(ctx.Scene.SceneGeom.Pos)):
			top.closePopupAndBelow()
		}
		return true
	}
	return false
}

// parentMenuPress handles a mouse press on a menu below the given top menu
// stage, in the menus of the same chain (such as the menu of a submenu),
// returning whether it did: the menus above that menu are closed, and it
// gets the press, so that its items can be clicked, such as to open
// another submenu.
func (pm *stages) parentMenuPress(top *Stage, e events.Event) bool {
	if top.Type != MenuStage || e.Type() != events.MouseDown {
		return false
	}
	pos := e.WindowPos()
	n := pm.stack.Len()
	for i := n - 2; i >= 0; i-- {
		s := pm.stack.ValueByIndex(i)
		if s.Type != MenuStage {
			return false
		}
		if s.ignoreEvents || s.Scene == nil || !pos.In(s.Scene.SceneGeom.Bounds()) {
			continue
		}
		var above []*Stage
		for j := n - 1; j > i; j-- {
			above = append(above, pm.stack.ValueByIndex(j))
		}
		for _, a := range above {
			pm.deleteStage(a)
		}
		s.popupHandleEvent(e)
		return true
	}
	return false
}

// abs returns the absolute value of the given int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// mouseClone returns a copy of the given mouse event with the given type,
// not handled, with the mouse button ([events.Event.NewFromClone] only
// copies the base event).
func mouseClone(e events.Event, typ events.Types) events.Event {
	me, ok := e.(*events.Mouse)
	if !ok {
		return e.NewFromClone(typ)
	}
	c := *me
	c.Typ = typ
	c.ClearHandled()
	return &c
}
