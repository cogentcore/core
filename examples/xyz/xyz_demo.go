// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"log"
	"math"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/xyzv"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/xyz"
	"goki.dev/xyz/examples/assets"
	_ "goki.dev/xyz/io/obj"

	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

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
	Ang float32 `inactive:"+"`

	// the time.Ticker for animating the scene
	Ticker *time.Ticker `view:"-"`

	// the scene
	Scene *xyzv.Scene3D

	// the torus
	Torus *xyz.Solid

	// the gopher
	Gopher *xyz.Group

	// original position
	TorusPosOrig mat32.Vec3

	// original position
	GopherPosOrig mat32.Vec3
}

// Start starts the animation ticker timer -- if on is true, then
// animation will actually start too.
func (an *Anim) Start(se *xyzv.Scene3D, on bool) {
	an.Scene = se
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
	se := &an.Scene.Scene
	torusi := se.ChildByName("torus", 0)
	if torusi == nil {
		return
	}
	an.Torus = torusi.(*xyz.Solid)
	an.TorusPosOrig = an.Torus.Pose.Pos

	ggp := se.ChildByName("go-group", 0)
	if ggp == nil {
		return
	}
	gophi := ggp.Child(1)
	if gophi == nil {
		return
	}
	an.Gopher = gophi.(*xyz.Group)
	an.GopherPosOrig = an.Gopher.Pose.Pos
}

// Animate
func (an *Anim) Animate() {
	for {
		if an.Ticker == nil || an.Scene.This() == nil {
			return
		}
		<-an.Ticker.C // wait for tick
		if !an.On || an.Scene.This() == nil || an.Scene.Is(ki.Deleted) || an.Torus == nil || an.Gopher == nil {
			continue
		}

		updt := an.Scene.UpdateStart3D()
		radius := float32(0.3)

		if an.DoTorus {
			tdx := radius * mat32.Cos(an.Ang)
			tdz := radius * mat32.Sin(an.Ang)
			tp := an.TorusPosOrig
			tp.X += tdx
			tp.Z += tdz
			an.Torus.SetPosePos(tp)
		}

		if an.DoGopher {
			gdx := 0.1 * radius * mat32.Cos(an.Ang+math.Pi)
			gdz := 0.1 * radius * mat32.Sin(an.Ang+math.Pi)
			gp := an.GopherPosOrig
			gp.X += gdx
			gp.Z += gdz
			an.Gopher.SetPosePos(gp)
		}

		an.Scene.UpdateEndUpdate3D(updt)
		an.Ang += an.Speed
	}
}

