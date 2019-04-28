// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"log"

	"github.com/goki/gi/mat32"
	"github.com/goki/ki/kit"
)

// https://www.khronos.org/opengl/wiki/Vertex_Specification_Best_Practices

// Object represents an individual 3D object or object element.
// It has its own unique transforms, and a material and mesh structure.
type Object struct {
	Node3DBase
	Mesh    MeshName `desc:"name of the mesh shape information used for rendering this object -- all meshes are collected on the Scene"`
	Mat     Material `view:"inline" desc:"material properties of the surface (color, shininess, texture, etc)"`
	MeshPtr Mesh     `view:"-" desc:"cached pointer to mesh"`
}

var KiT_Object = kit.Types.AddType(&Object{}, nil)

func (obj *Object) IsObject() bool {
	return true
}

func (obj *Object) AsObject() *Object {
	return obj
}

// AddNewObject adds a new object of given name and mesh
func (obj *Object) AddNewObject(sc *Scene, name string, meshName string) *Object {
	nobj := obj.AddNewChild(KiT_Object, name).(*Object)
	nobj.SetMesh(sc, meshName)
	return nobj
}

// AddNewGroup adds a new group of given name and mesh
func (obj *Object) AddNewGroup(name string) *Group {
	ngp := obj.AddNewChild(KiT_Group, name).(*Group)
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

// Render3D is called by Scene Render3D on main thread,
// everything ready to go..
func (obj *Object) Render3D(sc *Scene) {
	var nm mat32.Mat3
	nm.GetNormalMatrix(&obj.Pose.MVMatrix)
	sc.Rends.SetMatrix(obj.Pose.MVMatrix, obj.Pose.MVPMatrix, nm)
	obj.MeshPtr.Activate(sc)  // meshes have al been prep'd
	obj.MeshPtr.TransferAll() // todo: need to optimize
	var rnd Render
	switch {
	case obj.Mat.TexPtr != nil:
		// obj.Mat.TexPtr.Activate()
		rnd = sc.Rends.Renders["RenderTexture"]
	case obj.MeshPtr.HasColor():
		rnd = sc.Rends.Renders["RenderVertexColor"]
	default:
		rndu := sc.Rends.Renders["RenderUniformColor"].(*RenderUniformColor)
		rndu.SetColors(obj.Mat.Color, obj.Mat.Emissive)
		rnd = rndu
	}
	rnd.VtxFragProg().Activate()
	obj.MeshPtr.Render3D()
}
