// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/goosi"

// PopupStage supports Popup types (Menu, Tooltip, Snakbar, Chooser),
// which are transitory and simple, without additional decor,
// and are associated with and managed by a MainStage element (Window, etc).
type PopupStage struct {
	StageBase

	// Main is the MainStage that owns this Popup (via its PopupMgr)
	Main *MainStage
}

// AsPopup returns this stage as a PopupStage (for Popup types)
// returns nil for MainStage types.
func (st *PopupStage) AsPopup() *PopupStage {
	return st
}

func (st *PopupStage) MainMgr() *MainStageMgr {
	if st.Main == nil {
		return nil
	}
	return st.Main.StageMgr
}

func (st *PopupStage) RenderCtx() *RenderContext {
	if st.Main == nil {
		return nil
	}
	return st.Main.RenderCtx()
}

func (st *PopupStage) Delete() {
	if st.Scene != nil {
		st.Scene.Delete()
	}
	st.Scene = nil
	st.Main = nil
}

func (st *PopupStage) StageAdded(smi StageMgr) {
	pm := smi.AsPopupMgr()
	st.Main = pm.Main
	// todo: ?
	// if pfoc != nil {
	// 	sm.EventMgr.PushFocus(pfoc)
	// } else {
	// 	sm.EventMgr.PushFocus(st)
	// }
}

func (st *PopupStage) HandleEvent(evi goosi.Event) {
	evi.SetLocalOff(st.Scene.Geom.Pos)
	// todo: need a simpler event filter delivery guy:
	// st.EventMgr.HandleEvent(st.Scene, evi)
}

// NewPopupStage returns a new PopupStage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewPopupStage call.
// Use Run call at the end to start the Stage running.
func NewPopupStage(typ StageTypes, sc *Scene, ctx Widget) *PopupStage {
	st := &PopupStage{}
	st.Stage = st
	st.SetType(typ)
	st.SetScene(sc)
	st.CtxWidget = ctx
	if ctx == nil {
		// todo: error -- need context for widget
		return st
	}
	cwb := ctx.AsWidget()
	if cwb.Sc == nil || cwb.Sc.Stage == nil {
		// todo: error
		return st
	}
	cst := cwb.Sc.Stage
	mst := cst.AsMain()
	if mst != nil {
		st.Main = mst
	} else {
		pst := cst.AsPopup()
		st.Main = pst.Main
	}
	return st
}

// NewMenu returns a new Menu stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewMenu(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Menu, sc, ctx)
}

// NewTooltip returns a new Tooltip stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewTooltip(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Tooltip, sc, ctx)
}

// NewSnackbar returns a new Snackbar stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSnackbar(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Snackbar, sc, ctx)
}

// NewChooser returns a new Chooser stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewChooser(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Chooser, sc, ctx)
}

// RunPopup runs a popup-style Stage on top of current
// active stage in current active RenderWin.
func (st *PopupStage) RunPopup() *PopupStage {
	mm := st.MainMgr()
	if mm == nil {
		// todo: error
		return st
	}
	mm.RenderCtx.Mu.RLock()
	defer mm.RenderCtx.Mu.RUnlock()
	mm.Push(st)
	// todo: what else!?
	return st
}