func app() {
	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.EventTrace = true
	// gi.WinEventTrace = true
	// xyz.Update3DTrace = true
	// gi.Update2DTrace = true
	// vgpu.Debug = true

	gi.SetAppName("xyz")
	gi.SetAppAbout(`This is a demo of the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/xyz/README.md">README</a> page for this example app has further info.</p>`)

	anim := &Anim{}
	b := gi.NewBody()

	gi.NewLabel(b).SetText(`This is a demonstration of the
<a href="https://goki.dev/gi/v2">GoGi</a> <i>3D</i> Framework<br>
See <a href="https://goki.dev/gi/v2/blob/master/examples/xyz/README.md">README</a> for detailed info and things to try.`).
		SetType(gi.LabelHeadlineSmall).Style(func(s *styles.Style) {
		s.Text.Align = styles.Center
		s.Text.AlignV = styles.Center
	})

	gi.NewButton(b).SetText("Toggle Anim").OnClick(func(e events.Event) {
		anim.On = !anim.On
	})

	s3 := xyzv.NewScene3D(b)
	se := &s3.Scene

	// options - must be set here
	// sc.MultiSample = 1
	// se.Wireframe = true
	// sc.NoNav = true

	// first, add lights, set camera
	se.BackgroundColor = colors.FromRGB(230, 230, 255) // sky blue-ish
	xyz.NewAmbientLight(se, "ambient", 0.3, xyz.DirectSun)

	// se.Camera.Pose.Pos.Set(-2, 9, 3)
	se.Camera.Pose.Pos.Set(-2, 2, 10)
	// se.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	se.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	dir := xyz.NewDirLight(se, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := xyz.NewPointLight(sc, "point", 1, xyz.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := xyz.NewSpotLight(sc, "spot", 1, xyz.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	grtx := xyz.NewTextureFileFS(assets.Content, se, "ground", "ground.png")
	// _ = grtx
	// wdtx := xyz.NewTextureFile(sc, "wood", "wood.png")

	cbm := xyz.NewBox(se, "cube1", 1, 1, 1)
	cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := xyz.NewGroup(se, "r-b-group")

	xyz.NewSolid(rbgp, "red-cube").SetMesh(cbm).
		SetColor(colors.Red).SetShiny(500).SetPos(-1, 0, 0)

	bcb := xyz.NewSolid(rbgp, "blue-cube").SetMesh(cbm).
		SetColor(colors.Blue).SetShiny(10).SetReflective(0.2).
		SetPos(1, 1, 0)
	bcb.Pose.Scale.X = 2

	// alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!
	xyz.NewSolid(rbgp, "green-trans-cube").SetMesh(cbm).
		SetColor(color.RGBA{0, 255, 0, 128}).SetShiny(20).SetPos(0, 0, 1)

	floorp := xyz.NewPlane(se, "floor-plane", 100, 100)
	floor := xyz.NewSolid(se, "floor").SetMesh(floorp).
		SetColor(colors.Tan).SetTexture(grtx).SetPos(0, -5, 0)
	floor.Mat.Tiling.Repeat.Set(40, 40)

	// floor.Mat.Emissive.SetName("brown")
	// floor.Mat.Bright = 2 // .5 for wood / brown
	// floor.SetDisabled() // not selectable

	lnsm := xyz.NewLines(se, "Lines", []mat32.Vec3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, mat32.Vec2{.2, .1}, xyz.CloseLines)
	lns := xyz.NewSolid(se, "hi-line").SetMesh(lnsm).SetColor(color.RGBA{255, 255, 0, 128})
	lns.Pose.Pos.Set(0, 0, 1)

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	cyan := colors.FromRGB(0, 255, 255)
	xyz.NewArrow(se, se, "arrow", mat32.Vec3{-1.5, -.5, .5}, mat32.Vec3{2, 1, 1}, .05, cyan, xyz.StartArrow, xyz.EndArrow, 4, .5, 4)

	// bbclr := styles.Color{}
	// bbclr.SetUInt8(255, 255, 0, 255)
	// xyz.NewLineBox(sc, sc, "bbox", "bbox", mat32.Box3{Min: mat32.Vec3{-2, -2, -1}, Max: mat32.Vec3{-1, -1, .5}}, .01, bbclr, xyz.Active)

	cylm := xyz.NewCylinder(se, "cylinder", 1.5, .5, 32, 1, true, true)
	xyz.NewSolid(se, "cylinder").SetMesh(cylm).SetPos(-2.25, 0, 0)

	capm := xyz.NewCapsule(se, "capsule", 1.5, .5, 32, 1)
	xyz.NewSolid(se, "capsule").SetMesh(capm).SetColor(colors.Tan).
		SetPos(3.25, 0, 0)

	sphm := xyz.NewSphere(se, "sphere", .75, 32)
	sph := xyz.NewSolid(se, "sphere").SetMesh(sphm).SetColor(colors.Orange)
	sph.Mat.Color.A = 200
	sph.Pose.Pos.Set(0, -2, 0)

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo, err := se.OpenToLibraryFS(assets.Content, "gopher.obj", "")
	if err != nil {
		log.Println(err)
	}
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	gogp := xyz.NewGroup(se, "go-group")

	bgo, _ := se.AddFmLibrary("gopher", gogp)
	bgo.SetScale(.5, .5, .5).SetPos(1.4, -2.5, 0).SetAxisRotation(0, 1, 0, -160)

	sgo, _ := se.AddFmLibrary("gopher", gogp)
	sgo.SetPos(-1.5, -2, 0).SetScale(.2, .2, .2)

	trsm := xyz.NewTorus(se, "torus", .75, .1, 32)
	trs := xyz.NewSolid(se, "torus").SetMesh(trsm).SetColor(colors.White).
		SetPos(-1.6, -1.6, -.2).SetAxisRotation(1, 0, 0, 90)
	trs.Mat.Color.A = 200

	txt := xyz.NewText2D(se, "text").SetText("Text2D can put <b>HTML</b> formatted<br>Text anywhere you might <i>want</i>")
	// txt.Styles.BackgroundColor.SetSolid(colors.White)
	txt.Styles.Color = colors.Black
	txt.Styles.Text.Align = styles.Start
	txt.Pose.Scale.SetScalar(0.2)
	txt.SetPos(0, 2.2, 0)

	tcg := xyz.NewGroup(se, xyz.TrackCameraName) // automatically tracks camera -- FPS effect
	xyz.NewSolid(tcg, "first-person-gun").SetMesh(cbm).
		SetScale(.1, .1, 1).SetPos(.5, -.5, -2.5). // in front of camera
		SetColor(color.RGBA{255, 0, 255, 128})

	///////////////////////////////////////////////////
	//  Animation & Embedded controls

	anim.Start(s3, false) // start without animation running

	/*

		emb := xyz.NewEmbed2D(sc, sc, "embed-but", 150, 100, xyz.FitContent)
		emb.Pose.Pos.Set(-2, 2, 0)
		// emb.Zoom = 1.5   // this is how to rescale overall size
		evlay := gi.NewFrame(emb.Viewport, "vlay", gi.LayoutVert)
		evlay.SetProp("margin", units.Ex(1))

		eabut := gi.NewCheckBox(evlay, "anim-but")
		eabut.SetText("Animate")
		eabut.Tooltip = "toggle animation on and off"
		eabut.ButtonSig.Connect(win.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.ButtonToggled) {
				anim.On = eabut.IsChecked()
			}
		})

		cmb := gi.NewButton(evlay, "anim-ctrl")
		cmb.SetText("Anim Ctrl")
		cmb.Tooltip = "options for what is animated (note: menu only works when not animating -- checkboxes would be more useful here but wanted to test menu function)"
		cmb.Menu.AddAction(gi.ActOpts{Label: "Toggle Torus"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				anim.DoTorus = !anim.DoTorus
			})
		cmb.Menu.AddAction(gi.ActOpts{Label: "Toggle Gopher"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				anim.DoGopher = !anim.DoGopher
			})
		cmb.Menu.AddAction(gi.ActOpts{Label: "Edit Anim"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				giv.StructViewDialog(vp, anim, giv.DlgOpts{Title: "Animation Parameters"}, nil, nil)
			})

		sprw := gi.NewLayout(evlay, "speed-lay", gi.LayoutHoriz)
		gi.NewLabel(sprw, "speed-lbl", "Speed: ")
		sb := gi.NewSpinBox(sprw, "anim-speed")
		sb.SetMin(0.01)
		sb.Step = 0.01
		sb.SetValue(anim.Speed)
		sb.Tooltip = "determines the speed of rotation (step size)"

		spsld := gi.NewSlider(evlay, "speed-slider")
		spsld.Dim = mat32.X
		spsld.Min = 0.01
		spsld.Max = 1
		spsld.Step = 0.01
		spsld.PageStep = 0.1
		spsld.SetMinPrefWidth(units.Em(20))
		spsld.SetMinPrefHeight(units.Em(2))
		spsld.SetValue(anim.Speed)
		// spsld.Tracking = true
		spsld.Icon = icons.RadioButtonUnchecked

		sb.SpinBoxSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			anim.Speed = sb.Value
			spsld.SetValue(anim.Speed)
		})
		spsld.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			if gi.SliderSignals(sig) == gi.SliderValueChanged {
				anim.Speed = data.(float32)
				sb.SetValue(anim.Speed)
			}
		})
	*/

	b.NewWindow().Run().Wait()
}
