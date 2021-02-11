// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"sort"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
// the camera (i.e., its Pose is copied) -- Solids in that group can
// set their relative Pos etc to display relative to the camera, to achieve
// "first person" effects.
type Scene struct {
	gi.WidgetBase
	Geom          gi.Geom2DInt       `desc:"Viewport-level viewbox within any parent Viewport2D"`
	Camera        Camera             `desc:"camera determines view onto scene"`
	BgColor       gist.Color         `desc:"background color"`
	Wireframe     bool               `desc:"if true, render as wireframe instead of filled"`
	Lights        map[string]Light   `desc:"all lights used in the scene"`
	Meshes        map[string]Mesh    `desc:"all meshes used in the scene"`
	Textures      map[string]Texture `desc:"all textures used in the scene"`
	Library       map[string]*Group  `desc:"library of objects that can be used in the scene"`
	NoNav         bool               `desc:"don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene"`
	SavedCams     map[string]Camera  `desc:"saved cameras -- can Save and Set these to view the scene from different angles"`
	Win           *gi.Window         `copy:"-" json:"-" xml:"-" desc:"our parent window that we render into"`
	Renders       Renderers          `view:"-" desc:"rendering programs"`
	Frame         gpu.Framebuffer    `view:"-" desc:"direct render target for scene"`
	Tex           gpu.Texture2D      `view:"-" desc:"the texture that the framebuffer returns, which should be rendered into the window"`
	SetDragCursor bool               `view:"-" desc:"has dragging cursor been set yet?"`
	SelMode       SelModes           `desc:"how to deal with selection / manipulation events"`
	CurSel        Node3D             `copy:"-" json:"-" xml:"-" view:"-" desc:"currently selected node"`
	CurManipPt    *ManipPt           `copy:"-" json:"-" xml:"-" view:"-" desc:"currently selected manipulation control point"`
	SelParams     SelParams          `view:"inline" desc:"parameters for selection / manipulation box"`
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
	sc.SelParams.Defaults()
}

func (sc *Scene) Disconnect() {
	if sc.Win != nil && sc.Win.IsVisible() {
		sc.DeleteResources()
	}
	sc.WidgetBase.Disconnect()
}

// Update is a global update of everything: Init3D and re-render
func (sc *Scene) Update() {
	updt := sc.UpdateStart()
	sc.Init3D()
	sc.UpdateEnd(updt)
}

// AddMesh adds given mesh to mesh collection.  Any existing mesh of the
// same name is deleted.
// see AddNewX for convenience methods to add specific shapes
func (sc *Scene) AddMesh(ms Mesh) {
	if sc.Meshes == nil {
		sc.Meshes = make(map[string]Mesh)
	}
	nm := ms.Name()
	ems, has := sc.Meshes[nm]
	if has {
		oswin.TheApp.RunOnMain(func() {
			ems.Delete(sc)
		})
	}
	sc.Meshes[nm] = ms
}

// AddMeshUniqe adds given mesh to mesh collection, ensuring that it has
// a unique name if one already exists.
func (sc *Scene) AddMeshUnique(ms Mesh) {
	nm := ms.Name()
	if sc.Meshes == nil {
		sc.Meshes = make(map[string]Mesh)
		sc.Meshes[nm] = ms
	}
	_, has := sc.Meshes[nm]
	if has {
		nm += fmt.Sprintf("_%d", len(sc.Meshes))
		ms.SetName(nm)
	}
	sc.Meshes[nm] = ms
}

// MeshByName looks for mesh by name -- returns nil if not found
func (sc *Scene) MeshByName(nm string) Mesh {
	if sc.Meshes == nil {
		sc.Meshes = make(map[string]Mesh)
	}
	ms, ok := sc.Meshes[nm]
	if ok {
		return ms
	}
	return nil
}

