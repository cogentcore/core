// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

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
	ctx = NonNilContext(ctx)
	st := &Stage{}
	st.SetType(typ)
	st.SetScene(sc)
	st.Context = ctx
	st.Pos = ctx.ContextMenuPos(nil)
	sc.Stage = st
	// note: not setting all the connections until run
	return st
}

// RunPopup runs a popup-style Stage in context widget's popups.
func (st *Stage) RunPopup() *Stage {
	ctx := st.Context.AsWidget()
	// if our context stage is nil, we wait until
	// our context is shown and then try again
	if ctx.Scene.Stage == nil {
		ctx.OnShow(func(e events.Event) {
			st.RunPopup()
		})
		return st
	}

	if st.Type == SnackbarStage {
		st.Scene.ConfigSceneBars()
	}
	st.Scene.ConfigSceneWidgets()
	sc := st.Scene

	ms := ctx.Scene.Stage.Main
	msc := ms.Scene

	// note: completer and potentially other things drive popup creation asynchronously
	// so we need to protect here *before* pushing the new guy on the stack, and during closing.
	ms.RenderCtx.Mu.RLock()
	defer ms.RenderCtx.Mu.RUnlock()

	if st.Type == SnackbarStage {
		// only one snackbar can exist
		ms.PopupMgr.PopDeleteType(SnackbarStage)
	}

	ms.PopupMgr.Push(st)
	st.SetPopupMgr(ms) // sets all pointers

	maxSz := msc.SceneGeom.Size

	// original size and position, which is that of the context widget / location for a tooltip
	osz := sc.SceneGeom.Size
	opos := sc.SceneGeom.Pos

	sc.SceneGeom.Size = maxSz
	sc.SceneGeom.Pos = st.Pos
	sz := sc.PrefSize(maxSz)
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
		b := msc.SceneGeom.Bounds()
		// Go in the middle [(max - min) / 2], and then subtract
		// half of the size because we are specifying starting point,
		// not the center. This results in us being centered.
		sc.SceneGeom.Pos.X = (b.Max.X - b.Min.X - sz.X) / 2
		// get enough space to fit plus 10 extra pixels of margin
		sc.SceneGeom.Pos.Y = b.Max.Y - sz.Y - 10
	case TooltipStage:
		// on x axis, we center on the original (widget) position
		// on y axis, we put our bottom 10 above the top of the original (widget) position
		wc := opos.X + osz.X/2
		sc.SceneGeom.Pos.X = wc - sz.X/2

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
	sc.FitInWindow(msc.SceneGeom) // does resize
	sc.ShowIter = 0

	if st.Timeout > 0 {
		time.AfterFunc(st.Timeout, func() {
			if st.Main == nil {
				return
			}
			st.PopupMgr.DeleteStage(st)
		})
	}

	return st
}

// ClosePopup closes this stage as a popup
func (st *Stage) ClosePopup() {
	// note: this is critical for Completer to not crash due to async closing:
	if st.Main == nil || st.PopupMgr == nil || st.MainMgr == nil {
		// fmt.Println("popup already gone")
		return
	}
	// note: essential to lock here for async popups like completer
	st.MainMgr.RenderCtx.Mu.RLock()
	defer st.MainMgr.RenderCtx.Mu.RUnlock()

	st.PopupMgr.DeleteStage(st)
}

func (st *Stage) PopupHandleEvent(e events.Event) {
	if st.Scene == nil {
		return
	}
	if e.IsHandled() {
		return
	}
	e.SetLocalOff(st.Scene.SceneGeom.Pos)
	// fmt.Println("pos:", evi.Pos(), "local:", evi.LocalPos())
	st.Scene.EventMgr.HandleEvent(e)
}

//////////////////////////////////////////////////////////////////////////////
// 	StageMgr for Popup

// TopIsModal returns true if there is a Top PopupStage and it is Modal.
func (pm *StageMgr) TopIsModal() bool {
	top := pm.Top()
	if top == nil {
		return false
	}
	return top.Modal
}

// PopupHandleEvent processes Popup events.
// requires outer RenderContext mutex.
func (pm *StageMgr) PopupHandleEvent(e events.Event) {
	top := pm.Top()
	if top == nil {
		return
	}
	ts := top.Scene

	// we must get the top stage that does not ignore events
	if top.IgnoreEvents {
		var ntop *Stage
		for i := pm.Stack.Len() - 1; i >= 0; i-- {
			s := pm.Stack.ValByIdx(i)
			if !s.IgnoreEvents {
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
		if pos.In(ts.SceneGeom.Bounds()) {
			top.PopupHandleEvent(e)
			e.SetHandled()
			return
		}
		if top.ClickOff && e.Type() == events.MouseUp {
			pm.PopDelete()
		}
		if top.Modal { // absorb any other events!
			e.SetHandled()
			return
		}
		// otherwise not Handled, so pass on to first lower stage
		// that accepts events and is in bounds
		for i := pm.Stack.Len() - 1; i >= 0; i-- {
			s := pm.Stack.ValByIdx(i)
			ss := s.Scene
			if !s.IgnoreEvents && pos.In(ss.SceneGeom.Bounds()) {
				s.PopupHandleEvent(e)
				e.SetHandled()
				return
			}
		}
	} else { // typically focus, so handle even if not in bounds
		top.PopupHandleEvent(e) // could be set as Handled or not
	}
}
