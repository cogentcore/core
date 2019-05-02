// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/ki/kit"
)

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Object represents an individual 3D object or object element.
// It has its own unique transforms, and a material and mesh structure.
type Object struct {
	Node3DBase
	Mesh    MeshName `desc:"name of the mesh shape information used for rendering this object -- all meshes are collected on the Scene"`
	Mat     Material `desc:"material properties of the surface (color, shininess, texture, etc)"`
	MeshPtr Mesh     `view:"-" desc:"cached pointer to mesh"`
}

var KiT_Object = kit.Types.AddType(&Object{}, nil)

func (obj *Object) IsObject() bool {
	return true
}

func (obj *Object) AsObject() *Object {
	return obj
}

// Defaults sets default initial settings for object params -- important
// to call this before setting specific values, as the initial zero
// values for some parameters are degenerate
func (obj *Object) Defaults() {
	obj.Pose.Defaults()
	obj.Mat.Defaults()
}

// AddNewObject adds a new object of given name and mesh
func (obj *Object) AddNewObject(sc *Scene, name string, meshName string) *Object {
	nobj := obj.AddNewChild(KiT_Object, name).(*Object)
	nobj.SetMesh(sc, meshName)
	nobj.Defaults()
	return nobj
}

// AddNewGroup adds a new group of given name and mesh
func (obj *Object) AddNewGroup(name string) *Group {
	ngp := obj.AddNewChild(KiT_Group, name).(*Group)
	ngp.Defaults()
	return ngp
}

// SetMesh sets mesh to given mesh name -- requires Scene to lookup mesh by name
func (obj *Object) SetMesh(sc *Scene, meshName string) error {
	ms, ok := sc.Meshes[meshName]
	if !ok {
		err := fmt.Errorf("gi3d.Object: %s SetMesh name: %s not found in scene", obj.PathUnique(), meshName)
		log.Println(err)
		return err
	}
	obj.MeshPtr = ms
	return nil
}

func (obj *Object) Init3D(sc *Scene) {
	err := obj.Validate(sc)
	if err != nil {
		obj.SetInvisible()
	}
	obj.Node3DBase.Init3D(sc)
}

// Validate checks that object has valid mesh and texture settings, etc
func (obj *Object) Validate(sc *Scene) error {
	if obj.Mesh == "" {
		err := fmt.Errorf("gi3d.Object: %s Mesh name is empty", obj.PathUnique())
		log.Println(err)
		return err
	}
	if obj.MeshPtr == nil || obj.MeshPtr.Name() != string(obj.Mesh) {
		err := obj.SetMesh(sc, string(obj.Mesh))
		if err != nil {
			return err
		}
	}
	return obj.Mat.Validate(sc)
}

func (obj *Object) IsVisible() bool {
	if obj.MeshPtr == nil {
		return false
	}
	return obj.Node3DBase.IsVisible()
}

func (obj *Object) IsTransparent() bool {
	if obj.MeshPtr == nil {
		return false
	}
	if obj.MeshPtr.HasColor() {
		return obj.MeshPtr.IsTransparent()
	}
	return obj.Mat.IsTransparent()
}

// BBox returns the bounding box information for this node -- from Mesh or aggregate for groups
func (obj *Object) BBox() *BBox {
	if obj.MeshPtr == nil {
		return nil
	}
	return &(obj.MeshPtr.AsMeshBase().BBox)
}

// TrackCamera moves this object to position of camera
func (obj *Object) TrackCamera(sc *Scene) {
	obj.Pose = sc.Camera.Pose
}

// TrackLight moves this object to position of light of given name
// Does not work for Ambient Lights
func (obj *Object) TrackLight(sc *Scene, lightName string) {
	lt, ok := sc.Lights[lightName]
	if !ok {
		// todo: error
		return
	}
	// todo: do rest..
	switch l := lt.(type) {
	case *DirLight:
		obj.Pose.Pos = l.Pos
	}
}

/////////////////////////////////////////////////////////////////////////////
//   Rendering

// RenderClass returns the class of rendering for this object
// used for organizing the ordering of rendering
func (obj *Object) RenderClass() RenderClasses {
	switch {
	case obj.Mat.TexPtr != nil:
		return RClassTexture
	case obj.MeshPtr.HasColor():
		if obj.MeshPtr.IsTransparent() {
			return RClassTransVertex
		}
		return RClassOpaqueVertex
	default:
		if obj.Mat.IsTransparent() {
			return RClassTransUniform
		}
		return RClassOpaqueUniform
	}
}

// Render3D activates this object for rendering ()
func (obj *Object) Render3D(sc *Scene, rc RenderClasses, rnd Render) {
	switch rc {
	case RClassOpaqueUniform, RClassTransUniform:
		rndu := rnd.(*RenderUniformColor)
		rndu.SetMat(&obj.Mat)
	case RClassTexture:
		// obj.Mat.TexPtr.Activate()
	}
	// gpu.Draw.DepthTest(true)
	sc.Renders.SetMatrix(&obj.Pose)
	obj.MeshPtr.Activate(sc) // meshes have all been prep'd
	// obj.MeshPtr.TransferAll() // todo: need to optimize
	obj.MeshPtr.Render3D()
	gpu.TheGPU.ErrCheck("obj render")
}
