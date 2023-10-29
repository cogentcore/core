// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"sort"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Group collects individual elements in a scene but does not have a Mesh or Material of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	NodeBase
}

func (gp *Group) CopyFieldsFrom(frm any) {
	fr := frm.(*Group)
	gp.NodeBase.CopyFieldsFrom(&fr.NodeBase)
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (gp *Group) UpdateMeshBBox() {
	// todo: radial, etc
	gp.MeshBBox.BBox.SetEmpty()
	for _, kid := range gp.Kids {
		nii, ni := AsNode(kid)
		if nii == nil {
			continue
		}
		ni.PoseMu.RLock()
		nbb := ni.MeshBBox.BBox.MulMat4(&ni.Pose.Matrix)
		ni.PoseMu.RUnlock()
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

// SetPos sets the [Pose.Pos] position of the solid
func (gp *Group) SetPos(x, y, z float32) *Group {
	gp.Pose.Pos.Set(x, y, z)
	return gp
}

// SetScale sets the [Pose.Scale] scale of the solid
func (gp *Group) SetScale(x, y, z float32) *Group {
	gp.Pose.Scale.Set(x, y, z)
	return gp
}

// SetAxisRotation sets the [Pose.Quat] rotation of the solid,
// from local axis and angle in degrees.
func (gp *Group) SetAxisRotation(x, y, z, angle float32) *Group {
	gp.Pose.SetAxisRotation(x, y, z, angle)
	return gp
}

// SetEulerRotation sets the [Pose.Quat] rotation of the solid,
// from euler angles in degrees
func (gp *Group) SetEulerRotation(x, y, z float32) *Group {
	gp.Pose.SetEulerRotation(x, y, z)
	return gp
}

// SolidPoint contains a Solid and a Point on that solid
type SolidPoint struct {
	Solid *Solid
	Point mat32.Vec3
}

// RaySolidIntersections returns a list of solids whose bounding box intersects
// with the given ray, with the point of intersection.  Results are sorted
// from closest to furthest.
func (gp *Group) RaySolidIntersections(ray mat32.Ray) []*SolidPoint {
	var sp []*SolidPoint
	gp.WalkPre(func(k ki.Ki) bool {
		ni, nb := AsNode(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		pt, has := ray.IntersectBox(nb.WorldBBox.BBox)
		if !has {
			return ki.Break
		}
		if !ni.IsSolid() {
			return ki.Continue
		}
		sd := ni.AsSolid()
		sp = append(sp, &SolidPoint{sd, pt})
		return ki.Break
	})

	sort.Slice(sp, func(i, j int) bool {
		di := sp[i].Point.DistTo(ray.Origin)
		dj := sp[j].Point.DistTo(ray.Origin)
		return di < dj
	})

	return sp
}

// test for impl
var _ Node = &Group{}
