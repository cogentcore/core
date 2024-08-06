// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"sort"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/vgpu"
)

// render notes:
//
// # Config:
//
// Unlike core, xyz Config can only be successfully done after the
// GPU framework has been initialized, because it is all about allocating
// GPU resources.
// [Scene.NeedsConfig] indicates if any of these resources
// have been changed and a new Config is required.
// * ConfigFrame -- allocates the rendertarget -- based on Geom.Size
// * Lights, Meshes, Textures
// * ConfigNodes to update and validate node-specific settings.
//   Most nodes do not require this, but Text2D and Embed2D do.
//
// # Update:
//
// Update involves updating Pose

// DoUpdate handles needed updates based on Scene Flags.
// If no updates are required, then false is returned, else true.
// NeedsConfig is NOT handled here because it must be done on main thread,
// so this must be checked separately (e.g., in xyzcore.Scene, as it requires
// a separate RunOnMainThread call).
func (sc *Scene) DoUpdate() bool {
	switch {
	case sc.NeedsUpdate:
		sc.UpdateNodes()
		sc.Render()
		sc.NeedsUpdate = false
		sc.NeedsRender = false
	case sc.NeedsRender:
		sc.Render()
		sc.NeedsRender = false
	default:
		return false
	}
	return true
}

// SetNeedsRender sets [Scene.NeedsRender] to true.
func (sc *Scene) SetNeedsRender() {
	sc.NeedsRender = true
}

// SetNeedsUpdate sets [Scene.SetNeedsUpdate] to true.
func (sc *Scene) SetNeedsUpdate() {
	sc.NeedsUpdate = true
}

// SetNeedsConfig sets [Scene.SetNeedsConfig] to true.
func (sc *Scene) SetNeedsConfig() {
	sc.NeedsConfig = true
}

// UpdateNodesIfNeeded can be called to update prior to an ad-hoc render
// if the NeedsUpdate flag has been set (resets flag)
func (sc *Scene) UpdateNodesIfNeeded() {
	if sc.NeedsUpdate {
		sc.UpdateNodes()
		sc.NeedsUpdate = false
	}
}

// ConfigFrameFromSurface configures framebuffer for GPU rendering
// Using GPU and Device from given vgpu.Surface
func (sc *Scene) ConfigFrameFromSurface(surf *vgpu.Surface) {
	sc.ConfigFrame(surf.GPU, &surf.Device)
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
		sc.Frame = vgpu.NewRenderTexture(gpu, dev, sz)
		sc.Frame.Format.SetMultisample(sc.MultiSample)
		sy := sc.Phong.Sys
		sy.InitGraphics(gpu, "vphong.Phong", dev)
		sy.ConfigRenderNonSurface(&sc.Frame.Format, vgpu.Depth32)
		sc.Frame.SetRender(&sy.Render)
		sc.Phong.ConfigSys()
	} else {
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
	}
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
}

// Image returns the current rendered image from the Frame RenderTexture.
// This version returns a direct pointer to the underlying host version of
// the GPU image, and should only be used immediately (for saving or writing
// to another image).  You must call ImageDone() when done with the image.
// See [ImageCopy] for a version that returns a copy of the image, which
// will be usable until the next call to ImageCopy.
func (sc *Scene) Image() (*image.RGBA, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("xyz.Scene Image: Scene does not have a Frame")
	}
	sy := &sc.Phong.Sys
	tcmd := sy.MemCmdStart()
	fr.GrabImage(tcmd, 0) // note: re-uses a persistent Grab image
	sy.MemCmdEndSubmitWaitFree()
	img, err := fr.Render.Grab.DevGoImage()
	if err == nil {
		return img, err
	}
	return nil, err
}

// ImageDone must be called when done using the image returned by [Scene.Image].
func (sc *Scene) ImageDone() {
	if sc.Frame == nil {
		return
	}
	sc.Frame.Render.Grab.UnmapDev()
}

// ImageCopy returns a copy of the current rendered image
// from the Frame RenderTexture. A re-used image.RGBA is returned.
// This same image is used across calls to avoid large memory allocations,
// so it will automatically update after the next ImageCopy call.
// The underlying image is in the [ImgCopy] field.
// If a persistent image is required, call [imagex.CloneAsRGBA].
func (sc *Scene) ImageCopy() (*image.RGBA, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("xyz.Scene ImageCopy: Scene does not have a Frame")
	}
	sy := &sc.Phong.Sys
	tcmd := sy.MemCmdStart()
	fr.GrabImage(tcmd, 0) // note: re-uses a persistent Grab image
	sy.MemCmdEndSubmitWaitFree()
	err := fr.Render.Grab.DevGoImageCopy(&sc.imgCopy)
	if err == nil {
		return &sc.imgCopy, err
	}
	return nil, err
}

