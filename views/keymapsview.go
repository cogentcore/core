// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
)

// KeyMapButton represents a [keymap.MapName] value with a button.
type KeyMapButton struct {
	core.Button
	MapName keymap.MapName
}

func (km *KeyMapButton) WidgetValue() any { return &km.MapName }

func (km *KeyMapButton) Init() {
	km.Button.Init()
	km.SetType(core.ButtonTonal)
	km.Updater(func() {
		km.SetText(string(km.MapName))
	})
	core.InitValueButton(km, false, func(d *core.Body) {
		d.SetTitle("Select a key map")
		si := 0
		_, curRow, _ := keymap.AvailableMaps.MapByName(km.MapName)
		tv := NewTableView(d).SetSlice(&keymap.AvailableMaps).SetSelectedIndex(curRow).BindSelect(&si)
		tv.OnChange(func(e events.Event) {
			name := keymap.AvailableMaps[si]
			km.MapName = keymap.MapName(name.Name)
		})
	})
}
