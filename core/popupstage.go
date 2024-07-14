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
	rc.lock()
	defer rc.unlock()
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
		ctx.OnShow(func(e events.Event) {
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

	maxSz := msc.sceneGeom.Size

	// original size and position, which is that of the context widget / location for a tooltip
	osz := sc.sceneGeom.Size
	opos := sc.sceneGeom.Pos

	sc.sceneGeom.Size = maxSz
	sc.sceneGeom.Pos = st.Pos
	sz := sc.prefSize(maxSz)
	scrollWd := int(sc.Styles.ScrollBarWidth.Dots)
	fontHt := 16
	if sc.Styles.Font.Face != nil {
		fontHt = int(sc.Styles.Font.Face.Metrics.Height)
	}

	switch st.Type {
	case MenuStage:
		sz.X += scrollWd * 2
		maxht := int(SystemSettings.MenuMaxHeight * fontHt)
		sz.Y = min(maxht, sz.Y)
	case SnackbarStage:
		b := msc.sceneGeom.Bounds()
		// Go in the middle [(max - min) / 2], and then subtract
		// half of the size because we are specifying starting point,
		// not the center. This results in us being centered.
		sc.sceneGeom.Pos.X = (b.Max.X - b.Min.X - sz.X) / 2
		// get enough space to fit plus 10 extra pixels of margin
		sc.sceneGeom.Pos.Y = b.Max.Y - sz.Y - 10
	case TooltipStage:
		sc.sceneGeom.Pos.X = opos.X
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
		sc.sceneGeom.Pos.Y = ypos
	}

	sc.sceneGeom.Size = sz
	sc.fitInWindow(msc.sceneGeom) // does resize
	sc.showIter = 0

	if st.Timeout > 0 {
		time.AfterFunc(st.Timeout, func() {
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
	rc.lock()
	defer rc.unlock()
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
	e.SetLocalOff(st.Scene.sceneGeom.Pos)
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
		// fmt.Println("pos:", pos, "top geom:", ts.SceneGeom)
		if pos.In(ts.sceneGeom.Bounds()) {
			top.popupHandleEvent(e)
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
			if !s.ignoreEvents && pos.In(ss.sceneGeom.Bounds()) {
				s.popupHandleEvent(e)
				e.SetHandled()
				return
			}
		}
	} else { // typically focus, so handle even if not in bounds
		top.popupHandleEvent(e) // could be set as Handled or not
	}
}
