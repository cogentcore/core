// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package physics

import (
	"sort"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// Group is a container of bodies, joints, or other groups
// it should be used strategically to partition the space
// and its BBox is used to optimize tree-based collision detection.
// Use a group for the top-level World node as well.
type Group struct {
	NodeBase
}

func (gp *Group) Update() {
	gp.RunUpdaters()
	gp.WalkDown(func(n tree.Node) bool {
		if n.AsTree().This == gp.This {
			return tree.Continue
		}
		n.(Node).Update()
		return tree.Continue
	})
}

func (gp *Group) InitAbs(par *NodeBase) {
	gp.InitAbsBase(par)
}

func (gp *Group) RelToAbs(par *NodeBase) {
	gp.RelToAbsBase(par) // yes we can move groups
}

func (gp *Group) Step(step float32) {
	// groups do NOT update physics
}

func (gp *Group) GroupBBox() {
	hasDyn := false
	gp.BBox.BBox.SetEmpty()
	gp.BBox.VelBBox.SetEmpty()
	for _, kid := range gp.Children {
		n, nb := AsNode(kid)
		if n == nil {
			continue
		}
		gp.BBox.BBox.ExpandByBox(nb.BBox.BBox)
		gp.BBox.VelBBox.ExpandByBox(nb.BBox.VelBBox)
		if nb.Dynamic {
			hasDyn = true
		}
	}
	gp.SetDynamic(hasDyn)
}

// WorldDynGroupBBox does a GroupBBox on all dynamic nodes
func (gp *Group) WorldDynGroupBBox() {
	gp.WalkDownPost(func(tn tree.Node) bool {
		n, nb := AsNode(tn)
		if n == nil {
			return false
		}
		if !nb.Dynamic {
			return false
		}
		return true
	}, func(tn tree.Node) bool {
		n, nb := AsNode(tn)
		if n == nil {
			return false
		}
		if !nb.Dynamic {
			return false
		}
		n.GroupBBox()
		return true
	})
}

// WorldInit does the full tree InitAbs and GroupBBox updates
func (gp *Group) WorldInit() {
	gp.Update()
	gp.WalkDown(func(tn tree.Node) bool {
		n, _ := AsNode(tn)
		if n == nil {
			return false
		}
		_, pi := AsNode(tn.AsTree().Parent)
		n.InitAbs(pi)
		return true
	})

	gp.WalkDownPost(func(tn tree.Node) bool {
		n, _ := AsNode(tn)
		if n == nil {
			return false
		}
		return true
	}, func(tn tree.Node) bool {
		n, _ := AsNode(tn)
		if n == nil {
			return false
		}
		n.GroupBBox()
		return true
	})

}

// WorldRelToAbs does a full RelToAbs update for all Dynamic groups, for
// Scripted mode updates with manual updating of Rel values.
func (gp *Group) WorldRelToAbs() {
	gp.WalkDown(func(tn tree.Node) bool {
		n, nb := AsNode(tn)
		if n == nil {
			return false // going into a different type of thing, bail
		}
		if !nb.Dynamic {
			return false
		}
		_, pi := AsNode(tn.AsTree().Parent)
		n.RelToAbs(pi)
		return true
	})

	gp.WorldDynGroupBBox()
}

// WorldStep does a full Step update for all Dynamic nodes, for
// either physics or scripted mode, based on current velocities.
func (gp *Group) WorldStep(step float32) {
	gp.WalkDown(func(tn tree.Node) bool {
		n, nb := AsNode(tn)
		if n == nil {
			return false // going into a different type of thing, bail
		}
		if !nb.Dynamic {
			return false
		}
		n.Step(step)
		return true
	})

	gp.WorldDynGroupBBox()
}

const (
	// DynsTopGps is passed to WorldCollide when all dynamic objects are in separate top groups
	DynsTopGps = true

	// DynsSubGps is passed to WorldCollide when all dynamic objects are in separate groups under top
	// level (i.e., one level deeper)
	DynsSubGps
)

// WorldCollide does first pass filtering step of collision detection
// based on separate dynamic vs. dynamic and dynamic vs. static groups.
// If dynTop is true, then each Dynamic group is separate at the top level --
// otherwise they are organized at the next group level.
// Contacts are organized by dynamic group, when non-nil, for easier
// processing.
func (gp *Group) WorldCollide(dynTop bool) []Contacts {
	var stats []Node
	var dyns []Node
	for _, kid := range gp.Children {
		n, nb := AsNode(kid)
		if n == nil {
			continue
		}
		if nb.Dynamic {
			dyns = append(dyns, n)
		} else {
			stats = append(stats, n)
		}
	}

	var sdyns []Node
	if !dynTop {
		for _, d := range dyns {
			for _, dk := range d.AsTree().Children {
				nii, _ := AsNode(dk)
				if nii == nil {
					continue
				}
				sdyns = append(sdyns, nii)
			}
		}
		dyns = sdyns
	}

	var cts []Contacts
	for i, d := range dyns {
		var dct Contacts
		for _, s := range stats {
			cc := BodyVelBBoxIntersects(d, s)
			dct = append(dct, cc...)
		}
		for di := 0; di < i; di++ {
			od := dyns[di]
			cc := BodyVelBBoxIntersects(d, od)
			dct = append(dct, cc...)
		}
		if len(dct) > 0 {
			cts = append(cts, dct)
		}
	}
	return cts
}

// BodyPoint contains a Body and a Point on that body
type BodyPoint struct {
	Body  Body
	Point math32.Vector3
}

// RayBodyIntersections returns a list of bodies whose bounding box intersects
// with the given ray, with the point of intersection
func (gp *Group) RayBodyIntersections(ray math32.Ray) []*BodyPoint {
	var bs []*BodyPoint
	gp.WalkDown(func(k tree.Node) bool {
		nii, ni := AsNode(k)
		if nii == nil {
			return false // going into a different type of thing, bail
		}
		pt, has := ray.IntersectBox(ni.BBox.BBox)
		if !has {
			return false
		}
		bd := nii.AsBody()
		if bd == nil {
			return true
		}
		bs = append(bs, &BodyPoint{bd, pt})
		return false
	})

	sort.Slice(bs, func(i, j int) bool {
		di := bs[i].Point.DistanceTo(ray.Origin)
		dj := bs[j].Point.DistanceTo(ray.Origin)
		return di < dj
	})

	return bs
}