// MeshByNameTry looks for mesh by name -- returns error if not found
func (sc *Scene) MeshByNameTry(nm string) (Mesh, error) {
	if sc.Meshes == nil {
		sc.Meshes = make(map[string]Mesh)
	}
	ms, ok := sc.Meshes[nm]
	if ok {
		return ms, nil
	}
	return nil, fmt.Errorf("Mesh named: %v not found in Scene: %v", nm, sc.Nm)
}

// MeshList returns a list of available meshes (e.g., for chooser)
func (sc *Scene) MeshList() []string {
	sz := len(sc.Meshes)
	if sz == 0 {
		return nil
	}
	sl := make([]string, sz)
	ctr := 0
	for k := range sc.Meshes {
		sl[ctr] = k
		ctr++
	}
	return sl
}

// DeleteMesh removes given mesh -- returns error if mesh not found.
func (sc *Scene) DeleteMesh(nm string) error {
	ms, ok := sc.Meshes[nm]
	if ok {
		oswin.TheApp.RunOnMain(func() {
			ms.Delete(sc)
		})
		delete(sc.Meshes, nm)
		return nil
	}
	return fmt.Errorf("Mesh named: %v not found in Scene: %v", nm, sc.Nm)
}

// DeleteMeshes removes all meshes
func (sc *Scene) DeleteMeshes() {
	oswin.TheApp.RunOnMain(func() {
		for _, ms := range sc.Meshes {
			ms.Delete(sc)
		}
	})
	sc.Meshes = make(map[string]Mesh)
}

// PlaneMesh2D returns the special Plane mesh used for Text2D and Embed2D
// (creating it if it does not yet exist).
// This is a 1x1 plane with a normal pointing in +Z direction.
func (sc *Scene) PlaneMesh2D() Mesh {
	nm := Plane2DMeshName
	tm, ok := sc.Meshes[nm]
	if ok {
		return tm
	}
	tmp := AddNewPlane(sc, nm, 1, 1)
	tmp.NormAxis = mat32.Z
	tmp.NormNeg = false
	return tmp
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
	nm := tx.Name()
	if _, has := sc.Textures[nm]; has {
		sc.DeleteTexture(nm) // prevent memory leak from not deleting
	}
	sc.Textures[nm] = tx
}

// TextureByName looks for texture by name -- returns nil if not found
func (sc *Scene) TextureByName(nm string) Texture {
	if sc.Textures == nil {
		sc.Textures = make(map[string]Texture)
	}
	tx, ok := sc.Textures[nm]
	if ok {
		return tx
	}
	return nil
}

// TextureByNameTry looks for texture by name -- returns error if not found
func (sc *Scene) TextureByNameTry(nm string) (Texture, error) {
	if sc.Textures == nil {
		sc.Textures = make(map[string]Texture)
	}
	tx, ok := sc.Textures[nm]
	if ok {
		return tx, nil
	}
	return nil, fmt.Errorf("Texture named: %v not found in Scene: %v", nm, sc.Nm)
}

// TextureList returns a list of available textures (e.g., for chooser)
func (sc *Scene) TextureList() []string {
	sz := len(sc.Textures)
	if sz == 0 {
		return nil
	}
	sl := make([]string, sz)
	ctr := 0
	for k := range sc.Textures {
		sl[ctr] = k
		ctr++
	}
	return sl
}

// DeleteTexture deletes texture of given name -- returns error if not found
func (sc *Scene) DeleteTexture(nm string) error {
	tx, ok := sc.Textures[nm]
	if ok {
		oswin.TheApp.RunOnMain(func() {
			tx.Delete(sc)
		})
		delete(sc.Textures, nm)
		return nil
	}
	return fmt.Errorf("Texture named: %v not found in Scene: %v", nm, sc.Nm)
}

