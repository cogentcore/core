// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"errors"
	"image"
	"sort"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// ConfigOffscreenFromSurface configures offscreen [gpu.RenderTexture]
// using GPU and Device from given [gpu.Surface].
func (sc *Scene) ConfigOffscreenFromSurface(surf *gpu.Surface) {
	sc.ConfigOffscreen(surf.GPU, surf.Device())
}

// ConfigOffscreen configures offscreen [gpu.RenderTexture]
// using given gpu and device, and size set in Geom.Size.
// If Frame already exists, it ensures that the Size is correct.
func (sc *Scene) ConfigOffscreen(gp *gpu.GPU, dev *gpu.Device) {
	sc.Lock()
	defer sc.Unlock()

	sz := sc.Geom.Size
	if sz == (image.Point{}) {
		sz = image.Point{480, 320}
	}
	if sc.Frame == nil {
		sc.Frame = gpu.NewRenderTexture(gp, dev, sz, sc.MultiSample, gpu.Depth32)
		sc.Phong = phong.NewPhong(gp, sc.Frame)
		sc.ConfigNewPhong()
	} else {
		sc.Frame.SetSize(sz) // nop if same
	}
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
}

// Rebuild updates all the data resources.
// Is only effective when the GPU render is active.
func (sc *Scene) Rebuild() {
	sc.Update()
	if !sc.IsLive() {
		return
	}
	sc.Phong.ResetAll()
	sc.ConfigNewPhong()
}

func (sc *Scene) ConfigNewPhong() {
	sc.Frame.Render().ClearColor = sc.Background.At(0, 0)
	sc.ConfigNodes()
	UpdateWorldMatrix(sc.This)
	sc.setAllLights()
	sc.setAllMeshes()
	sc.setAllTextures()
}

// UseAltFrame sets Phong to use the AltFrame [gpu.RenderTexture]
// using given size.
// If AltFrame already exists, it ensures that the Size is correct.
// Call UseMainFrame to return to Frame.
func (sc *Scene) UseAltFrame(sz image.Point) {
	sc.Lock()
	defer sc.Unlock()

	if sc.AltFrame == nil {
		sc.AltFrame = gpu.NewRenderTexture(sc.Phong.System.GPU(), sc.Phong.System.Device(), sz, sc.MultiSample, gpu.Depth32)
	} else {
		sc.AltFrame.SetSize(sz) // nop if same
	}
	sc.AltFrame.Render().ClearColor = sc.Background.At(0, 0)
	sc.Camera.Aspect = float32(sz.X) / float32(sz.Y)
	sc.Phong.System.Renderer = sc.AltFrame
}

// UseMainFrame sets Phong to return to using the Frame [gpu.RenderTexture].
func (sc *Scene) UseMainFrame() {
	sc.Lock()
	defer sc.Unlock()
	sc.Phong.System.Renderer = sc.Frame
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
}

// Render renders the scene to the Frame framebuffer.
// Returns false if currently already rendering.
func (sc *Scene) Render() bool {
	if sc.Frame == nil {
		return false
	}
	sc.render(false)
	return true
}

// RenderGrabImage renders the scene to the Frame framebuffer.
// and returns the resulting image as an [image.NRGBA]
// which could be nil if there are any issues.
// The image data is a copy and can be modified etc.
func (sc *Scene) RenderGrabImage() *image.NRGBA {
	if sc.Frame == nil {
		return nil
	}
	return sc.render(true)
}

// render renders the scene to the Frame framebuffer.
// Returns false if currently already rendering.
func (sc *Scene) render(grabImage bool) *image.NRGBA {
	sc.Lock()
	defer sc.Unlock()
	sc.Frame.Render().ClearColor = sc.Background.At(0, 0)

	if len(sc.SavedCams) == 0 {
		sc.SaveCamera("default")
	}
	sc.TrackCamera()
	UpdateWorldMatrix(sc.This)
	sc.UpdateMeshBBox()
	sc.Camera.Aspect = float32(sc.Geom.Size.X) / float32(sc.Geom.Size.Y)
	sc.Camera.UpdateMatrix()
	sc.UpdateMVPMatrix()
	return sc.renderImpl(grabImage)
}

// AssertImage asserts the [Scene.Image] at the given filename using [imagex.Assert].
// It first configures, updates, and renders the scene.
func (sc *Scene) AssertImage(t imagex.TestingT, filename string) {
	sc.Rebuild()
	img := sc.RenderGrabImage()
	if img == nil {
		t.Errorf("xyz.Scene.AssertImage: failure getting image")
		return
	}
	imagex.Assert(t, img, filename)
}

// DepthImage returns the current rendered depth image
func (sc *Scene) DepthImage() ([]float32, error) {
	fr := sc.Frame
	if fr == nil {
		return nil, errors.New("xyz.Scene DepthImage: Scene does not have a Frame")
	}
	// sy := &sc.Phong.System
	// tcmd := sy.MemCmdStart()
	// fr.GrabDepthImage(tcmd)
	// sy.MemCmdEndSubmitWaitFree()
	// depth, err := fr.Render.DepthImageArray()
	// if err == nil {
	// 	return depth, err
	// }
	return nil, nil //err
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
	return true
}

////////  renderImpl

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

// renderImpl renders the scene to the framebuffer.
// all scene-level resources must be initialized and activated at this point.
// if grabImage is true, the resulting rendered image is returned.
func (sc *Scene) renderImpl(grabImage bool) *image.NRGBA {
	ph := sc.Phong
	ph.SetCamera(&sc.Camera.ViewMatrix, &sc.Camera.ProjectionMatrix)

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
			obj.PreRender()
		}
	}

	rp, err := ph.RenderStart()
	if err != nil {
		return nil
	}
	for rci, objs := range rcs {
		rc := RenderClasses(rci)
		if len(objs) == 0 {
			continue
		}
		lastrc := RClassOpaqueVertex
		for _, obj := range objs {
			rc = obj.RenderClass()
			if rc >= RClassTransTexture && rc != lastrc {
				lastrc = rc
			}
			obj.Render(rp)
		}
	}
	if grabImage {
		return ph.RenderEndGrabImage(rp)
	}
	ph.RenderEnd(rp)
	return nil
}
