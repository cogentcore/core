// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3dv

import (
	"log"
	"reflect"
	"sort"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi3d"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  MeshValue

// Value restylesers MeshValue as the viewer of MeshName
// func (mn gi3d.MeshName) Value() giv.Value {
// 	return &MeshValue{}
// }

// MeshValue presents an action for displaying a MeshName and selecting
// meshes from a ChooserDialog
type MeshValue struct {
	giv.ValueBase
}

func (vv *MeshValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *MeshValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	bt.SetText(txt)
	bt.Update()
}

func (vv *MeshValue) ConfigWidget(widg gi.Widget, sc *gi.Scene) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *MeshValue) HasDialog() bool {
	return true
}

func (vv *MeshValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	if vv.OwnKind != reflect.Struct {
		return
	}
	ndi, ok := vv.Owner.(gi3d.Node)
	if !ok {
		return
	}
	sci, err := ndi.ParentByTypeTry(gi3d.SceneType, ki.Embeds)
	if err != nil {
		log.Println(err)
		return
	}
	sc := sci.(*gi3d.Scene)
	sl := sc.MeshList()
	sort.Strings(sl)

	si := 0
	cur := laser.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	d := gi.NewDialog(ctx).Title("Select a mesh").Prompt(desc).FullWindow(true)
	giv.NewSliceView(d).SetSlice(&sl).SetSelVal(cur).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			ms := sl[si]
			vv.SetValue(ms)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TexValue

/*

This doesn't work because texture is on Material which doesn't have a pointer to the
Scene!

// Value restylesers TexValue as the viewer of TexName
func (mn TexName) Value() giv.Value {
	vv := TexValue{}
	vv.Init(&vv)
	return &vv
}

// TexValue presents an action for displaying a TexName and selecting
// textures from a ChooserDialog
type TexValue struct {
	giv.ValueBase
}

func (vv *TexValue) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.TypeAction
	return vv.WidgetTyp
}

func (vv *TexValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *TexValue) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Button)
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
		vvv, _ := recv.Embed(TypeTexValue).(*TexValue)
		ac := vvv.Widget.(*gi.Button)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *TexValue) HasAction() bool {
	return true
}

func (vv *TexValue) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	if vv.OwnKind != reflect.Struct {
		return
	}
	mati, ok := vv.Owner.(*Material)
	if !ok {
		return
	}
	sci, err := ndi.ParentByTypeTry(TypeScene, ki.Embeds)
	if err != nil {
		log.Println(err)
		return
	}
	sc := sci.Embed(TypeScene).(*Scene)
	sl := sc.TextureList()
	sort.Strings(sl)

	cur := laser.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	giv.SliceViewSelectDialog(vp, &sl, cur, giv.DlgOpts{Title: "Select a Texture", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				si := giv.SliceViewSelectDialogValue(ddlg)
				if si >= 0 {
					vv.SetValue(sl[si])
					vv.UpdateWidget()
				}
			}
			if dlgRecv != nil && dlgFunc != nil {
				dlgFunc(dlgRecv, send, sig, data)
			}
		})
}
*/
