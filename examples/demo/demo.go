// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"image/color"
	"log"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi3d"
	"goki.dev/gi3d/examples/assets"
	"goki.dev/gi3d/gi3dv"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/vgpu/v2/vgpu"

	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

/*
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
	Scene *gi3d.Scene

	// the torus
	Torus *gi3d.Solid

	// the gopher
	Gopher *gi3d.Group

	// original position
	TorusPosOrig mat32.Vec3

	// original position
	GopherPosOrig mat32.Vec3
}

// Start starts the animation ticker timer -- if on is true, then
// animation will actually start too.
func (an *Anim) Start(sc *gi3d.Scene, on bool) {
	an.Scene = sc
	an.On = on
	an.DoTorus = true
	an.DoGopher = true
	an.Speed = .1
	an.GetObjs()
	an.Ticker = time.NewTicker(10 * time.Millisecond) // 100 fps max
	go an.Animate()
}

// GetObjs gets the objects to animate
func (an *Anim) GetObjs() {
	torusi := an.Scene.ChildByName("torus", 0)
	if torusi == nil {
		return
	}
	an.Torus = torusi.(*gi3d.Solid)
	an.TorusPosOrig = an.Torus.Pose.Pos

	ggp := an.Scene.ChildByName("go-group", 0)
	if ggp == nil {
		return
	}
	gophi := ggp.Child(1)
	if gophi == nil {
		return
	}
	an.Gopher = gophi.(*gi3d.Group)
	an.GopherPosOrig = an.Gopher.Pose.Pos
}

// Animate
func (an *Anim) Animate() {
	for {
		if an.Ticker == nil || an.Scene == nil {
			return
		}
		<-an.Ticker.C // wait for tick
		if !an.On || an.Scene == nil || an.Torus == nil || an.Gopher == nil {
			continue
		}

		updt := an.Scene.UpdateStart()
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

		an.Scene.UpdateEnd(updt) // triggers re-render -- don't need a full Update() which updates meshes
		an.Ang += an.Speed
	}
}
*/

