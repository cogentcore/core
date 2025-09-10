// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"math"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/examples/assets"
	_ "cogentcore.org/core/xyz/io/obj"
	"cogentcore.org/core/xyz/xyzcore"

	"cogentcore.org/core/math32"
)

func main() {
	animOn := false
	animSpeed := float32(0.01)
	animGopher := true
	animHoop := true
	animAngle := float32(0)

	b := core.NewBody("XYZ Demo")

	core.NewText(b).SetText(`This is a demonstration of <b>XYZ</b>, the <a href="https://cogentcore.org/core">Cogent Core</a> <i>3D</i> framework`).
		SetType(core.TextHeadlineSmall).
		Styler(func(s *styles.Style) {
			s.Text.Align = text.Center
			s.Text.AlignV = text.Center
		})

	core.NewButton(b).SetText("Toggle animation").OnClick(func(e events.Event) {
		animOn = !animOn
	})

	se := xyzcore.NewSceneEditor(b)
	se.UpdateWidget()
	sw := se.SceneWidget()
	sc := se.SceneXYZ()
	sw.SelectionMode = xyzcore.Manipulable

	// options - must be set here
	// sc.MultiSample = 1
	// se.Wireframe = true
	// sc.NoNav = true

	se.Styler(func(s *styles.Style) {
		sc.Background = colors.Scheme.Select.Container
	})

	xyz.NewAmbient(sc, "ambient", 0.3, xyz.DirectSun)
	xyz.NewDirectional(sc, "directional", 1, xyz.DirectSun).Pos.Set(0, 2, 1)
	// xyz.NewPoint(sc, "point", 1, xyz.DirectSun).Pos.Set(-5, 0, 2)
	// spot := xyz.NewSpot(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)
	// spot.LookAtOrigin()

	// se.Camera.Pose.Pos.Set(-2, 9, 3)
	sc.Camera.Pose.Pos.Set(0, 2, 10)
	// se.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	grtx := xyz.NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	_ = grtx

	cube := xyz.NewBox(sc, "cube1", 1, 1, 1)
	cube.Segs.Set(10, 10, 10) // not clear if any diff really..

	tree.AddChild(sc, func(g *xyz.Group) {
		tree.AddChild(g, func(n *xyz.Solid) {
			n.SetMesh(cube).SetColor(colors.Red).SetShiny(500).SetPos(-1, 0, 0)
		})
		tree.AddChild(g, func(n *xyz.Solid) {
			n.SetMesh(cube).SetColor(colors.Blue).SetShiny(10).SetReflective(0.2).
				SetPos(1, 1, 0)
			n.Pose.Scale.X = 2
		})
		tree.AddChild(g, func(n *xyz.Solid) {
			// alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!
			n.SetMesh(cube).SetColor(color.RGBA{0, 255, 0, 128}).SetShiny(20).SetPos(0, 0, 1)
		})
	})

	floorp := xyz.NewPlane(sc, "floor-plane", 100, 100)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(floorp).SetColor(colors.Tan).SetTexture(grtx).SetPos(0, -5, 0)
		n.Material.Tiling.Repeat.Set(40, 40)
		// n.Mat.Emissive.SetName("brown")
		// n.Mat.Bright = 2 // .5 for wood / brown
		// n.SetDisabled() // not selectable
	})

	lines := xyz.NewLines(sc, "Lines", []math32.Vector3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, math32.Vec2(.2, .1), xyz.CloseLines)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(lines).SetColor(color.RGBA{255, 255, 0, 128}).SetPos(0, 0, 1)
	})

	// xyz.NewArrow(sc, sc, "arrow", math32.Vec3(-1.5, -.5, .5), math32.Vec3(2, 1, 1), .05, colors.Cyan, xyz.StartArrow, xyz.EndArrow, 4, .5, 8)

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	// xyz.NewLineBox(sc, sc, "bbox", "bbox", math32.Box3{Min: math32.Vec3(-2, -2, -1), Max: math32.Vec3(-1, -1, .5)}, .01, color.RGBA{255, 255, 0, 255}, xyz.Active)

	cylinder := xyz.NewCylinder(sc, "cylinder", 1.5, .5, 32, 1, true, true)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(cylinder).SetPos(-2.25, 0, 0)
	})

	capsule := xyz.NewCapsule(sc, "capsule", 1.5, .5, 32, 1)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(capsule).SetColor(colors.Tan).SetPos(3.25, 0, 0)
	})

	sphere := xyz.NewSphere(sc, "sphere", .75, 32)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(sphere).SetColor(colors.Orange).SetPos(0, -2, 0)
		n.Material.Color.A = 200
	})

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo := errors.Log1(sc.OpenToLibraryFS(assets.Content, "gopher.obj", ""))
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	var smallGo *xyz.Group
	var goPos math32.Vector3

	tree.AddChildAt(sc, "go-group", func(g *xyz.Group) {
		// todo: need a new type for this
		bgo, _ := sc.AddFromLibrary("gopher", g)
		bgo.SetScale(.5, .5, .5).SetPos(1.4, -2.5, 0).SetAxisRotation(0, 1, 0, -160)

		smallGo, _ = sc.AddFromLibrary("gopher", g)
		smallGo.SetPos(-1.5, -2, 0).SetScale(.2, .2, .2)
		goPos = smallGo.Pose.Pos
	})

	torus := xyz.NewTorus(sc, "torus", .75, .1, 32)
	var hoop *xyz.Solid
	var hoopPos math32.Vector3
	tree.AddChildAt(sc, "torus", func(n *xyz.Solid) {
		n.SetMesh(torus).SetColor(colors.White).SetPos(-1.6, -1.6, -.2).SetAxisRotation(1, 0, 0, 90)
		n.Material.Color.A = 200
		hoop = n
		hoopPos = hoop.Pose.Pos
	})

	tree.AddChild(sc, func(n *xyz.Text2D) {
		n.SetText("Text2D can put <b>HTML</b> formatted Text anywhere you might <i>want</i>").SetPos(0, 2.2, 0)
		n.Styles.Text.Align = text.Center
		n.Pose.Scale.SetScalar(0.2)
	})

	// automatically tracks camera -- FPS effect
	tree.AddChildAt(sc, xyz.TrackCameraName, func(g *xyz.Group) {
		tree.AddChild(g, func(n *xyz.Solid) {
			// in front of camera
			n.SetMesh(cube).SetScale(.1, .1, 1).SetPos(.5, -.5, -2.5).
				SetColor(color.RGBA{255, 0, 255, 128})
		})
	})

	///////  Animation & Embedded controls

	radius := float32(0.3)
	sw.Animate(func(a *core.Animation) {
		if !animOn {
			return
		}
		if animHoop {
			tdx := radius * math32.Cos(animAngle)
			tdz := radius * math32.Sin(animAngle)
			tp := hoopPos
			tp.X += tdx
			tp.Z += tdz
			hoop.SetPosePos(tp)
		}

		if animGopher {
			gdx := 0.1 * radius * math32.Cos(animAngle+math.Pi)
			gdz := 0.1 * radius * math32.Sin(animAngle+math.Pi)
			gp := goPos
			gp.X += gdx
			gp.Z += gdz
			smallGo.SetPosePos(gp)
		}
		sw.NeedsRender()
		animAngle += animSpeed * a.Dt
	})

	/*
		emb := xyz.NewEmbed2D(sc, sc, "embed-but", 150, 100, xyz.FitContent)
		emb.Pose.Pos.Set(-2, 2, 0)
		// emb.Zoom = 1.5   // this is how to rescale overall size
		evlay := core.NewFrame(emb.Viewport, "vlay", core.LayoutVert)
		evlay.SetProp("margin", units.Ex(1))

		eabut := core.NewCheckBox(evlay, "anim-but")
		eabut.SetText("Animate")
		eabut.Tooltip = "toggle animation on and off"
		eabut.ButtonSig.Connect(win.This, func(recv, send tree.Node, sig int64, data any) {
			if sig == int64(core.ButtonToggled) {
				anim.On = eabut.IsChecked()
			}
		})

		cmb := core.NewButton(evlay, "anim-ctrl")
		cmb.SetText("Anim Ctrl")
		cmb.Tooltip = "options for what is animated (note: menu only works when not animating -- checkboxes would be more useful here but wanted to test menu function)"
		cmb.Menu.AddAction(core.ActOpts{Label: "Toggle Torus"},
			win.This, func(recv, send tree.Node, sig int64, data any) {
				anim.DoTorus = !anim.DoTorus
			})
		cmb.Menu.AddAction(core.ActOpts{Label: "Toggle Gopher"},
			win.This, func(recv, send tree.Node, sig int64, data any) {
				anim.DoGopher = !anim.DoGopher
			})
		cmb.Menu.AddAction(core.ActOpts{Label: "Edit Anim"},
			win.This, func(recv, send tree.Node, sig int64, data any) {
				core.FormDialog(vp, anim, core.DlgOpts{Title: "Animation Parameters"}, nil, nil)
			})

		sprw := core.NewFrame(evlay, "speed-lay", core.LayoutHoriz)
		core.NewText(sprw, "speed-text", "Speed: ")
		sb := core.NewSpinBox(sprw, "anim-speed")
		sb.SetMin(0.01)
		sb.Step = 0.01
		sb.SetValue(anim.Speed)
		sb.Tooltip = "determines the speed of rotation (step size)"

		spsld := core.NewSlider(evlay, "speed-slider")
		spsld.Dim = math32.X
		spsld.Min = 0.01
		spsld.Max = 1
		spsld.Step = 0.01
		spsld.PageStep = 0.1
		spsld.SetMinPrefWidth(units.Em(20))
		spsld.SetMinPrefHeight(units.Em(2))
		spsld.SetValue(anim.Speed)
		// spsld.Tracking = true
		spsld.Icon = icons.RadioButtonUnchecked

		sb.SpinBoxSig.Connect(rec.This, func(recv, send tree.Node, sig int64, data any) {
			anim.Speed = sb.Value
			spsld.SetValue(anim.Speed)
		})
		spsld.SliderSig.Connect(rec.This, func(recv, send tree.Node, sig int64, data any) {
			if core.SliderSignals(sig) == core.SliderValueChanged {
				anim.Speed = data.(float32)
				sb.SetValue(anim.Speed)
			}
		})
	*/

	sc.Update()
	b.RunMainWindow()
}
