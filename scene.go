// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/icons"
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/cursor"
	"goki.dev/gi/v2/oswin/key"
	"goki.dev/gi/v2/oswin/mouse"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/vgpu"
	"goki.dev/vgpu/v2/vphong"
)

const (
	// TrackCameraName is a reserved top-level Group name -- this group
	// will have its Pose updated to match that of the camera automatically.
	TrackCameraName = "TrackCamera"

	// SelBoxName is the reserved top-level Group name for holding
	// a bounding box or manipulator for currently selected object.
	// also used for meshes representing the box.
	SelBoxName = "__SelectedBox"

	// ManipBoxName is the reserved top-level name for meshes
	// representing the manipulation box.
	ManipBoxName = "__ManipBox"

	// Plane2DMeshName is the reserved name for the 2D plane mesh
	// used for Text2D and Embed2D
	Plane2DMeshName = "__Plane2D"

	// LineMeshName is the reserved name for a unit-sized Line segment
	LineMeshName = "__UnitLine"

	// ConeMeshName is the reserved name for a unit-sized Cone segment.
	// Has the number of segments appended.
	ConeMeshName = "__UnitCone"
)

// Set Update3DTrace to true to get a trace of 3D updating
var Update3DTrace = false

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own vgpu.RenderFrame, the Image of which is then copied
// into the window vgpu.Drawer images for subsequent compositing into the
// window directly, as a DurectWinUpload element.
//
// There is default navigation event processing (disabled by setting NoNav)
// where mouse drag events Orbit the camera (Shift = Pan, Alt = PanTarget)
// and arrow keys do Orbit, Pan, PanTarget with same key modifiers.
// Spacebar restores original "default" camera, and numbers save (1st time)
// or restore (subsequently) camera views (Control = always save)
//
// A Group at the top-level named "TrackCamera" will automatically track
// the camera (i.e., its Pose is copied) -- Solids in that group can
// set their relative Pos etc to display relative to the camera, to achieve
// "first person" effects.
type Scene struct {
	gi.WidgetBase

	// Viewport-level viewbox within any parent Viewport2D
	Geom gi.Geom2DInt `desc:"Viewport-level viewbox within any parent Viewport2D"`

	// [def: 4] number of samples in multisampling -- must be a power of 2, and must be 1 if grabbing the Depth buffer back from the RenderFrame
	MultiSample int `def:"4" desc:"number of samples in multisampling -- must be a power of 2, and must be 1 if grabbing the Depth buffer back from the RenderFrame"`

	// [def: false] render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system (i.e., just after Scene is made)
	Wireframe bool `def:"false" desc:"render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system (i.e., just after Scene is made)"`

	// camera determines view onto scene
	Camera Camera `desc:"camera determines view onto scene"`

	// background color
	BackgroundColor color.RGBA `desc:"background color"`

	// all lights used in the scene
	Lights ordmap.Map[string, Light] `desc:"all lights used in the scene"`

	// meshes -- holds all the mesh data -- must be configured prior to rendering
	Meshes ordmap.Map[string, Mesh] `desc:"meshes -- holds all the mesh data -- must be configured prior to rendering"`

	// textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability
	Textures ordmap.Map[string, Texture] `desc:"textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability"`

	// library of objects that can be used in the scene
	Library map[string]*Group `desc:"library of objects that can be used in the scene"`

	// don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene
	NoNav bool `desc:"don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene"`

	// saved cameras -- can Save and Set these to view the scene from different angles
	SavedCams map[string]Camera `desc:"saved cameras -- can Save and Set these to view the scene from different angles"`

	// our parent window that we render into
	Win *gi.Window `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`

	// [view: -] has dragging cursor been set yet?
	SetDragCursor bool `view:"-" desc:"has dragging cursor been set yet?"`

	// how to deal with selection / manipulation events
	SelMode SelModes `desc:"how to deal with selection / manipulation events"`

	// [view: -] currently selected node
	CurSel Node3D `copy:"-" json:"-" xml:"-" view:"-" desc:"currently selected node"`

	// [view: -] currently selected manipulation control point
	CurManipPt *ManipPt `copy:"-" json:"-" xml:"-" view:"-" desc:"currently selected manipulation control point"`

	// [view: inline] parameters for selection / manipulation box
	SelParams SelParams `view:"inline" desc:"parameters for selection / manipulation box"`

	// the vphong rendering system
	Phong vphong.Phong `desc:"the vphong rendering system"`

	// the vgpu render frame holding the rendered scene
	Frame *vgpu.RenderFrame `desc:"the vgpu render frame holding the rendered scene"`

	// index in list of window direct uploading images
	DirUpIdx int `desc:"index in list of window direct uploading images"`

	// [view: -] mutex on rendering
	RenderMu sync.Mutex `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex on rendering"`
}