func app() {
	// turn these on to see a traces of various stages of processing..
	// ki.SignalTrace = true
	// gi.EventTrace = true
	// gi.WinEventTrace = true
	// gi3d.Update3DTrace = true
	// gi.Update2DTrace = true
	vgpu.Debug = true

	goosi.ZoomFactor = 1.5

	gi.SetAppName("gi3d")
	gi.SetAppAbout(`This is a demo of the 3D graphics aspect of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/gi3d/README.md">README</a> page for this example app has further info.</p>`)

	sc := gi.NewScene("gi3d-demo").SetTitle("Gi3D Demo")

	trow := gi.NewLayout(sc, "trow").SetLayout(gi.LayoutHoriz).
		Style(func(s *styles.Style) {
			s.SetStretchMaxWidth()
		})

	title := gi.NewLabel(trow, "title")
	title.SetText(`This is a demonstration of the
<a href="https://goki.dev/gi/v2">GoGi</a> <i>3D</i> Framework<br>
See <a href="https://goki.dev/gi/v2/blob/master/examples/gi3d/README.md">README</a> for detailed info and things to try.`)
	title.SetType(gi.LabelHeadlineSmall).
		SetStretchMax().
		Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNormal
			s.Text.Align = styles.AlignCenter
			s.Text.AlignV = styles.AlignCenter
			s.Font.Family = "Times New Roman, serif"
			// s.Font.Size = units.Dp(24) // todo: "x-large"?
			// s.Text.LineHeight = units.Em(1.5)
		})

	gi.NewSpace(sc, "scspc")
	scrow := gi.NewLayout(sc, "scrow").SetLayout(gi.LayoutHoriz)
	scrow.Style(func(s *styles.Style) {
		s.SetStretchMax()
	})

	s3 := gi3dv.NewScene3D(scrow, "scene")
	se := &s3.Scene

	// options - must be set here
	// sc.MultiSample = 1
	// se.Wireframe = true
	// sc.NoNav = true

	// first, add lights, set camera
	se.BackgroundColor = colors.FromRGB(230, 230, 255) // sky blue-ish
	gi3d.NewAmbientLight(se, "ambient", 0.3, gi3d.DirectSun)

	se.Camera.Pose.Pos.Set(3, 5, 10)              // default position
	se.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	dir := gi3d.NewDirLight(se, "dir", 1, gi3d.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	// point := gi3d.NewPointLight(sc, "point", 1, gi3d.DirectSun)
	// point.Pos.Set(0, 5, 5)

	// spot := gi3d.NewSpotLight(sc, "spot", 1, gi3d.DirectSun)
	// spot.Pose.Pos.Set(0, 5, 5)

	grtx := gi3d.NewTextureFileFS(assets.Content, se, "ground", "ground.png")
	// _ = grtx
	// wdtx := gi3d.NewTextureFile(sc, "wood", "wood.png")

	cbm := gi3d.NewBox(se, "cube1", 1, 1, 1)
	cbm.Segs.Set(10, 10, 10) // not clear if any diff really..

	rbgp := gi3d.NewGroup(se, "r-b-group")

	rcb := gi3d.NewSolid(rbgp, "red-cube").SetMesh(cbm)
	rcb.Pose.Pos.Set(-1, 0, 0)
	rcb.Mat.SetColor(colors.Red).SetShiny(500)

	bcb := gi3d.NewSolid(rbgp, "blue-cube").SetMesh(cbm)
	bcb.Pose.Pos.Set(1, 1, 0)
	bcb.Pose.Scale.X = 2 // somehow causing to not render
	bcb.Mat.SetColor(colors.Blue).SetShiny(10).SetReflective(0.2)

	gcb := gi3d.NewSolid(rbgp, "green-trans-cube").SetMesh(cbm)
	gcb.Pose.Pos.Set(0, 0, 1)
	gcb.Mat.SetColor(color.RGBA{0, 255, 0, 230}).SetShiny(30) // alpha = .5 -- note: colors are NOT premultiplied here: will become so when rendered!
	// fmt.Println(gcb.Mat)

	floorp := gi3d.NewPlane(se, "floor-plane", 100, 100)
	floor := gi3d.NewSolid(se, "floor").SetMesh(floorp)
	floor.Pose.Pos.Set(0, -5, 0)
	floor.Mat.Color = colors.Tan
	// floor.Mat.Emissive.SetName("brown")
	floor.Mat.Bright = 2 // .5 for wood / brown
	floor.Mat.SetTexture(grtx)
	floor.Mat.Tiling.Repeat.Set(40, 40)
	// floor.SetDisabled() // not selectable

	// sc.Config()
	// sc.UpdateNodes()
	if !se.Render() {
		log.Println("no render")
	}

	gi.NewWindow(sc).Run().Wait()

}

