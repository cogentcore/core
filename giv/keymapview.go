// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *gi.KeyMaps) {
	winm := "gogi-key-maps"
	width := 800
	height := 800
	win := gi.NewWindow2D(winm, "GoGi Key Maps", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	title := mfr.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.SetText("Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map")
	title.SetProp("width", units.NewValue(30, units.Ch)) // need for wrap
	title.SetStretchMaxWidth()
	title.SetProp("white-space", gi.WhiteSpaceNormal) // wrap

	tv := mfr.AddNewChild(KiT_TableView, "tv").(*TableView)
	tv.Viewport = vp
	tv.SetSlice(km, nil)
	tv.SetStretchMaxWidth()
	tv.SetStretchMaxHeight()

	gi.AvailKeyMapsChanged = false
	tv.ViewSig.Connect(mfr.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		gi.AvailKeyMapsChanged = true
	})

	mmen := win.MainMenu
	MainMenuView(km, win, mmen)

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if gi.AvailKeyMapsChanged { // only for main avail map..
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save KeyMaps Before Closing?",
				Prompt: "Do you want to save any changes to std preferences to std keymaps file before closing, or Cancel the close and do a Save to a different file?"},
				[]string{"Save and Close", "Discard and Close", "Cancel"},
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						km.SavePrefs()
						fmt.Printf("Preferences Saved to %v\n", gi.PrefsKeyMapsFileName)
						w.Close()
					case 1:
						if km == &gi.AvailKeyMaps {
							km.OpenPrefs() // revert
						}
						w.Close()
					case 2:
						// default is to do nothing, i.e., cancel
					}
				})
		} else {
			w.Close()
		}
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// KeyMapValueView presents an action for displaying an KeyMapName and selecting
// icons from KeyMapChooserDialog
type KeyMapValueView struct {
	ValueViewBase
}

var KiT_KeyMapValueView = kit.Types.AddType(&KeyMapValueView{}, nil)

func (vv *KeyMapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	ac.SetFullReRender()
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_KeyMapValueView).(*KeyMapValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasAction() bool {
	return true
}

func (vv *KeyMapValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	cur := kit.ToString(vv.Value.Interface())
	_, curRow, _ := gi.AvailKeyMaps.MapByName(gi.KeyMapName(cur))
	desc, _ := vv.Tag("desc")
	TableViewSelectDialog(vp, &gi.AvailKeyMaps, DlgOpts{Title: "Select a KeyMap", Prompt: desc}, curRow, nil,
		vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.Dialog)
				si := TableViewSelectDialogValue(ddlg)
				if si >= 0 {
					km := gi.AvailKeyMaps[si]
					vv.SetValue(km.Name)
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
