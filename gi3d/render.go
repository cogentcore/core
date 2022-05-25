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

// ConfigFrame configures framebuffer for GPU rendering context
// returns false if not possible
func (sc *Scene) ConfigFrame() bool {
	if sc.Frame == nil {
		oswin.TheApp.RunOnMain(func() {
			drw := sc.Win.OSWin.Drawer()
			sf := drw.Surf
			sz := sc.Geom.Size
			if sz == image.ZP {
				sz = image.Point{480, 320}
			}
			sc.Frame = vgpu.NewRenderFrame(sf.GPU, &sf.Device, sz)
			sy := &sc.Phong.Sys
			sy.InitGraphics(sf.GPU, "vphong.Phong", &sf.Device)
			sy.ConfigRenderNonSurface(&sc.Frame.Format, vgpu.Depth32)
			sc.Frame.SetRender(&sy.Render)
			sc.Phong.ConfigSys()
			sy.SetRasterization(vk.PolygonModeFill, vk.CullModeBackBit, vk.FrontFaceCounterClockwise, 1.0)
		})
	} else {
		sc.Frame.SetSize(sc.Geom.Size) // nop if same
	}
	sc.Camera.CamMu.Lock()
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	sc.Camera.CamMu.Unlock()
	clr := mat32.NewVec3Color(sc.BgColor)
	sc.Frame.Render.SetClearColor(clr.X, clr.Y, clr.Z, 1)
	// gpu.Draw.Wireframe(sc.Wireframe)
	return true
}

// DeleteResources deletes all GPU resources -- sets context and runs on main.
// This is called during Disconnect and before the window is closed.
func (sc *Scene) DeleteResources() {
	oswin.TheApp.RunOnMain(func() {
		sc.Phong.Destroy()
		sc.Frame.Destroy()
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
	sc.Phong.Config()
	sc.SetMeshes()
}

func (sc *Scene) Init3D() {
	if sc.Camera.FOV == 0 {
		sc.Defaults()
	}
	sc.UpdateWorldMatrix()
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
	sc.ConfigRender()
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
	if !sc.ConfigFrame() {
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
	sc.Render3D(false) // not offscreen

	drw := sc.Win.OSWin.Drawer()
	drw.SetFrameImage(sc.DirUpIdx, sc.Frame.Frames[0])
	sc.Win.DirDraws.Nodes.Order[sc.DirUpIdx-sc.Win.DirDraws.StartIdx].Val = sc.WinBBox
	drw.SyncImages()
	sc.ClearFlag(int(Rendering))
	return true
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

	sc.Phong.SetViewPrjn(&sc.Camera.ViewMatrix, &sc.Camera.VkPrjnMatrix)

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

	sy := &sc.Phong.Sys
	cmd := sy.CmdPool.Buff
	descIdx := 0
	sy.ResetBeginRenderPass(cmd, sc.Frame.Frames[0], descIdx)

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

		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				lastrc = rc
			}
			obj.Render3D(sc)
		}
	}

	sy.EndRenderPass(cmd)
	sc.Frame.SubmitRender(cmd) // this is where it waits for the 16 msec
	sc.Frame.WaitForRender()

}