// DeleteTextures removes all textures
func (sc *Scene) DeleteTextures() {
	oswin.TheApp.RunOnMain(func() {
		for _, tx := range sc.Textures {
			tx.Delete(sc)
		}
	})
	sc.Textures = make(map[string]Texture)
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
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
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
	sc.Init3D()
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
		csz := sc.Frame.Size()
		if csz == nwsz {
			sc.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sc.Frame != nil {
		oswin.TheApp.RunOnMain(func() {
			if !sc.Win.OSWin.Activate() {
				return
			}
			sc.Frame.SetSize(nwsz)
		})
	}
	sc.Geom.Size = nwsz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
}

func (sc *Scene) Init2D() {
	sc.Init2DWidget()
	sc.SetCurWin()
	// note: Viewport will automatically update us for any update sigs
	sc.Init3D()
	sc.Win.AddDirectUploader(sc)
}

func (sc *Scene) Style2D() {
	sc.StyMu.Lock()
	defer sc.StyMu.Unlock()

	sc.SetCanFocusIfActive() // we get all key events
	sc.SetCurWin()
	sc.Style2DWidget()
	sc.LayState.SetFromStyle(&sc.Sty.Layout) // also does reset
	// note: we do Style3D in Init3D
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
	// note: children are automatically moved in Node3DBase.UpdateBBox2D
	// by adding scene's ObjBBox.Min - BBox.Min offset in their own VpBBox computation
	// publish makes the thing update after scrolling -- doesn't otherwise.
	sc.Win.PublishFullReRender()
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
	sc.ConnectEvent(oswin.MouseDragEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ssc := recv.Embed(KiT_Scene).(*Scene)
		if ssc.NoNav {
			return
		}
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		if ssc.IsDragging() {
			orbDel := float32(.2)
			panDel := float32(.01)
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
	// sc.ConnectEvent(oswin.MouseMoveEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
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
	sc.ConnectEvent(oswin.MouseScrollEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ssc := recv.Embed(KiT_Scene).(*Scene)
		if ssc.NoNav {
			return
		}
		me := d.(*mouse.ScrollEvent)
		me.SetProcessed()
		if ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
			ssc.SetDragCursor = false
		}
		zoom := float32(me.NonZeroDelta(false))
		zoomPct := float32(.01)
		zoomDel := float32(.01)
		switch {
		case key.HasAllModifierBits(me.Modifiers, key.Alt):
			ssc.Camera.PanTarget(0, 0, zoom*zoomDel)
		default:
			ssc.Camera.Zoom(zoomPct * zoom)
		}
		ssc.UpdateSig()
	})
	sc.ConnectEvent(oswin.MouseEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ssc := recv.Embed(KiT_Scene).(*Scene)
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
		if !ssc.IsInactive() && !ssc.HasFocus() {
			ssc.GrabFocus()
		}
		// if ssc.CurManipPt == nil {
		ssc.SetSel(nil) // clear any selection at this point
		// }
	})
	sc.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		ssc := recv.Embed(KiT_Scene).(*Scene)
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

/////////////////////////////////////////////////////////////////////////////////////
// 		Rendering

// ActivateWin activates the window context for GPU rendering context (on the
// main thread -- all GPU rendering actions must be performed on main thread)
// returns false if not possible (i.e., Win nil, not visible)
func (sc *Scene) ActivateWin() bool {
	if sc.Win == nil || !sc.Win.IsVisible() {
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
		log.Printf("gi3d.Scene: %s not able to activate window\n", sc.Path())
		return false
	}
	oswin.TheApp.RunOnMain(func() {
		if sc.Frame == nil {
			msamp := 4
			if !gi.Prefs.Params.Smooth3D {
				msamp = 0
			}
			sc.Frame = gpu.TheGPU.NewFramebuffer(sc.Nm+"-frame", sc.Geom.Size, msamp)
		}
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
		sc.Camera.CamMu.Lock()
		sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
		sc.Camera.CamMu.Unlock()
		// fmt.Printf("aspect ratio: %v\n", sc.Camera.Aspect)
		sc.Frame.Activate()
		sc.Renders.DrawState()
		clr := ColorToVec3f(sc.BgColor)
		gpu.Draw.ClearColor(clr.X, clr.Y, clr.Z)
		gpu.Draw.Clear(true, true) // clear color and depth
		gpu.Draw.Wireframe(sc.Wireframe)
	})
	return true
}

// ActivateOffFrame creates (if necc) and activates given offscreen framebuffer
// for GPU rendering context, of given size, and multisampling number (4 = default
// for good antialiasing, 0 if not hardware accelerated).
func (sc *Scene) ActivateOffFrame(frame *gpu.Framebuffer, name string, size image.Point, msamp int) error {
	var err error
	oswin.TheApp.RunOnMain(func() {
		if *frame == nil {
			*frame = gpu.TheGPU.NewFramebuffer(name, size, msamp)
		}
		fr := *frame
		fr.SetSize(size) // nop if same
		sc.Camera.CamMu.Lock()
		sc.Camera.Aspect = float32(size.X) / float32(size.Y)
		sc.Camera.CamMu.Unlock()
		err = fr.Activate()
		if err == nil {
			sc.Renders.DrawState()
			clr := ColorToVec3f(sc.BgColor)
			gpu.Draw.ClearColor(clr.X, clr.Y, clr.Z)
			gpu.Draw.Clear(true, true) // clear color and depth
		}
	})
	return err
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
		InitMesh(ms, sc)
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

// InitMesh does a full init and gpu transfer of the given mesh name.
func (sc *Scene) InitMesh(nm string) error {
	if !sc.ActivateWin() {
		return fmt.Errorf("InitMesh: %v in Scene: %v  Could not activate window", nm, sc.Nm)
	}
	ms, ok := sc.Meshes[nm]
	if ok {
		oswin.TheApp.RunOnMain(func() {
			InitMesh(ms, sc)
		})
		return nil
	}
	return fmt.Errorf("Mesh named: %v not found in Scene: %v", nm, sc.Nm)
}

// UpdateMeshesInCtxt calls Update on all the meshes in context on main thread.
// Update is responsible for doing any transfers.
func (sc *Scene) UpdateMeshesInCtxt() bool {
	for _, ms := range sc.Meshes {
		ms.Update(sc)
	}
	return true
}

// UpdateMeshes calls Update on all meshes (for dynamically updating meshes).
// This version is to be used by external users -- sets context and runs on main.
func (sc *Scene) UpdateMeshes() {
	if !sc.ActivateWin() {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		sc.UpdateMeshesInCtxt()
	})
}

// DeleteResources deletes all GPU resources -- sets context and runs on main.
// This is called during Disconnect and before the window is closed.
func (sc *Scene) DeleteResources() {
	if sc.Win == nil {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		// sc.Win.OSWin.Activate()
		for _, tx := range sc.Textures {
			tx.Delete(sc)
		}
		for _, ms := range sc.Meshes {
			ms.Delete(sc)
		}
		if sc.Tex != nil {
			sc.Tex.Delete()
		}
		if sc.Frame != nil {
			sc.Frame.Delete()
		}
	})
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sc *Scene) UpdateMeshBBox() {
	for _, kid := range sc.Kids {
		kii, _ := KiToNode3D(kid)
		if kii == nil {
			continue
		}
		kii.FuncDownMeLast(0, kii.This(),
			func(k ki.Ki, level int, d interface{}) bool {
				nii, _ := KiToNode3D(k)
				if nii == nil {
					return ki.Break // going into a different type of thing, bail
				}
				return ki.Continue
			},
			func(k ki.Ki, level int, d interface{}) bool {
				nii, _ := KiToNode3D(k)
				if nii == nil {
					return ki.Break // going into a different type of thing, bail
				}
				nii.UpdateMeshBBox()
				return ki.Continue
			})
	}
}