var TypeScene = kit.Types.AddType(&Scene{}, SceneProps)

// AddNewScene adds a new scene to given parent node, with given name.
func AddNewScene(parent ki.Ki, name string) *Scene {
	sc := parent.AddNewChild(TypeScene, name).(*Scene)
	sc.Defaults()
	return sc
}

// Defaults sets default scene params (camera, bg = white)
func (sc *Scene) Defaults() {
	sc.MultiSample = 4
	sc.Camera.Defaults()
	sc.BackgroundColor = colors.White
	sc.SelParams.Defaults()
}

func (sc *Scene) Disconnect() {
	sc.WidgetBase.Disconnect()
}

// Update is a global update of everything: Init3D and re-render
func (sc *Scene) Update() {
	updt := sc.UpdateStart()
	sc.Init3D()
	sc.UpdateEnd(updt)
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
	// fmt.Printf("saved camera %s: %v\n", name, sc.Camera.Pose.GenGoSet(".Pose"))
}

// SetCamera sets the current camera to that of given name -- error if not found.
// "default" is a special name that is automatically saved on first render, and
// restored with the spacebar under default NavEvents.
// Numbered cameras 0-9 also saved / restored with corresponding keys.
func (sc *Scene) SetCamera(name string) error {
	cam, ok := sc.SavedCams[name]
	if !ok {
		return fmt.Errorf("gi3d.Scene: %v saved camera of name: %v not found", sc.Nm, name)
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
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		if ni.IsInvisible() {
			return ki.Break
		}
		err := nii.Validate(sc)
		if err != nil {
			hasError = true
		}
		return ki.Continue
	})
	if hasError {
		return fmt.Errorf("gi3d.Scene: %v Validate found at least one error (see log)", sc.Path())
	}
	return nil
}

/////////////////////////////////////////////////////////////////////////////////////
//  Library

// AddToLibrary adds given Group to library, using group's name as unique key
// in Library map.
func (sc *Scene) AddToLibrary(gp *Group) {
	if sc.Library == nil {
		sc.Library = make(map[string]*Group)
	}
	sc.Library[gp.Name()] = gp
}

// NewInLibrary makes a new Group in library, using given name as unique key
// in Library map.
func (sc *Scene) NewInLibrary(nm string) *Group {
	gp := &Group{}
	gp.InitName(gp, nm)
	sc.AddToLibrary(gp)
	return gp
}

