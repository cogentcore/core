// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Group collects individual elements in a scene but does not have a Mesh or Material of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	Node3DBase
}

var KiT_Group = kit.Types.AddType(&Group{}, GroupProps)

// AddNewGroup adds a new group of given name to given parent
func AddNewGroup(sc *Scene, parent ki.Ki, name string) *Group {
	gp := parent.AddNewChild(KiT_Group, name).(*Group)
	gp.Defaults()
	return gp
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (gp *Group) UpdateMeshBBox() {
	// todo: radial, etc
	gp.MeshBBox.BBox.SetEmpty()
	for _, kid := range gp.Kids {
		nii, ni := KiToNode3D(kid)
		if nii == nil {
			continue
		}
		nbb := ni.MeshBBox.BBox.MulMat4(&ni.Pose.Matrix)
		gp.MeshBBox.BBox.ExpandByPoint(nbb.Min)
		gp.MeshBBox.BBox.ExpandByPoint(nbb.Max)
	}
	// fmt.Printf("gp: %v  bbox: %v\n", gp.Nm, gp.MeshBBox.BBox)
}

func (gp *Group) Defaults() {
	gp.Pose.Defaults()
}

func (gp *Group) RenderClass() RenderClasses {
	return RClassNone
}

// test for impl
var _ Node3D = &Group{}

var GroupProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
