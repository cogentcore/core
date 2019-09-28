// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
)

// Node3D is the common interface for all gi3d scenegraph nodes
type Node3D interface {
	gi.Node

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

	// UpdateMVPMatrix updates this node's MVP matrix based on given view and prjn matrix from camera
	// Called during rendering.
	UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4)

	// UpdateMeshBBox updates the Mesh-based BBox info for all nodes.
	// groups aggregate over elements.  called from FuncDownMeLast traversal
	UpdateMeshBBox()

	// UpdateBBox2D updates this node's 2D bounding-box information based on scene
	// size and other scene bbox info from scene
	UpdateBBox2D(size mat32.Vec2, sc *Scene)

	// RayPick converts a given 2D point in scene image coordinates
	// into a ray from the camera position pointing through line of sight of camera
	// into *local* coordinates of the object.
	// This can be used to find point of intersection in local coordinates relative
	// to a given plane of interest, for example (see Ray methods for intersections)
	RayPick(pos image.Point, sc *Scene) mat32.Ray

	// WorldMatrix returns the world matrix for this node
	WorldMatrix() *mat32.Mat4

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

	// Init3D does 3D intialization
	Init3D(sc *Scene)

	// RenderClass returns the class of rendering for this object
	// used for organizing the ordering of rendering
	RenderClass() RenderClasses

	// Render3D is called by Scene Render3D on main thread,
	// everything ready to go..
	Render3D(sc *Scene, rc RenderClasses, rnd Render)

	// ConnectEvents3D: setup connections to window events -- called in
	// Render3D if in bounds.  It can be useful to create modular methods for
	// different event types that can then be mix-and-matched in any more
	// specialized types.
	ConnectEvents3D(sc *Scene)
}

// Node3DBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc.
// There are only two different kinds of Nodes: Group and Object
type Node3DBase struct {
	gi.NodeBase
	Pose      Pose       `desc:"complete specification of position and orientation"`
	MeshBBox  BBox       `desc:"mesh-based local bounding box (aggregated for groups)"`
	WorldBBox BBox       `desc:"world coordinates bounding box"`
	NDCBBox   mat32.Box3 `desc:"normalized display coordinates bounding box"`
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
// If a nil matrix is passed, then the previously-set parent world matrix is used.
// This sets the WorldMatrixUpdated flag but does not check that flag -- calling
// routine can optionally do so.
func (nb *Node3DBase) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	nb.Pose.UpdateMatrix() // note: can do this in special ways to bake in other
	// automatic transforms as needed
	nb.Pose.UpdateWorldMatrix(parWorld)
	nb.SetFlag(int(WorldMatrixUpdated))
}

// UpdateMVPMatrix updates this node's MVP matrix based on given view, prjn matricies from camera.
// Called during rendering.
func (nb *Node3DBase) UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4) {
	nb.Pose.UpdateMVPMatrix(viewMat, prjnMat)
}

// UpdateBBox2D updates this node's 2D bounding-box information based on scene
// size and min offset position.
func (nb *Node3DBase) UpdateBBox2D(size mat32.Vec2, sc *Scene) {
	off := mat32.Vec2{}
	nb.WorldBBox.BBox = nb.MeshBBox.BBox.MulMat4(&nb.Pose.WorldMatrix)
	nb.NDCBBox = nb.MeshBBox.BBox.MVProjToNDC(&nb.Pose.MVPMatrix)
	Wmin := nb.NDCBBox.Min.NDCToWindow(size, off, 0, 1, true) // true = flipY
	Wmax := nb.NDCBBox.Max.NDCToWindow(size, off, 0, 1, true) // true = filpY
	// BBox is always relative to scene
	nb.BBox = image.Rectangle{Min: image.Point{int(Wmin.X), int(Wmax.Y)}, Max: image.Point{int(Wmax.X), int(Wmin.Y)}}
	scbounds := image.Rectangle{Max: sc.Geom.Size}
	bbvis := nb.BBox.Intersect(scbounds)
	if bbvis != image.ZR { // filter out invisible at objbbox level
		nb.ObjBBox = bbvis.Add(sc.BBox.Min)
		nb.VpBBox = nb.ObjBBox.Add(sc.ObjBBox.Min.Sub(sc.BBox.Min)) // move amount
		nb.VpBBox = nb.VpBBox.Intersect(sc.VpBBox)
		if nb.VpBBox != image.ZR {
			nb.WinBBox = nb.VpBBox.Add(sc.WinBBox.Min.Sub(sc.VpBBox.Min))
		} else {
			nb.WinBBox = nb.VpBBox
		}
	} else {
		nb.ObjBBox = image.ZR
		nb.VpBBox = image.ZR
		nb.WinBBox = image.ZR
	}
}

