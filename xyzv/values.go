// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

import (
	"log/slog"
	"reflect"
	"sort"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
)

func init() {
	giv.AddValue(xyz.MeshName(""), func() giv.Value { return &MeshValue{} })
}

// MeshValue represents an [xyz.MeshName] with a button.
type MeshValue struct {
	giv.ValueBase[*gi.Button]
}

func (v *MeshValue) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetIcon(icons.DeployedCode)
	giv.ConfigDialogWidget(v, false)
}

func (v *MeshValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *MeshValue) ConfigDialog(d *gi.Body) (bool, func()) {
	d.SetTitle("Select a mesh")
	if v.OwnKind != reflect.Struct {
		return false, nil
	}
	ndi, ok := v.Owner.(xyz.Node)
	if !ok {
		return false, nil
	}
	sci := ndi.ParentByType(xyz.SceneType, tree.Embeds)
	if sci == nil {
		slog.Error("missing parent scene for node", "node", ndi)
		return false, nil
	}
	sc := xyz.AsScene(sci)
	sl := sc.MeshList()
	sort.Strings(sl)

	si := 0
	cur := laser.ToString(v.Value.Interface())
	giv.NewSliceView(d).SetSlice(&sl).SetSelectedValue(cur).BindSelect(&si)

	return true, func() {
		if si >= 0 {
			ms := sl[si]
			v.SetValue(ms)
			v.Update()
		}
	}
}

/*

TODO: This doesn't work because texture is on Material which doesn't have a pointer to the Scene!

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

func (vv *TexValue) Config(widg gi.Node2D) {
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
