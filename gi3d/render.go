// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
	"github.com/goki/vgpu/vgpu"

	vk "github.com/goki/vulkan"
)

// RenderClasses define the different classes of rendering
type RenderClasses int32

const (
	RClassNone          RenderClasses = iota
	RClassOpaqueTexture               // textures tend to be in background
	RClassOpaqueUniform
	RClassOpaqueVertex
	RClassTransTexture
	RClassTransUniform
	RClassTransVertex
	RenderClassesN
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
	if sc.Win == nil {
		return false
	}
	drw := sc.Win.OSWin.Drawer()
	sf := drw.Surf
	newFrame := sc.ConfigFrameImpl(sf.GPU, &sf.Device)
	if newFrame {
		sc.Win.Phongs = append(sc.Win.Phongs, &sc.Phong) // for destroying in sequence
		sc.Win.Frames = append(sc.Win.Frames, sc.Frame)  // for destroying in sequence
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
		oswin.TheApp.RunOnMain(func() {
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
	clr := mat32.NewVec3Color(sc.BgColor).SRGBToLinear()
	sc.Frame.Render.SetClearColor(clr.X, clr.Y, clr.Z, 1)
	// gpu.Draw.Wireframe(sc.Wireframe)
	return wasConfig
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
			func(k ki.Ki, level int, d any) bool {
				nii, _ := KiToNode3D(k)
				if nii == nil {
					return ki.Break // going into a different type of thing, bail
				}
				return ki.Continue
			},
			func(k ki.Ki, level int, d any) bool {
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
		kii.FuncDownMeFirst(0, kii.This(), func(k ki.Ki, level int, d any) bool {
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

	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
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

// ConfigRender configures all the rendering elements: Phong system and frame
func (sc *Scene) ConfigRender() {
	sc.ConfigFrame()
	oswin.TheApp.RunOnMain(func() {
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
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
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
	sc.ConfigRender()
}

func (sc *Scene) Style3D() {
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
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
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
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
	sc.SetFlag(int(Rendering))
	sc.RenderImpl(false) // not offscreen

	drw := sc.Win.OSWin.Drawer()
	drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
	sc.Win.DirDraws.SetWinBBox(sc.DirUpIdx, sc.WinBBox)
	drw.SyncImages()
	sc.ClearFlag(int(Rendering))
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
	sc.SetFlag(int(Rendering))
	sc.RenderImpl(true) // yes offscreen
	sc.ClearFlag(int(Rendering))
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
	return sc.HasFlag(int(Rendering))
}

func (sc *Scene) DirectWinUpload() {
	if !sc.IsVisible() {
		return
	}
	if Update3DTrace {
		fmt.Printf("3D Update: Window %s from Scene: %s at: %v\n", sc.Win.Nm, sc.Nm, sc.WinBBox.Min)
	}
	sc.Render()
}

// Render3D renders the scene to the framebuffer
// all scene-level resources must be initialized and activated at this point
func (sc *Scene) Render3D(offscreen bool) {
	sc.Phong.UpdtMu.Lock()
	sc.Phong.SetViewPrjn(&sc.Camera.ViewMatrix, &sc.Camera.VkPrjnMatrix)
	sc.Phong.UpdtMu.Unlock()
	sc.Phong.Sync()

	var rcs [RenderClassesN][]Node3D
	sc.FuncDownMeFirst(0, sc.This(), func(k ki.Ki, level int, d any) bool {
		if k == sc.This() {
			return ki.Continue
		}
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		if !offscreen {
			ni.BBoxMu.RLock()
			if ni.IsInvisible() || ni.ObjBBox == (image.Rectangle{}) { // objbbox is intersection of scene and obj
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