/*
	lnsm := gi3d.NewLines(sc, "Lines", []mat32.Vec3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, mat32.Vec2{.2, .1}, gi3d.CloseLines)
	lns := gi3d.NewSolid(sc, sc, "hi-line", lnsm.Name())
	lns.Pose.Pos.Set(0, 0, 1)
	lns.Mat.Color = color.RGBA{255, 255, 0, 128} // alpha = .5
	// sc.Wireframe = true                      // debugging

	// this line should go from lower left front of red cube to upper vertex of above hi-line
	cyan := colors.FromRGB(0, 255, 255)
	gi3d.NewArrow(sc, sc, "arrow", mat32.Vec3{-1.5, -.5, .5}, mat32.Vec3{2, 1, 1}, .05, cyan, gi3d.StartArrow, gi3d.EndArrow, 4, .5, 4)

	// bbclr := styles.Color{}
	// bbclr.SetUInt8(255, 255, 0, 255)
	// gi3d.NewLineBox(sc, sc, "bbox", "bbox", mat32.Box3{Min: mat32.Vec3{-2, -2, -1}, Max: mat32.Vec3{-1, -1, .5}}, .01, bbclr, gi3d.Active)

	cylm := gi3d.NewCylinder(sc, "cylinder", 1.5, .5, 32, 1, true, true)
	cyl := gi3d.NewSolid(sc, sc, "cylinder", cylm.Name())
	cyl.Pose.Pos.Set(-2.25, 0, 0)

	capm := gi3d.NewCapsule(sc, "capsule", 1.5, .5, 32, 1)
	caps := gi3d.NewSolid(sc, sc, "capsule", capm.Name())
	caps.Pose.Pos.Set(3.25, 0, 0)
	caps.Mat.Color = colors.Tan

	sphm := gi3d.NewSphere(sc, "sphere", .75, 32)
	sph := gi3d.NewSolid(sc, sc, "sphere", sphm.Name())
	sph.Pose.Pos.Set(0, -2, 0)
	sph.Mat.Color = colors.Orange
	sph.Mat.Color.A = 200

	// Good strategy for objects if used in multiple places is to load
	// into library, then add from there.
	lgo, err := sc.OpenToLibraryFS(content, "gopher.obj", "")
	if err != nil {
		log.Println(err)
	}
	lgo.Pose.SetAxisRotation(0, 1, 0, -90) // for all cases

	gogp := gi3d.NewGroup(sc, sc, "go-group")

	bgo, _ := sc.AddFmLibrary("gopher", gogp)
	bgo.Pose.Scale.Set(.5, .5, .5)
	bgo.Pose.Pos.Set(1.4, -2.5, 0)
	bgo.Pose.SetAxisRotation(0, 1, 0, -160)

	sgo, _ := sc.AddFmLibrary("gopher", gogp)
	sgo.Pose.Pos.Set(-1.5, -2, 0)
	sgo.Pose.Scale.Set(.2, .2, .2)

	trsm := gi3d.NewTorus(sc, "torus", .75, .1, 32)
	trs := gi3d.NewSolid(sc, sc, "torus", trsm.Name())
	trs.Pose.Pos.Set(-1.6, -1.6, -.2)
	trs.Pose.SetAxisRotation(1, 0, 0, 90)
	trs.Mat.Color = colors.White
	trs.Mat.Color.A = 200

	grtx := gi3d.NewTextureFileFS(content, sc, "ground", "ground.png")
	// _ = grtx
	// wdtx := gi3d.NewTextureFile(sc, "wood", "wood.png")

	floorp := gi3d.NewPlane(sc, "floor-plane", 100, 100)
	floor := gi3d.NewSolid(sc, sc, "floor", floorp.Name())
	floor.Pose.Pos.Set(0, -5, 0)
	floor.Mat.Color = colors.Tan
	// floor.Mat.Emissive.SetName("brown")
	floor.Mat.Bright = 2 // .5 for wood / brown
	floor.Mat.SetTexture(sc, grtx)
	floor.Mat.Tiling.Repeat.Set(40, 40)
	floor.SetDisabled() // not selectable

	txt := gi3d.NewText2D(sc, sc, "text", "Text2D can put <b>HTML</b> formatted<br>Text anywhere you might <i>want</i>")
	// 	txt.SetProp("background-color", styles.Color{0, 0, 0, 0}) // transparent -- default
	// txt.SetProp("background-color", "white")
	txt.SetProp("color", "black") // default depends on Light / Dark mode, so we set this
	// txt.SetProp("margin", units.NewPt(4)) // default is 2 px
	// txt.Mat.Bright = 5 // no dim text -- key if using a background and want it to be bright..
	txt.SetProp("text-align", styles.AlignLeft) // gi.AlignCenter)
	txt.Pose.Scale.SetScalar(0.2)
	txt.Pose.Pos.Set(0, 2.2, 0)

	tcg := gi3d.NewGroup(sc, sc, gi3d.TrackCameraName) // automatically tracks camera -- FPS effect
	fpgun := gi3d.NewSolid(sc, tcg, "first-person-gun", cbm.Name())
	fpgun.Pose.Scale.Set(.1, .1, 1)
	fpgun.Pose.Pos.Set(.5, -.5, -2.5)              // in front of camera
	fpgun.Mat.Color = color.RGBA{255, 0, 255, 128} // alpha = .5

	sc.Camera.Pose.Pos.Set(0, 0, 10)              // default position
	sc.Camera.LookAt(mat32.Vec3Zero, mat32.Vec3Y) // defaults to looking at origin

	///////////////////////////////////////////////////
	//  Animation & Embedded controls

	anim := &Anim{}

	emb := gi3d.NewEmbed2D(sc, sc, "embed-but", 150, 100, gi3d.FitContent)
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
