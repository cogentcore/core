// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/icons"
)

// SettingsViewToolbarBase is the base toolbar configuration function used in [SettingsView].
func SettingsViewToolbarBase(tb *gi.Toolbar) {
	tb.AddOverflowMenu(func(m *gi.Scene) {
		NewFuncButton(m, gi.ResetAllSettings).SetText("Reset settings").SetIcon(icons.Delete).SetConfirm(true)
		gi.NewButton(m).SetText("App version").SetIcon(icons.Info).OnClick(func(e events.Event) {
			d := gi.NewBody().AddTitle("App version")
			gi.NewLabel(d).SetText("App version: " + goosi.AppVersion)
			gi.NewLabel(d).SetText("Core version: " + goosi.CoreVersion)
			d.AddOkOnly().NewDialog(tb).Run()
		})

		NewFuncButton(m, gi.AppearanceSettings.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
		NewFuncButton(m, gi.ProfileToggle).SetText("Profile performance").SetIcon(icons.Analytics)
		gi.NewSeparator(m)
	})
}

// SettingsWindow makes and runs a new window for viewing user settings.
func SettingsWindow() {
	if gi.ActivateExistingMainWindow(&gi.AllSettings) {
		return
	}
	d := gi.NewBody("settings").SetTitle("Settings").SetData(&gi.AllSettings)
	SettingsView(d)
	d.NewWindow().Run()
}

// SettingsView adds to the given body a view of user settings
func SettingsView(b *gi.Body) {
	b.AddAppBar(func(tb *gi.Toolbar) {
		SettingsViewToolbarBase(tb)
		for _, se := range gi.AllSettings {
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
