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
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own Framebuffer, which is then drawn directly onto the window.
type Scene struct {
	gi.WidgetBase
	Geom     gi.Geom2DInt        `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Win      *gi.Window          `json:"-" xml:"-" desc:"our parent window that we render into"`
	Camera   Camera              `desc:"camera determines view onto scene"`
	Lights   map[string]Light    `desc:"all lights used in the scene"`
	Meshes   map[string]Mesh     `desc:"all meshes used in the scene"`
	Textures map[string]*Texture `desc:"all textures used in the scene"`
	Rends    Renderers           `desc:"rendering programs"`
	Frame    gpu.Framebuffer     `view:"-" desc:"direct render target for scene"`
}

var KiT_Scene = kit.Types.AddType(&Scene{}, nil)

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
	return obj
}

// AddNewGroup adds a new group of given name and mesh
func (sc *Scene) AddNewGroup(name string) *Group {
	ngp := sc.AddNewChild(KiT_Group, name).(*Group)
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

func (sc *Scene) Init2D() {
	sc.Init2DWidget()
	sc.SetCurWin()
	// we update ourselves whenever any node update event happens
	sc.NodeSig.Connect(sc.This(), func(recsc, sendk ki.Ki, sig int64, data interface{}) {
		rsci, _ := gi.KiToNode2D(recsc)
		rsc := rsci.(*Scene)
		if gi.Update2DTrace {
			fmt.Printf("Update: Scene: %v full render due to signal: %v from node: %v\n", rsc.PathUnique(), ki.NodeSignals(sig), sendk.PathUnique())
		}
		if !sc.IsDeleted() && !sc.IsDestroyed() {
			sc.Render()
		}
	})
}

func (sc *Scene) Style2D() {
	sc.SetCurWin()
	sc.Style2DWidget()
	sc.LayData.SetFromStyle(&sc.Sty.Layout) // also does reset
}

func (sc *Scene) Size2D(iter int) {
	sc.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := sc.Sty.Layout.PosDots().ToPoint()
	if pos != image.ZP {
		sc.Geom.Pos = pos
	}
	if sc.Geom.Size != image.ZP {
		sc.LayData.AllocSize.SetPoint(sc.Geom.Size)
	}
}

func (sc *Scene) Layout2D(parBBox image.Rectangle, iter int) bool {
	sc.Layout2DBase(parBBox, true, iter)
	return sc.Layout2DChildren(iter)
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
		sc.Render()
		sc.PopBounds()
	}
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
			sc.Frame = gpu.TheGPU.NewFramebuffer(sc.Nm+"-frame", sc.Geom.Size, 4) // 4 samples default
		}
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
		sc.Frame.Activate()
		gpu.Draw.Clear(true, true) // clear color and depth
		gpu.Draw.DepthTest(true)
	})
	return true
}

// UploadToWin uploads our viewport image into the parent window -- e.g., called
// by popups when updating separately
func (sc *Scene) UploadToWin() {
	if sc.Win == nil {
		return
	}
	// todo: fix this!
	// 	sc.Win.Uploadsc(sc, sc.WinBBox.Min)
}

// OpenTextures opens all the textures if not already opened, and establishes
// the GPU resources for them.  Must be called with context on main thread.
func (sc *Scene) OpenTextures() bool {
	// todo
	return true
}

// PrepMeshes makes sure all the Meshes are ready for rendering
// called on main thread with context
func (sc *Scene) PrepMeshes() bool {
	for _, ms := range sc.Meshes {
		// todo: here is where we need some kind of dirty bit.
		// basic logic should be to only make if not made at all
		// updating from that point is job of user
		ms.Make()
		ms.MakeVectorsImpl(sc)
	}
	return true
}

// Render renders the scene to the Frame framebuffer
// Fully self-contained, including window update
func (sc *Scene) Render() bool {
	if !sc.ActivateFrame() {
		return false
	}
	_, err := sc.Rends.Init()
	if err != nil {
		return false
	}
	sc.Camera.UpdateMatrix()
	oswin.TheApp.RunOnMain(func() {
		sc.Rends.SetLightsUnis(sc)
		sc.OpenTextures()
		sc.PrepMeshes()
		sc.Render3D()
		sc.UploadToWin()
	})
	return true
}

// Render3D renders the scene to the framebuffer
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) Render3D() {
	opaque := make([]*Object, 100)
	trans := make([]*Object, 100)

	// Prepare for frustum culling
	var proj mat32.Mat4
	proj.MultiplyMatrices(&sc.Camera.PrjnMatrix, &sc.Camera.ViewMatrix)
	frustum := mat32.NewFrustumFromMatrix(&proj)

	nb.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
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
		bb.ApplyMat4(&ni.Pose.WorldMatrix)
		if true || frustum.IntersectsBox(&bb) { // todo: remove true..
			if nii.IsTransparent() {
				trans = append(trans, obj)
			} else {
				opaque = append(opaque, obj)
			}
		}
		return true
	})

	// todo: zsort objects?
	gpu.Draw.Op(draw.Src)
	for _, obj := range opaque {
		obj.This().(Node3D).Render3D(sc)
	}
	gpu.Draw.Op(draw.Over) // alpha
	for _, obj := range trans {
		obj.This().(Node3D).Render3D(sc)
	}
}