// ImageUpdate configures, updates, and renders the scene, then returns [Scene.Image].
func (sc *Scene) ImageUpdate() (*image.RGBA, error) {
	sc.Config()
	sc.UpdateNodes()
	sc.Render()
	return sc.Image()
}

// AssertImage asserts the [Scene.Image] at the given filename using [imagex.Assert].
// It first configures, updates, and renders the scene.
func (sc *Scene) AssertImage(t imagex.TestingT, filename string) {
	img, err := sc.ImageUpdate()
	if err != nil {
		t.Errorf("xyz.Scene.AssertImage: error getting image: %w", err)
		return
	}
	imagex.Assert(t, img, filename)
	sc.ImageDone()
}

// DepthImage returns the current rendered depth image
func (sc *Scene) DepthImage() ([]float32, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("xyz.Scene DepthImage: Scene does not have a Frame")
	}
	sy := &sc.Phong.Sys
	tcmd := sy.MemCmdStart()
	fr.GrabDepthImage(tcmd)
	sy.MemCmdEndSubmitWaitFree()
	depth, err := fr.Render.DepthImageArray()
	if err == nil {
		return depth, err
	}
	return nil, err
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sc *Scene) UpdateMeshBBox() {
	for _, c := range sc.Children {
		cn, _ := AsNode(c)
		if cn == nil {
			continue
		}
		cn.AsTree().WalkDownPost(func(k tree.Node) bool {
			ni, _ := AsNode(k)
			if ni == nil {
				return tree.Break
			}
			return tree.Continue
		}, func(k tree.Node) bool {
			ni, _ := AsNode(k)
			if ni == nil {
				return tree.Break
			}
			ni.UpdateMeshBBox()
			return tree.Continue
		})
	}
}

// UpdateWorldMatrix updates the world matrix for node and everything inside it
func UpdateWorldMatrix(n tree.Node) {
	idmtx := math32.Identity4()
	n.AsTree().WalkDown(func(cn tree.Node) bool {
		ni, _ := AsNode(cn)
		if ni == nil {
			return tree.Continue
		}
		_, pd := AsNode(cn.AsTree().Parent)
		if pd == nil {
			ni.UpdateWorldMatrix(idmtx)
		} else {
			ni.UpdateWorldMatrix(&pd.Pose.WorldMatrix)
		}
		return tree.Continue
	})
}

// UpdateMVPMatrix updates the Model-View-Projection matrix for all scene elements
// and BBox2D
func (sc *Scene) UpdateMVPMatrix() {
	sc.Camera.Pose.UpdateMatrix()
	sz := sc.Geom.Size
	size := math32.Vec2(float32(sz.X), float32(sz.Y))

	sc.WalkDown(func(cn tree.Node) bool {
		if cn == sc.This {
			return tree.Continue
		}
		n, nb := AsNode(cn)
		if n == nil {
			return tree.Break // going into a different type of thing, bail
		}
		nb.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.ProjectionMatrix)
		nb.UpdateBBox2D(size)
		return tree.Continue
	})
}

// ConfigNodes runs Config on all nodes
func (sc *Scene) ConfigNodes() {
	sc.WalkDown(func(k tree.Node) bool {
		if k == sc.This {
			return tree.Continue
		}
		ni, _ := AsNode(k)
		if ni == nil {
			return tree.Break
		}
		ni.Config()
		return tree.Continue
	})
}

// Config configures the Scene to prepare for rendering.
// The Frame should already have been configured.
// This includes the Phong system and frame.
// It must be called before the first render, or after
// any change in the lights, meshes, textures, or any
// changes to the nodes that require Config updates.
// This must be called on the main thread.
func (sc *Scene) Config() {
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	clr := math32.NewVector3Color(colors.ToUniform(sc.Background)).SRGBToLinear()
	sc.Frame.Render.SetClearColor(clr.X, clr.Y, clr.Z, 1)
	// gpu.Draw.Wireframe(sc.Wireframe)
	sc.ConfigNodes()
	UpdateWorldMatrix(sc.This)
	sc.ConfigLights()
	sc.ConfigMeshesTextures()
	sc.NeedsConfig = false
	sc.NeedsUpdate = true
}

// ConfigMeshesTextures configures the meshes and the textures to the Phong
// rendering system.  Called by ConfigRender -- can be called
// separately if just these elements are updated -- see also ReconfigMeshes
// and ReconfigTextures
func (sc *Scene) ConfigMeshesTextures() {
	sc.ConfigMeshes()
	sc.ConfigTextures()
	sc.Phong.Wireframe = sc.Wireframe
	sc.Phong.Config()
	sc.SetMeshes()
}