// UpdateWorldMatrix updates the world matrix for all scene elements
// called during Init3D and rendering
func (sc *Scene) UpdateWorldMatrix() {
	idmtx := mat32.NewMat4()
	for _, kid := range sc.Kids {
		kii, _ := KiToNode3D(kid)
		if kii == nil {
			continue
		}
		kii.UpdateWorldMatrix(idmtx)
		kii.FuncDownMeFirst(0, kii.This(), func(k ki.Ki, level int, d interface{}) bool {
			if k == kid {
				return ki.Continue // skip, already did
			}
			nii, _ := KiToNode3D(k)
			if nii == nil {
				return ki.Break // going into a different type of thing, bail
			}
			pii, pi := KiToNode3D(k.Parent())
			if pii == nil {
				return ki.Break
			}
			pi.PoseMu.RLock()
			nii.UpdateWorldMatrix(&pi.Pose.WorldMatrix)
			pi.PoseMu.RUnlock()
			return ki.Continue
		})
	}
}

// UpdateMVPMatrix updates the Model-View-Projection matrix for all scene elements
// and BBox2D
func (sc *Scene) UpdateMVPMatrix() {
	sc.Camera.CamMu.Lock()
	defer sc.Camera.CamMu.Unlock()

	sc.Camera.Pose.UpdateMatrix()
	sz := sc.Geom.Size
	size := mat32.Vec2{float32(sz.X), float32(sz.Y)}

	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, _ := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		nii.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.PrjnMatrix)
		nii.UpdateBBox2D(size, sc)
		return ki.Continue
	})
}

