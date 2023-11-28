// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"
	"time"

	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

// func (st *Stage) MainMgr() *StageMgr {
// 	if st.Main == nil {
// 		return nil
// 	}
// 	return st.Main.StageMgr
// }

func (st *Stage) PopupRenderCtx() *RenderContext {
	if st.Main == nil {
		return nil
	}
	return st.Main.RenderCtx()
}

func (st *Stage) DeletePopup() {
	if st.Scene != nil {
		st.Scene.Delete(ki.DestroyKids)
	}
	st.Scene = nil
	st.Main = nil
}

func (st *Stage) PopupStageAdded(smi StageMgr) {
	pm := smi.AsPopupMgr()
	st.Main = pm.Main
}

func (st *Stage) PopupHandleEvent(evi events.Event) {
	if st.Scene == nil {
		return
	}
	if evi.IsHandled() {
		return
	}
	evi.SetLocalOff(st.Scene.SceneGeom.Pos)
	// fmt.Println("pos:", evi.Pos(), "local:", evi.LocalPos())
	st.Scene.EventMgr.HandleEvent(evi)
}

// NewPopupStage returns a new PopupStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewPopupStage call.
// Use Run call at the end to start the Stage running.
func NewPopupStage(typ StageTypes, sc *Scene, ctx Widget) *Stage {
	if ctx == nil {
		slog.Error("NewPopupStage needs a context Widget")
		return nil
	}
	cwb := ctx.AsWidget()
	if cwb.Sc == nil || cwb.Sc.Stage == nil {
		slog.Error("NewPopupStage context doesn't have a Stage")
		return nil
	}
	st := &Stage{}
	st.This = st
	st.SetType(typ)
	st.SetScene(sc)
	st.Context = ctx
	cst := cwb.Sc.Stage
	mst := cst.AsMain()
	if mst != nil {
		st.Main = mst
	} else {
		pst := cst.AsPopup()
		st.Main = pst.Main
	}

	switch typ {
	case MenuStage:
		MenuSceneConfigStyles(sc)
	}

	return st
}

// NewCompleter returns a new [CompleterStage] with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewCompleter(sc *Scene, ctx Widget) *Stage {
	return NewPopupStage(CompleterStage, sc, ctx)
}

// RunPopup runs a popup-style Stage in context widget's popups.
func (st *Stage) RunPopup() *Stage {
	st.Scene.ConfigSceneWidgets()
	mm := st.MainMgr()
	if mm == nil {
		slog.Error("popupstage has no MainMgr")
		return st
	}
	mm.RenderCtx.Mu.RLock()
	defer mm.RenderCtx.Mu.RUnlock()

	ms := mm.Top().AsMain() // main stage -- put all popups here
	msc := ms.Scene

	cmgr := &ms.PopupMgr
	cmgr.Push(st)
	sc := st.Scene
	maxSz := msc.SceneGeom.Size

	sc.SceneGeom.Size = maxSz
	sz := sc.PrefSize(maxSz)
	// fmt.Println(sz, maxSz)
	scrollWd := int(sc.Styles.ScrollBarWidth.Dots)
	fontHt := 16
	if sc.Styles.Font.Face != nil {
		fontHt = int(sc.Styles.Font.Face.Metrics.Height)
	}

	switch st.Type {
	case MenuStage:
		sz.X += scrollWd * 2
		maxht := int(MenuMaxHeight * fontHt)
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
		// on x axis, we center on the widget widget
		// on y axis, we put our bottom 10 above the top of the widget
		wb := st.Context.AsWidget()
		bb := wb.WinBBox()
		wc := bb.Min.X + bb.Size().X/2
		sc.SceneGeom.Pos.X = wc - sz.X/2

		// default to tooltip above element
		ypos := bb.Min.Y - sz.Y - 10
		if ypos < 0 {
			ypos = 0
		}
		// however, if we are within 10 pixels of the element,
		// we put the tooltip below it instead of above it
		maxy := ypos + sz.Y
		if maxy > bb.Min.Y-10 {
			ypos = bb.Max.Y + 10
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
			st.Main.PopupMgr.PopDeleteType(st.Type)
		})
	}

	return st
}

// Close closes this stage as a popup
func (st *Stage) Close() {
	mn := st.Main
	if mn == nil {
		slog.Error("popupstage has no Main")
		return
	}
	mm := st.MainMgr()
	if mm == nil {
		slog.Error("popupstage has no MainMgr")
		return
	}
	mm.RenderCtx.Mu.RLock()
	defer mm.RenderCtx.Mu.RUnlock()

	cmgr := &mn.PopupMgr
	cmgr.PopDelete()
}

//////////////////////////////////////////////////////////////////////////////
//		Main StageMgr

// TopIsModal returns true if there is a Top PopupStage and it is Modal.
func (pm *StageMgr) TopIsModal() bool {
	top := pm.Top()
	if top == nil {
		return false
	}
	return top.AsBase().Modal
}

// PopupHandleEvent processes Popup events.
// requires outer RenderContext mutex.
func (pm *StageMgr) PopupHandleEvent(evi events.Event) {
	top := pm.Top()
	if top == nil {
		return
	}
	tb := top.AsBase()
	ts := tb.Scene

	// we must get the top stage that does not ignore events
	if tb.IgnoreEvents {
		var ntop Stage
		for i := pm.Stack.Len() - 1; i >= 0; i-- {
			s := pm.Stack.ValByIdx(i)
			if !s.AsBase().IgnoreEvents {
				ntop = s
				break
			}
		}
		if ntop == nil {
			return
		}
		top = ntop
		tb = top.AsBase()
		ts = tb.Scene
	}

	if evi.HasPos() {
		pos := evi.Pos()
		// fmt.Println("pos:", pos, "top geom:", ts.SceneGeom)
		if pos.In(ts.SceneGeom.Bounds()) {
			top.HandleEvent(evi)
			evi.SetHandled()
			return
		}
		if tb.ClickOff && evi.Type() == events.MouseUp {
			pm.PopDelete()
		}
		if tb.Modal { // absorb any other events!
			evi.SetHandled()
			return
		}
		// otherwise not Handled, so pass on to first lower stage
		// that accepts events and is in bounds
		for i := pm.Stack.Len() - 1; i >= 0; i-- {
			s := pm.Stack.ValByIdx(i)
			sb := s.AsBase()
			ss := sb.Scene
			if !sb.IgnoreEvents && pos.In(ss.SceneGeom.Bounds()) {
				s.HandleEvent(evi)
				evi.SetHandled()
				return
			}
		}
	} else { // typically focus, so handle even if not in bounds
		top.HandleEvent(evi) // could be set as Handled or not
	}
}
