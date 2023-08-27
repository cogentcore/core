// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gist"
	"goki.dev/gi/v2/histyle"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/units"
)

////////////////////////////////////////////////////////////////////////////////////////
//  HiStyleValueView

// HiStyleValueView presents an action for displaying a mat32.Y and selecting
// from styles
type HiStyleValueView struct {
	ValueViewBase
}

var TypeHiStyleValueView = kit.Types.AddType(&HiStyleValueView{}, nil)

func (vv *HiStyleValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.TypeAction
	return vv.WidgetTyp
}

func (vv *HiStyleValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	ac.SetText(txt)
}

func (vv *HiStyleValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.Px(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeHiStyleValueView).(*HiStyleValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *HiStyleValueView) HasAction() bool {
	return true
}

func (vv *HiStyleValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	SliceViewSelectDialog(vp, &histyle.StyleNames, cur, DlgOpts{Title: "Select a HiStyle Highlighting Style", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					hs := histyle.StyleNames[si]
					vv.SetValue(hs)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}

//////////////////////////////////////////////////////////////////////////////////////
//  HiStylesView

// HiStylesView opens a view of highlighting styles
func HiStylesView(st *histyle.Styles) {
	winm := "hi-styles"
	width := 1280
	height := 800
	win, recyc := gi.RecycleMainWindow(st, winm, "Syntax Hilighting Styles", width, height)
	if recyc {
		return
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.TypeLabel, "title").(*gi.Label)
	title.SetText("Hilighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	title.SetProp("width", units.Ch(30)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(TypeMapView, "tv").(*MapView)
	tv.Viewport = vp
	tv.SetMap(st)
	tv.SetStretchMax()

	histyle.StylesChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		histyle.StylesChanged = true
	})

	mmen := win.MainMenu
	MainMenuView(st, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !histyle.StylesChanged || st != &histyle.CustomStyles { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Styles Before Closing?",
			Prompt: "Do you want to save any changes to std preferences styles file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					st.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", histyle.PrefsStylesFileName)
					win.Close()
				case 1:
					st.OpenPrefs() // revert
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.OSWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
