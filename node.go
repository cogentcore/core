// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"log"
	"reflect"
	"sync"

	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Node is the common interface for all xyz scenegraph nodes
type Node interface {
	ki.Ki

	// IsSolid returns true if this is an Solid node (else a Group)
	IsSolid() bool

	// AsNode3D returns a generic NodeBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNode() *NodeBase

	// AsSolid returns a node as Solid (nil if not)
	AsSolid() *Solid

	// Validate checks that scene element is valid
	Validate() error

	// UpdateWorldMatrix updates this node's local and world matrix based on parent's world matrix
	// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
	// routine can optionally do so.
	UpdateWorldMatrix(parWorld *mat32.Mat4)

	// UpdateMVPMatrix updates this node's MVP matrix based on
	// given view and prjn matrix from camera.
	// Called during rendering.
	UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4)

	// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
	// groups aggregate over elements.  called from WalkPost traversal
	UpdateMeshBBox()

	// UpdateBBox2D updates this node's 2D bounding-box information based on scene
	// size and other scene bbox info from scene
	UpdateBBox2D(size mat32.Vec2, sc *Scene)

	// RayPick converts a given 2D point in scene image coordinates
	// into a ray from the camera position pointing through line of sight of camera
	// into *local* coordinates of the solid.
	// This can be used to find point of intersection in local coordinates relative
	// to a given plane of interest, for example (see Ray methods for intersections).
	RayPick(pos image.Point, sc *Scene) mat32.Ray

	// WorldMatrix returns the world matrix for this node, under read-lock protection.
	WorldMatrix() *mat32.Mat4

	// NormDCBBox returns the normalized display coordinates bounding box
	// which is used for clipping.  This is read-lock protected.
	NormDCBBox() mat32.Box3

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

	// IsTransparent returns true if solid has transparent color
	IsTransparent() bool

	// Config performs
	Config(sc *Scene)

	// UpdateNode does arbitrary node updating during render process
	UpdateNode(sc *Scene)

	// RenderClass returns the class of rendering for this solid.
	// used for organizing the ordering of rendering
	RenderClass() RenderClasses

	// Render is called by Scene Render on main thread,
	// everything ready to go..
	Render(sc *Scene)

	// Convenience methods for external setting of Pose values with appropriate locking

	// SetPosePos sets Pose.Pos position to given value, under write lock protection
	SetPosePos(pos mat32.Vec3)

	// SetPoseScale sets Pose.Scale scale to given value, under write lock protection
	SetPoseScale(scale mat32.Vec3)

	// SetPoseQuat sets Pose.Quat to given value, under write lock protection
	SetPoseQuat(quat mat32.Quat)
}

// NodeBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc.
// There are only two different kinds of Nodes: Group and Solid
type NodeBase struct {
	ki.Node

	// complete specification of position and orientation
	Pose Pose `set:"-"`

	// Sc is the cached Scene
	Sc *Scene `set:"-"`

	// mutex on pose access -- needed for parallel updating
	PoseMu sync.RWMutex `view:"-" copy:"-" json:"-" xml:"-"  set:"-"`

	// mesh-based local bounding box (aggregated for groups)
	MeshBBox BBox `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// world coordinates bounding box
	WorldBBox BBox `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// normalized display coordinates bounding box, used for frustrum clipping
	NDCBBox mat32.Box3 `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// raw original bounding box for the widget within its parent Scene.
	// This is prior to intersecting with Frame bounds.
	BBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// 2D bounding box for region occupied within Scene Frame that we render onto.
	// This is BBox intersected with Frame bounds.
	ScBBox image.Rectangle `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`
}

// NodeFlags extend ki.Flags to hold 3D node state
type NodeFlags ki.Flags //enums:bitflag

const (
	// WorldMatrixUpdated means that the Pose.WorldMatrix has been updated
	WorldMatrixUpdated NodeFlags = NodeFlags(ki.FlagsN) + iota

	// VectorsUpdated means that the rendering vectors information is updated
	VectorsUpdated

	// Invisible marks this node as invisible
	Invisible
)

// AsNode converts Ki to a Node interface and a NodeBase obj -- nil if not.
func AsNode(k ki.Ki) (Node, *NodeBase) {
	if k == nil || k.This() == nil { // this also checks for destroyed
		return nil, nil
	}
	ni, ok := k.(Node)
	if ok {
		return ni, ni.AsNode()
	}
	return nil, nil
}

// AsNodeBase converts Ki to a *NodeBase -- use when known to be at
// least of this type, not-nil, etc
func AsNodeBase(k ki.Ki) *NodeBase {
	return k.(Node).AsNode()
}

func (nb *NodeBase) CopyFieldsFrom(frm any) {
	fr := frm.(*NodeBase)
	nb.Pose = fr.Pose
	nb.MeshBBox = fr.MeshBBox
	nb.WorldBBox = fr.WorldBBox
	nb.NDCBBox = fr.NDCBBox
	nb.BBox = fr.BBox
	nb.ScBBox = fr.ScBBox
}

// AsNode returns a generic NodeBase for our node -- gives generic
// access to all the base-level data structures without requiring
// interface methods.
func (nb *NodeBase) AsNode() *NodeBase {
	return nb
}

// UpdateStart sets the scene ScUpdating flag to prevent
// render updates during construction on a scene.
func (nb *NodeBase) UpdateStart() bool {
	updt := nb.Node.UpdateStart()
	if updt && nb.Sc != nil {
		nb.Sc.SetFlag(true, ScUpdating)
	}
	return updt
}

// UpdateEnd resets the scene ScUpdating flag
func (nb *NodeBase) UpdateEnd(updt bool) {
	if updt && nb.Sc != nil {
		nb.Sc.SetFlag(false, ScUpdating)
	}
	nb.Node.UpdateEnd(updt)
}

func (nb *NodeBase) BaseIface() reflect.Type {
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
	if nb == nil || nb.This() == nil || nb.Is(Invisible) {
		return false
	}
	return true
}

func (nb *NodeBase) IsTransparent() bool {
	return false
}

// UpdateWorldMatrix updates this node's world matrix based on parent's world matrix.
// If a nil matrix is passed, then the previously-set parent world matrix is used.
// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
// routine can optionally do so.
func (nb *NodeBase) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	nb.PoseMu.Lock()
	defer nb.PoseMu.Unlock()
	nb.Pose.UpdateMatrix() // note: can do this in special ways to bake in other
	// automatic transforms as needed
	nb.Pose.UpdateWorldMatrix(parWorld)
	nb.SetFlag(true, WorldMatrixUpdated)
}

// UpdateMVPMatrix updates this node's MVP matrix based on given view, prjn matricies from camera.
// Called during rendering.
func (nb *NodeBase) UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4) {
	nb.PoseMu.Lock()
	nb.Pose.UpdateMVPMatrix(viewMat, prjnMat)
	nb.PoseMu.Unlock()
}

// UpdateBBox2D updates this node's 2D bounding-box information based on scene
// size and min offset position.
func (nb *NodeBase) UpdateBBox2D(size mat32.Vec2, sc *Scene) {
	off := mat32.Vec2{}
	nb.PoseMu.RLock()
	nb.WorldBBox.BBox = nb.MeshBBox.BBox.MulMat4(&nb.Pose.WorldMatrix)
	nb.NDCBBox = nb.MeshBBox.BBox.MVProjToNDC(&nb.Pose.MVPMatrix)
	nb.PoseMu.RUnlock()
	Wmin := nb.NDCBBox.Min.NDCToWindow(size, off, 0, 1, true) // true = flipY
	Wmax := nb.NDCBBox.Max.NDCToWindow(size, off, 0, 1, true) // true = filpY
	// BBox is always relative to scene
	nb.BBox = image.Rectangle{Min: image.Point{int(Wmin.X), int(Wmax.Y)}, Max: image.Point{int(Wmax.X), int(Wmin.Y)}}
	// note: BBox is inaccurate for objects extending behind camera
	isvis := sc.Camera.Frustum.IntersectsBox(nb.WorldBBox.BBox)
	if isvis { // filter out invisible at objbbox level
		scbounds := image.Rectangle{Max: sc.Geom.Size}
		bbvis := nb.BBox.Intersect(scbounds)
		nb.ScBBox = bbvis
	} else {
		// fmt.Printf("not vis: %v  wbb: %v\n", nb.Name(), nb.WorldBBox.BBox)
		nb.ScBBox = image.Rectangle{}
	}
}

// RayPick converts a given 2D point in scene image coordinates
// into a ray from the camera position pointing through line of sight of camera
// into *local* coordinates of the solid.
// This can be used to find point of intersection in local coordinates relative
// to a given plane of interest, for example (see Ray methods for intersections).
// To convert mouse window-relative coords into scene-relative coords
// subtract the sc.ObjBBox.Min which includes any scrolling effects
func (nb *NodeBase) RayPick(pos image.Point, sc *Scene) mat32.Ray {
	// nb.PoseMu.RLock()
	// sc.Camera.CamMu.RLock()
	sz := sc.Geom.Size
	size := mat32.Vec2{float32(sz.X), float32(sz.Y)}
	fpos := mat32.Vec2{float32(pos.X), float32(pos.Y)}
	ndc := fpos.WindowToNDC(size, mat32.Vec2{}, true) // flipY
	var err error
	ndc.Z = -1 // at closest point
	cdir := mat32.NewVec4FromVec3(ndc, 1).MulMat4(&sc.Camera.InvPrjnMatrix)
	cdir.Z = -1
	cdir.W = 0 // vec
	// get world position / transform of camera: matrix is inverse of ViewMatrix
	wdir := cdir.MulMat4(&sc.Camera.Pose.Matrix)
	wpos := sc.Camera.Pose.Matrix.Pos()
	// sc.Camera.CamMu.RUnlock()
	invM, err := nb.Pose.WorldMatrix.Inverse()
	// nb.PoseMu.RUnlock()
	if err != nil {
		log.Println(err)
	}
	lpos := mat32.NewVec4FromVec3(wpos, 1).MulMat4(invM)
	ldir := wdir.MulMat4(invM)
	ldir.SetNormal()
	ray := mat32.NewRay(mat32.Vec3{lpos.X, lpos.Y, lpos.Z}, mat32.Vec3{ldir.X, ldir.Y, ldir.Z})
	return *ray
}

// WorldMatrix returns the world matrix for this node, under read lock protection
func (nb *NodeBase) WorldMatrix() *mat32.Mat4 {
	nb.PoseMu.RLock()
	defer nb.PoseMu.RUnlock()
	return &nb.Pose.WorldMatrix
}

// NormDCBBox returns the normalized display coordinates bounding box
// which is used for clipping.  This is read-lock protected.
func (nb *NodeBase) NormDCBBox() mat32.Box3 {
	return nb.NDCBBox
}

func (nb *NodeBase) Config(sc *Scene) {
	// nop by default -- could connect to scene for update signals or something
}

func (nb *NodeBase) UpdateNode(sc *Scene) {
}

func (nb *NodeBase) Render(sc *Scene) {
	// nop
}

// SetPosePos sets Pose.Pos position to given value, under write lock protection
func (nb *NodeBase) SetPosePos(pos mat32.Vec3) {
	nb.PoseMu.Lock()
	nb.Pose.Pos = pos
	nb.PoseMu.Unlock()
}

// SetPoseScale sets Pose.Scale scale to given value, under write lock protection
func (nb *NodeBase) SetPoseScale(scale mat32.Vec3) {
	nb.PoseMu.Lock()
	nb.Pose.Scale = scale
	nb.PoseMu.Unlock()
}

// SetPoseQuat sets Pose.Quat to given value, under write lock protection
func (nb *NodeBase) SetPoseQuat(quat mat32.Quat) {
	nb.PoseMu.Lock()
	nb.Pose.Quat = quat
	nb.PoseMu.Unlock()
}

// TrackCamera moves this node to pose of camera
func (nb *NodeBase) TrackCamera(sc *Scene) {
	nb.PoseMu.Lock()
	sc.Camera.CamMu.RLock()
	nb.Pose.CopyFrom(&sc.Camera.Pose)
	sc.Camera.CamMu.RUnlock()
	nb.PoseMu.Unlock()
}

// TrackLight moves node to position of light of given name.
// For SpotLight, copies entire Pose. Does not work for Ambient light
// which has no position information.
func (nb *NodeBase) TrackLight(sc *Scene, lightName string) error {
	nb.PoseMu.Lock()
	defer nb.PoseMu.Unlock()
	lt, ok := sc.Lights.ValByKeyTry(lightName)
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