// AddFmLibrary adds a Clone of named item in the Library under given parent
// in the scenegraph.  Returns an error if item not found.
func (sc *Scene) AddFmLibrary(nm string, parent ki.Ki) (*Group, error) {
	gp, ok := sc.Library[nm]
	if !ok {
		return nil, fmt.Errorf("Scene AddFmLibrary: Library item: %s not found", nm)
	}
	updt := sc.UpdateStart()
	nwgp := gp.Clone().(*Group)
	parent.AddChild(nwgp)
	sc.UpdateEnd(updt)
	return nwgp, nil
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
		csz := sc.Frame.Format.Size
		if csz == nwsz {
			sc.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sc.Frame != nil {
		sc.Frame.SetSize(nwsz)
	}
	sc.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

func (sc *Scene) Init2D() {
	sc.Init2DWidget()
	sc.SetCurWin()
	// note: Viewport will automatically update us for any update sigs
	sc.Init3D()
	sc.DirUpIdx = sc.Win.AddDirectUploader(sc)
}

func (sc *Scene) Style2D() {
	sc.StyMu.Lock()
	defer sc.StyMu.Unlock()

	sc.SetCanFocusIfActive() // we get all key events
	sc.SetCurWin()
	sc.Style2DWidget()
	sc.LayState.SetFromStyle(&sc.Style) // also does reset
	// note: we do Style3D in Init3D
}

func (sc *Scene) Size2D(iter int) {
	sc.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	pos := sc.Style.PosDots().ToPoint()
	if pos != (image.Point{}) {
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
	if sz != (image.Point{}) {
		sc.Geom.Size = sz
	} else {
		bb.Max = bb.Min.Add(image.Point{64, 64}) // min size for zero case
	}
	return bb
}

func (sc *Scene) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	if sc.Viewport != nil {
		sc.ComputeBBox2DBase(parBBox, delta)
	}
	sc.Geom.Pos = sc.LayState.Alloc.Pos.ToPointFloor()
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
		sc.BBoxMu.RLock()
		wbi := sc.WinBBox.Intersect(sc.Viewport.WinBBox)
		sc.BBoxMu.RUnlock()
		if wbi.Empty() {
			return false
		}
	}
	bb := sc.Geom.Bounds()
	// rs := &sc.Render
	// rs.PushBounds(bb)
	if gi.Render2DTrace {
		fmt.Printf("Render: %v at %v\n", sc.Path(), bb)
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
}

func (sc *Scene) Render2D() {
	if sc.PushBounds() {
		sc.NavEvents()
		if gi.Render2DTrace {
			fmt.Printf("3D Render2D: %v\n", sc.Path())
		}
		sc.Render()
		sc.PopBounds()
	} else {
		sc.DisconnectAllEvents(gi.RegPri)
	}
}

// NavEvents handles standard viewer navigation events
func (sc *Scene) NavEvents() {
	sc.ConnectEvent(oswin.MouseDragEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
		ssc := recv.Embed(TypeScene).(*Scene)
		if ssc.NoNav {
			return
		}
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		if ssc.IsDragging() {
			cdist := mat32.Max(ssc.Camera.DistTo(ssc.Camera.Target), 1.0)
			orbDel := 0.025 * cdist
			panDel := 0.001 * cdist

			if !ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Push(cursor.HandOpen)
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
				oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
				ssc.SetDragCursor = false
			}
		}
	})
	// sc.ConnectEvent(oswin.MouseMoveEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(*mouse.MoveEvent)
	// 	me.SetProcessed()
	// 	ssc := recv.Embed(TypeScene).(*Scene)
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
	sc.ConnectEvent(oswin.MouseScrollEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
		ssc := recv.Embed(TypeScene).(*Scene)
		if ssc.NoNav {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		if ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
			ssc.SetDragCursor = false
		}
		pt := me.Where.Sub(sc.ObjBBox.Min)
		sz := ssc.Geom.Size
		cdist := mat32.Max(ssc.Camera.DistTo(ssc.Camera.Target), 1.0)
		zoom := float32(me.NonZeroDelta(false))
		zoomDel := float32(.001) * cdist
		switch {
		case key.HasAllModifierBits(me.Modifiers, key.Alt):
			ssc.Camera.PanTarget(0, 0, zoom*zoomDel)
		default:
			ssc.Camera.ZoomTo(pt, sz, zoom*zoomDel)
		}
		ssc.UpdateSig()
	})
	sc.ConnectEvent(oswin.MouseEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
		ssc := recv.Embed(TypeScene).(*Scene)
		if ssc.NoNav {
			return
		}
		me := d.(*mouse.Event)
		if ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
			ssc.SetDragCursor = false
		}
		if me.Action != mouse.Press {
			return
		}
		if !ssc.IsDisabled() && !ssc.HasFocus() {
			ssc.GrabFocus()
		}
		// if ssc.CurManipPt == nil {
		ssc.SetSel(nil) // clear any selection at this point
		// }
	})
	sc.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
		ssc := recv.Embed(TypeScene).(*Scene)
		if ssc.NoNav {
			return
		}
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
	case "+", "=", "Shift++":
		sc.Camera.Zoom(-zoomPct)
		kt.SetProcessed()
	case "-", "_", "Shift+_":
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
		obj := sc.Child(0).(*Solid)
		fmt.Printf("updated obj: %v\n", obj.Path())
		obj.UpdateSig()
		return
	}
	sc.UpdateSig()
}

// TrackCamera -- a Group at the top-level named "TrackCamera"
// will automatically track the camera (i.e., its Pose is copied).
// Solids in that group can set their relative Pos etc to display
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

// SolidsIntersectingPoint finds all the solids that contain given 2D window coordinate
func (sc *Scene) SolidsIntersectingPoint(pos image.Point) []Node3D {
	var objs []Node3D
	for _, kid := range sc.Kids {
		kii, _ := KiToNode3D(kid)
		if kii == nil {
			continue
		}
		kii.FuncDownMeFirst(0, kii.This(), func(k ki.Ki, level int, d any) bool {
			nii, ni := KiToNode3D(k)
			if nii == nil {
				return ki.Break // going into a different type of thing, bail
			}
			if !nii.IsSolid() {
				return ki.Continue
			}
			if ni.PosInWinBBox(pos) {
				objs = append(objs, nii)
			}
			return ki.Continue
		})
	}
	return objs
}

// SceneProps define the ToolBar and MenuBar for StructView
var SceneProps = ki.Props{
	ki.EnumTypeFlag: TypeSceneFlags,
	"ToolBar": ki.PropSlice{
		{"Update", ki.Props{
			"icon": icons.Refresh,
		}},
	},
}

// SceneFlags extend gi.NodeFlags to hold 3D node state
type SceneFlags int

var TypeSceneFlags = kit.Enums.AddEnumExt(gi.TypeNodeFlags, SceneFlagsN, kit.BitFlag, nil)

const (
	// Rendering means that the scene is currently rendering
	Rendering SceneFlags = SceneFlags(gi.NodeFlagsN) + iota

	SceneFlagsN
)
