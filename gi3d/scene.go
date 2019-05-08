// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"strings"

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
// It renders to its own Framebuffer, the Texture of which is then drawn
// directly onto the window WinTex using the DirectWinUpload protocol.
//
// There is default navigation event processing (disabled by setting NoNav)
// where mouse drag events Orbit the camera (Shift = Pan, Alt = PanTarget)
// and arrow keys do Orbit, Pan, PanTarget with same key modifiers.
// Spacebar restores original "default" camera, and numbers save (1st time)
// or restore (subsequently) camera views (Control = always save)
//
// A Group at the top-level named "TrackCamera" will automatically track
// the camera (i.e., its Pose is copied) -- Objects in that group can
// set their relative Pos etc to display relative to the camera, to achieve
// "first person" effects.
type Scene struct {
	gi.WidgetBase
	Geom          gi.Geom2DInt       `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Camera        Camera             `desc:"camera determines view onto scene"`
	BgColor       gi.Color           `desc:"background color"`
	Lights        map[string]Light   `desc:"all lights used in the scene"`
	Meshes        map[string]Mesh    `desc:"all meshes used in the scene"`
	Textures      map[string]Texture `desc:"all textures used in the scene"`
	NoNav         bool               `desc:"don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene"`
	SavedCams     map[string]Camera  `desc:"saved cameras -- can Save and Set these to view the scene from different angles"`
	Win           *gi.Window         `json:"-" xml:"-" desc:"our parent window that we render into"`
	Renders       Renderers          `view:"-" desc:"rendering programs"`
	Frame         gpu.Framebuffer    `view:"-" desc:"direct render target for scene"`
	Tex           gpu.Texture2D      `view:"-" desc:"the texture that the framebuffer returns, which should be rendered into the window"`
	SetDragCursor bool               `view:"-" desc:"has dragging cursor been set yet?"`
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

// AddTexture adds given texture to texture collection
// see AddNewTextureFile to add a texture that loads from file
func (sc *Scene) AddTexture(tx Texture) {
	if sc.Textures == nil {
		sc.Textures = make(map[string]Texture)
	}
	sc.Textures[tx.Name()] = tx
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

// SaveCamera saves the current camera with given name -- can be restored later with SetCamera.
// "default" is a special name that is automatically saved on first render, and
// restored with the spacebar under default NavEvents.
// Numbered cameras 0-9 also saved / restored with corresponding keys.
func (sc *Scene) SaveCamera(name string) {
	if sc.SavedCams == nil {
		sc.SavedCams = make(map[string]Camera)
	}
	sc.SavedCams[name] = sc.Camera
}

// SetCamera sets the current camera to that of given name -- error if not found.
// "default" is a special name that is automatically saved on first render, and
// restored with the spacebar under default NavEvents.
// Numbered cameras 0-9 also saved / restored with corresponding keys.
func (sc *Scene) SetCamera(name string) error {
	cam, ok := sc.SavedCams[name]
	if !ok {
		return fmt.Errorf("gi3d.Scene: %v saved camera of name: %v not found", name)
	}
	sc.Camera = cam
	return nil
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
	if sc.Par == nil || sc.Par.This() == nil {
		return false
	}
	return sc.Par.This().(gi.Node2D).IsVisible()
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
	// note: Viewport will automatically update us for any update sigs
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
	if !sc.This().(gi.Node2D).IsVisible() {
		return false
	}
	// if we are completely invisible, no point in rendering..
	if sc.Viewport != nil {
		wbi := sc.WinBBox.Intersect(sc.Viewport.WinBBox)
		if wbi.Empty() {
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
		if gi.Render2DTrace {
			fmt.Printf("3D Render2D: %v\n", sc.PathUnique())
		}
		sc.Render()
		sc.PopBounds()
	} else {
		sc.DisconnectAllEvents(gi.RegPri)
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
			panDel := float32(.01)
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
				ssc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
			}
			ssc.UpdateSig()
		} else {
			if ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Pop()
				ssc.SetDragCursor = false
			}
		}
	})
	// sc.ConnectEvent(oswin.MouseMoveEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
	// 	me := d.(*mouse.MoveEvent)
	// 	me.SetProcessed()
	// 	ssc := recv.Embed(KiT_Scene).(*Scene)
	// 	orbDel := float32(.2)
	// 	panDel := float32(.05)
	// 	del := me.Where.Sub(me.From)
	// 	dx := float32(del.X)
	// 	dy := float32(del.Y)
	// 	switch {
	// 	case key.HasAllModifierBits(me.Modifiers, key.Shift):
	// 		ssc.Camera.Pan(dx*panDel, -dy*panDel)
	// 	case key.HasAllModifierBits(me.Modifiers, key.Control):
	// 		ssc.Camera.PanAxis(dx*panDel, -dy*panDel)
	// 	case key.HasAllModifierBits(me.Modifiers, key.Alt):
	// 		ssc.Camera.PanTarget(dx*panDel, -dy*panDel, 0)
	// 	default:
	// 		if mat32.Abs(dx) > mat32.Abs(dy) {
	// 			dy = 0
	// 		} else {
	// 			dx = 0
	// 		}
	// 		ssc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
	// 	}
	// 	ssc.UpdateSig()
	// })
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
	panDel := float32(.2)
	zoomPct := float32(.05)
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
		sc.Camera.Orbit(orbDeg, 0)
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
		sc.Camera.Orbit(-orbDeg, 0)
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
	case "+", "=", "Shift+=":
		sc.Camera.Zoom(-zoomPct)
		kt.SetProcessed()
	case "-", "_", "Shift+-":
		sc.Camera.Zoom(zoomPct)
		kt.SetProcessed()
	case " ", "Escape":
		err := sc.SetCamera("default")
		if err != nil {
			sc.Camera.DefaultPose()
		}
		kt.SetProcessed()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		err := sc.SetCamera(ch)
		if err != nil {
			sc.SaveCamera(ch)
			fmt.Printf("Saved camera to: %v\n", ch)
		} else {
			fmt.Printf("Restored camera from: %v\n", ch)
		}
		kt.SetProcessed()
	case "Control+0", "Control+1", "Control+2", "Control+3", "Control+4", "Control+5", "Control+6", "Control+7", "Control+8", "Control+9":
		cnm := strings.TrimPrefix(ch, "Control+")
		sc.SaveCamera(cnm)
		fmt.Printf("Saved camera to: %v\n", cnm)
		kt.SetProcessed()
	case "t":
		kt.SetProcessed()
		obj := sc.Child(0).(*Object)
		fmt.Printf("updated obj: %v\n", obj.PathUnique())
		obj.UpdateSig()
		return
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

// InitTexturesInCtxt opens all the textures if not already opened, and establishes
// the GPU resources for them.  Must be called with context on main thread.
func (sc *Scene) InitTexturesInCtxt() bool {
	for _, tx := range sc.Textures {
		tx.Init(sc)
	}
	return true
}

// InitTextures opens all the textures if not already opened, and establishes
// the GPU resources for them.  This version can be called externally and
// activates the context and runs on main thread
func (sc *Scene) InitTextures() bool {
	if !sc.ActivateWin() {
		return true
	}
	oswin.TheApp.RunOnMain(func() {
		sc.InitTexturesInCtxt()
	})
	return true
}

// InitMeshesInCtxt does a full init and gpu transfer of all the meshes
// This version must be called on main thread with context
func (sc *Scene) InitMeshesInCtxt() bool {
	for _, ms := range sc.Meshes {
		ms.Make(sc)
		ms.MakeVectors(sc)
		ms.Activate(sc)
		ms.TransferAll()
	}
	return true
}

// InitMeshes does a full init and gpu transfer of all the meshes
// This version us to be used by external users -- sets context and runs on main
func (sc *Scene) InitMeshes() {
	if !sc.ActivateWin() {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		sc.InitMeshesInCtxt()
	})
}

// UpdateMeshesInCtxt calls Update on all the meshes in context on main thread
// Update is responsible for doing any transfers
func (sc *Scene) UpdateMeshesInCtxt() bool {
	for _, ms := range sc.Meshes {
		ms.Update(sc)
	}
	return true
}

// UpdateMeshes calls Update on all meshes
// This version us to be used by external users -- sets context and runs on main
func (sc *Scene) UpdateMeshes() {
	if !sc.ActivateWin() {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		sc.UpdateMeshesInCtxt()
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
		sc.InitTexturesInCtxt()
		sc.InitMeshesInCtxt()
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
	if len(sc.SavedCams) == 0 {
		sc.SaveCamera("default")
	}
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateWorldMatrix() // inexpensive -- just do it to be sure..
	oswin.TheApp.RunOnMain(func() {
		sc.Renders.SetLightsUnis(sc)
		sc.Render3D()
		gpu.Draw.Flush()
		sc.Tex = sc.Frame.Texture()
	})
	return true
}

func (sc *Scene) IsDirectWinUpload() bool {
	return true
}

func (sc *Scene) DirectWinUpload() bool {
	if !sc.IsVisible() {
		return true
	}
	if Update3DTrace {
		fmt.Printf("3D Update: Window %s from Scene: %s at: %v\n", sc.Win.Nm, sc.Nm, sc.WinBBox.Min)
	}

	sc.Render()

	// https://learnopengl.com/Advanced-Lighting/Gamma-Correction
	// todo: will need to mess with gamma correction -- can already see that colors
	// are too bright..  generate some good test inputs (just a bunch of greyscale cubes)
	// render in svg at top of screen too for comparison
	wt := sc.Win.OSWin.WinTex()
	oswin.TheApp.RunOnMain(func() {
		sc.Win.OSWin.Activate()
		// wt.Copy(sc.WinBBox.Min, sc.Tex, sc.Tex.Bounds(), draw.Src, &oswin.DrawOptions{FlipSrcY: true})
		wt.Copy(sc.WinBBox.Min, sc.Tex, sc.Tex.Bounds(), draw.Src, nil)
	})
	sc.Win.UpdateSig() // trigger publish
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

// TrackCamera -- a Group at the top-level named "TrackCamera"
// will automatically track the camera (i.e., its Pose is copied).
// Objects in that group can set their relative Pos etc to display
// relative to the camera, to achieve "first person" effects.
func (sc *Scene) TrackCamera() bool {
	tci, err := sc.ChildByNameTry("TrackCamera", 0)
	if err != nil {
		return false
	}
	tc, ok := tci.(*Group)
	if !ok {
		return false
	}
	tc.TrackCamera(sc)
	return true
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
