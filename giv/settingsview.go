// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/events"
	"goki.dev/gi"
	"goki.dev/icons"
)

func init() {
	gi.AppearanceSettings.TBConfig = SettingsConfigToolbar
}

func SettingsConfigToolbar(tb *gi.Toolbar) {
	as := gi.AppearanceSettings
	NewFuncButton(tb, as.SaveZoom).SetIcon(icons.ZoomIn)
	// NewFuncButton(tb, pf.EditKeyMaps).SetIcon(icons.Keyboard)
	// NewFuncButton(tb, pf.EditHiStyles).SetIcon(icons.InkHighlighter)
	tb.AddOverflowMenu(func(m *gi.Scene) {
		NewFuncButton(m, as.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
		gi.NewSeparator(tb)
	})
}

// SettingsWindow makes and runs a new window for viewing user settings.
func SettingsWindow() {
	if gi.ActivateExistingMainWindow(gi.AllSettings) {
		return
	}
	d := gi.NewBody("settings").SetTitle("Settings").SetData(gi.AllSettings)
	SettingsView(d)
	d.NewWindow().Run()
}

// SettingsView adds to the given body a view of user settings
func SettingsView(b *gi.Body) {
	b.AddAppBar(func(tb *gi.Toolbar) {
		for _, se := range gi.AllSettings {
			se := se
			se.ConfigToolbar(tb)
		}
	})

	tabs := gi.NewTabs(b)

	for _, se := range gi.AllSettings {
		se := se
		fr := tabs.NewTab(se.Label())

		NewStructView(fr).SetStruct(se).OnChange(func(e events.Event) {
			if tab := b.GetTopAppBar(); tab != nil {
				tab.UpdateBar()
			}
			se.Apply()
			gi.ErrorSnackbar(fr, gi.SaveSettings(se), "Error saving "+se.Label()+" settings")
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
