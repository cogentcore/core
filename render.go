// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"sort"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

// render notes:
//
// # Config:
//
// Unlike gi, gi3d Config can only be successfully done after the
// GPU framework has been initialized, because it is all about allocating
// GPU resources.
// The Scene ScNeedsConfig flag indicates if any of these resources
// have been changed and a new Config is required.
// * ConfigFrame -- allocates the renderframe -- based on Geom.Size
// * Lights, Meshes, Textures
// * ConfigNodes to update and validate node-specific settings.
//   Most nodes do not require this, but Text2D and Embed2D do.
//
// # Update:
//
// Update involves updating Pose

// DoUpdate handles needed updates based on Scene Flags.
// if ScUpdating is set, then an update is in progress and false is returned.
// If no updates are required, then false is also returned, else true.
// ScNeedsConfig is NOT handled here because it must be done on main thread,
// so this must be checked separately (e.g., in gi3dv.Scene3D, it is requires
// a separate RunOnMainThread call).
func (sc *Scene) DoUpdate() bool {
	if sc.Is(ScUpdating) {
		return false
	}
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	switch {
	case sc.Is(ScNeedsUpdate):
		sc.UpdateNodes()
		sc.Render()
		sc.SetFlag(false, ScNeedsUpdate, ScNeedsRender)
	case sc.Is(ScNeedsRender):
		sc.Render()
		sc.SetFlag(false, ScNeedsRender)
	default:
		return false
	}
	return true
}

