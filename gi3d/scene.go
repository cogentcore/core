// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"image/draw"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Set Update3DTrace to true to get a trace of 3D updating
var Update3DTrace = false

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own Framebuffer, which is then drawn directly onto the window.
type Scene struct {
	gi.WidgetBase
	Geom          gi.Geom2DInt        `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Camera        Camera              `desc:"camera determines view onto scene"`
	BgColor       gi.Color            `desc:"background color"`
	Lights        map[string]Light    `desc:"all lights used in the scene"`
	Meshes        map[string]Mesh     `desc:"all meshes used in the scene"`
	Textures      map[string]*Texture `desc:"all textures used in the scene"`
	NoNav         bool                `desc:"don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene"`
	Win           *gi.Window          `json:"-" xml:"-" desc:"our parent window that we render into"`
	Renders       Renderers           `view:"-" desc:"rendering programs"`
	Frame         gpu.Framebuffer     `view:"-" desc:"direct render target for scene"`
	Tex           gpu.Texture2D       `view:"-" desc:"the texture that the framebuffer returns, which should be rendered into the window"`
	SetDragCursor bool                `view:"-" desc:"has dragging cursor been set yet?"`
}

var KiT_Scene = kit.Types.AddType(&Scene{}, SceneProps)

// AddNewScene adds a new scene to given parent node, with given name.
func AddNewScene(parent ki.Ki, name string) *Scene {
	sc := parent.AddNewChild(KiT_Scene, name).(*Scene)
	sc.Defaults()
	return sc
}

// Defaults sets default scene params (camera, bg = white)
func (sc *Scene) Defaults() {
	sc.Camera.Defaults()
	sc.BgColor.SetUInt8(255, 255, 255, 255)
}

// AddMesh adds given mesh to mesh collection
// see AddNewX for convenience methods to add specific shapes
func (sc *Scene) AddMesh(ms Mesh) {
	if sc.Meshes == nil {
		sc.Meshes = make(map[string]Mesh)
	}
	sc.Meshes[ms.Name()] = ms
}

// AddLight adds given light to lights
// see AddNewX for convenience methods to add specific lights
func (sc *Scene) AddLight(lt Light) {
	if sc.Lights == nil {
		sc.Lights = make(map[string]Light)
	}
	sc.Lights[lt.Name()] = lt
}

// AddNewTexture adds a new texture of given name and filename
func (sc *Scene) AddNewTexture(name string, filename string) *Texture {
	if sc.Textures == nil {
		sc.Textures = make(map[string]*Texture)
	}
	tx := &Texture{Name: name, File: gi.FileName(filename)}
	sc.Textures[name] = tx
	return tx
}

// AddNewObject adds a new object of given name and mesh
func (sc *Scene) AddNewObject(name string, meshName string) *Object {
	obj := sc.AddNewChild(KiT_Object, name).(*Object)
	obj.Mesh = MeshName(meshName)
	obj.Defaults()
	return obj
}

// AddNewGroup adds a new group of given name and mesh
func (sc *Scene) AddNewGroup(name string) *Group {
	ngp := sc.AddNewChild(KiT_Group, name).(*Group)
	ngp.Defaults()
	return ngp
}

// DeleteUnusedMeshes deletes all unused meshes
func (sc *Scene) DeleteUnusedMeshes() {
	// used := make(map[string]struct{})
	// iterate over scene, add to used, then iterate over mats and if not used, delete.
}

// Validate traverses the scene and validates all the elements -- errors are logged
// and a non-nil return indicates that at least one error was found.
func (sc *Scene) Validate() error {
	// var errs []error // todo -- could do this
	// if err != nil {
	// 	*errs = append(*errs, err)
	// }
	hasError := false
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return true
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return false // going into a different type of thing, bail
		}
		if ni.IsInvisible() {
			return false
		}
		err := nii.Validate(sc)
		if err != nil {
			hasError = true
		}
		return true
	})
	if hasError {
		return fmt.Errorf("gi3d.Scene: %v Validate found at least one error (see log)", sc.PathUnique())
	}
	return nil
}

/////////////////////////////////////////////////////////////////////////////////////
//  Node2D Interface

func (sc *Scene) IsVisible() bool {
	if sc == nil || sc.This() == nil || sc.IsInvisible() || sc.Win == nil {
		return false
	}
	return sc.Win.IsVisible()
}

// set our window pointer to point to the current window we are under
func (sc *Scene) SetCurWin() {
	pwin := sc.ParentWindow()
	if pwin != nil { // only update if non-nil -- otherwise we could be setting
		// temporarily to give access to DPI etc
		sc.Win = pwin
	}
}

func (sc *Scene) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 || sc.Win == nil || !sc.Win.IsVisible() {
		return
	}
	if sc.Frame != nil {
		csz := sc.Frame.Size()
		if csz == nwsz {
			sc.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sc.Frame != nil {
		oswin.TheApp.RunOnMain(func() {
			sc.Win.OSWin.Activate()
			sc.Frame.SetSize(nwsz)
		})
	}
	sc.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.PathUnique(), nwsz, vp.Pixels.Bounds())
}

func (sc *Scene) Init2D() {
	sc.Init2DWidget()
	sc.SetCurWin()
	// we update ourselves whenever any node update event happens
	sc.NodeSig.Connect(sc.This(), func(recsc, sendk ki.Ki, sig int64, data interface{}) {
		rsci, _ := gi.KiToNode2D(recsc)
		rsc := rsci.(*Scene)
		if rsc.IsVisible() {
			if Update3DTrace {
				fmt.Printf("Update: Scene: %v full render due to signal: %v from node: %v\n", rsc.PathUnique(), ki.NodeSignals(sig), sendk.PathUnique())
			}
			rsc.Render()
			rsc.Win.DirectUpdate(rsc)
		}
	})
	sc.Init3D()
	sc.Win.AddDirectUploader(sc)
}

func (sc *Scene) Style2D() {
	if !sc.NoNav {
		sc.SetCanFocusIfActive()
	}
	sc.SetCurWin()
	sc.Style2DWidget()
	sc.LayData.SetFromStyle(&sc.Sty.Layout) // also does reset
	sc.Init3D()                             // todo: is this needed??
}

func (sc *Scene) Size2D(iter int) {
	sc.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := sc.Sty.Layout.PosDots().ToPoint()
	if pos != image.ZP {
		sc.Geom.Pos = pos
	}
}

func (sc *Scene) Layout2D(parBBox image.Rectangle, iter int) bool {
	sc.Layout2DBase(parBBox, true, iter)
	return false
}

func (sc *Scene) BBox2D() image.Rectangle {
	bb := sc.BBoxFromAlloc()
	sz := bb.Size()
	if sz != image.ZP {
		sc.Resize(sz)
	} else {
		bb.Max = bb.Min.Add(image.Point{64, 64}) // min size for zero case
	}
	return bb
}

func (sc *Scene) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	if sc.Viewport != nil {
		sc.ComputeBBox2DBase(parBBox, delta)
	}
	sc.Geom.Pos = sc.LayData.AllocPos.ToPointFloor()
}

func (sc *Scene) ChildrenBBox2D() image.Rectangle {
	return sc.Geom.Bounds()
}

// we use our own render for these -- Viewport member is our parent!
func (sc *Scene) PushBounds() bool {
	if sc.VpBBox.Empty() {
		return false
	}
	// if we are completely invisible, no point in rendering..
	if sc.Viewport != nil {
		wbi := sc.WinBBox.Intersect(sc.Viewport.WinBBox)
		if wbi.Empty() {
			fmt.Printf("not rendering sc %v bc empty winbox -- ours: %v par: %v\n", sc.Nm, sc.WinBBox, sc.Viewport.WinBBox)
			return false
		}
	}
	bb := sc.Geom.Bounds()
	// rs := &sc.Render
	// rs.PushBounds(bb)
	if gi.Render2DTrace {
		fmt.Printf("Render: %v at %v\n", sc.PathUnique(), bb)
	}
	return true
}

func (sc *Scene) PopBounds() {
	// rs := &vp.Render
	// rs.PopBounds()
}

func (sc *Scene) Move2D(delta image.Point, parBBox image.Rectangle) {
	if sc == nil {
		return
	}
	sc.Move2DBase(delta, parBBox)
	// sc.Move2DChildren(image.ZP) // reset delta here -- we absorb the delta in our placement relative to the parent
}

func (sc *Scene) Render2D() {
	if sc.PushBounds() {
		if !sc.NoNav {
			sc.NavEvents()
		}
		sc.Render()
		sc.PopBounds()
	}
}

// NavEvents handles standard viewer navigation events
func (sc *Scene) NavEvents() {
	sc.ConnectEvent(oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		ssc := recv.Embed(KiT_Scene).(*Scene)
		if ssc.IsDragging() {
			orbDel := float32(.2)
			panDel := float32(.05)
			if !ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Push(cursor.HandOpen)
				ssc.SetDragCursor = true
			}
			del := me.Where.Sub(me.From)
			dx := float32(del.X)
			dy := float32(del.Y)
			switch {
			case key.HasAllModifierBits(me.Modifiers, key.Shift):
				ssc.Camera.Pan(dx*panDel, -dy*panDel)
			case key.HasAllModifierBits(me.Modifiers, key.Control):
				ssc.Camera.PanAxis(dx*panDel, -dy*panDel)
			case key.HasAllModifierBits(me.Modifiers, key.Alt):
				ssc.Camera.PanTarget(dx*panDel, -dy*panDel, 0)
			default:
				if mat32.Abs(dx) > mat32.Abs(dy) {
					dy = 0
				} else {
					dx = 0
				}
				ssc.Camera.Orbit(dx*orbDel, -dy*orbDel)
			}
			ssc.UpdateSig()
		} else {
			if ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Pop()
				ssc.SetDragCursor = false
			}
		}
	})
	sc.ConnectEvent(oswin.MouseScrollEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		ssc := recv.Embed(KiT_Scene).(*Scene)
		if ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Pop()
			ssc.SetDragCursor = false
		}
		zoom := float32(me.NonZeroDelta(false))
		zoomPct := float32(.05)
		zoomDel := float32(.05)
		switch {
		case key.HasAllModifierBits(me.Modifiers, key.Alt):
			ssc.Camera.PanTarget(0, 0, zoom*zoomDel)
		default:
			ssc.Camera.Zoom(zoomPct * zoom)
		}
		ssc.UpdateSig()
	})
	sc.ConnectEvent(oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		me.SetProcessed()
		ssc := recv.Embed(KiT_Scene).(*Scene)
		if ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Pop()
			ssc.SetDragCursor = false
		}
		if !ssc.IsInactive() && !ssc.HasFocus() {
			ssc.GrabFocus()
		}
		// obj := ssc.FirstContainingPoint(me.Where, true)
		// if me.Action == mouse.Release && me.Button == mouse.Right {
		// 	me.SetProcessed()
		// 	if obj != nil {
		// 		giv.StructViewDialog(ssc.Viewport, obj, giv.DlgOpts{Title: "sc Element View"}, nil, nil)
		// 	}
		// }
	})
	sc.ConnectEvent(oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		me.SetProcessed()
		// ssc := recv.Embed(KiT_Scene).(*Scene)
		// obj := ssc.FirstContainingPoint(me.Where, true)
		// if obj != nil {
		// 	pos := me.Where
		// 	ttxt := fmt.Sprintf("element name: %v -- use right mouse click to edit", obj.Name())
		// 	gi.PopupTooltip(obj.Name(), pos.X, pos.Y, sc.Viewport, ttxt)
		// }
	})
	sc.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ssc := recv.Embed(KiT_Scene).(*Scene)
		kt := d.(*key.ChordEvent)
		ssc.NavKeyEvents(kt)
	})
}

// NavKeyEvents handles standard viewer keyboard navigation events
func (sc *Scene) NavKeyEvents(kt *key.ChordEvent) {
	ch := string(kt.Chord())
	// fmt.Printf(ch)
	orbDeg := float32(5)
	panDel := float32(.1)
	zoomPct := float32(.1)
	switch ch {
	case "UpArrow":
		sc.Camera.Orbit(0, orbDeg)
		kt.SetProcessed()
	case "Shift+UpArrow":
		sc.Camera.Pan(0, panDel)
		kt.SetProcessed()
	case "Control+UpArrow":
		sc.Camera.PanAxis(0, panDel)
		kt.SetProcessed()
	case "Alt+UpArrow":
		sc.Camera.PanTarget(0, panDel, 0)
		kt.SetProcessed()
	case "DownArrow":
		sc.Camera.Orbit(0, -orbDeg)
		kt.SetProcessed()
	case "Shift+DownArrow":
		sc.Camera.Pan(0, -panDel)
		kt.SetProcessed()
	case "Control+DownArrow":
		sc.Camera.PanAxis(0, -panDel)
		kt.SetProcessed()
	case "Alt+DownArrow":
		sc.Camera.PanTarget(0, -panDel, 0)
		kt.SetProcessed()
	case "LeftArrow":
		sc.Camera.Orbit(-orbDeg, 0)
		kt.SetProcessed()
	case "Shift+LeftArrow":
		sc.Camera.Pan(-panDel, 0)
		kt.SetProcessed()
	case "Control+LeftArrow":
		sc.Camera.PanAxis(-panDel, 0)
		kt.SetProcessed()
	case "Alt+LeftArrow":
		sc.Camera.PanTarget(-panDel, 0, 0)
		kt.SetProcessed()
	case "RightArrow":
		sc.Camera.Orbit(orbDeg, 0)
		kt.SetProcessed()
	case "Shift+RightArrow":
		sc.Camera.Pan(panDel, 0)
		kt.SetProcessed()
	case "Control+RightArrow":
		sc.Camera.PanAxis(panDel, 0)
		kt.SetProcessed()
	case "Alt+RightArrow":
		sc.Camera.PanTarget(panDel, 0, 0)
		kt.SetProcessed()
	case "Alt++", "Alt+=":
		sc.Camera.PanTarget(0, 0, panDel)
		kt.SetProcessed()
	case "Alt+-", "Alt+_":
		sc.Camera.PanTarget(0, 0, -panDel)
		kt.SetProcessed()
	case "+", "=":
		sc.Camera.Zoom(-zoomPct)
		kt.SetProcessed()
	case "-", "_":
		sc.Camera.Zoom(zoomPct)
		kt.SetProcessed()
	case " ":
		sc.Camera.DefaultPose()
		kt.SetProcessed()
	}
	sc.UpdateSig()
}

/////////////////////////////////////////////////////////////////////////////////////
// 		Rendering

// ActivateWin activates the window context for GPU rendering context (on the
// main thread -- all GPU rendering actions must be performed on main thread)
// returns false if not possible (i.e., Win nil)
func (sc *Scene) ActivateWin() bool {
	if sc.Win == nil {
		return false
	}
	oswin.TheApp.RunOnMain(func() {
		sc.Win.OSWin.Activate()
	})
	return true
}

// ActivateFrame creates (if necc) and activates framebuffer for GPU rendering context
// returns false if not possible
func (sc *Scene) ActivateFrame() bool {
	if !sc.ActivateWin() {
		log.Printf("gi3d.Scene: %s not able to activate window\n", sc.PathUnique())
		return false
	}
	oswin.TheApp.RunOnMain(func() {
		if sc.Frame == nil {
			msamp := 4
			if !gi.Prefs.Smooth3D {
				msamp = 0
			}
			sc.Frame = gpu.TheGPU.NewFramebuffer(sc.Nm+"-frame", sc.Geom.Size, msamp)
		}
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
		sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
		// fmt.Printf("aspect ratio: %v\n", sc.Camera.Aspect)
		sc.Frame.Activate()
		sc.Renders.DrawState()
		clr := ColorToVec3f(sc.BgColor)
		gpu.Draw.ClearColor(clr.X, clr.Y, clr.Z)
		gpu.Draw.Clear(true, true) // clear color and depth
	})
	return true
}

// InitTextures opens all the textures if not already opened, and establishes
// the GPU resources for them.  Must be called with context on main thread.
func (sc *Scene) InitTextures() bool {
	// todo
	return true
}

// InitMeshes makes sure all the Meshes are ready for rendering
// Must be called on main thread with context
func (sc *Scene) InitMeshes() bool {
	for _, ms := range sc.Meshes {
		ms.Make()
		ms.MakeVectors(sc)
		ms.Activate(sc)
		ms.TransferAll()
	}
	return true
}

// UpdateMeshes makes sure all the Meshes are ready for rendering
// This is the version for external use -- can be called on any thread
func (sc *Scene) UpdateMeshes() {
	if !sc.ActivateWin() {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		sc.InitMeshes()
	})
}

// UpdateWorldMatrix updates the world matrix for all scene elements
// called during Init3D and subsequent updates are triggered by local
// update signals on each node
func (sc *Scene) UpdateWorldMatrix() {
	idmtx := mat32.NewMat4()
	for _, kid := range sc.Kids {
		nii, _ := KiToNode3D(kid)
		if nii != nil {
			nii.UpdateWorldMatrix(idmtx)
			nii.UpdateWorldMatrixChildren()
		}
	}
}

func (sc *Scene) Init3D() {
	sc.UpdateWorldMatrix()
	if !sc.ActivateWin() {
		return
	}
	_, err := sc.Renders.Init(sc)
	if err != nil {
		log.Println(err)
	}
	oswin.TheApp.RunOnMain(func() {
		sc.InitTextures()
		sc.InitMeshes()
	})
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return true
		}
		nii, _ := KiToNode3D(k)
		if nii == nil {
			return false // going into a different type of thing, bail
		}
		nii.Init3D(sc)
		return true
	})
}

// Render renders the scene to the Frame framebuffer
func (sc *Scene) Render() bool {
	if !sc.ActivateFrame() {
		return false
	}
	sc.Camera.UpdateMatrix()
	oswin.TheApp.RunOnMain(func() {
		// sc.Win.OSWin.Activate() // render to screen..
		sc.Renders.SetLightsUnis(sc)
		gpu.TheGPU.ErrCheck("scene set light")
		sc.Render3D()
		gpu.TheGPU.ErrCheck("scene render3d")
		gpu.Draw.Flush()
		sc.Tex = sc.Frame.Texture()
	})
	return true
}

func (sc *Scene) IsDirectWinUpload() bool {
	return true
}

func (sc *Scene) DirectWinUpload() bool {
	if sc.Tex == nil || !sc.IsVisible() {
		return true
	}
	if Update3DTrace {
		fmt.Printf("Update: Window %s from Scene: %s at: %v, bounds: %v\n", sc.Win.Nm, sc.Nm, sc.WinBBox.Min, sc.Tex.Bounds())
	}

	// https://learnopengl.com/Advanced-Lighting/Gamma-Correction
	// todo: will need to mess with gamma correction -- can already see that colors
	// are too bright..  generate some good test inputs (just a bunch of greyscale cubes)
	// render in svg at top of screen too for comparison
	wt := sc.Win.OSWin.WinTex()
	oswin.TheApp.RunOnMain(func() {
		sc.Win.OSWin.Activate()
		// wt.Copy(sc.WinBBox.Min, sc.Tex, sc.Tex.Bounds(), draw.Over, &oswin.DrawOptions{FlipSrcY: true})
		wt.Copy(sc.WinBBox.Min, sc.Tex, sc.Tex.Bounds(), draw.Over, nil)
	})
	return true
}

// Render3D renders the scene to the framebuffer
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) Render3D() {
	var rcs [RenderClassesN][]*Object

	sc.Camera.Pose.UpdateMatrix()
	// Prepare for frustum culling
	proj := sc.Camera.PrjnMatrix.Mul(&sc.Camera.ViewMatrix)
	frustum := mat32.NewFrustumFromMatrix(proj)

	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return true
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return false // going into a different type of thing, bail
		}
		if ni.IsInvisible() {
			return false
		}
		if !nii.IsObject() {
			return true
		}
		obj := nii.AsObject()
		nii.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.PrjnMatrix)
		bba := nii.BBox()
		bb := bba.BBox
		bb.MulMat4(&ni.Pose.WorldMatrix)
		if true || frustum.IntersectsBox(bb) { // todo: remove true..
			rc := obj.RenderClass()
			rcs[rc] = append(rcs[rc], obj)
		}
		return true
	})

	// todo: zsort objects..  opaque front-to-back, trans back-to-front
	for rci, objs := range rcs {
		rc := RenderClasses(rci)
		if len(objs) == 0 {
			continue
		}
		var rnd Render
		switch rc {
		case RClassOpaqueUniform:
			rnd = sc.Renders.Renders["RenderUniformColor"]
			gpu.Draw.Op(draw.Src)
		case RClassTransUniform:
			rnd = sc.Renders.Renders["RenderUniformColor"]
			gpu.Draw.Op(draw.Over) // alpha
		case RClassTexture:
			rnd = sc.Renders.Renders["RenderTexture"]
			gpu.Draw.Op(draw.Src)
		case RClassOpaqueVertex:
			rnd = sc.Renders.Renders["RenderVertexColor"]
			gpu.Draw.Op(draw.Src)
		case RClassTransVertex:
			rnd = sc.Renders.Renders["RenderVertexColor"]
			gpu.Draw.Op(draw.Over) // alpha
		}
		rnd.Activate(&sc.Renders) // use same program for all..
		for _, obj := range objs {
			obj.This().(Node3D).Render3D(sc, rc, rnd)
		}
	}
}

// SceneProps define the ToolBar and MenuBar for StructView
var SceneProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"UpdateSig", ki.Props{
			"label": "Update",
			"icon":  "update",
		}},
	},
}
