// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

import (
	"fmt"

	"cogentcore.org/core/abilities"
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

func (sv *SceneView) Config() {
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
		sm.SetCurrentValue(sw.SelectionMode)
	}
}

func (sv *SceneView) ConfigToolbar(tb *gi.Toolbar) {
	sw := sv.SceneWidget()
	sc := sv.SceneXYZ()
	gi.NewButton(tb).SetIcon(icons.Update).SetTooltip("reset to default initial display").
		OnClick(func(e events.Event) {
			sc.SetCamera("default")
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.ZoomIn).SetTooltip("zoom in").
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Zoom(-.05)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.ZoomOut).SetTooltip("zoom out").
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Zoom(.05)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewSeparator(tb)
	gi.NewLabel(tb).SetText("Rot:").SetTooltip("rotate display")
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowLeft).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Orbit(5, 0)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowUp).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Orbit(0, 5)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowDown).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Orbit(0, -5)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowRight).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Orbit(-5, 0)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewSeparator(tb)

	gi.NewLabel(tb).SetText("Pan:").SetTooltip("pan display")
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowLeft).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Pan(-.2, 0)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowUp).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Pan(0, .2)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowDown).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Pan(0, -.2)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewButton(tb).SetIcon(icons.KeyboardArrowRight).
		Style(func(s *styles.Style) {
			s.SetAbilities(true, abilities.RepeatClickable)
		}).
		OnClick(func(e events.Event) {
			sc.Camera.Pan(.2, 0)
			sc.NeedsUpdate()
			sv.NeedsRender()
		})
	gi.NewSeparator(tb)

	gi.NewLabel(tb).SetText("Save:")
	for i := 1; i <= 4; i++ {
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
				sc.NeedsUpdate()
				sv.NeedsRender()
			})
	}
	gi.NewSeparator(tb)

	sm := gi.NewChooser(tb, "selmode").SetEnum(sw.SelectionMode)
	sm.OnChange(func(e events.Event) {
		sw.SelectionMode = sm.CurrentItem.Value.(SelectionModes)
	})
	sm.SetCurrentValue(sw.SelectionMode)

	gi.NewButton(tb).SetText("Edit").SetIcon(icons.Edit).
		SetTooltip("edit the currently-selected object").
		OnClick(func(e events.Event) {
			if sw.CurrentSelected == nil {
				return
			}
			d := gi.NewBody().AddTitle("Selected Node")
			giv.NewStructView(d).SetStruct(sw.CurrentSelected)
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
