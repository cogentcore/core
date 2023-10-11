// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *gi.KeyMaps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	sc := gi.StageScene("gogi-key-maps")
	sc.Title = "Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map"
	sc.Lay = gi.LayoutVert
	sc.Data = km

	sc.AddStyles(func(s *styles.Style) {
		s.Margin.Set(units.Dp(8))
	})

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.AddStyles(func(s *styles.Style) {
		s.Width.SetCh(30) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := sc.NewChild(TableViewType, "tv").(*TableView)
	tv.SetSlice(km)
	tv.SetStretchMax()

	gi.AvailKeyMapsChanged = false
	tv.OnChange(func(e events.Event) {
		gi.AvailKeyMapsChanged = true
	})

	/* todo: menu, close
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
	*/

	gi.NewWindow(sc).Run()
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
	ac.SetText(txt)
}

func (vv *KeyMapValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValueView) HasDialog() bool {
	return true
}

func (vv *KeyMapValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsInactive() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := gi.AvailKeyMaps.MapByName(gi.KeyMapName(cur))
	desc, _ := vv.Tag("desc")
	TableViewSelectDialog(ctx, DlgOpts{Title: "Select a KeyMap", Prompt: desc}, &gi.AvailKeyMaps, curRow, nil, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			si := TableViewSelectDialogValue(dlg)
			if si >= 0 {
				km := gi.AvailKeyMaps[si]
				vv.SetValue(km.Name)
				vv.UpdateWidget()
			}
		}
		if fun != nil {
			fun(dlg)
		}
	})
}
