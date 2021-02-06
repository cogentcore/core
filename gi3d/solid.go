// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Solid represents an individual 3D solid element.
// It has its own unique spatial transforms and material properties,
// and points to a mesh structure defining the shape of the solid.
type Solid struct {
	Node3DBase
	Mesh    MeshName `desc:"name of the mesh shape information used for rendering this solid -- all meshes are collected on the Scene"`
	Mat     Material `view:"add-fields" desc:"material properties of the surface (color, shininess, texture, etc)"`
	MeshPtr Mesh     `view:"-" desc:"cached pointer to mesh"`
}

var KiT_Solid = kit.Types.AddType(&Solid{}, SolidProps)

// AddNewSolid adds a new solid of given name and mesh to given parent
func AddNewSolid(sc *Scene, parent ki.Ki, name string, meshName string) *Solid {
	sld := parent.AddNewChild(KiT_Solid, name).(*Solid)
	sld.SetMeshName(sc, meshName)
	sld.Defaults()
	return sld
}

func (sld *Solid) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Solid)
	sld.Node3DBase.CopyFieldsFrom(&fr.Node3DBase)
	sld.Mesh = fr.Mesh
	sld.Mat = fr.Mat
	sld.MeshPtr = fr.MeshPtr
}

func (sld *Solid) IsSolid() bool {
	return true
}

func (sld *Solid) AsSolid() *Solid {
	return sld
}

func (sld *Solid) Disconnect() {
	sld.Node3DBase.Disconnect()
	sld.MeshPtr = nil
	sld.Mat.Disconnect()
}

// Defaults sets default initial settings for solid params -- important
// to call this before setting specific values, as the initial zero
// values for some parameters are degenerate
func (sld *Solid) Defaults() {
	sld.Pose.Defaults()
	sld.Mat.Defaults()
}

// SetMeshName sets mesh to given mesh name.
func (sld *Solid) SetMeshName(sc *Scene, meshName string) error {
	if meshName == "" {
		return nil
	}
	ms, ok := sc.Meshes[meshName]
	if !ok {
		err := fmt.Errorf("gi3d.Solid: %s SetMesh name: %s not found in scene", sld.Path(), meshName)
		log.Println(err)
		return err
	}
	sld.Mesh = MeshName(meshName)
	sld.MeshPtr = ms
	return nil
}

// SetMesh sets mesh
func (sld *Solid) SetMesh(sc *Scene, ms Mesh) {
	sld.MeshPtr = ms
	if sld.MeshPtr != nil {
		sld.Mesh = MeshName(sld.MeshPtr.Name())
	} else {
		sld.Mesh = ""
	}
}

func (sld *Solid) Init3D(sc *Scene) {
	err := sld.Validate(sc)
	if err != nil {
		sld.SetInvisible()
	}
	sld.Node3DBase.Init3D(sc)
}

// ParentMaterial returns parent's material or nil if not avail
func (sld *Solid) ParentMaterial() *Material {
	if sld.Par == nil {
		return nil
	}
	psi := sld.Par.Embed(KiT_Solid)
	if psi == nil {
		return nil
	}
	return &(psi.(*Solid).Mat)
}

func (sld *Solid) Style3D(sc *Scene) {
	styprops := *sld.Properties()
	parMat := sld.ParentMaterial()
	sld.Mat.SetMatProps(parMat, styprops, sc.Viewport)

	pagg := sld.ParentCSSAgg()
	if pagg != nil {
		gi.AggCSS(&sld.CSSAgg, *pagg)
	} else {
		sld.CSSAgg = nil // restart
	}
	gi.AggCSS(&sld.CSSAgg, sld.CSS)
	sld.Mat.StyleCSS(sld.This().(Node3D), sld.CSSAgg, "", sc.Viewport)
}

// Validate checks that solid has valid mesh and texture settings, etc
func (sld *Solid) Validate(sc *Scene) error {
	if sld.Mesh == "" {
		err := fmt.Errorf("gi3d.Solid: %s Mesh name is empty", sld.Path())
		log.Println(err)
		return err
	}
	if sld.MeshPtr == nil || sld.MeshPtr.Name() != string(sld.Mesh) {
		err := sld.SetMeshName(sc, string(sld.Mesh))
		if err != nil {
			return err
		}
	}
	return sld.Mat.Validate(sc)
}

func (sld *Solid) IsVisible() bool {
	if sld.MeshPtr == nil {
		return false
	}
	return sld.Node3DBase.IsVisible()
}

func (sld *Solid) IsTransparent() bool {
	if sld.MeshPtr == nil {
		return false
	}
	if sld.MeshPtr.HasColor() {
		return sld.MeshPtr.IsTransparent()
	}
	return sld.Mat.IsTransparent()
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sld *Solid) UpdateMeshBBox() {
	if sld.MeshPtr != nil {
		sld.BBoxMu.Lock()
		mesh := sld.MeshPtr.AsMeshBase()
		mesh.BBoxMu.RLock()
		sld.MeshBBox = mesh.BBox
		mesh.BBoxMu.RUnlock()
		sld.BBoxMu.Unlock()
	}
}

/////////////////////////////////////////////////////////////////////////////
//   Rendering

// RenderClass returns the class of rendering for this solid
// used for organizing the ordering of rendering
func (sld *Solid) RenderClass() RenderClasses {
	switch {
	case sld.Mat.TexPtr != nil:
		return RClassOpaqueTexture
	case sld.MeshPtr.HasColor():
		if sld.MeshPtr.IsTransparent() {
			return RClassTransVertex
		}
		return RClassOpaqueVertex
	default:
		if sld.Mat.IsTransparent() {
			return RClassTransUniform
		}
		return RClassOpaqueUniform
	}
}

// Render3D activates this solid for rendering
func (sld *Solid) Render3D(sc *Scene, rc RenderClasses, rnd Render) {
	switch rc {
	case RClassOpaqueUniform, RClassTransUniform:
		rndu := rnd.(*RenderUniformColor)
		rndu.SetMat(&sld.Mat, sc)
	case RClassOpaqueVertex, RClassTransVertex:
		rndv := rnd.(*RenderVertexColor)
		rndv.SetMat(&sld.Mat, sc)
	case RClassOpaqueTexture, RClassTransTexture:
		rndt := rnd.(*RenderTexture)
		rndt.SetMat(&sld.Mat, sc)
	}
	sld.PoseMu.RLock()
	sc.Renders.SetMatrix(&sld.Pose)
	sld.PoseMu.RUnlock()
	sld.MeshPtr.Render3D(sc)
	gpu.TheGPU.ErrCheck("sld render")
}

var SolidProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
