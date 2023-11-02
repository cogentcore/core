// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  HiStyleValue

// HiStyleValue presents an action for displaying a mat32.Y and selecting
// from styles
type HiStyleValue struct {
	ValueBase
}

func (vv *HiStyleValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *HiStyleValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	bt.SetText(txt)
}

func (vv *HiStyleValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *HiStyleValue) HasDialog() bool {
	return true
}

func (vv *HiStyleValue) OpenDialog(ctx gi.Widget, fun func(d *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	d := gi.NewDialog(ctx).Title("Select a HiStyle Highlighting Style").Prompt(vv.Doc()).FullWindow(true)
	NewSliceView(d).SetSlice(&histyle.StyleNames).SetSelVal(cur).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			hs := histyle.StyleNames[si]
			vv.SetValue(hs)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  HiStylesView

// HiStylesView opens a view of highlighting styles
func HiStylesView(st *histyle.Styles) {
	if gi.ActivateExistingMainWindow(st) {
		return
	}

	sc := gi.NewScene("hi-styles")
	sc.Title = "Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards."
	sc.Lay = gi.LayoutVert
	sc.Data = st

	title := gi.NewLabel(sc, "title").SetText(sc.Title)
	title.Style(func(s *styles.Style) {
		s.SetMinPrefWidth(units.Ch(30)) // need for wrap
		s.SetStretchMaxWidth()
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	mv := NewMapView(sc).SetMap(st)
	mv.SetStretchMax()

	histyle.StylesChanged = false
	mv.OnChange(func(e events.Event) {
		histyle.StylesChanged = true
	})

	sc.TopAppBar = func(tb *gi.Toolbar) {
		gi.DefaultTopAppBar(tb)
		oj := NewFuncButton(tb, st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
		oj.Args[0].SetTag(".ext", ".histy")
		sj := NewFuncButton(tb, st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
		sj.Args[0].SetTag(".ext", ".histy")
		gi.NewSeparator(tb)
		mv.MapDefaultToolbar(tb)
	}

	// mmen := win.MainMenu
	// MainMenuView(st, win, mmen)

	// todo: close prompt
	/*
		inClosePrompt := false
		win.RenderWin.SetCloseReqFunc(func(w goosi.RenderWin) {
			if !histyle.StylesChanged || st != &histyle.CustomStyles { // only for main avail map..
				win.Close()
				return
			}
			if inClosePrompt {
				return
			}
			inClosePrompt = true
			gi.ChoiceDialog(sc, gi.DlgOpts{Title: "Save Styles Before Closing?",
				Prompt: "Do you want to save any changes to std preferences styles file before closing, or Cancel the close and do a Save to a different file?"},
				[]string{"Save and Close", "Discard and Close", "Cancel"}, func(dlg *gi.Dialog) {
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
	*/

	gi.NewWindow(sc).Run() // todo: should be a dialog instead?
}
