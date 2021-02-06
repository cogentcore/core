// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SceneView provides a toolbar controller for a gi3d.Scene
type SceneView struct {
	gi.Layout
}

var KiT_SceneView = kit.Types.AddType(&SceneView{}, nil)

// AddNewSceneView adds a new SceneView to given parent node, with given name.
func AddNewSceneView(parent ki.Ki, name string) *SceneView {
	return parent.AddNewChild(KiT_SceneView, name).(*SceneView)
}

// Config configures the overall view widget
func (sv *SceneView) Config() {
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(KiT_Scene, "scene")
	config.Add(gi.KiT_ToolBar, "tbar")
	mods, updt := sv.ConfigChildren(config)
	if mods {
		sc := sv.Scene()
		sc.Defaults()
		sc.SetStretchMaxWidth()
		sc.SetStretchMaxHeight()
		sv.ToolbarConfig()
	}
	sv.UpdateEnd(updt)
}

// IsConfiged returns true if widget is fully configured
func (sv *SceneView) IsConfiged() bool {
	if len(sv.Kids) == 0 {
		return false
	}
	return true
}

func (sv *SceneView) Scene() *Scene {
	return sv.ChildByName("scene", 0).(*Scene)
}

func (sv *SceneView) Toolbar() *gi.ToolBar {
	tbi := sv.ChildByName("tbar", 1)
	if tbi == nil {
		return nil
	}
	return tbi.(*gi.ToolBar)
}

func (sv *SceneView) ToolbarConfig() {
	tbar := sv.Toolbar()
	if len(tbar.Kids) != 0 {
		return
	}
	tbar.SetStretchMaxWidth()
	tbar.AddAction(gi.ActOpts{Icon: "update", Tooltip: "reset to default initial display, and rebuild everything"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.SetCamera("default")
			scc.Update()
		})
	tbar.AddAction(gi.ActOpts{Icon: "zoom-in", Tooltip: "zoom in"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Zoom(-.05)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "zoom-out", Tooltip: "zoom out"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Zoom(.05)
			scc.UpdateSig()
		})
	tbar.AddSeparator("rot")
	gi.AddNewLabel(tbar, "rot", "Rot:")
	tbar.AddAction(gi.ActOpts{Icon: "wedge-left"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Orbit(5, 0)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-up"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Orbit(0, 5)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-down"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Orbit(0, -5)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-right"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Orbit(-5, 0)
			scc.UpdateSig()
		})
	tbar.AddSeparator("pan")
	gi.AddNewLabel(tbar, "pan", "Pan:")
	tbar.AddAction(gi.ActOpts{Icon: "wedge-left"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Pan(-.2, 0)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-up"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Pan(0, .2)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-down"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Pan(0, -.2)
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Icon: "wedge-right"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			scc.Camera.Pan(.2, 0)
			scc.UpdateSig()
		})
	tbar.AddSeparator("save")
	gi.AddNewLabel(tbar, "save", "Save:")
	tbar.AddAction(gi.ActOpts{Label: "1", Icon: "save", Tooltip: "first click (or + Shift) saves current view, second click restores to saved state"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			em := svv.EventMgr2D()
			scc := svv.Scene()
			cam := "1"
			if key.HasAllModifierBits(em.LastModBits, key.Shift) {
				scc.SaveCamera(cam)
			} else {
				err := scc.SetCamera(cam)
				if err != nil {
					scc.SaveCamera(cam)
				}
			}
			fmt.Printf("Camera %s: %v\n", cam, scc.Camera.GenGoSet(""))
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Label: "2", Icon: "save", Tooltip: "first click saves current view, second click restores to saved state"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			em := svv.EventMgr2D()
			scc := svv.Scene()
			cam := "2"
			if key.HasAllModifierBits(em.LastModBits, key.Shift) {
				scc.SaveCamera(cam)
			} else {
				err := scc.SetCamera(cam)
				if err != nil {
					scc.SaveCamera(cam)
				}
			}
			fmt.Printf("Camera %s: %v\n", cam, scc.Camera.GenGoSet(""))
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Label: "3", Icon: "save", Tooltip: "first click (or + Shift) saves current view, second click restores to saved state"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			em := svv.EventMgr2D()
			scc := svv.Scene()
			cam := "3"
			if key.HasAllModifierBits(em.LastModBits, key.Shift) {
				scc.SaveCamera(cam)
			} else {
				err := scc.SetCamera(cam)
				if err != nil {
					scc.SaveCamera(cam)
				}
			}
			fmt.Printf("Camera %s: %v\n", cam, scc.Camera.GenGoSet(""))
			scc.UpdateSig()
		})
	tbar.AddAction(gi.ActOpts{Label: "4", Icon: "save", Tooltip: "first click (or + Shift) saves current view, second click restores to saved state"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			em := svv.EventMgr2D()
			scc := svv.Scene()
			cam := "4"
			if key.HasAllModifierBits(em.LastModBits, key.Shift) {
				scc.SaveCamera(cam)
			} else {
				err := scc.SetCamera(cam)
				if err != nil {
					scc.SaveCamera(cam)
				}
			}
			fmt.Printf("Camera %s: %v\n", cam, scc.Camera.GenGoSet(""))
			scc.UpdateSig()
		})
	tbar.AddSeparator("sel")
	cb := gi.AddNewComboBox(tbar, "selmode")
	cb.ItemsFromEnum(KiT_SelModes, true, 25)
	cb.SetCurIndex(int(sv.Scene().SelMode))
	cb.ComboSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		svv := recv.Embed(KiT_SceneView).(*SceneView)
		cbb := send.(*gi.ComboBox)
		scc := svv.Scene()
		scc.SelMode = SelModes(cbb.CurIndex)
		scc.UpdateSig()
	})
	tbar.AddAction(gi.ActOpts{Label: "Edit", Icon: "edit", Tooltip: "edit the currently-selected object"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			if scc.CurSel != nil {
				giv.StructViewDialog(svv.Viewport, scc.CurSel, giv.DlgOpts{Title: "Select Node"}, nil, nil)
			}
		})
	tbar.AddAction(gi.ActOpts{Label: "Edit Scene", Icon: "edit", Tooltip: "edit the overall scene object (for access to meshes, etc)"}, sv.This(),
		func(recv, send ki.Ki, sig int64, data interface{}) {
			svv := recv.Embed(KiT_SceneView).(*SceneView)
			scc := svv.Scene()
			giv.StructViewDialog(svv.Viewport, scc, giv.DlgOpts{Title: "Scene"}, nil, nil)
		})
}
