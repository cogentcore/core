// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
)

// KeyChordButton represents a [key.Chord] value with a button.
type KeyChordButton struct {
	core.Button
	Chord key.Chord
}

func (kc *KeyChordButton) WidgetValue() any { return &kc.Chord }

func (kc *KeyChordButton) Config() {
	kc.OnKeyChord(func(e events.Event) {
		if !kc.StateIs(states.Focused) {
			return
		}
		kc.Chord = e.KeyChord()
		e.SetHandled()
		kc.SendChange()
		kc.Update()
	})
	kc.Updater(func() {
		kc.SetText(kc.Chord.Label())
	})
	kc.AddContextMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).OnClick(func(e events.Event) {
			kc.Chord = ""
			kc.SendChange()
			kc.Update()
		})
	})
}
