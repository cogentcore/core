// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
)

// KeyMapButton represents a [keymap.MapName] value with a button.
type KeyMapButton struct {
	Button
	MapName keymap.MapName
}

func (km *KeyMapButton) WidgetValue() any { return &km.MapName }

func (km *KeyMapButton) Init() {
	km.Button.Init()
	km.SetType(ButtonTonal)
	km.Updater(func() {
		km.SetText(string(km.MapName))
	})
	InitValueButton(km, false, func(d *Body) {
		d.SetTitle("Select a key map")
		si := 0
		_, curRow, _ := keymap.AvailableMaps.MapByName(km.MapName)
		tv := NewTable(d).SetSlice(&keymap.AvailableMaps).SetSelectedIndex(curRow).BindSelect(&si)
		tv.OnChange(func(e events.Event) {
			name := keymap.AvailableMaps[si]
			km.MapName = keymap.MapName(name.Name)
		})
	})
}

// KeyChordButton represents a [key.Chord] value with a button.
type KeyChordButton struct {
	Button
	Chord key.Chord
}

func (kc *KeyChordButton) WidgetValue() any { return &kc.Chord }

func (kc *KeyChordButton) Init() {
	kc.Button.Init()
	kc.SetType(ButtonTonal)
	kc.OnKeyChord(func(e events.Event) {
		if !kc.StateIs(states.Focused) {
			return
		}
		kc.Chord = e.KeyChord()
		e.SetHandled()
		kc.UpdateChange()
	})
	kc.Updater(func() {
		kc.SetText(kc.Chord.Label())
	})
	kc.AddContextMenu(func(m *Scene) {
		NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).OnClick(func(e events.Event) {
			kc.Chord = ""
			kc.UpdateChange()
		})
	})
}
