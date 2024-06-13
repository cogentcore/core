// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// SettingsEditorToolbarBase is the base toolbar configuration function used in [SettingsEditor].
func SettingsEditorToolbarBase(p *core.Plan) {
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(core.AppearanceSettings.SaveScreenZoom).
			SetAfterFunc(func() {
				core.AppearanceSettings.Apply()
				core.UpdateAll()
			}).SetIcon(icons.ZoomIn)
		// todo: update..
	})

	/*
		tb.AddOverflowMenu(func(m *core.Scene) {
			NewFuncButton(m, core.ResetAllSettings).SetText("Reset settings").SetIcon(icons.Delete).SetConfirm(true)

			NewFuncButton(m, core.AppearanceSettings.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
			NewFuncButton(m, core.ProfileToggle).SetText("Profile performance").SetIcon(icons.Analytics)
			core.NewSeparator(m)
		})
	*/
}

// SettingsWindow makes and runs a new window for editing user settings.
func SettingsWindow() {
	if core.RecycleMainWindow(&core.AllSettings) {
		return
	}
	d := core.NewBody("settings").SetTitle("Settings").SetData(&core.AllSettings)
	SettingsEditor(d)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// SettingsEditor adds to the given body an editor of user settings.
func SettingsEditor(b *core.Body) {
	b.AddAppBar(func(p *core.Plan) {
		SettingsEditorToolbarBase(p)
		for _, se := range core.AllSettings {
			se.MakeToolbar(p)
		}
	})

	tabs := core.NewTabs(b)

	for _, se := range core.AllSettings {
		fr := tabs.NewTab(se.Label())

		NewStructView(fr).SetStruct(se).OnChange(func(e events.Event) {
			core.UpdateSettings(fr, se)
		})
	}
}
