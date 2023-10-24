// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

//go:generate goki generate

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"sync"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
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
	Geom mat32.Geom2DInt

	// number of samples in multisampling -- must be a power of 2, and must be 1 if grabbing the Depth buffer back from the RenderFrame
	MultiSample int `def:"4"`

	// render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system (i.e., just after Scene is made)
	Wireframe bool `def:"false"`

	// camera determines view onto scene
	Camera Camera `set:"-"`

	// background color
	BackgroundColor color.RGBA

	// all lights used in the scene
	Lights ordmap.Map[string, Light]

	// meshes -- holds all the mesh data -- must be configured prior to rendering
	Meshes ordmap.Map[string, Mesh] `set:"-"`

	// textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability
	Textures ordmap.Map[string, Texture]

	// library of objects that can be used in the scene
	Library map[string]*Group

	// don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene
	NoNav bool

	// saved cameras -- can Save and Set these to view the scene from different angles
	SavedCams map[string]Camera

	// has dragging cursor been set yet?
	SetDragCursor bool `view:"-"`

	// how to deal with selection / manipulation events
	SelMode SelModes

	// currently selected node
	CurSel Node `copy:"-" json:"-" xml:"-" view:"-"`

	// currently selected manipulation control point
	CurManipPt *ManipPt `copy:"-" json:"-" xml:"-" view:"-"`

	// parameters for selection / manipulation box
	SelParams SelParams `view:"inline"`

	// the vphong rendering system
	Phong vphong.Phong

	// the vgpu render frame holding the rendered scene
	Frame *vgpu.RenderFrame

	// index in list of window direct uploading images
	DirUpIdx int

	// mutex on rendering
	RenderMu sync.Mutex `view:"-" copy:"-" json:"-" xml:"-"`
}

