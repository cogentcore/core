// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// SettingsEditorToolbarBase is the base toolbar configuration function used in [SettingsEditor].
func SettingsEditorToolbarBase(p *Plan) {
	Add(p, func(w *FuncButton) {
		w.SetFunc(AppearanceSettings.SaveScreenZoom).
			SetAfterFunc(func() {
				AppearanceSettings.Apply()
				UpdateAll()
			}).SetIcon(icons.ZoomIn)
		// todo: update..
	})

	/*
		tb.AddOverflowMenu(func(m *Scene) {
			NewFuncButton(m).SetFunc(ResetAllSettings).SetText("Reset settings").SetIcon(icons.Delete).SetConfirm(true)

			NewFuncButton(m).SetFunc(AppearanceSettings.DeleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
			NewFuncButton(m).SetFunc(ProfileToggle).SetText("Profile performance").SetIcon(icons.Analytics)
			NewSeparator(m)
		})
	*/
}

// SettingsWindow opens a window for editing user settings.
func SettingsWindow() { //types:add
	if RecycleMainWindow(&AllSettings) {
		return
	}
	d := NewBody("settings").SetTitle("Settings").SetData(&AllSettings)
	SettingsEditor(d)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// SettingsEditor adds to the given body an editor of user settings.
func SettingsEditor(b *Body) {
	b.AddAppBar(func(p *Plan) {
		SettingsEditorToolbarBase(p)
		for _, se := range AllSettings {
			se.MakeToolbar(p)
		}
	})

	tabs := NewTabs(b)

	for _, se := range AllSettings {
		fr := tabs.NewTab(se.Label())

		NewForm(fr).SetStruct(se).OnChange(func(e events.Event) {
			UpdateSettings(fr, se)
		})
	}
}
