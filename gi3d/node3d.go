// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/ki/ki"
)

// Node3D is the common interface for all gi3d scenegraph nodes
type Node3D interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in
	// all Node3D elements.
	ki.Ki

	// IsObject returns true if this is an Object node (else a Group)
	IsObject() bool

	// AsNode3D returns a generic Node3DBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNode3D() *Node3DBase

	// AsObject returns a node as Object (nil if not)
	AsObject() *Object

	// Validate checks that scene element is valid
	Validate(sc *Scene) error

	// UpdateWorldMatrix updates this node's local and world matrix based on parent's world matrix
	// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
	// routine can optionally do so.
	UpdateWorldMatrix(parWorld *mat32.Mat4)

	// UpdateWorldMatrixChildren updates the world matrix for all children of this node
	UpdateWorldMatrixChildren()

	// UpdateMVPMatrix updates this node's MVP matrix based on given view and prjn matrix from camera
	// Called during rendering.
	UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4)

	// BBox returns the bounding box information for this node -- from Mesh or aggregate for groups
	BBox() *BBox

	// IsVisible provides the definitive answer as to whether a given node
	// is currently visible.  It is only entirely valid after a render pass
	// for widgets in a visible window, but it checks the window and viewport
	// for their visibility status as well, which is available always.
	// Non-visible nodes are automatically not rendered and not connected to
	// window events.  The Invisible flag is one key element of the IsVisible
	// calculus -- it is set by e.g., TabView for invisible tabs, and is also
	// set if a widget is entirely out of render range.  But again, use
	// IsVisible as the main end-user method.
	// For robustness, it recursively calls the parent -- this is typically
	// a short path -- propagating the Invisible flag properly can be
	// very challenging without mistakenly overwriting invisibility at various
	// levels.
	IsVisible() bool

	// IsTransparent returns true if object has transparent color
	IsTransparent() bool

	// Render3D is called by Scene Render3D on main thread,
	// everything ready to go..
	Render3D(sc *Scene)
}

// Node3DBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc.
// There are only two different kinds of Nodes: Group and Object
type Node3DBase struct {
	gi.NodeBase
	Pose Pose `desc:"complete specification of position and orientation"`
}

// gi3d.NodeFlags extend gi.NodeFlags to hold 3D node state
const (
	// WorldMatrixUpdated means that the Pose.WorldMatrix has been updated
	WorldMatrixUpdated gi.NodeFlags = gi.NodeFlagsN + iota

	// VectorsUpdated means that the rendering vectors information is updated
	VectorsUpdated

	NodeFlagsN
)

// KiToNode3D converts Ki to a Node3D interface and a Node3DBase obj -- nil if not.
func KiToNode3D(k ki.Ki) (Node3D, *Node3DBase) {
	if k == nil || k.This() == nil { // this also checks for destroyed
		return nil, nil
	}
	nii, ok := k.(Node3D)
	if ok {
		return nii, nii.AsNode3D()
	}
	return nil, nil
}

// KiToNode3DBase converts Ki to a *Node3DBase -- use when known to be at
// least of this type, not-nil, etc
func KiToNode3DBase(k ki.Ki) *Node3DBase {
	return k.(Node3D).AsNode3D()
}

// AsNode3D returns a generic Node3DBase for our node -- gives generic
// access to all the base-level data structures without requiring
// interface methods.
func (nb *Node3DBase) AsNode3D() *Node3DBase {
	return nb
}

func (nb *Node3DBase) IsObject() bool {
	return false
}

func (nb *Node3DBase) AsObject() *Object {
	return nil
}

func (nb *Node3DBase) Validate(sc *Scene) error {
	return nil
}

func (nb *Node3DBase) IsVisible() bool {
	if nb == nil || nb.This() == nil || nb.IsInvisible() {
		return false
	}
	if nb.Par == nil || nb.Par.This() == nil {
		return false
	}
	// if nb.Par == nb.Scene { // cutoff at top
	// 	return true
	// }
	return nb.Par.This().(Node3D).IsVisible()
}

func (nb *Node3DBase) IsTransparent() bool {
	return false
}

func (nb *Node3DBase) WorldMatrixUpdated() bool {
	return nb.HasFlag(int(WorldMatrixUpdated))
}

// UpdateWorldMatrix updates this node's world matrix based on parent's world matrix.
// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
// routine can optionally do so.
func (nb *Node3DBase) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	nb.Pose.UpdateWorldMatrix(parWorld)
	nb.SetFlag(int(WorldMatrixUpdated))
}

// UpdateWorldMatrixChildren updates the world matrix for all children of this node
// (and their children, and so on..)
func (nb *Node3DBase) UpdateWorldMatrixChildren() {
	for _, kid := range nb.Kids {
		nii, _ := KiToNode3D(kid)
		if nii != nil {
			nii.UpdateWorldMatrix(&nb.Pose.WorldMatrix)
			nii.UpdateWorldMatrixChildren()
		}
	}
}

// UpdateMVPMatrix updates this node's MVP matrix based on given view, prjn matricies from camera
// Called during rendering.
func (nb *Node3DBase) UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4) {
	nb.Pose.UpdateMVPMatrix(viewMat, prjnMat)
}

func (nb *Node3DBase) Render3D(sc *Scene) {
	// nop
}