// RayPick converts a given 2D point in scene image coordinates
// into a ray from the camera position pointing through line of sight of camera
// into *local* coordinates of the object.
// This can be used to find point of intersection in local coordinates relative
// to a given plane of interest, for example (see Ray methods for intersections).
// To convert mouse window-relative coords into scene-relative coords
// subtract the sc.ObjBBox.Min which includes any scrolling effects
func (nb *Node3DBase) RayPick(pos image.Point, sc *Scene) mat32.Ray {
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
	invM, err := nb.Pose.WorldMatrix.Inverse()
	if err != nil {
		log.Println(err)
	}
	lpos := mat32.NewVec4FromVec3(wpos, 1).MulMat4(invM)
	ldir := wdir.MulMat4(invM)
	ldir.SetNormal()
	ray := mat32.NewRay(mat32.Vec3{lpos.X, lpos.Y, lpos.Z}, mat32.Vec3{ldir.X, ldir.Y, ldir.Z})
	return *ray
}

// WorldMatrix returns the world matrix for this node
func (nb *Node3DBase) WorldMatrix() *mat32.Mat4 {
	return &nb.Pose.WorldMatrix
}

func (nb *Node3DBase) Init3D(sc *Scene) {
	// todo: instead, just trigger a scene update.
	// nb.NodeSig.Connect(nb.This(), func(recnb, sendk ki.Ki, sig int64, data interface{}) {
	// 	rnbi, rnb := KiToNode3D(recnb)
	// 	if Update3DTrace {
	// 		fmt.Printf("3D Update: Node: %v update scene due to signal: %v from node: %v\n", rnbi.PathUnique(), ki.NodeSignals(sig), sendk.PathUnique())
	// 	}
	// 	if !rnb.IsDeleted() && !rnb.IsDestroyed() {
	// 		scci := rnb.ParentByType(KiT_Scene, true)
	// 		if scci != nil {
	// 			rnbi.UpdateWorldMatrix(nil) // nil = use cached last one
	// 			rnbi.UpdateWorldMatrixChildren()
	// 			scci.(*Scene).DirectWinUpload()
	// 		}
	// 	}
	// })
}

func (nb *Node3DBase) Render3D(sc *Scene, rc RenderClasses, rnd Render) {
	// nop
}

/////////////////////////////////////////////////////////////////
// Events

func (nb *Node3DBase) ConnectEvents3D(sc *Scene) {
	// nop -- add connect event calls here as needed in derived types
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (nb *Node3DBase) ConnectEvent(win *gi.Window, et oswin.EventType, pri gi.EventPris, fun ki.RecvFunc) {
	win.ConnectEvent(nb.This(), et, pri, fun)
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (nb *Node3DBase) DisconnectEvent(win *gi.Window, et oswin.EventType, pri gi.EventPris) {
	win.DisconnectEvent(nb.This(), et, pri)
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities.
// This goes down the entire tree from this node on down, as typically everything under
// will not get an explicit disconnect call because no further updating will happen
func (nb *Node3DBase) DisconnectAllEvents(win *gi.Window, pri gi.EventPris) {
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, ni := KiToNode3D(k)
		if ni == nil {
			return false // going into a different type of thing, bail
		}
		win.DisconnectAllEvents(ni.This(), pri)
		return true
	})
}

// TrackCamera moves this node to pose of camera
func (nb *Node3DBase) TrackCamera(sc *Scene) {
	nb.Pose.CopyFrom(&sc.Camera.Pose)
}

// TrackLight moves node to position of light of given name.
// For SpotLight, copies entire Pose. Does not work for Ambient light
// which has no position information.
func (nb *Node3DBase) TrackLight(sc *Scene, lightName string) error {
	lt, ok := sc.Lights[lightName]
	if !ok {
		return fmt.Errorf("gi3d Node: %v TrackLight named: %v not found", nb.PathUnique(), lightName)
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
