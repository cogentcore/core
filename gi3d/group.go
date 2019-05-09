// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Group collects individual elements in a scene but does not have a Mesh or Material of
// its own.  It does have a transform that applies to all nodes under it.
type Group struct {
	Node3DBase
	BBx BBox `desc:"bounding box aggregated over all child nodes"`
}

var KiT_Group = kit.Types.AddType(&Group{}, nil)

// AddNewGroup adds a new group of given name and mesh to given parent
func AddNewGroup(sc *Scene, parent ki.Ki, name string) *Group {
	gp := parent.AddNewChild(KiT_Group, name).(*Group)
	gp.Defaults()
	return gp
}

// BBox returns the bounding box information for this node -- from Mesh or aggregate for groups
func (gp *Group) BBox() *BBox {
	// todo: compute bbox
	return &gp.BBx
}

func (gp *Group) Defaults() {
	gp.Pose.Defaults()
}

func (gp *Group) RenderClass() RenderClasses {
	return RClassNone
}

// test for impl
var _ Node3D = &Group{}
