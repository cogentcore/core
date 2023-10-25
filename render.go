// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image"
	"sort"

	"goki.dev/goosi"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

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

/////////////////////////////////////////////////////////////////////////////////////
// 		Rendering

// IsConfiged Returns true if the scene has already been configured
func (sc *Scene) IsConfiged() bool {
	return sc.Frame != nil
}

// ConfigFrame configures framebuffer for GPU rendering context
// returns false if not possible
// designed to be called prior to each render, to ensure ready.
func (sc *Scene) ConfigFrame() bool {
	// drw := sc.Win.OSWin.Drawer()
	var drw vdraw.Drawer
	sf := drw.Surf
	newFrame := sc.ConfigFrameImpl(sf.GPU, &sf.Device)
	if newFrame {
		// todo:
		// sc.Win.Phongs = append(sc.Win.Phongs, &sc.Phong) // for destroying in sequence
		// sc.Win.Frames = append(sc.Win.Frames, sc.Frame)  // for destroying in sequence
	}
	return true
}

// ConfigFrameImpl configures framebuffer for GPU rendering,
// using given gpu and device.
// designed to be called prior to each render, to ensure ready.
// returns true if the frame was nil and thus configured.
func (sc *Scene) ConfigFrameImpl(gpu *vgpu.GPU, dev *vgpu.Device) bool {
	wasConfig := false
	if sc.Frame == nil {
		wasConfig = true
		goosi.TheApp.RunOnMain(func() {
			sz := sc.Geom.Size
			if sz == (image.Point{}) {
				sz = image.Point{480, 320}
			}
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
		})
	} else {
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
	}
	sc.Camera.CamMu.Lock()
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	sc.Camera.CamMu.Unlock()
	clr := mat32.NewVec3Color(sc.BackgroundColor).SRGBToLinear()
	sc.Frame.Render.SetClearColor(clr.X, clr.Y, clr.Z, 1)
	// gpu.Draw.Wireframe(sc.Wireframe)
	return wasConfig
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sc *Scene) UpdateMeshBBox() {
	for _, kid := range sc.Kids {
		kii, _ := AsNode3D(kid)
		if kii == nil {
			continue
		}
		kii.WalkPost(func(k ki.Ki) bool {
			ni, _ := AsNode3D(k)
			if ni == nil {
				return ki.Break // going into a different type of thing, bail
			}
			return ki.Continue
		},
			func(k ki.Ki) bool {
				ni, _ := AsNode3D(k)
				if ni == nil {
					return ki.Break // going into a different type of thing, bail
				}
				ni.UpdateMeshBBox()
				return ki.Continue
			})
	}
}

// UpdateWorldMatrix updates the world matrix for all scene elements
// called during Init3D and rendering
func (sc *Scene) UpdateWorldMatrix() {
	idmtx := mat32.NewMat4()
	for _, kid := range sc.Kids {
		kii, _ := AsNode3D(kid)
		if kii == nil {
			continue
		}
		kii.UpdateWorldMatrix(idmtx)
		kii.WalkPre(func(k ki.Ki) bool {
			if k == kid {
				return ki.Continue // skip, already did
			}
			ni, _ := AsNode3D(k)
			if ni == nil {
				return ki.Break // going into a different type of thing, bail
			}
			pi, pd := AsNode3D(k.Parent())
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
		ni, _ := AsNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		ni.UpdateMVPMatrix(&sc.Camera.ViewMatrix, &sc.Camera.PrjnMatrix)
		ni.UpdateBBox2D(size, sc)
		return ki.Continue
	})
}

// ConfigRender configures all the rendering elements: Phong system and frame
func (sc *Scene) ConfigRender() {
	sc.ConfigFrame()
	goosi.TheApp.RunOnMain(func() {
		sc.ConfigLights()
		sc.ConfigMeshesTextures()
	})
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

func (sc *Scene) Init3D() {
	if sc.Camera.FOV == 0 {
		sc.Defaults()
	}
	sc.UpdateWorldMatrix()
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		ni.Init3D(sc)
		return ki.Continue
	})
	sc.Style3D()
	sc.ConfigRender()
}

