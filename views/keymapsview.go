// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/keymap"
)

// KeyMapValue represents a [keymap.MapName] value with a button.
type KeyMapValue struct {
	ValueBase[*core.Button]
}

func (v *KeyMapValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	ConfigDialogWidget(v, false)
}

func (v *KeyMapValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

func (v *KeyMapValue) ConfigDialog(d *core.Body) (bool, func()) {
	d.SetTitle("Select a key map")
	si := 0
	cur := reflectx.ToString(v.Value.Interface())
	_, curRow, _ := keymap.AvailableMaps.MapByName(keymap.MapName(cur))
	NewTableView(d).SetSlice(&keymap.AvailableMaps).SetSelectedIndex(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			km := keymap.AvailableMaps[si]
			v.SetValue(keymap.MapName(km.Name))
			v.Update()
		}
	}
}
