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

// PopupStage supports Popup types (Menu, Tooltip, Snackbar, Completer),
// which are transitory and simple, without additional decor,
// and are associated with and managed by a MainStage element (Window, etc).
type PopupStage struct {
	StageBase

	// Main is the MainStage that owns this Popup (via its PopupMgr)
	Main *MainStage

	// if > 0, disappears after a timeout duration
	Timeout time.Duration
}

// AsPopup returns this stage as a PopupStage (for Popup types)
// returns nil for MainStage types.
func (st *PopupStage) AsPopup() *PopupStage {
	return st
}

func (st *PopupStage) String() string {
	return "PopupStage: " + st.StageBase.String()
}

func (st *PopupStage) MainMgr() *MainStageMgr {
	if st.Main == nil {
		return nil
	}
	return st.Main.StageMgr
}

func (st *PopupStage) SetTimeout(dur time.Duration) *PopupStage {
	st.Timeout = dur
	return st
}

func (st *PopupStage) RenderCtx() *RenderContext {
	if st.Main == nil {
		return nil
	}
	return st.Main.RenderCtx()
}

func (st *PopupStage) Delete() {
	if st.Scene != nil {
		st.Scene.Delete(ki.DestroyKids)
	}
	st.Scene = nil
	st.Main = nil
}

func (st *PopupStage) StageAdded(smi StageMgr) {
	pm := smi.AsPopupMgr()
	st.Main = pm.Main
}

func (st *PopupStage) HandleEvent(evi events.Event) {
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
func NewPopupStage(typ StageTypes, sc *Scene, ctx Widget) *PopupStage {
	if ctx == nil {
		slog.Error("NewPopupStage needs a context Widget")
		return nil
	}
	cwb := ctx.AsWidget()
	if cwb.Sc == nil || cwb.Sc.Stage == nil {
		slog.Error("NewPopupStage context doesn't have a Stage")
		return nil
	}
	st := &PopupStage{}
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
func NewCompleter(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(CompleterStage, sc, ctx)
}

// RunPopup runs a popup-style Stage in context widget's popups.
func (st *PopupStage) RunPopup() *PopupStage {
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
	sc.ShowLayoutIter = 0

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
func (st *PopupStage) Close() {
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