func (sc *Scene) Init3D() {
	if sc.Camera.FOV == 0 {
		sc.Defaults()
	}
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
			return ki.Continue
		}
		nii, _ := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		nii.Init3D(sc)
		return ki.Continue
	})
	sc.Style3D()
}

func (sc *Scene) Style3D() {
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, _ := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		nii.Style3D(sc)
		return ki.Continue
	})
}

func (sc *Scene) UpdateNodes3D() {
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, _ := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		nii.UpdateNode3D(sc)
		return ki.Continue
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
	sc.SetFlag(int(Rendering))
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateNodes3D()
	sc.UpdateWorldMatrix()
	sc.UpdateMeshBBox()
	sc.UpdateMVPMatrix()
	oswin.TheApp.RunOnMain(func() {
		sc.Renders.SetLightsUnis(sc)
		sc.Render3D(false) // not offscreen
		gpu.Draw.Flush()
		gpu.Draw.Wireframe(false)
		sc.Frame.Rendered()
		sc.Tex = sc.Frame.Texture()
		sc.Tex.SetBotZero(true) // this has Y=0 at bottom!
	})
	sc.ClearFlag(int(Rendering))
	return true
}

// RenderOffFrame renders the scene to currently-activated offscreen framebuffer
// must call ActivateOffFrame first and call Frame.Rendered() after!
func (sc *Scene) RenderOffFrame() bool {
	sc.SetFlag(int(Rendering))
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateWorldMatrix()
	sc.UpdateMeshBBox()
	sc.UpdateMVPMatrix()
	oswin.TheApp.RunOnMain(func() {
		sc.Renders.SetLightsUnis(sc)
		sc.Render3D(true) //  yes offscreen
		gpu.Draw.Flush()
	})
	sc.ClearFlag(int(Rendering))
	return true
}

func (sc *Scene) IsDirectWinUpload() bool {
	return true
}

func (sc *Scene) IsRendering() bool {
	return sc.HasFlag(int(Rendering))
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
	if wt != nil {
		oswin.TheApp.RunOnMain(func() {
			if !sc.Win.OSWin.Activate() {
				return
			}
			// limit upload to vpbbox region
			rvp := sc.VpBBox
			tvp := rvp
			sz := sc.Tex.Size()
			tvp.Max = tvp.Min.Add(sz)
			tvp = rvp.Intersect(tvp)
			// mvoff is amount left edge of scene has been clipped by VpBbox and is no longer
			// visible -- thus how much the texture blit must move over accordingly.
			mvoff := sc.VpBBox.Min.Sub(sc.ObjBBox.Min)
			tb := image.Rectangle{Min: mvoff, Max: mvoff.Add(tvp.Size())}
			fb := tb
			fb.Min.Y = sz.Y - tb.Max.Y // flip Y
			fb.Max.Y = sz.Y - tb.Min.Y
			sc.BBoxMu.RLock()
			wt.Copy(sc.WinBBox.Min, sc.Tex, fb, draw.Src, nil)
			sc.BBoxMu.RUnlock()
		})
	}
	if !sc.Win.IsUpdating() {
		sc.Win.UpdateSig() // trigger publish
	}
	return true
}

