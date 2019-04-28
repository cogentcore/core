// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/ki/kit"

// Group collects individual elements in a scene but does not have a Mesh or Material of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	Node3DBase
	BBx BBox `desc:"bounding box aggregated over all child nodes"`
}

var KiT_Group = kit.Types.AddType(&Group{}, nil)

// AddNewObject adds a new object of given name and mesh
func (gp *Group) AddNewObject(name string, meshName string) *Object {
	obj := gp.AddNewChild(KiT_Object, name).(*Object)
	obj.Mesh = MeshName(meshName)
	return obj
}

// AddNewGroup adds a new group of given name and mesh
func (gp *Group) AddNewGroup(name string) *Group {
	ngp := gp.AddNewChild(KiT_Group, name).(*Group)
	return ngp
}

// BBox returns the bounding box information for this node -- from Mesh or aggregate for groups
func (gp *Group) BBox() *BBox {
	// todo: compute bbox
	return &gp.BBx
}
