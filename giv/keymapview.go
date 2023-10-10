// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *gi.KeyMaps) {
	winm := "gogi-key-maps"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainRenderWin(km, winm, "GoGi Key Maps", width, height)
	if recyc {
		return
	}

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert
	mfr.AddStyles(func(s *styles.Style) {
		s.Margin.Set(units.Dp(8 * gi.Prefs.DensityMul()))
	})

	title := gi.NewLabel(mfr, "title", "Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map")
	title.Type = gi.LabelHeadlineSmall
	title.AddStyles(func(s *styles.Style) {
		s.Width.SetCh(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := mfr.NewChild(TableViewType, "tv").(*TableView)
	tv.Scene = vp
	tv.SetSlice(km)
	tv.SetStretchMax()

	gi.AvailKeyMapsChanged = false
	tv.ViewSig.Connect(mfr.This(), func(recv, send ki.Ki, sig int64, data any) {
		gi.AvailKeyMapsChanged = true
	})

	mmen := win.MainMenu
	MainMenuView(km, win, mmen)

	inClosePrompt := false
	win.RenderWin.SetCloseReqFunc(func(w goosi.RenderWin) {
		if !gi.AvailKeyMapsChanged || km != &gi.AvailKeyMaps { // only for main avail map..
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save KeyMaps Before Closing?",
			Prompt: "Do you want to save any changes to std preferences keymaps file before closing, or Cancel the close and do a Save to a different file?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"}, func(dlg *gi.Dialog) {
				switch sig {
				case 0:
					km.SavePrefs()
					fmt.Printf("Preferences Saved to %v\n", gi.PrefsKeyMapsFileName)
					win.Close()
				case 1:
					if km == &gi.AvailKeyMaps {
						km.OpenPrefs() // revert
					}
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasFlag(WinHasGeomPrefs) { // resize to contents
		vpsz := vp.PrefSize(win.RenderWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValueView

// KeyMapValueView presents an action for displaying a KeyMapName and selecting
// from chooser
type KeyMapValueView struct {
	ValueViewBase
}

func (vv *KeyMapValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *KeyMapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	ac.SetFullReRender()
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeKeyMapValueView).(*KeyMapValueView)
		ac := vvv.Widget.(*gi.Button)
		vvv.OpenDialog(ac.Sc, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasAction() bool {
	return true
}

func (vv *KeyMapValueView) OpenDialog(vp *gi.Scene, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := gi.AvailKeyMaps.MapByName(gi.KeyMapName(cur))
	desc, _ := vv.Tag("desc")
	TableViewSelectDialog(vp, &gi.AvailKeyMaps, DlgOpts{Title: "Select a KeyMap", Prompt: desc}, curRow, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg, _ := send.(*gi.DialogStage)
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