// Render3D renders the scene to the framebuffer
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) Render3D(offscreen bool) {
	var rcs [RenderClassesN][]Node3D
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		if !offscreen {
			ni.BBoxMu.RLock()
			if ni.IsInvisible() || ni.ObjBBox == image.ZR { // objbbox is intersection of scene and obj
				ni.BBoxMu.RUnlock()
				ni.DisconnectAllEvents(sc.Win, gi.AllPris)
				return ki.Break
			}
			ni.BBoxMu.RUnlock()
			nii.ConnectEvents3D(sc) // only connect visible
		}
		if !nii.IsSolid() {
			return ki.Continue
		}
		rc := nii.RenderClass()
		if rc > RClassTransTexture { // all in one group b/c z-sorting is key
			rc = RClassTransTexture
		}
		rcs[rc] = append(rcs[rc], nii)
		return ki.Continue
	})

	for rci, objs := range rcs {
		rc := RenderClasses(rci)
		if len(objs) == 0 {
			continue
		}
		if rc >= RClassTransTexture { // sort back-to-front for transparent
			sort.Slice(objs, func(i, j int) bool {
				return objs[i].NormDCBBox().Max.Z > objs[j].NormDCBBox().Max.Z
			})
		} else { // sort front-to-back for opaque to allow "early z rejection"
			sort.Slice(objs, func(i, j int) bool {
				return objs[i].NormDCBBox().Min.Z < objs[j].NormDCBBox().Min.Z
			})
		}
		// fmt.Printf("\nRender class: %v\n", rc)
		// for i := range objs {
		// 	fmt.Printf("obj: %s  max z: %g   min z: %g\n", objs[i].Name(), objs[i].AsNode3D().NDCBBox.Max.Z, objs[i].AsNode3D().NDCBBox.Min.Z)
		// }

		var rnd Render
		if rc < RClassTransTexture {
			switch rc {
			case RClassOpaqueTexture:
				rnd = sc.Renders.Renders["RenderTexture"]
			case RClassOpaqueUniform:
				rnd = sc.Renders.Renders["RenderUniformColor"]
			case RClassOpaqueVertex:
				rnd = sc.Renders.Renders["RenderVertexColor"]
			}
			gpu.Draw.Op(draw.Src)     // opaque
			rnd.Activate(&sc.Renders) // use same program for all..
		}
		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				switch rc {
				case RClassTransTexture:
					rnd = sc.Renders.Renders["RenderTexture"]
				case RClassTransUniform:
					rnd = sc.Renders.Renders["RenderUniformColor"]
				case RClassTransVertex:
					rnd = sc.Renders.Renders["RenderVertexColor"]
				}
				gpu.Draw.Op(draw.Over) // alpha
				rnd.Activate(&sc.Renders)
				lastrc = rc
			}
			obj.Render3D(sc, rc, rnd)
		}
	}
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
		kii.FuncDownMeFirst(0, kii.This(), func(k ki.Ki, level int, d interface{}) bool {
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
	"EnumType:Flag": KiT_SceneFlags,
	"ToolBar": ki.PropSlice{
		{"Update", ki.Props{
			"icon": "update",
		}},
	},
}

// SceneFlags extend gi.NodeFlags to hold 3D node state
type SceneFlags int

//go:generate stringer -type=SceneFlags

var KiT_SceneFlags = kit.Enums.AddEnumExt(gi.KiT_NodeFlags, SceneFlagsN, kit.BitFlag, nil)

const (
	// Rendering means that the scene is currently rendering
	Rendering SceneFlags = SceneFlags(gi.NodeFlagsN) + iota

	SceneFlagsN
)
