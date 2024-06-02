// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

/* TODO(config)
import (
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles/states"
)

// KeyChordValue represents a [key.Chord] value with a button.
type KeyChordValue struct {
	ValueBase[*core.Button]
}

func (v *KeyChordValue) Config() {
	v.Widget.On(events.KeyChord, func(e events.Event) {
		if !v.Widget.StateIs(states.Focused) {
			return
		}
		if !v.SetValue(e.KeyChord()) {
			return
		}
		e.SetHandled()
		v.Update()
		v.SendChange()
	})
	v.Widget.AddContextMenu(func(m *core.Scene) {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).OnClick(func(e events.Event) {
			if !v.SetValue(key.Chord("")) {
				return
			}
			v.Update()
			v.SendChange()
		})
	})
}

func (v *KeyChordValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	v.Widget.SetText(txt)
}
*/
