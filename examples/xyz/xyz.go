// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"log"
	"math"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/examples/assets"
	_ "cogentcore.org/core/xyz/io/obj"
	"cogentcore.org/core/xyz/xyzcore"

	"cogentcore.org/core/math32"
)

// Anim has control for animating
type Anim struct {

	// run the animation
	On bool

	// angular speed (in radians)
	Speed float32 `min:"0.01" step:"0.01"`

	// animate the torus
	DoTorus bool

	// animate the gopher
	DoGopher bool

	// current angle
	Ang float32 `edit:"-"`

	// the time.Ticker for animating the scene
	Ticker *time.Ticker `display:"-"`

	// the scene editor
	SceneEditor *xyzcore.SceneEditor

	// the torus
	Torus *xyz.Solid

	// the gopher
	Gopher *xyz.Group

	// original position
	TorusPosOrig math32.Vector3

	// original position
	GopherPosOrig math32.Vector3
}

// Start starts the animation ticker timer -- if on is true, then
// animation will actually start too.
func (an *Anim) Start(sv *xyzcore.SceneEditor, on bool) {
	an.SceneEditor = sv
	an.On = on
	an.DoTorus = true
	an.DoGopher = true
	an.Speed = .1
	an.GetObjs()
	an.Ticker = time.NewTicker(time.Second / 30) // 30 fps probably smoother
	go an.Animate()
}

// GetObjs gets the objects to animate
func (an *Anim) GetObjs() {
	sc := an.SceneEditor.SceneXYZ()
	torusi := sc.ChildByName("torus", 0)
	if torusi == nil {
		return
	}
	an.Torus = torusi.(*xyz.Solid)
	an.TorusPosOrig = an.Torus.Pose.Pos

	ggp := sc.ChildByName("go-group", 0)
	if ggp == nil {
		return
	}
	gophi := ggp.AsTree().Child(1)
	if gophi == nil {
		return
	}
	an.Gopher = gophi.(*xyz.Group)
	an.GopherPosOrig = an.Gopher.Pose.Pos
}

// Animate
func (an *Anim) Animate() {
	for {
		if an.Ticker == nil || an.SceneEditor.This == nil {
			return
		}
		<-an.Ticker.C // wait for tick
		if !an.On || an.SceneEditor.This == nil || an.Torus == nil || an.Gopher == nil {
			continue
		}
		sc := an.SceneEditor.SceneXYZ()
		radius := float32(0.3)

		if an.DoTorus {
			tdx := radius * math32.Cos(an.Ang)
			tdz := radius * math32.Sin(an.Ang)
			tp := an.TorusPosOrig
			tp.X += tdx
			tp.Z += tdz
			an.Torus.SetPosePos(tp)
		}

		if an.DoGopher {
			gdx := 0.1 * radius * math32.Cos(an.Ang+math.Pi)
			gdz := 0.1 * radius * math32.Sin(an.Ang+math.Pi)
			gp := an.GopherPosOrig
			gp.X += gdx
			gp.Z += gdz
			an.Gopher.SetPosePos(gp)
		}

		sc.SetNeedsUpdate()
		an.SceneEditor.SceneWidget().NeedsRender()
		an.Ang += an.Speed
	}
}

