// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzview

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/views"
	"cogentcore.org/core/xyz"
)

// SceneView provides a toolbar controller for an [xyz.Scene],
// and manipulation abilities.
type SceneView struct {
	core.Frame
}

func (sv *SceneView) Init() {
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

// SceneWidget returns the core.Widget Scene (xyzv.Scene)
func (sv *SceneView) SceneWidget() *Scene {
	return sv.ChildByName("scene", 0).(*Scene)
}

// Scene returns the xyz.Scene
func (sv *SceneView) SceneXYZ() *xyz.Scene {
	return sv.SceneWidget().XYZ
}

func (sv *SceneView) Toolbar() *core.Toolbar {
	tbi := sv.ChildByName("tb", 1)
	if tbi == nil {
		return nil
	}
	return tbi.(*core.Toolbar)
}

func (sv *SceneView) UpdateToolbar() {
	tb := sv.Toolbar()
	if tb == nil {
		return
	}
	sw := sv.SceneWidget()
	smi := tb.ChildByName("selmode", 10)
	if smi != nil {
		sm := smi.(*core.Chooser)
		sm.SetCurrentValue(sw.SelectionMode)
	}
}

func (sv *SceneView) MakeToolbar(p *core.Plan) {
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

	core.AddAt(p, "selmode", func(w *core.Chooser) {
		w.SetEnum(sw.SelectionMode)
		w.OnChange(func(e events.Event) {
			sw.SelectionMode = w.CurrentItem.Value.(SelectionModes)
		})
		w.SetCurrentValue(sw.SelectionMode)
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Edit").SetIcon(icons.Edit).
			SetTooltip("edit the currently selected object").
			OnClick(func(e events.Event) {
				if sw.CurrentSelected == nil {
					return
				}
				d := core.NewBody().AddTitle("Selected Node")
				views.NewForm(d).SetStruct(sw.CurrentSelected)
				d.RunWindowDialog(sv)
			})
	})

	core.Add(p, func(w *core.Button) {
		w.SetText("Edit Scene").SetIcon(icons.Edit).
			SetTooltip("edit the 3D Scene object (for access to meshes, textures etc)").
			OnClick(func(e events.Event) {
				d := core.NewBody().AddTitle("xyz.Scene")
				views.NewForm(d).SetStruct(sv.SceneXYZ())
				d.RunWindowDialog(sv)
			})
	})
}
