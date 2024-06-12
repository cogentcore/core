// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"log"
	"reflect"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// Node is the common interface for all xyz 3D tree nodes.
// [Solid] and [Group] are the two main types of nodes,
// which both extend [NodeBase] for the core functionality.
type Node interface {
	tree.Node

	// AsNodeBase returns the [NodeBase] for our node, which gives
	// access to all the base-level data structures and methods
	// without requiring interface methods.
	AsNodeBase() *NodeBase

	// IsSolid returns true if this is an [Solid] node (otherwise a [Group]).
	IsSolid() bool

	// AsSolid returns the node as a [Solid] (nil if not).
	AsSolid() *Solid

	// Validate checks that scene element is valid.
	Validate() error

	// UpdateWorldMatrix updates this node's local and world matrix based on parent's world matrix
	// This sets the WorldMatrixUpdated flag but does not check that flag; calling
	// routine can optionally do so.
	UpdateWorldMatrix(parWorld *math32.Matrix4)

	// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
	// groups aggregate over elements. It is called from WalkPost traversal.
	UpdateMeshBBox()

	// IsVisible provides the definitive answer as to whether a given node
	// is currently visible.  It is only entirely valid after a render pass
	// for widgets in a visible window, but it checks the window and viewport
	// for their visibility status as well, which is available always.
	// Non-visible nodes are automatically not rendered and not connected to
	// window events.  The Invisible flag is one key element of the IsVisible
	// calculus; it is set by e.g., TabView for invisible tabs, and is also
	// set if a widget is entirely out of render range.  But again, use
	// IsVisible as the main end-user method.
	// For robustness, it recursively calls the parent; this is typically
	// a short path; propagating the Invisible flag properly can be
	// very challenging without mistakenly overwriting invisibility at various
	// levels.
	IsVisible() bool

	// IsTransparent returns true if solid has transparent color.
	IsTransparent() bool

	// Config configures the node.
	Config()

	// RenderClass returns the class of rendering for this solid.
	// It is used for organizing the ordering of rendering.
	RenderClass() RenderClasses

	// Render is called by Scene Render on main thread
	// when everything is ready to go.
	Render()
}

// NodeBase is the basic 3D tree node, which has the full transform information
// relative to parent, and computed bounding boxes, etc.
// It implements the [Node] interface and contains the core functionality
// common to all 3D nodes.
type NodeBase struct {
	tree.NodeBase

	// Pose is the complete specification of position and orientation.
	Pose Pose `set:"-"`

	// Scene is the cached [Scene].
	Scene *Scene `copier:"-" set:"-"`

	// mesh-based local bounding box (aggregated for groups)
	MeshBBox BBox `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// world coordinates bounding box
	WorldBBox BBox `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// normalized display coordinates bounding box, used for frustrum clipping
	NDCBBox math32.Box3 `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// raw original bounding box for the widget within its parent Scene.
	// This is prior to intersecting with Frame bounds.
	BBox image.Rectangle `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// 2D bounding box for region occupied within Scene Frame that we render onto.
	// This is BBox intersected with Frame bounds.
	SceneBBox image.Rectangle `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`
}

// NodeFlags extend [tree.Flags] to hold 3D node state.
type NodeFlags tree.Flags //enums:bitflag

const (
	// WorldMatrixUpdated means that the Pose.WorldMatrix has been updated
	WorldMatrixUpdated NodeFlags = NodeFlags(tree.FlagsN) + iota

	// VectorsUpdated means that the rendering vectors information is updated
	VectorsUpdated

	// Invisible marks this node as invisible
	Invisible
)

// AsNode converts the given tree node to a [Node] and [NodeBase],
// returning nil if that is not possible.
func AsNode(n tree.Node) (Node, *NodeBase) {
	ni, ok := n.(Node)
	if ok {
		return ni, ni.AsNodeBase()
	}
	return nil, nil
}

// AsNodeBase returns a generic NodeBase for our node, giving generic
// access to all the base-level data structures without requiring
// interface methods.
func (nb *NodeBase) AsNodeBase() *NodeBase {
	return nb
}

// OnAdd is called when nodes are added to a parent.
// It sets the scene of the node to that of its parent.
// It should be called by all other OnAdd functions defined by node types.
func (nb *NodeBase) OnAdd() {
	if nb.Parent == nil {
		return
	}
	if sc, ok := nb.Parent.(*Scene); ok {
		nb.Scene = sc
		return
	}
	if _, pnb := AsNode(nb.Parent); pnb != nil {
		nb.Scene = pnb.Scene
		return
	}
	fmt.Println(nb, "not set from parent")
}

func (nb *NodeBase) BaseInterface() reflect.Type {
	return reflect.TypeOf((*NodeBase)(nil)).Elem()
}

func (nb *NodeBase) IsSolid() bool {
	return false
}

func (nb *NodeBase) AsSolid() *Solid {
	return nil
}

func (nb *NodeBase) Validate() error {
	return nil
}

func (nb *NodeBase) IsVisible() bool {
	if nb == nil || nb.This == nil || nb.Is(Invisible) {
		return false
	}
	return true
}

func (nb *NodeBase) IsTransparent() bool {
	return false
}

// UpdateWorldMatrix updates this node's world matrix based on parent's world matrix.
// If a nil matrix is passed, then the previously set parent world matrix is used.
// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
// routine can optionally do so.
func (nb *NodeBase) UpdateWorldMatrix(parWorld *math32.Matrix4) {
	nb.Pose.UpdateMatrix() // note: can do this in special ways to bake in other
	// automatic transforms as needed
	nb.Pose.UpdateWorldMatrix(parWorld)
	nb.SetFlag(true, WorldMatrixUpdated)
}

