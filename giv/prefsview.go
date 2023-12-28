// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
)

// SettingsViewWindow makes and runs a new window for viewing user settings.
func SettingsViewWindow() {
	if gi.ActivateExistingMainWindow(gi.AllSettings) {
		return
	}
	d := gi.NewBody("settings").SetTitle("Settings").SetData(gi.AllSettings)
	SettingsView(d)
	d.NewWindow().Run()
}

// SettingsView adds to the given body a view of user settings
func SettingsView(b *gi.Body) {
	/*
		b.AddAppBar(func(tb *gi.Toolbar) {
			NewFuncButton(tb, pf.ScreenInfo).SetShowReturn(true).SetIcon(icons.Info)
			NewFuncButton(tb, pf.VersionInfo).SetShowReturn(true).SetIcon(icons.Info)
			gi.NewSeparator(tb)
			NewFuncButton(tb, pf.EditKeyMaps).SetIcon(icons.Keyboard)
			NewFuncButton(tb, pf.EditHiStyles).SetIcon(icons.InkHighlighter)
			NewFuncButton(tb, pf.EditDetailed).SetIcon(icons.Description)
			NewFuncButton(tb, pf.EditDebug).SetIcon(icons.BugReport)
			tb.AddOverflowMenu(func(m *gi.Scene) {
				NewFuncButton(m, pf.Open).SetKey(keyfun.Open)
				NewFuncButton(m, pf.Delete).SetConfirm(true)
				NewFuncButton(m, pf.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
				gi.NewSeparator(tb)
			})
		})
	*/

	tabs := gi.NewTabs(b)

	for _, kv := range gi.AllSettings.Order {
		nm := kv.Key
		se := kv.Val

		fr := tabs.NewTab(nm)

		NewStructView(fr).SetStruct(se).OnChange(func(e events.Event) {
			if tab := b.GetTopAppBar(); tab != nil {
				tab.UpdateBar()
			}
			se.Apply()
			gi.ErrorSnackbar(fr, gi.SaveSettings(se), "Error saving "+nm+" settings")
			gi.UpdateAll()
		})
	}
}

/*
// PrefsDetView opens a view of user detailed preferences
func PrefsDetView(pf *gi.PrefsDetailed) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}

	d := gi.NewBody("gogi-prefs-det").SetTitle("GoGi Detailed Preferences")

	sv := NewStructView(d, "sv")
	sv.SetStruct(pf)

	d.Sc.Data = pf

	d.AddAppBar(func(tb *gi.Toolbar) {
		NewFuncButton(tb, pf.Apply).SetIcon(icons.Refresh)
		gi.NewSeparator(tb)
		NewFuncButton(tb, pf.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(pf.Changed) })
		tb.AddOverflowMenu(func(m *gi.Scene) {
			NewFuncButton(m, pf.Open).SetKey(keyfun.Open)
			gi.NewSeparator(tb)
		})
	})

	d.NewWindow().Run()
}

// PrefsDbgView opens a view of user debugging preferences
func PrefsDbgView(pf *gi.DebugSettingsData) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}
	d := gi.NewBody("gogi-prefs-dbg")
	d.Title = "GoGi Debugging Preferences"

	sv := NewStructView(d, "sv")
	sv.SetStruct(pf)

	d.Sc.Data = pf

	d.AddAppBar(func(tb *gi.Toolbar) {
		NewFuncButton(tb, pf.Profile).SetIcon(icons.LabProfile)
	})

	d.NewWindow().Run()
}
*/