func (sc *Scene) Style3D() {
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		ni.Style3D(sc)
		return ki.Continue
	})
}

func (sc *Scene) UpdateNodes3D() {
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, _ := AsNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		ni.UpdateNode(sc)
		return ki.Continue
	})
}

// Render renders the scene to the Frame framebuffer,
// and uploads to the window direct draw image texture.
// for onscreen rendering.
// returns false if currently already rendering.
func (sc *Scene) Render() bool {
	if sc.IsRendering() {
		return false
	}
	if !sc.ConfigFrame() {
		return false
	}
	sc.RenderMu.Lock()
	if len(sc.SavedCams) == 0 {
		sc.SaveCamera("default")
	}
	// sc.SetFlag(true, Rendering)
	sc.RenderImpl(false) // not offscreen

	var drw vdraw.Drawer
	// sc.Win.OSWin.Drawer()
	drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
	// sc.Win.DirDraws.SetWinBBox(sc.DirUpIdx, sc.WinBBox)
	drw.SyncImages()
	// sc.SetFlag(false, Rendering)
	sc.RenderMu.Unlock()
	return true
}

// RenderOffscreen does an offscreen render to the
// framebuffer, which can be accessed for its image.
// MUST call ConfigFrame / Impl on the Scene frame
// prior to rendering.
// returns false if currently already rendering.
func (sc *Scene) RenderOffscreen() bool {
	if sc.IsRendering() || sc.Frame == nil {
		return false
	}
	sc.RenderMu.Lock()
	// sc.SetFlag(true, Rendering)
	sc.RenderImpl(true) // yes offscreen
	// sc.SetFlag(false, Rendering)
	sc.RenderMu.Unlock()
	return true
}

// RenderImpl does the 3D rendering including updating
// the view / world matricies, and calling Render3D
func (sc *Scene) RenderImpl(offscreen bool) {
	sc.Camera.UpdateMatrix()
	sc.TrackCamera()
	sc.UpdateNodes3D()
	sc.UpdateWorldMatrix()
	sc.UpdateMeshBBox()
	sc.UpdateMVPMatrix()
	sc.Render3D(offscreen)
}

func (sc *Scene) IsDirectWinUpload() bool {
	return true
}

func (sc *Scene) IsRendering() bool {
	// return sc.HasFlag(Rendering)
	return false
}

func (sc *Scene) DirectWinUpload() {
	if !sc.IsVisible() {
		return
	}
	// if Update3DTrace {
	// 	fmt.Printf("3D Update: from Scene: %s at: %v\n", sc.Nm, sc.ScBBox.Min)
	// }
	// sc.Render()
}

// Render3D renders the scene to the framebuffer
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) Render3D(offscreen bool) {
	sc.Phong.UpdtMu.Lock()
	sc.Phong.SetViewPrjn(&sc.Camera.ViewMatrix, &sc.Camera.VkPrjnMatrix)
	sc.Phong.UpdtMu.Unlock()
	sc.Phong.Sync()

	var rcs [RenderClassesN][]Node
	sc.WalkPre(func(k ki.Ki) bool {
		if k == sc.This() {
			return ki.Continue
		}
		ni, nb := AsNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		if !offscreen {
			if !ni.IsVisible() || nb.ScBBox == (image.Rectangle{}) { // objbbox is intersection of scene and obj
				return ki.Break
			}
			// ni.ConnectEvents3D(sc) // only connect visible
		}
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
		// 	fmt.Printf("obj: %s  max z: %g   min z: %g\n", objs[i].Name(), objs[i].AsNode3D().NDCBBox.Max.Z, objs[i].AsNode3D().NDCBBox.Min.Z)
		// }

		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				lastrc = rc
			}
			obj.Render3D(sc)
		}
	}
	sc.Phong.UpdtMu.Lock()
	sy.EndRenderPass(cmd)
	sc.Frame.SubmitRender(cmd) // this is where it waits for the 16 msec
	sc.Frame.WaitForRender()
	sc.Phong.UpdtMu.Unlock()
}