// UpdateMVPMatrix updates this node's MVP matrix based on given view, projection matricies from camera.
// Called during rendering.
func (nb *NodeBase) UpdateMVPMatrix(viewMat, projectionMat *math32.Matrix4) {
	nb.Pose.UpdateMVPMatrix(viewMat, projectionMat)
}

// UpdateBBox2D updates this node's 2D bounding-box information based on scene
// size and min offset position.
func (nb *NodeBase) UpdateBBox2D(size math32.Vector2) {
	off := math32.Vector2{}
	nb.WorldBBox.BBox = nb.MeshBBox.BBox.MulMatrix4(&nb.Pose.WorldMatrix)
	nb.NDCBBox = nb.MeshBBox.BBox.MVProjToNDC(&nb.Pose.MVPMatrix)
	Wmin := nb.NDCBBox.Min.NDCToWindow(size, off, 0, 1, true) // true = flipY
	Wmax := nb.NDCBBox.Max.NDCToWindow(size, off, 0, 1, true) // true = filpY
	// BBox is always relative to scene
	nb.BBox = image.Rectangle{Min: image.Point{int(Wmin.X), int(Wmax.Y)}, Max: image.Point{int(Wmax.X), int(Wmin.Y)}}
	// note: BBox is inaccurate for objects extending behind camera
	if nb.Scene == nil {
		fmt.Println(nb, "Error: scene is nil")
		return
	}
	isvis := nb.Scene.Camera.Frustum.IntersectsBox(nb.WorldBBox.BBox)
	if isvis { // filter out invisible at objbbox level
		scbounds := image.Rectangle{Max: nb.Scene.Geom.Size}
		bbvis := nb.BBox.Intersect(scbounds)
		nb.SceneBBox = bbvis
	} else {
		// fmt.Printf("not vis: %v  wbb: %v\n", nb.Name, nb.WorldBBox.BBox)
		nb.SceneBBox = image.Rectangle{}
	}
}

// RayPick converts a given 2D point in scene image coordinates
// into a ray from the camera position pointing through line of sight of camera
// into *local* coordinates of the solid.
// This can be used to find point of intersection in local coordinates relative
// to a given plane of interest, for example (see Ray methods for intersections).
// To convert mouse window-relative coords into scene-relative coords
// subtract the sc.ObjBBox.Min which includes any scrolling effects
func (nb *NodeBase) RayPick(pos image.Point) math32.Ray {
	// nb.PoseMu.RLock()
	// nb.Sc.Camera.CamMu.RLock()
	sz := nb.Scene.Geom.Size
	size := math32.Vec2(float32(sz.X), float32(sz.Y))
	fpos := math32.Vec2(float32(pos.X), float32(pos.Y))
	ndc := fpos.WindowToNDC(size, math32.Vector2{}, true) // flipY
	var err error
	ndc.Z = -1 // at closest point
	cdir := math32.Vector4FromVector3(ndc, 1).MulMatrix4(&nb.Scene.Camera.InvProjectionMatrix)
	cdir.Z = -1
	cdir.W = 0 // vec
	// get world position / transform of camera: matrix is inverse of ViewMatrix
	wdir := cdir.MulMatrix4(&nb.Scene.Camera.Pose.Matrix)
	wpos := nb.Scene.Camera.Pose.Matrix.Pos()
	// nb.Sc.Camera.CamMu.RUnlock()
	invM, err := nb.Pose.WorldMatrix.Inverse()
	// nb.PoseMu.RUnlock()
	if err != nil {
		log.Println(err)
	}
	lpos := math32.Vector4FromVector3(wpos, 1).MulMatrix4(invM)
	ldir := wdir.MulMatrix4(invM)
	ldir.SetNormal()
	ray := math32.NewRay(math32.Vec3(lpos.X, lpos.Y, lpos.Z), math32.Vec3(ldir.X, ldir.Y, ldir.Z))
	return *ray
}

// WorldMatrix returns the world matrix for this node
func (nb *NodeBase) WorldMatrix() *math32.Matrix4 {
	return &nb.Pose.WorldMatrix
}

// NormDCBBox returns the normalized display coordinates bounding box
// which is used for clipping.
func (nb *NodeBase) NormDCBBox() math32.Box3 {
	return nb.NDCBBox
}

func (nb *NodeBase) Config() {
	// nop by default; could connect to scene for update signals or something
}

func (nb *NodeBase) Render() {
	// nop
}

// SetPosePos sets Pose.Pos position to given value
func (nb *NodeBase) SetPosePos(pos math32.Vector3) {
	nb.Pose.Pos = pos
}

// SetPoseScale sets Pose.Scale scale to given value
func (nb *NodeBase) SetPoseScale(scale math32.Vector3) {
	nb.Pose.Scale = scale
}

// SetPoseQuat sets Pose.Quat to given value
func (nb *NodeBase) SetPoseQuat(quat math32.Quat) {
	nb.Pose.Quat = quat
}

// TrackCamera moves this node to pose of camera
func (nb *NodeBase) TrackCamera() {
	nb.Pose.CopyFrom(&nb.Scene.Camera.Pose)

	UpdateWorldMatrix(nb.This)
}

// TrackLight moves node to position of light of given name.
// For SpotLight, copies entire Pose. Does not work for Ambient light
// which has no position information.
func (nb *NodeBase) TrackLight(lightName string) error {
	lt, ok := nb.Scene.Lights.ValueByKeyTry(lightName)
	if !ok {
		return fmt.Errorf("xyz Node: %v TrackLight named: %v not found", nb.Path(), lightName)
	}
	switch l := lt.(type) {
	case *DirLight:
		nb.Pose.Pos = l.Pos
	case *PointLight:
		nb.Pose.Pos = l.Pos
	case *SpotLight:
		nb.Pose.CopyFrom(&l.Pose)
	}
	return nil
}
