// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"sort"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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

func (gp *Group) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Group)
	gp.Node3DBase.CopyFieldsFrom(&fr.Node3DBase)
}

// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
// groups aggregate over elements
func (gp *Group) UpdateMeshBBox() {
	// todo: radial, etc
	gp.BBoxMu.Lock()
	gp.MeshBBox.BBox.SetEmpty()
	for _, kid := range gp.Kids {
		nii, ni := KiToNode3D(kid)
		if nii == nil {
			continue
		}
		ni.BBoxMu.Lock()
		ni.PoseMu.RLock()
		nbb := ni.MeshBBox.BBox.MulMat4(&ni.Pose.Matrix)
		ni.PoseMu.RUnlock()
		ni.BBoxMu.Unlock()
		gp.MeshBBox.BBox.ExpandByPoint(nbb.Min)
		gp.MeshBBox.BBox.ExpandByPoint(nbb.Max)
	}
	// fmt.Printf("gp: %v  bbox: %v\n", gp.Nm, gp.MeshBBox.BBox)
	gp.BBoxMu.Unlock()
}

func (gp *Group) Defaults() {
	gp.Pose.Defaults()
}

func (gp *Group) RenderClass() RenderClasses {
	return RClassNone
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
	gp.FuncDownMeFirst(0, gp.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, ni := KiToNode3D(k)
		if nii == nil {
			return ki.Break // going into a different type of thing, bail
		}
		pt, has := ray.IntersectBox(ni.WorldBBox.BBox)
		if !has {
			return ki.Break
		}
		if !nii.IsSolid() {
			return ki.Continue
		}
		sd := nii.AsSolid()
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
var _ Node3D = &Group{}

var GroupProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
