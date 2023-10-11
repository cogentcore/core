// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/goosi/mimedata"
	"goki.dev/gti"
	"goki.dev/laser"
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

func (vv *KeyChordValue) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	kc := vv.Widget.(*KeyChordEdit)
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
	FocusActive bool `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
}

func (kc *KeyChordEdit) OnInit() {
	kc.HandleKeyChordEvents()
	kc.KeyChordStyles()
}

func (kc *KeyChordEdit) KeyChordStyles() {
	kc.AddStyles(func(s *styles.Style) {
		s.Cursor = cursors.Pointer
		s.AlignV = styles.AlignTop
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		s.Width.SetCh(20)
		s.Padding.Set(units.Dp(8))
		s.SetStretchMaxWidth()
		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Tertiary.Container)
			s.Color = colors.Scheme.Tertiary.OnContainer
		} else {
			// STYTODO: get state styles working
			s.BackgroundColor.SetSolid(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		}
		if s.Is(states.Disabled) {
			s.Cursor = cursors.NotAllowed
		}
	})
}

func (kc *KeyChordEdit) MakeContextMenu(m *gi.Menu) {
	m.AddButton(gi.ActOpts{Label: "Clear"}, func(act *gi.Button) {
		kc.SetText("")
		kc.SendChange()
	})
}

/* todo: these should all be in WidgetBase now
func (kc *KeyChordEdit) HandleMouseEvent() {
	kc.AddFunc(events.MouseUp, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(events.Event)
		kcc := recv.Embed(TypeKeyChordEdit).(*KeyChordEdit)
		if me.Action == events.Press && me.Button == events.Left {
			if kcc.Selectable {
				me.SetHandled()
				kcc.SetSelected(!kcc.StateIs(states.Selected))
				if kcc.StateIs(states.Selected) {
					kcc.GrabFocus()
				}
				kcc.EmitSelectedSignal()
				kcc.UpdateSig()
			}
		}
		if me.Action == events.Release && me.Button == events.Right {
			me.SetHandled()
			kcc.EmitContextMenuSignal()
			kcc.This().(gi.Widget).ContextMenu()
		}
	})
}
*/

func (kc *KeyChordEdit) HandleKeyChord() {
	kc.On(events.KeyChord, func(e events.Event) {
		if kc.StateIs(states.Focused) {
			e.SetHandled()
			kc.SetText(string(e.KeyChord())) // that's easy!
			kc.EventMgr().ClipBoard().Write(mimedata.NewText(string(e.KeyChord())))
			kc.SendChange()
		}
	})
}

// func (kc *KeyChordEdit) ApplyStyle(sc *gi.Scene) {
// todo: are these still relevant?
// 	kc.SetCanFocusIfActive()
// 	kc.Selectable = true
// 	kc.Redrawable = true
// 	kc.StyleLabel()
// 	kc.LayoutLabel()
// }

func (kc *KeyChordEdit) HandleKeyChordEvents() {
	// kc.HoverEvent()
	// kc.MouseEvent()
	kc.HandleWidgetEvents()
	kc.HandleKeyChord()
}

// func (kc *KeyChordEdit) FocusChanged(change gi.FocusChanges) {
// 	switch change {
// 	case gi.FocusLost:
// 		kc.FocusActive = false
// 		kc.ClearSelected()
//		   kc.SendChange()
// 		kc.UpdateSig()
// 	case gi.FocusGot:
// 		kc.FocusActive = true
// 		kc.SetSelected(true)
// 		kc.ScrollToMe()
// 		kc.EmitFocusedSignal()
// 		kc.UpdateSig()
// 	case gi.FocusInactive:
// 		kc.FocusActive = false
// 		kc.ClearSelected()
// 		kc.ChordUpdated()
// 		kc.UpdateSig()
// 	case gi.FocusActive:
// 		// we don't re-activate on keypress here, so that you don't end up stuck
// 		// on a given keychord
// 		// kc.SetSelected()
// 		// kc.FocusActive = true
// 		// kc.ScrollToMe()
// 	}
// }
