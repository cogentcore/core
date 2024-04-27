// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/system"
)

// SettingsViewToolbarBase is the base toolbar configuration function used in [SettingsView].
func SettingsViewToolbarBase(tb *core.Toolbar) {
	NewFuncButton(tb, core.AppearanceSettings.SaveScreenZoom).SetIcon(icons.ZoomIn).
		SetAfterFunc(func() {
			core.AppearanceSettings.Apply()
			core.UpdateAll()
		})
		// todo: doesn't work to update..

	tb.AddOverflowMenu(func(m *core.Scene) {
		NewFuncButton(m, core.ResetAllSettings).SetText("Reset settings").SetIcon(icons.Delete).SetConfirm(true)
		core.NewButton(m).SetText("App version").SetIcon(icons.Info).OnClick(func(e events.Event) {
			d := core.NewBody().AddTitle("App version")
			core.NewText(d).SetText("App version: " + system.AppVersion)
			core.NewText(d).SetText("Core version: " + system.CoreVersion)
			d.AddOKOnly().RunDialog(tb)
		})

		NewFuncButton(m, core.AppearanceSettings.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
		NewFuncButton(m, core.ProfileToggle).SetText("Profile performance").SetIcon(icons.Analytics)
		core.NewSeparator(m)
	})
}

// SettingsWindow makes and runs a new window for viewing user settings.
func SettingsWindow() {
	if core.ActivateExistingMainWindow(&core.AllSettings) {
		return
	}
	d := core.NewBody("settings").SetTitle("Settings").SetData(&core.AllSettings)
	SettingsView(d)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// SettingsView adds to the given body a view of user settings
func SettingsView(b *core.Body) {
	b.AddAppBar(func(tb *core.Toolbar) {
		SettingsViewToolbarBase(tb)
		for _, se := range core.AllSettings {
			se.ConfigToolbar(tb)
		}
	})

	tabs := core.NewTabs(b)

	for _, se := range core.AllSettings {
		fr := tabs.NewTab(se.Label())

		NewStructView(fr).SetStruct(se).OnChange(func(e events.Event) {
			if tab := b.GetTopAppBar(); tab != nil {
				tab.ApplyStyleUpdate()
			}
			core.UpdateSettings(fr, se)
		})
	}
}