func main() {
	anim := &Anim{}
	b := core.NewBody("XYZ Demo")

	core.NewText(b).SetText(`This is a demonstration of <b>XYZ</b>, the <a href="https://cogentcore.org/core">Cogent Core</a> <i>3D</i> framework`).
		SetType(core.TextHeadlineSmall).
		Styler(func(s *styles.Style) {
			s.Text.Align = styles.Center
			s.Text.AlignV = styles.Center
		})

	core.NewButton(b).SetText("Toggle animation").OnClick(func(e events.Event) {
		anim.On = !anim.On
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
	xyz.NewAmbientLight(sc, "ambient", 0.3, xyz.DirectSun)

	dir := xyz.NewDirLight(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// se.Camera.Pose.Pos.Set(-2, 9, 3)
	sc.Camera.Pose.Pos.Set(0, 2, 10)
	// se.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // defaults to looking at origin

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	grtx := xyz.NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	// _ = grtx

	cbm := xyz.NewBox(sc, "cube1", 1, 1, 1)
	cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := xyz.NewGroup(sc)

	xyz.NewSolid(rbgp).SetMesh(cbm).
		SetColor(colors.Red).SetShiny(500).SetPos(-1, 0, 0)

	bcb := xyz.NewSolid(rbgp).SetMesh(cbm).
		SetColor(colors.Blue).SetShiny(10).SetReflective(0.2).
		SetPos(1, 1, 0)
	bcb.Pose.Scale.X = 2

	// alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!
	xyz.NewSolid(rbgp).SetMesh(cbm).
		SetColor(color.RGBA{0, 255, 0, 128}).SetShiny(20).SetPos(0, 0, 1)

	floorp := xyz.NewPlane(sc, "floor-plane", 100, 100)
	floor := xyz.NewSolid(sc).SetMesh(floorp).
		SetColor(colors.Tan).SetTexture(grtx).SetPos(0, -5, 0)
	floor.Material.Tiling.Repeat.Set(40, 40)

	// floor.Mat.Emissive.SetName("brown")
	// floor.Mat.Bright = 2 // .5 for wood / brown
	// floor.SetDisabled() // not selectable

	lnsm := xyz.NewLines(sc, "Lines", []math32.Vector3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, math32.Vec2(.2, .1), xyz.CloseLines)
	lns := xyz.NewSolid(sc).SetMesh(lnsm).SetColor(color.RGBA{255, 255, 0, 128})
	lns.Pose.Pos.Set(0, 0, 1)

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	cyan := colors.FromRGB(0, 255, 255)
	xyz.NewArrow(sc, sc, "arrow", math32.Vec3(-1.5, -.5, .5), math32.Vec3(2, 1, 1), .05, cyan, xyz.StartArrow, xyz.EndArrow, 4, .5, 4)

	// bbclr := styles.Color{}
	// bbclr.SetUInt8(255, 255, 0, 255)
	// xyz.NewLineBox(sc, sc, "bbox", "bbox", math32.Box3{Min: math32.Vec3(-2, -2, -1), Max: math32.Vec3(-1, -1, .5)}, .01, bbclr, xyz.Active)

	cylm := xyz.NewCylinder(sc, "cylinder", 1.5, .5, 32, 1, true, true)
	xyz.NewSolid(sc).SetMesh(cylm).SetPos(-2.25, 0, 0)

	capm := xyz.NewCapsule(sc, "capsule", 1.5, .5, 32, 1)
	xyz.NewSolid(sc).SetMesh(capm).SetColor(colors.Tan).
		SetPos(3.25, 0, 0)

	sphm := xyz.NewSphere(sc, "sphere", .75, 32)
	sph := xyz.NewSolid(sc).SetMesh(sphm).SetColor(colors.Orange)
	sph.Material.Color.A = 200
	sph.Pose.Pos.Set(0, -2, 0)

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo, err := sc.OpenToLibraryFS(assets.Content, "gopher.obj", "")
	if err != nil {
		log.Println(err)
	}
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	gogp := xyz.NewGroup(sc)
	gogp.SetName("go-group")

	bgo, _ := sc.AddFromLibrary("gopher", gogp)
	bgo.SetScale(.5, .5, .5).SetPos(1.4, -2.5, 0).SetAxisRotation(0, 1, 0, -160)

	sgo, _ := sc.AddFromLibrary("gopher", gogp)
	sgo.SetPos(-1.5, -2, 0).SetScale(.2, .2, .2)

	trsm := xyz.NewTorus(sc, "torus", .75, .1, 32)
	trs := xyz.NewSolid(sc).SetMesh(trsm).SetColor(colors.White).
		SetPos(-1.6, -1.6, -.2).SetAxisRotation(1, 0, 0, 90)
	trs.SetName("torus")
	trs.Material.Color.A = 200

	txt := xyz.NewText2D(sc).SetText("Text2D can put <b>HTML</b> formatted<br>Text anywhere you might <i>want</i>")
	txt.Styles.Text.Align = styles.Center
	txt.Pose.Scale.SetScalar(0.2)
	txt.SetPos(0, 2.2, 0)

	tcg := xyz.NewGroup(sc) // automatically tracks camera -- FPS effect
	tcg.SetName(xyz.TrackCameraName)
	xyz.NewSolid(tcg).SetMesh(cbm).
		SetScale(.1, .1, 1).SetPos(.5, -.5, -2.5). // in front of camera
		SetColor(color.RGBA{255, 0, 255, 128})

	///////////////////////////////////////////////////
	//  Animation & Embedded controls

	anim.Start(se, false) // start without animation running

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

	b.RunMainWindow()
}
