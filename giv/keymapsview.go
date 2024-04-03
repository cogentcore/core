// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
)

// KeyMapValue represents a [keyfun.MapName] value with a button.
type KeyMapValue struct {
	ValueBase[*gi.Button]
}

func (v *KeyMapValue) Config() {
	v.Widget.SetType(gi.ButtonTonal)
	ConfigDialogWidget(v, false)
}

func (v *KeyMapValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

func (v *KeyMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	d.SetTitle("Select a key map")
	si := 0
	cur := laser.ToString(v.Value.Interface())
	_, curRow, _ := keyfun.AvailableMaps.MapByName(keyfun.MapName(cur))
	NewTableView(d).SetSlice(&keyfun.AvailableMaps).SetSelectedIndex(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			km := keyfun.AvailableMaps[si]
			v.SetValue(keyfun.MapName(km.Name))
			v.Update()
		}
	}
}
