// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image/color"
	"log"
)

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Solid represents an individual 3D solid element.
// It has its own unique spatial transforms and material properties,
// and points to a mesh structure defining the shape of the solid.
type Solid struct {
	NodeBase

	// MeshName is the name of the mesh shape information used for rendering
	// this solid; all meshes are collected on the Scene.
	MeshName MeshName `set:"-"`

	// Material contains the material properties of the surface (color, shininess, texture, etc).
	Material Material `view:"add-fields"`

	// Mesh is the cached [Mesh] object set from [Solid.MeshName].
	Mesh Mesh `view:"-" set:"-"`
}

func (sld *Solid) Init() {
	sld.Defaults()
}

func (sld *Solid) IsSolid() bool {
	return true
}

func (sld *Solid) AsSolid() *Solid {
	return sld
}

// Defaults sets default initial settings for solid params.
// This is called automatically Init.
func (sld *Solid) Defaults() {
	sld.Pose.Defaults()
	sld.Material.Defaults()
}

// SetMeshName sets mesh to given mesh name.
func (sld *Solid) SetMeshName(meshName string) error {
	if meshName == "" {
		return nil
	}
	ms, err := sld.Scene.MeshByNameTry(meshName)
	if err != nil {
		log.Println(err)
		return err
	}
	sld.MeshName = MeshName(meshName)
	sld.Mesh = ms
	return nil
}

// SetMesh sets mesh
func (sld *Solid) SetMesh(ms Mesh) *Solid {
	sld.Mesh = ms
	if sld.Mesh != nil {
		sld.MeshName = MeshName(sld.Mesh.AsMeshBase().Name)
	} else {
		sld.MeshName = ""
	}
	return sld
}

// SetColor sets the [Material.Color]:
// prop: color = main color of surface, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering
func (sld *Solid) SetColor(v color.RGBA) *Solid {
	sld.Material.Color = v
	return sld
}

// SetEmissive sets the [Material.Emissive]:
// prop: emissive = color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object
func (sld *Solid) SetEmissive(v color.RGBA) *Solid {
	sld.Material.Emissive = v
	return sld
}

// SetShiny sets the [Material.Shiny]:
// prop: shiny = specular shininess factor -- how focally vs. broad the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128 or so but can go higher) having a smaller more focal specular reflection.  Also set Reflective factor to change overall shininess effect.
func (sld *Solid) SetShiny(v float32) *Solid {
	sld.Material.Shiny = v
	return sld
}

// SetReflective sets the [Material.Reflective]:
// prop: reflective = specular reflectiveness factor -- how much it shines back directional light.  The specular reflection color is always white * the incoming light.
func (sld *Solid) SetReflective(v float32) *Solid {
	sld.Material.Reflective = v
	return sld
}

// SetBright sets the [Material.Bright]:
// prop: bright = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters
func (sld *Solid) SetBright(v float32) *Solid {
	sld.Material.Bright = v
	return sld
}

// SetTextureName sets material to use given texture name
// (textures are accessed by name on Scene).
// If name is empty, then texture is reset
func (sld *Solid) SetTextureName(texName string) *Solid {
	sld.Material.SetTextureName(sld.Scene, texName)
	return sld
}

// SetTexture sets material to use given texture
func (sld *Solid) SetTexture(tex Texture) *Solid {
	sld.Material.SetTexture(tex)
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

func (sld *Solid) Config() {
	sld.Validate()
	sld.NodeBase.Config()
}

// ParentMaterial returns parent's material or nil if not avail
func (sld *Solid) ParentMaterial() *Material {
	if sld.Parent == nil {
		return nil
	}
	psi := sld.Parent.(Node).AsSolid()
	if psi == nil {
		return nil
	}
	return &(psi.Material)
}

// Validate checks that solid has valid mesh and texture settings, etc
func (sld *Solid) Validate() error {
	if sld.MeshName == "" {
		err := fmt.Errorf("xyz.Solid: %s Mesh name is empty", sld.Path())
		log.Println(err)
		return err
	}
	if sld.Mesh == nil || sld.Mesh.AsMeshBase().Name != string(sld.MeshName) {
		err := sld.SetMeshName(string(sld.MeshName))
		if err != nil {
			return err
		}
	}
	return sld.Material.Validate(sld.Scene)
}

func (sld *Solid) IsVisible() bool {
	if sld.Mesh == nil {
		return false
	}
	return sld.NodeBase.IsVisible()
}

func (sld *Solid) IsTransparent() bool {
	if sld.Mesh == nil {
		return false
	}
	if sld.Mesh.AsMeshBase().HasColor {
		return sld.Mesh.AsMeshBase().Transparent
	}
	return sld.Material.IsTransparent()
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (sld *Solid) UpdateMeshBBox() {
	if sld.Mesh != nil {
		mesh := sld.Mesh.AsMeshBase()
		sld.MeshBBox = mesh.BBox
	}
}

/////////////////////////////////////////////////////////////////////////////
//   Rendering

// RenderClass returns the class of rendering for this solid
// used for organizing the ordering of rendering
func (sld *Solid) RenderClass() RenderClasses {
	switch {
	case sld.Material.Texture != nil:
		return RClassOpaqueTexture
	case sld.Mesh.AsMeshBase().HasColor:
		if sld.Mesh.AsMeshBase().Transparent {
			return RClassTransVertex
		}
		return RClassOpaqueVertex
	default:
		if sld.Material.IsTransparent() {
			return RClassTransUniform
		}
		return RClassOpaqueUniform
	}
}

// Render activates this solid for rendering
func (sld *Solid) Render() {
	sld.Scene.Phong.UseMeshName(string(sld.MeshName))
	sld.Scene.Phong.SetModelMtx(&sld.Pose.WorldMatrix)
	sld.Material.Render(sld.Scene)
	sld.Scene.Phong.Render()
}
