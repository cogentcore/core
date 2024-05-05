// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzview

import (
	"log/slog"
	"reflect"
	"sort"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
	"cogentcore.org/core/xyz"
)

func init() {
	views.AddValue(xyz.MeshName(""), func() views.Value { return &MeshValue{} })
}

// MeshValue represents an [xyz.MeshName] with a button.
type MeshValue struct {
	views.ValueBase[*core.Button]
}

func (v *MeshValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.DeployedCode)
	views.ConfigDialogWidget(v, false)
}

func (v *MeshValue) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *MeshValue) ConfigDialog(d *core.Body) (bool, func()) {
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
	sc := sci.(*xyz.Scene)
	sl := sc.MeshList()
	sort.Strings(sl)

	si := 0
	cur := reflectx.ToString(v.Value.Interface())
	views.NewSliceView(d).SetSlice(&sl).SetSelectedValue(cur).BindSelect(&si)

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
func (mn TexName) Value() views.Value {
	vv := TexValue{}
	vv.Init(&vv)
	return &vv
}

// TexValue presents an action for displaying a TexName and selecting
// textures from a ChooserDialog
type TexValue struct {
	views.ValueBase
}

func (vv *TexValue) WidgetType() reflect.Type {
	vv.WidgetTyp = core.TypeAction
	return vv.WidgetTyp
}

func (vv *TexValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*core.Button)
	txt := reflectx.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *TexValue) Config(widg core.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*core.Button)
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send tree.Node, sig int64, data any) {
		vvv, _ := recv.Embed(TypeTexValue).(*TexValue)
		ac := vvv.Widget.(*core.Button)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *TexValue) HasAction() bool {
	return true
}

func (vv *TexValue) Activate(vp *core.Viewport2D, dlgRecv tree.Node, dlgFunc tree.RecvFunc) {
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
	sci, err := ndi.ParentByTypeTry(TypeScene, tree.Embeds)
	if err != nil {
		log.Println(err)
		return
	}
	sc := sci.Embed(TypeScene).(*Scene)
	sl := sc.TextureList()
	sort.Strings(sl)

	cur := reflectx.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	views.SliceViewSelectDialog(vp, &sl, cur, views.DlgOpts{Title: "Select a Texture", Prompt: desc}, nil,
		vv.This(), func(recv, send tree.Node, sig int64, data any) {
			if sig == int64(core.DialogAccepted) {
				ddlg := send.Embed(core.TypeDialog).(*core.Dialog)
				si := views.SliceViewSelectDialogValue(ddlg)
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
