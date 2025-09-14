// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"fmt"
	"image"
	"math/rand"
	"os"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/physics"
	"cogentcore.org/core/xyz/physics/world"
	"cogentcore.org/core/xyz/xyzcore"
)

var NoGUI bool

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-nogui" {
		NoGUI = true
	}
	ev := &Env{}
	ev.Defaults()
	if NoGUI {
		ev.NoGUIRun()
		return
	}
	// core.RenderTrace = true
	b := ev.ConfigGUI()
	b.RunMainWindow()
}

// Env encapsulates the virtual environment
type Env struct { //types:add

	// if true, emer is angry: changes face color
	EmerAngry bool

	// height of emer
	EmerHt float32

	// how far to move every step
	MoveStep float32

	// how far to rotate every step
	RotStep float32

	// width of room
	Width float32

	// depth of room
	Depth float32

	// height of room
	Height float32

	// thickness of walls of room
	Thick float32

	// current depth map
	DepthVals []float32

	// offscreen render camera settings
	Camera world.Camera

	// color map to use for rendering depth map
	DepthMap core.ColorMapName

	// The whole physics World, including visualization.
	World *world.World

	// 3D visualization of the Scene
	SceneEditor *xyzcore.SceneEditor

	// emer object
	Emer *physics.Group `display:"-"`

	// Right eye of emer
	EyeR physics.Body `display:"-"`

	// contacts from last step, for body
	Contacts physics.Contacts `display:"-"`

	// snapshot image
	EyeRImg *core.Image `display:"-"`

	// depth map image
	DepthImage *core.Image `display:"-"`
}

func (ev *Env) Defaults() {
	ev.Width = 10
	ev.Depth = 15
	ev.Height = 2
	ev.Thick = 0.2
	ev.EmerHt = 1
	ev.MoveStep = ev.EmerHt * .2
	ev.RotStep = 15
	ev.DepthMap = core.ColorMapName("ColdHot")
	ev.Camera.Defaults()
	ev.Camera.FOV = 90
}

// MakePhysicsWorld constructs a new virtual physics world.
func (ev *Env) MakePhysicsWorld() *physics.Group {
	pw := physics.NewGroup()
	pw.SetName("RoomWorld")

	ev.MakeRoom(pw, "room1", ev.Width, ev.Depth, ev.Height, ev.Thick)
	ev.MakeEmer(pw, "emer", ev.EmerHt)
	pw.WorldInit()
	return pw
}

func (ev *Env) MakeWorld(sc *xyz.Scene) {
	pw := ev.MakePhysicsWorld()
	sc.Background = colors.Scheme.Select.Container
	xyz.NewAmbient(sc, "ambient", 0.3, xyz.DirectSun)

	dir := xyz.NewDirectional(sc, "dir", 1, xyz.DirectSun)
	dir.Pos.Set(0, 2, 1) // default: 0,1,1 = above and behind us (we are at 0,0,X)

	ev.World = world.NewWorld(pw, sc)
}

// InitWorld does init on world.
func (ev *Env) WorldInit() { //types:add
	ev.World.Init()
}

// ConfigView3D makes the 3D view
func (ev *Env) ConfigView3D(sc *xyz.Scene) {
	// sc.MultiSample = 1 // we are using depth grab so we need this = 1
}

// RenderEyeImg returns a snapshot from the perspective of Emer's right eye
func (ev *Env) RenderEyeImg() image.Image {
	return ev.World.RenderFromNode(ev.EyeR, &ev.Camera)
}

// GrabEyeImg takes a snapshot from the perspective of Emer's right eye
func (ev *Env) GrabEyeImg() { //types:add
	img := ev.RenderEyeImg()
	if img != nil {
		ev.EyeRImg.SetImage(img)
		ev.EyeRImg.NeedsRender()
	}
	// depth, err := ev.View3D.DepthImage()
	// if err == nil && depth != nil {
	// 	ev.DepthVals = depth
	// 	ev.ViewDepth(depth)
	// }
}

// ViewDepth updates depth bitmap with depth data
func (ev *Env) ViewDepth(depth []float32) {
	cmap := colormap.AvailableMaps[string(ev.DepthMap)]
	img := image.NewRGBA(image.Rectangle{Max: ev.Camera.Size})
	ev.DepthImage.SetImage(img)
	world.DepthImage(img, depth, cmap, &ev.Camera)
	ev.DepthImage.NeedsRender()
}

// UpdateView tells 3D view it needs to update.
func (ev *Env) UpdateView() {
	if ev.SceneEditor.IsVisible() {
		ev.SceneEditor.NeedsRender()
	}
}

// WorldStep does one step of the world
func (ev *Env) WorldStep() {
	pw := ev.World.World
	pw.Update() // only need to call if there are updaters added to world
	pw.WorldRelToAbs()
	cts := pw.WorldCollide(physics.DynsTopGps)
	ev.Contacts = nil
	for _, cl := range cts {
		if len(cl) > 1 {
			for _, c := range cl {
				if c.A.AsTree().Name == "body" {
					ev.Contacts = cl
				}
				fmt.Printf("A: %v  B: %v\n", c.A.AsTree().Name, c.B.AsTree().Name)
			}
		}
	}
	ev.EmerAngry = false
	if len(ev.Contacts) > 1 { // turn around
		ev.EmerAngry = true
		fmt.Printf("hit wall: turn around!\n")
		rot := 100.0 + 90.0*rand.Float32()
		ev.Emer.Rel.RotateOnAxis(0, 1, 0, rot)
	}
	ev.World.Update()
	ev.GrabEyeImg()
	ev.UpdateView()
}

