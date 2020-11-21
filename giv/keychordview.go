// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  KeyChordValueView

// KeyChordValueView presents an KeyChordEdit for key.Chord
type KeyChordValueView struct {
	ValueViewBase
}

var KiT_KeyChordValueView = kit.Types.AddType(&KeyChordValueView{}, nil)

func (vv *KeyChordValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_KeyChordEdit
	return vv.WidgetTyp
}

func (vv *KeyChordValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	kc := vv.Widget.(*KeyChordEdit)
	txt := kit.ToString(vv.Value.Interface())
	kc.SetText(txt)
}

func (vv *KeyChordValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	kc := vv.Widget.(*KeyChordEdit)
	kc.KeyChordSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_KeyChordValueView).(*KeyChordValueView)
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
	FocusActive bool      `json:"-" xml:"-" desc:"true if the keyboard focus is active or not -- when we lose active focus we apply changes"`
	KeyChordSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal -- only one event, when chord is updated from key input"`
}

var KiT_KeyChordEdit = kit.Types.AddType(&KeyChordEdit{}, KeyChordEditProps)

func (kc *KeyChordEdit) Disconnect() {
	kc.Label.Disconnect()
	kc.KeyChordSig.DisconnectAll()
}

var KeyChordEditProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(2),
	"vertical-align":   gist.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Control,
	"border-width":     units.NewPx(1),
	"border-radius":    units.NewPx(4),
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gist.BorderSolid,
	"height":           units.NewEm(1),
	"width":            units.NewCh(20),
	"max-width":        -1,
	gi.LabelSelectors[gi.LabelActive]: ki.Props{
		"background-color": "lighter-0",
	},
	gi.LabelSelectors[gi.LabelInactive]: ki.Props{
		"color": "lighter-50",
	},
	gi.LabelSelectors[gi.LabelSelected]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
}

// ChordUpdated emits KeyChordSig when a new chord has been entered
func (kc *KeyChordEdit) ChordUpdated() {
	kc.KeyChordSig.Emit(kc.This(), 0, kc.Text)
}

func (kc *KeyChordEdit) MakeContextMenu(m *gi.Menu) {
	m.AddAction(gi.ActOpts{Label: "Clear"},
		kc, func(recv, send ki.Ki, sig int64, data interface{}) {
			kcc := recv.Embed(KiT_KeyChordEdit).(*KeyChordEdit)
			kcc.SetText("")
			kcc.ChordUpdated()
		})
}

func (kc *KeyChordEdit) MouseEvent() {
	kc.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		kcc := recv.Embed(KiT_KeyChordEdit).(*KeyChordEdit)
		if me.Action == mouse.Press && me.Button == mouse.Left {
			if kcc.Selectable {
				me.SetProcessed()
				kcc.SetSelectedState(!kcc.IsSelected())
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
	kc.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		kcc := recv.Embed(KiT_KeyChordEdit).(*KeyChordEdit)
		if kcc.HasFocus() && kcc.FocusActive {
			kt := d.(*key.ChordEvent)
			kt.SetProcessed()
			kcc.SetText(string(kt.Chord())) // that's easy!
			oswin.TheApp.ClipBoard(kc.ParentWindow().OSWin).Write(mimedata.NewText(string(kt.Chord())))
			kcc.ChordUpdated()
		}
	})
}

func (kc *KeyChordEdit) Style2D() {
	kc.SetCanFocusIfActive()
	kc.Selectable = true
	kc.Redrawable = true
	kc.StyleLabel()
	kc.StyMu.Lock()
	kc.LayState.SetFromStyle(&kc.Sty.Layout) // also does reset
	kc.StyMu.Unlock()
	kc.LayoutLabel()
}

func (kc *KeyChordEdit) ConnectEvents2D() {
	kc.HoverEvent()
	kc.MouseEvent()
	kc.KeyChordEvent()
}

func (kc *KeyChordEdit) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusLost:
		kc.FocusActive = false
		kc.ClearSelected()
		kc.ChordUpdated()
		kc.UpdateSig()
	case gi.FocusGot:
		kc.FocusActive = true
		kc.SetSelected()
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
