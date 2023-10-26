// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image"
	"sort"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
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

// IsConfiged Returns true if the scene has already been configured
func (sc *Scene) IsConfiged() bool {
	return sc.Frame != nil
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
				return ki.Break // going into a different type of thing, bail
			}
			return ki.Continue
		},
			func(k ki.Ki) bool {
				ni, _ := AsNode(k)
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
	sc.UpdateMVPMatrix()
}

// Render renders the scene to the Frame framebuffer.
// Returns false if currently already rendering.
func (sc *Scene) Render() bool {
	if sc.Is(ScUpdating) || sc.Frame == nil {
		return false
	}
	if len(sc.SavedCams) == 0 {
		sc.SaveCamera("default")
	}
	sc.SetFlag(true, ScUpdating)
	sc.Camera.UpdateMatrix()
	// sc.TrackCamera()
	sc.RenderImpl()
	sc.SetFlag(false, ScUpdating)
	return true
}

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
		ni, nb := AsNode(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		if !ni.IsVisible() || nb.ScBBox == (image.Rectangle{}) {
			return ki.Break
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