// StepForward moves Emer forward in current facing direction one step, and takes GrabEyeImg
func (ev *Env) StepForward() { //types:add
	ev.Emer.Rel.MoveOnAxis(0, 0, 1, -ev.MoveStep)
	ev.WorldStep()
}

// StepBackward moves Emer backward in current facing direction one step, and takes GrabEyeImg
func (ev *Env) StepBackward() { //types:add
	ev.Emer.Rel.MoveOnAxis(0, 0, 1, ev.MoveStep)
	ev.WorldStep()
}

// RotBodyLeft rotates emer left and takes GrabEyeImg
func (ev *Env) RotBodyLeft() { //types:add
	ev.Emer.Rel.RotateOnAxis(0, 1, 0, ev.RotStep)
	ev.WorldStep()
}

// RotBodyRight rotates emer right and takes GrabEyeImg
func (ev *Env) RotBodyRight() { //types:add
	ev.Emer.Rel.RotateOnAxis(0, 1, 0, -ev.RotStep)
	ev.WorldStep()
}

// RotHeadLeft rotates emer left and takes GrabEyeImg
func (ev *Env) RotHeadLeft() { //types:add
	hd := ev.Emer.ChildByName("head", 1).(*physics.Group)
	hd.Rel.RotateOnAxis(0, 1, 0, ev.RotStep)
	ev.WorldStep()
}

// RotHeadRight rotates emer right and takes GrabEyeImg
func (ev *Env) RotHeadRight() { //types:add
	hd := ev.Emer.ChildByName("head", 1).(*physics.Group)
	hd.Rel.RotateOnAxis(0, 1, 0, -ev.RotStep)
	ev.WorldStep()
}

// MakeRoom constructs a new room in given parent group with given params
func (ev *Env) MakeRoom(par *physics.Group, name string, width, depth, height, thick float32) {
	tree.AddChildAt(par, name, func(rm *physics.Group) {
		rm.Maker(func(p *tree.Plan) {
			tree.AddAt(p, "floor", func(n *physics.Box) {
				n.SetSize(math32.Vec3(width, thick, depth)).
					SetColor("grey").SetInitPos(math32.Vec3(0, -thick/2, 0))
			})
			tree.AddAt(p, "back-wall", func(n *physics.Box) {
				n.SetSize(math32.Vec3(width, height, thick)).
					SetColor("blue").SetInitPos(math32.Vec3(0, height/2, -depth/2))
			})
			tree.AddAt(p, "left-wall", func(n *physics.Box) {
				n.SetSize(math32.Vec3(thick, height, depth)).
					SetColor("red").SetInitPos(math32.Vec3(-width/2, height/2, 0))
			})
			tree.AddAt(p, "right-wall", func(n *physics.Box) {
				n.SetSize(math32.Vec3(thick, height, depth)).
					SetColor("green").SetInitPos(math32.Vec3(width/2, height/2, 0))
			})
			tree.AddAt(p, "front-wall", func(n *physics.Box) {
				n.SetSize(math32.Vec3(width, height, thick)).
					SetColor("yellow").SetInitPos(math32.Vec3(0, height/2, depth/2))
			})
		})
	})
}

// MakeEmer constructs a new Emer virtual robot of given height (e.g., 1).
func (ev *Env) MakeEmer(par *physics.Group, name string, height float32) {
	tree.AddChildAt(par, name, func(emr *physics.Group) {
		ev.Emer = emr
		emr.Maker(func(p *tree.Plan) {
			width := height * .4
			depth := height * .15
			tree.AddAt(p, "body", func(n *physics.Box) {
				n.SetSize(math32.Vec3(width, height, depth)).
					SetColor("purple").SetDynamic(true).
					SetInitPos(math32.Vec3(0, height/2, 0))
			})
			// body := physics.NewCapsule(emr, "body", math32.Vec3(0, height / 2, 0), height, width/2)
			// body := physics.NewCylinder(emr, "body", math32.Vec3(0, height / 2, 0), height, width/2)

			headsz := depth * 1.5
			eyesz := headsz * .2
			hhsz := .5 * headsz
			tree.AddAt(p, "head", func(n *physics.Group) {
				n.SetInitPos(math32.Vec3(0, height+hhsz, 0))
				n.Maker(func(p *tree.Plan) {
					tree.AddAt(p, "head", func(n *physics.Box) {
						n.SetSize(math32.Vec3(headsz, headsz, headsz)).
							SetColor("tan").SetDynamic(true).SetInitPos(math32.Vec3(0, 0, 0))
						n.InitView = func(vn tree.Node) {
							sld := vn.(*xyz.Solid)
							world.BoxInit(n, sld)
							sld.Updater(func() {
								clr := n.Color
								if ev.EmerAngry {
									clr = "pink"
								}
								world.UpdateColor(clr, n.View.(*xyz.Solid))
							})
						}
					})
					tree.AddAt(p, "eye-l", func(n *physics.Box) {
						n.SetSize(math32.Vec3(eyesz, eyesz*.5, eyesz*.2)).
							SetColor("green").SetDynamic(true).
							SetInitPos(math32.Vec3(-hhsz*.6, headsz*.1, -(hhsz + eyesz*.3)))
					})
					tree.AddAt(p, "eye-r", func(n *physics.Box) {
						ev.EyeR = n
						n.SetSize(math32.Vec3(eyesz, eyesz*.5, eyesz*.2)).
							SetColor("green").SetDynamic(true).
							SetInitPos(math32.Vec3(hhsz*.6, headsz*.1, -(hhsz + eyesz*.3)))
					})
				})
			})
		})
	})
}

