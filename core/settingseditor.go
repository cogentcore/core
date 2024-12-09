// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
)

// settingsEditorToolbarBase is the base toolbar configuration
// function used in [SettingsEditor].
func settingsEditorToolbarBase(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(AppearanceSettings.SaveScreenZoom).SetIcon(icons.ZoomIn)
		w.SetAfterFunc(func() {
			AppearanceSettings.Apply()
			UpdateAll()
		})
	})
}

// SettingsWindow opens a window for editing user settings.
func SettingsWindow() { //types:add
	if RecycleMainWindow(&AllSettings) {
		return
	}
	d := NewBody("Settings").SetData(&AllSettings)
	SettingsEditor(d)
	d.RunWindow()
}

// SettingsEditor adds to the given body an editor of user settings.
func SettingsEditor(b *Body) {
	b.AddTopBar(func(bar *Frame) {
		tb := NewToolbar(bar)
		tb.Maker(settingsEditorToolbarBase)
		for _, se := range AllSettings {
			tb.Maker(se.MakeToolbar)
		}
		tb.AddOverflowMenu(func(m *Scene) {
			NewFuncButton(m).SetFunc(resetAllSettings).SetConfirm(true).SetText("Reset settings").SetIcon(icons.Delete)
			NewFuncButton(m).SetFunc(AppearanceSettings.deleteSavedWindowGeometries).SetConfirm(true).SetIcon(icons.Delete)
			NewFuncButton(m).SetFunc(ProfileToggle).SetShortcut("Control+Alt+R").SetText("Profile performance").SetIcon(icons.Analytics)
		})
	})

	tabs := NewTabs(b)

	for _, se := range AllSettings {
		fr, _ := tabs.NewTab(se.Label())

		NewForm(fr).SetStruct(se).OnChange(func(e events.Event) {
			UpdateSettings(fr, se)
		})
	}
}