func (sc *Scene) UpdateNodes() {
	UpdateWorldMatrix(sc.This)
	sc.UpdateMeshBBox()
}

// TrackCamera -- a Group at the top-level named "TrackCamera"
// will automatically track the camera (i.e., its Pose is copied).
// Solids in that group can set their relative Pos etc to display
// relative to the camera, to achieve "first person" effects.
func (sc *Scene) TrackCamera() bool {
	tci := sc.ChildByName("TrackCamera", 0)
	if tci == nil {
		return false
	}
	tc, ok := tci.(*Group)
	if !ok {
		return false
	}
	tc.TrackCamera()
	sc.SetNeedsUpdate() // need to update world model for nodes
	return true
}

// Render renders the scene to the Frame framebuffer.
// Only the Camera pose view matrix is updated here.
// If nodes require their own pose etc updates, UpdateNodes
// must be called prior to render.
// Returns false if currently already rendering.
func (sc *Scene) Render() bool {
	if sc.Frame == nil {
		return false
	}
	if len(sc.SavedCams) == 0 {
		sc.SaveCamera("default")
	}
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateMVPMatrix()
	sc.RenderImpl()
	return true
}

////////////////////////////////////////////////////////////////////
// 	RenderImpl

// RenderClasses define the different classes of rendering
type RenderClasses int32 //enums:enum -trim-prefix RClass

const (
	RClassNone          RenderClasses = iota
	RClassOpaqueTexture               // textures tend to be in background
	RClassOpaqueUniform
	RClassOpaqueVertex
	RClassTransTexture
	RClassTransUniform
	RClassTransVertex
)

// RenderImpl renders the scene to the framebuffer.
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) RenderImpl() {
	sc.Phong.UpdateMu.Lock()
	sc.Phong.SetViewProjection(&sc.Camera.ViewMatrix, &sc.Camera.VkProjectionMatrix)
	sc.Phong.UpdateMu.Unlock()
	sc.Phong.Sync()

	var rcs [RenderClassesN][]Node
	sc.WalkDown(func(k tree.Node) bool {
		if k == sc.This {
			return tree.Continue
		}
		ni, _ := AsNode(k)
		if ni == nil {
			return tree.Break // going into a different type of thing, bail
		}
		// if !ni.IsVisible() || nb.ScBBox == (image.Rectangle{}) {
		// 	return tree.Break
		// }
		if !ni.IsSolid() {
			return tree.Continue
		}
		rc := ni.RenderClass()
		if rc > RClassTransTexture { // all in one group b/c z-sorting is key
			rc = RClassTransTexture
		}
		rcs[rc] = append(rcs[rc], ni)
		return tree.Continue
	})

	sc.Phong.UpdateMu.Lock()
	sy := &sc.Phong.Sys
	cmd := sy.CmdPool.Buff
	descIndex := 0
	sy.ResetBeginRenderPass(cmd, sc.Frame.Frames[0], descIndex)
	sc.Phong.UpdateMu.Unlock()

	for rci, objs := range rcs {
		rc := RenderClasses(rci)
		if len(objs) == 0 {
			continue
		}
		if rc >= RClassTransTexture { // sort back-to-front for transparent
			sort.Slice(objs, func(i, j int) bool { // TODO: use slices.SortFunc here and everywhere else we use sort.Slice
				return objs[i].AsNodeBase().NormDCBBox().Min.Z > objs[j].AsNodeBase().NormDCBBox().Min.Z
			})
		} else { // sort front-to-back for opaque to allow "early z rejection"
			sort.Slice(objs, func(i, j int) bool {
				return objs[i].AsNodeBase().NormDCBBox().Min.Z < objs[j].AsNodeBase().NormDCBBox().Min.Z
			})
		}
		// fmt.Printf("\nRender class: %v\n", rc)
		// for i := range objs {
		// 	fmt.Printf("obj: %s  max z: %g   min z: %g\n", objs[i].Name, objs[i].AsNode().NDCBBox.Max.Z, objs[i].AsNode().NDCBBox.Min.Z)
		// }

		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				lastrc = rc
			}
			obj.Render()
		}
	}
	sc.Phong.UpdateMu.Lock()
	sy.EndRenderPass(cmd)
	sc.Frame.SubmitRender(cmd) // this is where it waits for the 16 msec
	sc.Frame.WaitForRender()
	sc.Phong.UpdateMu.Unlock()
}
