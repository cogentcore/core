// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
)

// SettingsEditorToolbarBase is the base toolbar configuration function used in [SettingsEditor].
func SettingsEditorToolbarBase(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(AppearanceSettings.SaveScreenZoom).SetIcon(icons.ZoomIn)
		w.SetAfterFunc(func() {
			AppearanceSettings.Apply()
			UpdateAll()
		})

		p.Parent.(*Toolbar).AddOverflowMenu(func(m *Scene) {
			NewFuncButton(m).SetFunc(resetAllSettings).SetConfirm(true).SetText("Reset settings").SetIcon(icons.Delete)

			NewFuncButton(m).SetFunc(AppearanceSettings.deleteSavedWindowGeoms).SetConfirm(true).SetIcon(icons.Delete)
			NewFuncButton(m).SetFunc(ProfileToggle).SetShortcut("Control+Alt+R").SetText("Profile performance").SetIcon(icons.Analytics)
			NewSeparator(m)
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
	d.NewWindow().SetCloseOnBack(true).Run()
}

// SettingsEditor adds to the given body an editor of user settings.
func SettingsEditor(b *Body) {
	b.AddAppBar(func(p *tree.Plan) {
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