// Defaults sets default scene params (camera, bg = white)
func (sc *Scene) Defaults() {
	sc.MultiSample = 4
	sc.Camera.Defaults()
	sc.BackgroundColor = colors.White
	sc.SelParams.Defaults()
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
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode3D(k)
		if !ni.IsVisible() {
			return ki.Break
		}
		err := ni.Validate(sc)
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

func (sc *Scene) IsInvisible() bool {
	return false
}

func (sc *Scene) IsVisible() bool {
	if sc == nil || sc.This() == nil || sc.IsInvisible() {
		return false
	}
	if sc.Par == nil || sc.Par.This() == nil {
		return false
	}
	return true
	// return sc.Par.This().(gi.Node2D).IsVisible()
}

// set our window pointer to point to the current window we are under
func (sc *Scene) SetCurWin() {
	// pwin := sc.ParentWindow()
	// if pwin != nil { // only update if non-nil -- otherwise we could be setting
	// 	// temporarily to give access to DPI etc
	// 	sc.Win = pwin
	// }
}

func (sc *Scene) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
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

func (sc *Scene) Config(gsc *gi.Scene) {
	// sc.Init3D()
	// sc.DirUpIdx = sc.Win.AddDirectUploader(sc)
}

func (sc *Scene) Size2D(iter int) {
	// sc.InitLayout2D()
	// we listen to x,y styling for positioning within parent vp, if non-zero -- todo: only popup?
	// pos := sc.Style.PosDots().ToPoint()
	// if pos != (image.Point{}) {
	// 	sc.Geom.Pos = pos
	// }
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
	// if sc.Viewport != nil {
	// 	sc.ComputeBBox2DBase(parBBox, delta)
	// }
	sc.Geom.Pos = sc.LayState.Alloc.Pos.ToPointFloor()
}

func (sc *Scene) ChildrenBBox2D() image.Rectangle {
	return sc.Geom.Bounds()
}

// we use our own render for these -- Viewport member is our parent!
func (sc *Scene) PushBounds() bool {
	if sc.ScBBox.Empty() {
		return false
	}
	if !sc.IsInvisible() {
		return false
	}
	// if we are completely invisible, no point in rendering..
	// if sc.Viewport != nil {
	// 	sc.BBoxMu.RLock()
	// 	wbi := sc.WinBBox.Intersect(sc.Viewport.WinBBox)
	// 	sc.BBoxMu.RUnlock()
	// 	if wbi.Empty() {
	// 		return false
	// 	}
	// }
	bb := sc.Geom.Bounds()
	// rs := &sc.Render
	// rs.PushBounds(bb)
	if gi.RenderTrace {
		fmt.Printf("Render: %v at %v\n", sc.Path(), bb)
	}
	return true
}

func (sc *Scene) PopBounds() {
	// rs := &vp.Render
	// rs.PopBounds()
}

// func (sc *Scene) Render2D() {
// 	if sc.PushBounds() {
// 		sc.NavEvents()
// 		if gi.Render2DTrace {
// 			fmt.Printf("3D Render2D: %v\n", sc.Path())
// 		}
// 		sc.Render()
// 		sc.PopBounds()
// 	} else {
// 		sc.DisconnectAllEvents(gi.RegPri)
// 	}
// }

/*
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
*/

// NavKeyEvents handles standard viewer keyboard navigation events
func (sc *Scene) NavKeyEvents(kt events.Event) {
	ch := string(kt.KeyChord())
	// fmt.Printf(ch)
	orbDeg := float32(5)
	panDel := float32(.2)
	zoomPct := float32(.05)
	switch ch {
	case "UpArrow":
		sc.Camera.Orbit(0, orbDeg)
		kt.SetHandled()
	case "Shift+UpArrow":
		sc.Camera.Pan(0, panDel)
		kt.SetHandled()
	case "Control+UpArrow":
		sc.Camera.PanAxis(0, panDel)
		kt.SetHandled()
	case "Alt+UpArrow":
		sc.Camera.PanTarget(0, panDel, 0)
		kt.SetHandled()
	case "DownArrow":
		sc.Camera.Orbit(0, -orbDeg)
		kt.SetHandled()
	case "Shift+DownArrow":
		sc.Camera.Pan(0, -panDel)
		kt.SetHandled()
	case "Control+DownArrow":
		sc.Camera.PanAxis(0, -panDel)
		kt.SetHandled()
	case "Alt+DownArrow":
		sc.Camera.PanTarget(0, -panDel, 0)
		kt.SetHandled()
	case "LeftArrow":
		sc.Camera.Orbit(orbDeg, 0)
		kt.SetHandled()
	case "Shift+LeftArrow":
		sc.Camera.Pan(-panDel, 0)
		kt.SetHandled()
	case "Control+LeftArrow":
		sc.Camera.PanAxis(-panDel, 0)
		kt.SetHandled()
	case "Alt+LeftArrow":
		sc.Camera.PanTarget(-panDel, 0, 0)
		kt.SetHandled()
	case "RightArrow":
		sc.Camera.Orbit(-orbDeg, 0)
		kt.SetHandled()
	case "Shift+RightArrow":
		sc.Camera.Pan(panDel, 0)
		kt.SetHandled()
	case "Control+RightArrow":
		sc.Camera.PanAxis(panDel, 0)
		kt.SetHandled()
	case "Alt+RightArrow":
		sc.Camera.PanTarget(panDel, 0, 0)
		kt.SetHandled()
	case "Alt++", "Alt+=":
		sc.Camera.PanTarget(0, 0, panDel)
		kt.SetHandled()
	case "Alt+-", "Alt+_":
		sc.Camera.PanTarget(0, 0, -panDel)
		kt.SetHandled()
	case "+", "=", "Shift++":
		sc.Camera.Zoom(-zoomPct)
		kt.SetHandled()
	case "-", "_", "Shift+_":
		sc.Camera.Zoom(zoomPct)
		kt.SetHandled()
	case " ", "Escape":
		err := sc.SetCamera("default")
		if err != nil {
			sc.Camera.DefaultPose()
		}
		kt.SetHandled()
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		err := sc.SetCamera(ch)
		if err != nil {
			sc.SaveCamera(ch)
			fmt.Printf("Saved camera to: %v\n", ch)
		} else {
			fmt.Printf("Restored camera from: %v\n", ch)
		}
		kt.SetHandled()
	case "Control+0", "Control+1", "Control+2", "Control+3", "Control+4", "Control+5", "Control+6", "Control+7", "Control+8", "Control+9":
		cnm := strings.TrimPrefix(ch, "Control+")
		sc.SaveCamera(cnm)
		fmt.Printf("Saved camera to: %v\n", cnm)
		kt.SetHandled()
	case "t":
		kt.SetHandled()
		obj := sc.Child(0).(*Solid)
		fmt.Printf("updated obj: %v\n", obj.Path())
		// obj.UpdateSig()
		return
	}
	// sc.UpdateSig()
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
func (sc *Scene) SolidsIntersectingPoint(pos image.Point) []Node {
	var objs []Node
	for _, kid := range sc.Kids {
		kii, _ := AsNode3D(kid)
		if kii == nil {
			continue
		}
		kii.WalkPre(func(k ki.Ki) bool {
			ni, _ := AsNode3D(k)
			if ni == nil {
				return ki.Break // going into a different type of thing, bail
			}
			if !ni.IsSolid() {
				return ki.Continue
			}
			// if nb.PosInWinBBox(pos) {
			objs = append(objs, ni)
			// }
			return ki.Continue
		})
	}
	return objs
}
