// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

////////////////////////////////////////////////////////////////////////////////////////
//  KeyChordValue

// KeyChordValue presents an KeyChordEdit for key.Chord
type KeyChordValue struct {
	ValueBase
}

func (vv *KeyChordValue) WidgetType() *gti.Type {
	vv.WidgetTyp = KeyChordEditType
	return vv.WidgetTyp
}

func (vv *KeyChordValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	kc := vv.Widget.(*KeyChordEdit)
	txt := laser.ToString(vv.Value.Interface())
	kc.SetText(txt)
}

func (vv *KeyChordValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	kc := vv.Widget.(*KeyChordEdit)
	kc.Config()
	kc.OnChange(func(e events.Event) {
		if vv.SetValue(key.Chord(kc.Text)) {
			vv.UpdateWidget()
		}
		vv.SendChange()
	})
	vv.UpdateWidget()
}

func (vv *KeyChordValue) HasDialog() bool {
	return false
}

/////////////////////////////////////////////////////////////////////////////////
// KeyChordEdit

// KeyChordEdit is a label widget that shows a key chord string, and, when in
// focus (after being clicked) will update to whatever key chord is typed --
// used for representing and editing key chords.
type KeyChordEdit struct {
	gi.Label

	// true if the keyboard focus is active or not -- when we lose active focus we apply changes
	FocusActive bool `json:"-" xml:"-"`
}

func (kc *KeyChordEdit) OnInit() {
	kc.Label.OnInit()
	kc.HandleEvents()
	kc.SetStyles()
	kc.AddContextMenu(kc.ContextMenu)
}

func (kc *KeyChordEdit) SetStyles() {
	kc.Style(func(s *styles.Style) {
		if !kc.IsReadOnly() {
			s.Cursor = cursors.Pointer
		}
		s.Align.Self = styles.Start
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		s.Min.X.Ch(20)
		s.Padding.Set(units.Dp(8))
		s.SetTextWrap(false)
		if s.Is(states.Selected) {
			s.Background = colors.C(colors.Scheme.Select.Container)
			s.Color = colors.Scheme.Select.OnContainer
		} else {
			// STYTODO: get state styles working
			s.Background = colors.C(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		}
	})
}

func (kc *KeyChordEdit) ContextMenu(m *gi.Scene) {
	gi.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).OnClick(func(e events.Event) {
		kc.SetText("")
		kc.SendChange()
	})
}

func (kc *KeyChordEdit) HandleEvents() {
	// kc.HoverEvent()
	// kc.MouseEvent()
	kc.HandleKeys()
}

func (kc *KeyChordEdit) HandleKeys() {
	kc.On(events.KeyChord, func(e events.Event) {
		if kc.StateIs(states.Focused) {
			e.SetHandled()
			kc.SetText(string(e.KeyChord())) // that's easy!
			kc.Clipboard().Write(mimedata.NewText(string(e.KeyChord())))
			kc.SendChange()
		}
	})
}
