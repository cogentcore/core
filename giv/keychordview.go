// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mimedata"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  KeyChordValueView

// KeyChordValueView presents an KeyChordEdit for key.Chord
type KeyChordValueView struct {
	ValueViewBase
}

func (vv *KeyChordValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = TypeKeyChordEdit
	return vv.WidgetTyp
}

func (vv *KeyChordValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	kc := vv.Widget.(*KeyChordEdit)
	txt := laser.ToString(vv.Value.Interface())
	kc.SetText(txt)
}

func (vv *KeyChordValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	kc := vv.Widget.(*KeyChordEdit)
	kc.KeyChordSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeKeyChordValueView).(*KeyChordValueView)
		kcc := vvv.Widget.(*KeyChordEdit)
		if vvv.SetValue(key.Chord(kcc.Text)) {
			vvv.UpdateWidget()
		}
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyChordValueView) HasAction() bool {
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

	// [view: -] signal -- only one event, when chord is updated from key input
	KeyChordSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal -- only one event, when chord is updated from key input"`
}

func (kc *KeyChordEdit) OnInit() {
	kc.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.Cursor = cursor.HandPointing
		s.AlignV = gist.AlignTop
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius = gist.BorderRadiusFull
		s.Width.SetCh(20)
		s.Padding.Set(units.Px(8 * gi.Prefs.DensityMul()))
		s.SetStretchMaxWidth()
		if w.IsSelected() {
			s.BackgroundColor.SetSolid(gi.ColorScheme.TertiaryContainer)
			s.Color = gi.ColorScheme.OnTertiaryContainer
		} else {
			// STYTODO: get state styles working
			s.BackgroundColor.SetSolid(gi.ColorScheme.SecondaryContainer)
			s.Color = gi.ColorScheme.OnSecondaryContainer
		}
	})
}

func (kc *KeyChordEdit) Disconnect() {
	kc.Label.Disconnect()
	kc.KeyChordSig.DisconnectAll()
}

var KeyChordEditProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// ChordUpdated emits KeyChordSig when a new chord has been entered
func (kc *KeyChordEdit) ChordUpdated() {
	kc.KeyChordSig.Emit(kc.This(), 0, kc.Text)
}

func (kc *KeyChordEdit) MakeContextMenu(m *gi.Menu) {
	m.AddAction(gi.ActOpts{Label: "Clear"},
		kc, func(recv, send ki.Ki, sig int64, data any) {
			kcc := recv.Embed(TypeKeyChordEdit).(*KeyChordEdit)
			kcc.SetText("")
			kcc.ChordUpdated()
		})
}

func (kc *KeyChordEdit) MouseEvent() {
	kc.ConnectEvent(goosi.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		kcc := recv.Embed(TypeKeyChordEdit).(*KeyChordEdit)
		if me.Action == mouse.Press && me.Button == mouse.Left {
			if kcc.Selectable {
				me.SetProcessed()
				kcc.SetSelected(!kcc.IsSelected())
				if kcc.IsSelected() {
					kcc.GrabFocus()
				}
				kcc.EmitSelectedSignal()
				kcc.UpdateSig()
			}
		}
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			kcc.EmitContextMenuSignal()
			kcc.This().(gi.Node2D).ContextMenu()
		}
	})
}

func (kc *KeyChordEdit) KeyChordEvent() {
	kc.ConnectEvent(goosi.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		kcc := recv.Embed(TypeKeyChordEdit).(*KeyChordEdit)
		if kcc.HasFocus() && kcc.FocusActive {
			kt := d.(*key.ChordEvent)
			kt.SetProcessed()
			kcc.SetText(string(kt.Chord())) // that's easy!
			goosi.TheApp.ClipBoard(kc.ParentRenderWin().RenderWin).Write(mimedata.NewText(string(kt.Chord())))
			kcc.ChordUpdated()
		}
	})
}

func (kc *KeyChordEdit) SetStyle() {
	kc.SetCanFocusIfActive()
	kc.Selectable = true
	kc.Redrawable = true
	kc.StyleLabel()
	kc.StyMu.Lock()
	kc.LayState.SetFromStyle(&kc.Style) // also does reset
	kc.StyMu.Unlock()
	kc.LayoutLabel()
}

func (kc *KeyChordEdit) ConnectEvents() {
	kc.HoverEvent()
	kc.MouseEvent()
	kc.KeyChordEvent()
}

func (kc *KeyChordEdit) FocusChanged(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		kc.FocusActive = false
		kc.ClearSelected()
		kc.ChordUpdated()
		kc.UpdateSig()
	case gi.FocusGot:
		kc.FocusActive = true
		kc.SetSelected(true)
		kc.ScrollToMe()
		kc.EmitFocusedSignal()
		kc.UpdateSig()
	case gi.FocusInactive:
		kc.FocusActive = false
		kc.ClearSelected()
		kc.ChordUpdated()
		kc.UpdateSig()
	case gi.FocusActive:
		// we don't re-activate on keypress here, so that you don't end up stuck
		// on a given keychord
		// kc.SetSelected()
		// kc.FocusActive = true
		// kc.ScrollToMe()
	}
}
