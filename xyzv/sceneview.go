// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xyz"
)

// SceneView provides a toolbar controller for an xyz.Scene,
// and manipulation abilities.
type SceneView struct {
	gi.Layout
}

func (sv *SceneView) OnInit() {
	sv.Layout.OnInit()
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

func (sv *SceneView) ConfigWidget() {
	sv.ConfigSceneView()
}

func (sv *SceneView) ConfigSceneView() {
	if sv.HasChildren() {
		sv.UpdateToolbar()
		return
	}
	NewScene(sv, "scene")
	tb := gi.NewToolbar(sv, "tb")
	sv.ConfigToolbar(tb)
}

// SceneWidget returns the gi.Widget Scene (xyzv.Scene)
func (sv *SceneView) SceneWidget() *Scene {
	return sv.ChildByName("scene", 0).(*Scene)
}

// Scene returns the xyz.Scene
func (sv *SceneView) SceneXYZ() *xyz.Scene {
	return sv.SceneWidget().XYZ
}

func (sv *SceneView) Toolbar() *gi.Toolbar {
	tbi := sv.ChildByName("tb", 1)
	if tbi == nil {
		return nil
	}
	return tbi.(*gi.Toolbar)
}

func (sv *SceneView) UpdateToolbar() {
	tb := sv.Toolbar()
	if tb == nil {
		return
	}
	sw := sv.SceneWidget()
	smi := tb.ChildByName("selmode", 10)
	if smi != nil {
		sm := smi.(*gi.Chooser)
		sm.SetCurrentValue(sw.SelMode)
	}
}

func (sv *SceneView) ConfigToolbar(tb *gi.Toolbar) {
	sw := sv.SceneWidget()
	sc := sv.SceneXYZ()
	gi.NewButton(tb).SetIcon(icons.Update).SetTooltip("reset to default initial display").
		OnClick(func(e events.Event) {
			sc.SetCamera("default")
			sc.SetNeedsUpdate()
			sv.SetNeedsRender(true)
		})
	gi.NewButton(tb).SetIcon(icons.ZoomIn).SetTooltip("zoom in").
		OnClick(func(e events.Event) {
			sc.Camera.Zoom(-.05)
			sc.SetNeedsUpdate()
			sv.SetNeedsRender(true)
		})
	gi.NewButton(tb).SetIcon(icons.ZoomOut).SetTooltip("zoom out").
		OnClick(func(e events.Event) {
			sc.Camera.Zoom(.05)
			sc.SetNeedsUpdate()
			sv.SetNeedsRender(true)
		})
	gi.NewSeparator(tb)
	gi.NewLabel(tb).SetText("Rot:").SetTooltip("rotate display")
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowLeft).OnClick(func(e events.Event) {
		sc.Camera.Orbit(5, 0)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowUp).OnClick(func(e events.Event) {
		sc.Camera.Orbit(0, 5)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowDown).OnClick(func(e events.Event) {
		sc.Camera.Orbit(0, -5)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowRight).OnClick(func(e events.Event) {
		sc.Camera.Orbit(-5, 0)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewSeparator(tb)

	gi.NewLabel(tb).SetText("Pan:").SetTooltip("pan display")
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowLeft).OnClick(func(e events.Event) {
		sc.Camera.Pan(-.2, 0)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowUp).OnClick(func(e events.Event) {
		sc.Camera.Pan(0, .2)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowDown).OnClick(func(e events.Event) {
		sc.Camera.Pan(0, -.2)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowRight).OnClick(func(e events.Event) {
		sc.Camera.Pan(.2, 0)
		sc.SetNeedsUpdate()
		sv.SetNeedsRender(true)
	})
	gi.NewSeparator(tb)

	gi.NewLabel(tb).SetText("Save:")
	for i := 1; i <= 4; i++ {
		i := i
		nm := fmt.Sprintf("%d", i)
		gi.NewButton(tb).SetText(nm).
			SetTooltip("first click (or + Shift) saves current view, second click restores to saved state").
			OnClick(func(e events.Event) {
				cam := nm
				if e.HasAllModifiers(e.Modifiers(), key.Shift) {
					sc.SaveCamera(cam)
				} else {
					err := sc.SetCamera(cam)
					if err != nil {
						sc.SaveCamera(cam)
					}
				}
				fmt.Printf("Camera %s: %v\n", cam, sc.Camera.GenGoSet(""))
				sc.SetNeedsUpdate()
				sv.SetNeedsRender(true)
			})
	}
	gi.NewSeparator(tb)

	sm := gi.NewChooser(tb, "selmode").SetEnum(sw.SelMode)
	sm.OnChange(func(e events.Event) {
		sw.SelMode = sm.CurrentItem.Value.(SelModes)
	})
	sm.SetCurrentValue(sw.SelMode)

	gi.NewButton(tb).SetText("Edit").SetIcon(icons.Edit).
		SetTooltip("edit the currently-selected object").
		OnClick(func(e events.Event) {
			if sw.CurSel == nil {
				return
			}
			d := gi.NewBody().AddTitle("Selected Node")
			giv.NewStructView(d).SetStruct(sw.CurSel)
			d.NewFullDialog(sv).SetNewWindow(true).Run()
		})

	gi.NewButton(tb).SetText("Edit Scene").SetIcon(icons.Edit).
		SetTooltip("edit the 3D Scene object (for access to meshes, textures etc)").
		OnClick(func(e events.Event) {
			d := gi.NewBody().AddTitle("xyz.Scene")
			giv.NewStructView(d).SetStruct(sv.SceneXYZ())
			d.NewFullDialog(sv).SetNewWindow(true).Run()
		})
}
