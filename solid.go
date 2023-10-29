// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image/color"
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

// SetColor sets the [Material.Color]:
// prop: color = main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering
func (sld *Solid) SetColor(v color.RGBA) *Solid {
	sld.Mat.Color = v
	return sld
}

// SetEmissive sets the [Material.Emissive]:
// prop: emissive = color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object
func (sld *Solid) SetEmissive(v color.RGBA) *Solid {
	sld.Mat.Emissive = v
	return sld
}

// SetShiny sets the [Material.Shiny]:
// prop: shiny = specular shininess factor -- how focally vs. broad the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128 or so but can go higher) having a smaller more focal specular reflection.  Also set Reflective factor to change overall shininess effect.
func (sld *Solid) SetShiny(v float32) *Solid {
	sld.Mat.Shiny = v
	return sld
}

// SetReflective sets the [Material.Reflective]:
// prop: reflective = specular reflectiveness factor -- how much it shines back directional light.  The specular reflection color is always white * the incoming light.
func (sld *Solid) SetReflective(v float32) *Solid {
	sld.Mat.Reflective = v
	return sld
}

// SetBright sets the [Material.Bright]:
// prop: bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters
func (sld *Solid) SetBright(v float32) *Solid {
	sld.Mat.Bright = v
	return sld
}

// SetTextureName sets material to use given texture name
// (textures are accessed by name on Scene).
// If name is empty, then texture is reset
func (sld *Solid) SetTextureName(texName string) *Solid {
	sld.Mat.SetTextureName(sld.Sc, texName)
	return sld
}

// SetTexture sets material to use given texture
func (sld *Solid) SetTexture(tex Texture) *Solid {
	sld.Mat.SetTexture(tex)
	return sld
}

// SetPos sets the [Pose.Pos] position of the solid
func (sld *Solid) SetPos(x, y, z float32) *Solid {
	sld.Pose.Pos.Set(x, y, z)
	return sld
}

// SetScale sets the [Pose.Scale] scale of the solid
func (sld *Solid) SetScale(x, y, z float32) *Solid {
	sld.Pose.Scale.Set(x, y, z)
	return sld
}

// SetAxisRotation sets the [Pose.Quat] rotation of the solid,
// from local axis and angle in degrees.
func (sld *Solid) SetAxisRotation(x, y, z, angle float32) *Solid {
	sld.Pose.SetAxisRotation(x, y, z, angle)
	return sld
}

// SetEulerRotation sets the [Pose.Quat] rotation of the solid,
// from euler angles in degrees
func (sld *Solid) SetEulerRotation(x, y, z float32) *Solid {
	sld.Pose.SetEulerRotation(x, y, z)
	return sld
}

func (sld *Solid) Config(sc *Scene) {
	sld.Sc = sc
	sld.Validate()
	sld.NodeBase.Config(sc)
}

// ParentMaterial returns parent's material or nil if not avail
func (sld *Solid) ParentMaterial() *Material {
	if sld.Par == nil || sld.Par.This() != nil {
		return nil
	}
	psi := sld.Par.(Node).AsSolid()
	if psi == nil {
		return nil
	}
	return &(psi.Mat)
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
