// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

//go:generate core generate -add-types

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/vgpu"
	"cogentcore.org/core/vgpu/vphong"
)

// Set Update3DTrace to true to get a trace of 3D updating
var Update3DTrace = false

// Scene is the overall scenegraph containing nodes as children.
// It renders to its own vgpu.RenderFrame.
// The Image of this Frame is usable directly or, via xyzview.Scene,
// where it is copied into an overall core.Scene image.
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
//core:no-new
//core:embedder
type Scene struct {
	tree.NodeBase

	// Viewport-level viewbox within any parent Viewport2D
	Geom math32.Geom2DInt `set:"-"`

	// number of samples in multisampling -- must be a power of 2, and must be 1 if grabbing the Depth buffer back from the RenderFrame
	MultiSample int `default:"4"`

	// render using wireframe instead of filled polygons -- this must be set prior to configuring the Phong rendering system (i.e., just after Scene is made)
	Wireframe bool `default:"false"`

	// camera determines view onto scene
	Camera Camera `set:"-"`

	// background color, which is used directly as an RGB color in vulkan
	BackgroundColor color.RGBA

	// all lights used in the scene
	Lights ordmap.Map[string, Light] `set:"-"`

	// meshes -- holds all the mesh data -- must be configured prior to rendering
	Meshes ordmap.Map[string, Mesh] `set:"-"`

	// textures -- must be configured prior to rendering -- a maximum of 16 textures is supported for full cross-platform portability
	Textures ordmap.Map[string, Texture] `set:"-"`

	// library of objects that can be used in the scene
	Library map[string]*Group `set:"-"`

	// don't activate the standard navigation keyboard and mouse event processing to move around the camera in the scene
	NoNav bool

	// saved cameras -- can Save and Set these to view the scene from different angles
	SavedCams map[string]Camera `set:"-"`

	// has dragging cursor been set yet?
	SetDragCursor bool `view:"-" set:"-"`

	// the vphong rendering system
	Phong vphong.Phong `set:"-"`

	// the vgpu render frame holding the rendered scene
	Frame *vgpu.RenderFrame `set:"-"`

	// image used to hold a copy of the Frame image, for ImageCopy() call.
	// This is re-used across calls to avoid large memory allocations,
	// so it will automatically update after every ImageCopy call.
	// If a persistent image is required, call [iox/imagex.CloneAsRGBA].
	ImgCopy image.RGBA `set:"-"`

	// index in list of window direct uploading images
	DirUpIndex int `set:"-"`

	// mutex on rendering
	RenderMu sync.Mutex `view:"-" copier:"-" json:"-" xml:"-" set:"-"`
}

// Defaults sets default scene params (camera, bg = white)
func (sc *Scene) Defaults() {
	sc.MultiSample = 4
	sc.Camera.Defaults()
	sc.BackgroundColor = colors.Scheme.Background
}

// NewScene creates a new Scene to contain a 3D scenegraph.
func NewScene(name ...string) *Scene {
	sc := &Scene{}
	sc.Defaults()
	sc.InitName(sc, name...)
	return sc
}

// Update is a global update of everything: config, update, and re-render
func (sc *Scene) Update() {
	sc.Config()
	sc.NeedsUpdate()
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
		return fmt.Errorf("xyz.Scene: %v saved camera of name: %v not found", sc.Nm, name)
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
	sc.WalkDown(func(k tree.Node) bool {
		if k == sc.This() {
			return tree.Continue
		}
		ni, _ := AsNode(k)
		if !ni.IsVisible() {
			return tree.Break
		}
		err := ni.Validate()
		if err != nil {
			hasError = true
		}
		return tree.Continue
	})
	if hasError {
		return fmt.Errorf("xyz.Scene: %v Validate found at least one error (see log)", sc.Path())
	}
	return nil
}

//////////////////////////////////////////////////////////////////
//  Flags

// ScFlags has critical state information signaling when rendering,
// updating, or config needs to be done
type ScFlags tree.Flags //enums:bitflag

const (
	// ScNeedsConfig means that a GPU resource (Lights, Texture, Meshes,
	// or more complex Nodes that require ConfigNodes) has been changed
	// and a Config call is required.
	ScNeedsConfig ScFlags = ScFlags(tree.FlagsN) + iota

	// ScNeedsUpdate means that Node Pose has changed and an update pass
	// is required to update matrix and bounding boxes.
	ScNeedsUpdate

	// ScNeedsRender means that something has been updated (minimally the
	// Camera pose) and a new Render is required.
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
	sc.Phong.Destroy()
	if sc.Frame != nil {
		sc.Frame.Destroy()
		sc.Frame = nil
		// fmt.Println("Phong, Frame destroyed")
	}
}

// SolidsIntersectingPoint finds all the solids that contain given 2D window coordinate
func (sc *Scene) SolidsIntersectingPoint(pos image.Point) []Node {
	var objs []Node
	for _, kid := range sc.Kids {
		kii, _ := AsNode(kid)
		if kii == nil {
			continue
		}
		kii.WalkDown(func(k tree.Node) bool {
			ni, _ := AsNode(k)
			if ni == nil {
				return tree.Break // going into a different type of thing, bail
			}
			if !ni.IsSolid() {
				return tree.Continue
			}
			// if nb.PosInWinBBox(pos) {
			objs = append(objs, ni)
			// }
			return tree.Continue
		})
	}
	return objs
}
