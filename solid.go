// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"goki.dev/ki/v2"
)

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Solid represents an individual 3D solid element.
// It has its own unique spatial transforms and material properties,
// and points to a mesh structure defining the shape of the solid.
type Solid struct { //goki:no-new
	NodeBase

	// name of the mesh shape information used for rendering this solid -- all meshes are collected on the Scene
	Mesh MeshName `set:"-"`

	// material properties of the surface (color, shininess, texture, etc)
	Mat Material `view:"add-fields"`

	// cached pointer to mesh
	MeshPtr Mesh `view:"-" set:"-"`
}

var _ Node = (*Solid)(nil)

// NewSolid adds a new solid of given name and mesh to given parent
func NewSolid(parent ki.Ki, name ...string) *Solid {
	sld := parent.NewChild(SolidType, name...).(*Solid)
	sld.Defaults()
	return sld
}

func (sld *Solid) CopyFieldsFrom(frm any) {
	fr := frm.(*Solid)
	sld.NodeBase.CopyFieldsFrom(&fr.NodeBase)
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

// Defaults sets default initial settings for solid params -- important
// to call this before setting specific values, as the initial zero
// values for some parameters are degenerate
func (sld *Solid) Defaults() {
	sld.Pose.Defaults()
	sld.Mat.Defaults()
}

// SetMeshName sets mesh to given mesh name.
func (sld *Solid) SetMeshName(meshName string) error {
	if meshName == "" {
		return nil
	}
	ms, err := sld.Sc.MeshByNameTry(meshName)
	if err != nil {
		log.Println(err)
		return err
	}
	sld.Mesh = MeshName(meshName)
	sld.MeshPtr = ms
	return nil
}

// SetMesh sets mesh
func (sld *Solid) SetMesh(ms Mesh) *Solid {
	sld.MeshPtr = ms
	if sld.MeshPtr != nil {
		sld.Mesh = MeshName(sld.MeshPtr.Name())
	} else {
		sld.Mesh = ""
	}
	return sld
}

func (sld *Solid) Config(sc *Scene) {
	sld.Sc = sc
	sld.Validate()
	sld.NodeBase.Config(sc)
}

// ParentMaterial returns parent's material or nil if not avail
func (sld *Solid) ParentMaterial() *Material {
	if sld.Par == nil {
		return nil
	}
	// psi := sld.Par.Embed(TypeSolid)
	// if psi == nil {
	// 	return nil
	// }
	// return &(psi.(*Solid).Mat)
	return nil
}

// Validate checks that solid has valid mesh and texture settings, etc
func (sld *Solid) Validate() error {
	if sld.Mesh == "" {
		err := fmt.Errorf("gi3d.Solid: %s Mesh name is empty", sld.Path())
		log.Println(err)
		return err
	}
	if sld.MeshPtr == nil || sld.MeshPtr.Name() != string(sld.Mesh) {
		err := sld.SetMeshName(string(sld.Mesh))
		if err != nil {
			return err
		}
	}
	return sld.Mat.Validate(sld.Sc)
}

func (sld *Solid) IsVisible() bool {
	if sld.MeshPtr == nil {
		return false
	}
	return sld.NodeBase.IsVisible()
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
		mesh := sld.MeshPtr.AsMeshBase()
		sld.MeshBBox = mesh.BBox
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

// Render activates this solid for rendering
func (sld *Solid) Render(sc *Scene) {
	sc.Phong.UseMeshName(string(sld.Mesh))
	sld.PoseMu.RLock()
	sc.Phong.SetModelMtx(&sld.Pose.WorldMatrix)
	sld.PoseMu.RUnlock()
	sld.Mat.Render(sc)
	sc.Phong.Render()
}
