// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"log"
	"reflect"
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  MeshValueView

// ValueView registers MeshValueView as the viewer of MeshName
func (mn MeshName) ValueView() giv.ValueView {
	vv := &MeshValueView{}
	ki.InitNode(vv)
	return vv
}

// MeshValueView presents an action for displaying a MeshName and selecting
// meshes from a ChooserDialog
type MeshValueView struct {
	giv.ValueViewBase
}

var KiT_MeshValueView = kit.Types.AddType(&MeshValueView{}, nil)

func (vv *MeshValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *MeshValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *MeshValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_MeshValueView).(*MeshValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *MeshValueView) HasAction() bool {
	return true
}

func (vv *MeshValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
	if vv.IsInactive() {
		return
	}
	if vv.OwnKind != reflect.Struct {
		return
	}
	ndi, ok := vv.Owner.(Node3D)
	if !ok {
		return
	}
	sci, err := ndi.ParentByTypeTry(KiT_Scene, ki.Embeds)
	if err != nil {
		log.Println(err)
		return
	}
	sc := sci.Embed(KiT_Scene).(*Scene)
	sl := sc.MeshList()
	sort.Strings(sl)

	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	giv.SliceViewSelectDialog(vp, &sl, cur, giv.DlgOpts{Title: "Select a Mesh", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
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

////////////////////////////////////////////////////////////////////////////////////////
//  TexValueView

/*

This doesn't work because texture is on Material which doesn't have a pointer to the
Scene!

// ValueView registers TexValueView as the viewer of TexName
func (mn TexName) ValueView() giv.ValueView {
	vv := TexValueView{}
	vv.Init(&vv)
	return &vv
}

// TexValueView presents an action for displaying a TexName and selecting
// textures from a ChooserDialog
type TexValueView struct {
	giv.ValueViewBase
}

var KiT_TexValueView = kit.Types.AddType(&TexValueView{}, nil)

func (vv *TexValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *TexValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	txt := kit.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none, click to select)"
	}
	ac.SetText(txt)
}

func (vv *TexValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	ac := vv.Widget.(*gi.Action)
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_TexValueView).(*TexValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.ViewportSafe(), nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *TexValueView) HasAction() bool {
	return true
}

func (vv *TexValueView) Activate(vp *gi.Viewport2D, dlgRecv ki.Ki, dlgFunc ki.RecvFunc) {
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
	sci, err := ndi.ParentByTypeTry(KiT_Scene, ki.Embeds)
	if err != nil {
		log.Println(err)
		return
	}
	sc := sci.Embed(KiT_Scene).(*Scene)
	sl := sc.TextureList()
	sort.Strings(sl)

	cur := kit.ToString(vv.Value.Interface())
	desc, _ := vv.Tag("desc")
	giv.SliceViewSelectDialog(vp, &sl, cur, giv.DlgOpts{Title: "Select a Texture", Prompt: desc}, nil,
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
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