func (ev *Env) ConfigGUI() *core.Body {
	// vgpu.Debug = true

	b := core.NewBody("virtroom").SetTitle("Emergent Virtual Engine")
	split := core.NewSplits(b)

	tv := core.NewTree(core.NewFrame(split))
	sv := core.NewForm(split).SetStruct(ev)
	imfr := core.NewFrame(split)
	tbvw := core.NewTabs(split)
	scfr, _ := tbvw.NewTab("3D View")

	split.SetSplits(.1, .2, .2, .5)

	tv.OnSelect(func(e events.Event) {
		if len(tv.SelectedNodes) > 0 {
			sv.SetStruct(tv.SelectedNodes[0].AsCoreTree().SyncNode)
		}
	})

	////////    3D Scene

	etb := core.NewToolbar(scfr)
	ev.SceneEditor = xyzcore.NewSceneEditor(scfr)
	ev.SceneEditor.UpdateWidget()
	sc := ev.SceneEditor.SceneXYZ()
	ev.MakeWorld(sc)
	tv.SyncTree(ev.World.World)

	// local toolbar for manipulating emer
	etb.Maker(world.MakeStateToolbar(&ev.Emer.Rel, func() {
		ev.World.Update()
		ev.SceneEditor.NeedsRender()
	}))

	sc.Camera.Pose.Pos = math32.Vec3(0, 40, 3.5)
	sc.Camera.LookAt(math32.Vec3(0, 5, 0), math32.Vec3(0, 1, 0))
	sc.SaveCamera("3")

	sc.Camera.Pose.Pos = math32.Vec3(0, 20, 30)
	sc.Camera.LookAt(math32.Vec3(0, 5, 0), math32.Vec3(0, 1, 0))
	sc.SaveCamera("2")

	sc.Camera.Pose.Pos = math32.Vec3(-.86, .97, 2.7)
	sc.Camera.LookAt(math32.Vec3(0, .8, 0), math32.Vec3(0, 1, 0))
	sc.SaveCamera("1")
	sc.SaveCamera("default")

	////////    Image

	imfr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	core.NewText(imfr).SetText("Right Eye Image:")
	ev.EyeRImg = core.NewImage(imfr)
	ev.EyeRImg.SetName("eye-r-img")
	ev.EyeRImg.Image = image.NewRGBA(image.Rectangle{Max: ev.Camera.Size})

	core.NewText(imfr).SetText("Right Eye Depth:")
	ev.DepthImage = core.NewImage(imfr)
	ev.DepthImage.SetName("depth-img")
	ev.DepthImage.Image = image.NewRGBA(image.Rectangle{Max: ev.Camera.Size})

	////////    Toolbar

	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ev.MakeToolbar)
	})
	return b
}

func (ev *Env) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.WorldInit).SetText("Init").SetIcon(icons.Update)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.GrabEyeImg).SetText("Grab Image").SetIcon(icons.Image)
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.StepForward).SetText("Fwd").SetIcon(icons.SkipNext).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.StepBackward).SetText("Bkw").SetIcon(icons.SkipPrevious).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.RotBodyLeft).SetText("Body Left").SetIcon(icons.KeyboardArrowLeft).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.RotBodyRight).SetText("Body Right").SetIcon(icons.KeyboardArrowRight).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.RotHeadLeft).SetText("Head Left").SetIcon(icons.KeyboardArrowLeft).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(ev.RotHeadRight).SetText("Head Right").SetIcon(icons.KeyboardArrowRight).
			Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.RepeatClickable)
			})
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("README").SetIcon(icons.FileMarkdown).
			SetTooltip("Open browser on README.").
			OnClick(func(e events.Event) {
				core.TheApp.OpenURL("https://github.com/cogentcore/core/blob/master/xyz/examples/physics/README.md")
			})
	})
}

func (ev *Env) NoGUIRun() {
	gp, dev, err := gpu.NoDisplayGPU()
	if err != nil {
		panic(err)
	}
	sc := world.NoDisplayScene(gp, dev)
	ev.MakeWorld(sc)

	img := ev.RenderEyeImg()
	if img != nil {
		imagex.Save(img, "eyer_0.png")
	}
}
