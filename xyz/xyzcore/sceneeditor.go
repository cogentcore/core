// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzcore

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/xyz"
)

// SceneEditor provides a toolbar controller and manipulation abilities
// for a [Scene].
type SceneEditor struct {
	core.Frame
}

func (sv *SceneEditor) Init() {
	sv.Frame.Init()
	sv.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	core.AddChildAt(sv, "scene", func(w *Scene) {})
	core.AddChildAt(sv, "tb", func(w *core.Toolbar) {
		w.Maker(sv.MakeToolbar)
	})
}

// SceneWidget returns the [Scene] widget.
func (sv *SceneEditor) SceneWidget() *Scene {
	return sv.ChildByName("scene", 0).(*Scene)
}

// SceneXYZ returns the [xyz.Scene].
func (sv *SceneEditor) SceneXYZ() *xyz.Scene {
	return sv.SceneWidget().XYZ
}

func (sv *SceneEditor) MakeToolbar(p *core.Plan) {
	sw := sv.SceneWidget()
	sc := sv.SceneXYZ()
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Update).SetTooltip("reset to default initial display").
			OnClick(func(e events.Event) {
				sc.SetCamera("default")
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ZoomIn).SetTooltip("zoom in").
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Zoom(-.05)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ZoomOut).SetTooltip("zoom out").
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).OnClick(func(e events.Event) {
			sc.Camera.Zoom(.05)
			sc.SetNeedsUpdate()
			sv.NeedsRender()
		})
	})
	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Text) {
		w.SetText("Rot:").SetTooltip("rotate display")
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowLeft).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Orbit(5, 0)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowUp).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Orbit(0, 5)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowDown).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Orbit(0, -5)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowRight).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Orbit(-5, 0)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Text) {
		w.SetText("Pan:").SetTooltip("pan display")
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowLeft).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Pan(-.2, 0)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowUp).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Pan(0, .2)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowDown).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Pan(0, -.2)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Button) {
		w.SetIcon(icons.KeyboardArrowRight).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			}).
			OnClick(func(e events.Event) {
				sc.Camera.Pan(.2, 0)
				sc.SetNeedsUpdate()
				sv.NeedsRender()
			})
	})
	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Text) {
		w.SetText("Save:")
	})

	for i := 1; i <= 4; i++ {
		nm := fmt.Sprintf("%d", i)
		core.AddAt(p, "save_"+nm, func(w *core.Button) {
			w.SetText(nm).
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
					sv.NeedsRender()
				})
		})
	}
	core.Add(p, func(w *core.Separator) {})

	core.Add(p, func(w *core.Chooser) {
		core.Bind(&sw.SelectionMode, w)
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Edit").SetIcon(icons.Edit).
			SetTooltip("edit the currently selected object").
			OnClick(func(e events.Event) {
				if sw.CurrentSelected == nil {
					return
				}
				d := core.NewBody().AddTitle("Selected Node")
				core.NewForm(d).SetStruct(sw.CurrentSelected)
				d.RunWindowDialog(sv)
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Edit Scene").SetIcon(icons.Edit).
			SetTooltip("edit the 3D Scene object (for access to meshes, textures etc)").
			OnClick(func(e events.Event) {
				d := core.NewBody().AddTitle("xyz.Scene")
				core.NewForm(d).SetStruct(sv.SceneXYZ())
				d.RunWindowDialog(sv)
			})
	})
}