// IsConfiged Returns true if the scene has already been configured
func (sc *Scene) IsConfiged() bool {
	return sc.Frame != nil
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

// Image returns the current rendered image from the Frame RenderFrame.
// This version returns a direct pointer to the underlying host version of
// the GPU image, and should only be used immediately (for saving or writing
// to another image).  You must call ImageDone() when done with the image.
// See [ImageCopy] for a version that returns a copy of the image, which
// will be usable until the next call to ImageCopy.
func (sc *Scene) Image() (*image.RGBA, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("gi3d.Scene Image: Scene does not have a Frame")
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

// ImageDone must be called when done using the image returned by [Image].
func (sc *Scene) ImageDone() {
	if sc.Frame == nil {
		return
	}
	sc.Frame.Render.Grab.UnmapDev()
}

// ImageCopy returns a copy of the current rendered image
// from the Frame RenderFrame. A re-used image.RGBA is returned.
// This same image is used across calls to avoid large memory allocations,
// so it will automatically update after the next ImageCopy call.
// The underlying image is in the [ImgCopy] field.
// If a persistent image is required, call [glop/images.CloneAsRGBA].
func (sc *Scene) ImageCopy() (*image.RGBA, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, fmt.Errorf("gi3d.Scene ImageCopy: Scene does not have a Frame")
	}
	sy := &sc.Phong.Sys
	tcmd := sy.MemCmdStart()
	fr.GrabImage(tcmd, 0) // note: re-uses a persistent Grab image
	sy.MemCmdEndSubmitWaitFree()
	err := fr.Render.Grab.DevGoImageCopy(&sc.ImgCopy)
	if err == nil {
		return &sc.ImgCopy, err
	}
	return nil, err
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sc *Scene) UpdateMeshBBox() {
	for _, kid := range sc.Kids {
		kii, _ := AsNode(kid)
		if kii == nil {
			continue
		}
		kii.WalkPost(func(k ki.Ki) bool {
			ni, _ := AsNode(k)
			if ni == nil {
				return ki.Break
			}
			return ki.Continue
		},
			func(k ki.Ki) bool {
				ni, _ := AsNode(k)
				if ni == nil {
					return ki.Break
				}
				ni.UpdateMeshBBox()
				return ki.Continue
			})
	}
}

// UpdateWorldMatrix updates the world matrix for all scene elements.
func (sc *Scene) UpdateWorldMatrix() {
	idmtx := mat32.NewMat4()
	for _, kid := range sc.Kids { //
		kii, _ := AsNode(kid)
		if kii == nil {
			continue
		}
		kii.UpdateWorldMatrix(idmtx)
		kii.WalkPre(func(k ki.Ki) bool {
			if k == kid {
				return ki.Continue // skip, already did
			}
			ni, _ := AsNode(k)
			if ni == nil {
				return ki.Break // going into a different type of thing, bail
			}
			pi, pd := AsNode(k.Parent())
			if pi == nil {
				return ki.Break
			}
			pd.PoseMu.RLock()
			ni.UpdateWorldMatrix(&pd.Pose.WorldMatrix)
			pd.PoseMu.RUnlock()
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

	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		ni.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.PrjnMatrix)
		ni.UpdateBBox2D(size, sc)
		return ki.Continue
	})
}

// ConfigNodes runs Config on all nodes
func (sc *Scene) ConfigNodes() {
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode(k)
		if ni == nil {
			return ki.Break
		}
		ni.Config(sc)
		return ki.Continue
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
	if sc.Camera.FOV == 0 {
		sc.Defaults()
	}
	sc.Camera.CamMu.Lock()
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	sc.Camera.CamMu.Unlock()
	clr := mat32.NewVec3Color(sc.BackgroundColor).SRGBToLinear()
	sc.Frame.Render.SetClearColor(clr.X, clr.Y, clr.Z, 1)
	// gpu.Draw.Wireframe(sc.Wireframe)
	sc.UpdateWorldMatrix()
	sc.ConfigLights()
	sc.ConfigMeshesTextures()
	sc.ConfigNodes()
	sc.SetFlag(false, ScNeedsConfig)
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
	sc.UpdateWorldMatrix()
	sc.UpdateMeshBBox()
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
	sc.SetFlag(true, ScUpdating)
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateMVPMatrix()
	sc.RenderImpl()
	sc.SetFlag(false, ScUpdating)
	return true
}

////////////////////////////////////////////////////////////////////
// 	RenderImpl

// RenderClasses define the different classes of rendering
type RenderClasses int32 //enums:enum -trimprefix RClass

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
	sc.Phong.UpdtMu.Lock()
	sc.Phong.SetViewPrjn(&sc.Camera.ViewMatrix, &sc.Camera.VkPrjnMatrix)
	sc.Phong.UpdtMu.Unlock()
	sc.Phong.Sync()

	var rcs [RenderClassesN][]Node
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		// if !ni.IsVisible() || nb.ScBBox == (image.Rectangle{}) {
		// 	return ki.Break
		// }
		if !ni.IsSolid() {
			return ki.Continue
		}
		rc := ni.RenderClass()
		if rc > RClassTransTexture { // all in one group b/c z-sorting is key
			rc = RClassTransTexture
		}
		rcs[rc] = append(rcs[rc], ni)
		return ki.Continue
	})

	sc.Phong.UpdtMu.Lock()
	sy := &sc.Phong.Sys
	cmd := sy.CmdPool.Buff
	descIdx := 0
	sy.ResetBeginRenderPass(cmd, sc.Frame.Frames[0], descIdx)
	sc.Phong.UpdtMu.Unlock()

	for rci, objs := range rcs {
		rc := RenderClasses(rci)
		if len(objs) == 0 {
			continue
		}
		if rc >= RClassTransTexture { // sort back-to-front for transparent
			sort.Slice(objs, func(i, j int) bool {
				return objs[i].NormDCBBox().Min.Z > objs[j].NormDCBBox().Min.Z
			})
		} else { // sort front-to-back for opaque to allow "early z rejection"
			sort.Slice(objs, func(i, j int) bool {
				return objs[i].NormDCBBox().Min.Z < objs[j].NormDCBBox().Min.Z
			})
		}
		// fmt.Printf("\nRender class: %v\n", rc)
		// for i := range objs {
		// 	fmt.Printf("obj: %s  max z: %g   min z: %g\n", objs[i].Name(), objs[i].AsNode().NDCBBox.Max.Z, objs[i].AsNode().NDCBBox.Min.Z)
		// }

		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				lastrc = rc
			}
			obj.Render(sc)
		}
	}
	sc.Phong.UpdtMu.Lock()
	sy.EndRenderPass(cmd)
	sc.Frame.SubmitRender(cmd) // this is where it waits for the 16 msec
	sc.Frame.WaitForRender()
	sc.Phong.UpdtMu.Unlock()
}
