// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

//go:generate goki generate

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	vk "github.com/goki/vulkan"
	"goki.dev/colors"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/ordmap"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
	"goki.dev/vgpu/v2/vphong"
)

// Set Update3DTrace to true to get a trace of 3D updating
var Update3DTrace = false

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own vgpu.RenderFrame.
// The Image of this Frame is usable directly or, via gi3v.Scene3D,
// where it is copied into an overall gi.Scene image.
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
//
//goki:no-new
type Scene struct {
	ki.Node

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

// NewScene creates a new Scene to contain a 3D scenegraph.
func NewScene(name string) *Scene {
	sc := &Scene{}
	sc.Defaults()
	sc.InitName(sc, name)
	return sc
}

// Update is a global update of everything: Init3D and re-render
func (sc *Scene) Update() {
	updt := sc.UpdateStart()
	sc.Config()
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
		ni, _ := AsNode(k)
		if !ni.IsVisible() {
			return ki.Break
		}
		err := ni.Validate()
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

//////////////////////////////////////////////////////////////////
//  Flags

// ScFlags has critical state information signaling when rendering,
// styling etc need to be done
type ScFlags int64 //enums:bitflag -trim-prefix Sc

const (
	// ScUpdating means scene is in the process of updating:
	// set for any kind of tree-level update.
	// skip any further update passes until it goes off.
	ScUpdating ScFlags = ScFlags(ki.FlagsN) + iota

	// ScNeedsRender means nodes have flagged that they need a Render
	// update.
	ScNeedsRender
)

func (sc *Scene) SetSize(sz image.Point) *Scene {
	if sz.X == 0 || sz.Y == 0 {
		return sc
	}
	if sc.Frame != nil {
		csz := sc.Frame.Format.Size
		if csz == sz {
			sc.Geom.Size = sz // make sure
			return sc
		}
	}
	if sc.Frame != nil {
		sc.Frame.SetSize(sz)
	}
	sc.Geom.Size = sz // make sure
	// fmt.Printf("vp %v resized to: %v, bounds: %v\n", vp.Path(), nwsz, vp.Pixels.Bounds())
	return sc
}

func (sc *Scene) Destroy() {
	// todo: delete Phong, Frame
}

// ConfigFrameFromDrawer configures framebuffer for GPU rendering
// Using GPU and Device from a vgpu vdraw.Drawer.
func (sc *Scene) ConfigFrameFromDrawer(drw *vdraw.Drawer) {
	sf := drw.Surf
	sc.ConfigFrame(sf.GPU, &sf.Device)
}

// ConfigFrame configures framebuffer for GPU rendering,
// using given gpu and device, and size set in Geom.Size.
// Must be called on the main thread.
// If Frame already exists, it ensures that the Size is correct.
func (sc *Scene) ConfigFrame(gpu *vgpu.GPU, dev *vgpu.Device) {
	sz := sc.Geom.Size
	if sz == (image.Point{}) {
		sz = image.Point{480, 320}
	}
	if sc.Frame == nil {
		sc.Frame = vgpu.NewRenderFrame(gpu, dev, sz)
		sc.Frame.Format.SetMultisample(sc.MultiSample)
		sy := &sc.Phong.Sys
		sy.InitGraphics(gpu, "vphong.Phong", dev)
		sy.ConfigRenderNonSurface(&sc.Frame.Format, vgpu.Depth32)
		sc.Frame.SetRender(&sy.Render)
		sc.Phong.ConfigSys()
		if sc.Wireframe {
			sy.SetRasterization(vk.PolygonModeLine, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
		} else {
			sy.SetRasterization(vk.PolygonModeFill, vk.CullModeNone, vk.FrontFaceCounterClockwise, 1.0)
		}
	} else {
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
	}
}

// Image returns the current rendered image from RenderFrame
func (sc *Scene) Image() (*image.RGBA, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("gi3d.Scene Image: Scene does not have a Frame")
	}
	sy := &sc.Phong.Sys
	tcmd := sy.MemCmdStart()
	fr.GrabImage(tcmd, 0)
	sy.MemCmdEndSubmitWaitFree()
	img, err := fr.Render.Grab.DevGoImage()
	if err == nil {
		return img, err
	}
	return nil, err
}

// SolidsIntersectingPoint finds all the solids that contain given 2D window coordinate
func (sc *Scene) SolidsIntersectingPoint(pos image.Point) []Node {
	var objs []Node
	for _, kid := range sc.Kids {
		kii, _ := AsNode(kid)
		if kii == nil {
			continue
		}
		kii.WalkPre(func(k ki.Ki) bool {
			ni, _ := AsNode(k)
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
