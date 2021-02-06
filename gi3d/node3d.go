// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"log"
	"reflect"
	"sync"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Node3D is the common interface for all gi3d scenegraph nodes
type Node3D interface {
	gi.Node

	// IsSolid returns true if this is an Solid node (else a Group)
	IsSolid() bool

	// AsNode3D returns a generic Node3DBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsNode3D() *Node3DBase

	// AsSolid returns a node as Solid (nil if not)
	AsSolid() *Solid

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

	// Init3D does 3D intialization
	Init3D(sc *Scene)

	// Style3D does 3D styling using property values on nodes
	Style3D(sc *Scene)

	// UpdateNode3D does arbitrary node updating during render process
	UpdateNode3D(sc *Scene)

	// RenderClass returns the class of rendering for this solid.
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

	// Convenience methods for external setting of Pose values with appropriate locking

	// SetPosePos sets Pose.Pos position to given value, under write lock protection
	SetPosePos(pos mat32.Vec3)

	// SetPoseScale sets Pose.Scale scale to given value, under write lock protection
	SetPoseScale(scale mat32.Vec3)

	// SetPoseQuat sets Pose.Quat to given value, under write lock protection
	SetPoseQuat(quat mat32.Quat)
}

// Node3DBase is the basic 3D scenegraph node, which has the full transform information
// relative to parent, and computed bounding boxes, etc.
// There are only two different kinds of Nodes: Group and Solid
type Node3DBase struct {
	gi.NodeBase
	Pose      Pose         `desc:"complete specification of position and orientation"`
	PoseMu    sync.RWMutex `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex on pose access -- needed for parallel updating"`
	MeshBBox  BBox         `desc:"mesh-based local bounding box (aggregated for groups)"`
	WorldBBox BBox         `desc:"world coordinates bounding box"`
	NDCBBox   mat32.Box3   `desc:"normalized display coordinates bounding box, used for frustrum clipping"`
}

var KiT_Node3DBase = kit.Types.AddType(&Node3DBase{}, Node3DBaseProps)

var Node3DBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	"EnumType:Flag": KiT_NodeFlags,
}

// NodeFlags extend gi.NodeFlags to hold 3D node state
type NodeFlags int

//go:generate stringer -type=NodeFlags

var KiT_NodeFlags = kit.Enums.AddEnumExt(gi.KiT_NodeFlags, NodeFlagsN, kit.BitFlag, nil)

const (
	// WorldMatrixUpdated means that the Pose.WorldMatrix has been updated
	WorldMatrixUpdated NodeFlags = NodeFlags(gi.NodeFlagsN) + iota

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

func (nb *Node3DBase) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Node3DBase)
	nb.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	nb.Pose = fr.Pose
	nb.MeshBBox = fr.MeshBBox
	nb.WorldBBox = fr.WorldBBox
	nb.NDCBBox = fr.NDCBBox
}

// AsNode3D returns a generic Node3DBase for our node -- gives generic
// access to all the base-level data structures without requiring
// interface methods.
func (nb *Node3DBase) AsNode3D() *Node3DBase {
	return nb
}

func (nb *Node3DBase) BaseIface() reflect.Type {
	return reflect.TypeOf((*Node3DBase)(nil)).Elem()
}

func (nb *Node3DBase) IsSolid() bool {
	return false
}

func (nb *Node3DBase) AsSolid() *Solid {
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
	sc := nb.Par.Embed(KiT_Scene)
	if sc != nil {
		return sc.(*Scene).IsVisible()
	}
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
	nb.PoseMu.Lock()
	defer nb.PoseMu.Unlock()
	nb.Pose.UpdateMatrix() // note: can do this in special ways to bake in other
	// automatic transforms as needed
	nb.Pose.UpdateWorldMatrix(parWorld)
	nb.SetFlag(int(WorldMatrixUpdated))
}

// UpdateMVPMatrix updates this node's MVP matrix based on given view, prjn matricies from camera.
// Called during rendering.
func (nb *Node3DBase) UpdateMVPMatrix(viewMat, prjnMat *mat32.Mat4) {
	nb.PoseMu.Lock()
	nb.Pose.UpdateMVPMatrix(viewMat, prjnMat)
	nb.PoseMu.Unlock()
}

// UpdateBBox2D updates this node's 2D bounding-box information based on scene
// size and min offset position.
func (nb *Node3DBase) UpdateBBox2D(size mat32.Vec2, sc *Scene) {
	nb.BBoxMu.Lock()
	defer nb.BBoxMu.Unlock()
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
		nb.ObjBBox = bbvis.Add(sc.BBox.Min)
		nb.VpBBox = nb.ObjBBox.Add(sc.ObjBBox.Min.Sub(sc.BBox.Min)) // move amount
		nb.VpBBox = nb.VpBBox.Intersect(sc.VpBBox)
		if nb.VpBBox != image.ZR {
			nb.WinBBox = nb.VpBBox.Add(sc.WinBBox.Min.Sub(sc.VpBBox.Min))
		} else {
			nb.WinBBox = nb.VpBBox
		}
	} else {
		// fmt.Printf("not vis: %v  wbb: %v\n", nb.Name(), nb.WorldBBox.BBox)
		nb.ObjBBox = image.ZR
		nb.VpBBox = image.ZR
		nb.WinBBox = image.ZR
	}
}

// RayPick converts a given 2D point in scene image coordinates
// into a ray from the camera position pointing through line of sight of camera
// into *local* coordinates of the solid.
// This can be used to find point of intersection in local coordinates relative
// to a given plane of interest, for example (see Ray methods for intersections).
// To convert mouse window-relative coords into scene-relative coords
// subtract the sc.ObjBBox.Min which includes any scrolling effects
func (nb *Node3DBase) RayPick(pos image.Point, sc *Scene) mat32.Ray {
	nb.PoseMu.RLock()
	sc.Camera.CamMu.RLock()
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
	sc.Camera.CamMu.RUnlock()
	invM, err := nb.Pose.WorldMatrix.Inverse()
	nb.PoseMu.RUnlock()
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
func (nb *Node3DBase) WorldMatrix() *mat32.Mat4 {
	nb.PoseMu.RLock()
	defer nb.PoseMu.RUnlock()
	return &nb.Pose.WorldMatrix
}

// NormDCBBox returns the normalized display coordinates bounding box
// which is used for clipping.  This is read-lock protected.
func (nb *Node3DBase) NormDCBBox() mat32.Box3 {
	nb.BBoxMu.RLock()
	defer nb.BBoxMu.RUnlock()
	return nb.NDCBBox
}

func (nb *Node3DBase) Init3D(sc *Scene) {
	// nop by default -- could connect to scene for update signals or something
}

func (nb *Node3DBase) Style3D(sc *Scene) {
	pagg := nb.ParentCSSAgg()
	if pagg != nil {
		gi.AggCSS(&nb.CSSAgg, *pagg)
	} else {
		nb.CSSAgg = nil // restart
	}
	gi.AggCSS(&nb.CSSAgg, nb.CSS)
}

func (nb *Node3DBase) UpdateNode3D(sc *Scene) {
}

func (nb *Node3DBase) Render3D(sc *Scene, rc RenderClasses, rnd Render) {
	// nop
}

// SetPosePos sets Pose.Pos position to given value, under write lock protection
func (nb *Node3DBase) SetPosePos(pos mat32.Vec3) {
	nb.PoseMu.Lock()
	nb.Pose.Pos = pos
	nb.PoseMu.Unlock()
}

// SetPoseScale sets Pose.Scale scale to given value, under write lock protection
func (nb *Node3DBase) SetPoseScale(scale mat32.Vec3) {
	nb.PoseMu.Lock()
	nb.Pose.Scale = scale
	nb.PoseMu.Unlock()
}

// SetPoseQuat sets Pose.Quat to given value, under write lock protection
func (nb *Node3DBase) SetPoseQuat(quat mat32.Quat) {
	nb.PoseMu.Lock()
	nb.Pose.Quat = quat
	nb.PoseMu.Unlock()
}

/////////////////////////////////////////////////////////////////
// Events

// Default node can be selected / manipulated per the Scene SelMode settings
func (nb *Node3DBase) ConnectEvents3D(sc *Scene) {
	nb.ConnectEvent(sc.Win, oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		if me.Action != mouse.Press || !nb.IsVisible() || nb.IsInactive() {
			return
		}
		sci, err := recv.ParentByTypeTry(KiT_Scene, ki.Embeds)
		if err != nil {
			return
		}
		ssc := sci.Embed(KiT_Scene).(*Scene)
		ni := nb.This().(Node3D)
		if ssc.CurSel != ni {
			ssc.SetSel(ni)
			me.SetProcessed()
		}
	})
}

// ConnectEvent connects this node to receive a given type of GUI event
// signal from the parent window -- typically connect only visible nodes, and
// disconnect when not visible
func (nb *Node3DBase) ConnectEvent(win *gi.Window, et oswin.EventType, pri gi.EventPris, fun ki.RecvFunc) {
	win.EventMgr.ConnectEvent(nb.This(), et, pri, fun)
}

// DisconnectEvent disconnects this receiver from receiving given event
// type -- pri is priority -- pass AllPris for all priorities -- see also
// DisconnectAllEvents
func (nb *Node3DBase) DisconnectEvent(win *gi.Window, et oswin.EventType, pri gi.EventPris) {
	win.EventMgr.DisconnectEvent(nb.This(), et, pri)
}

// DisconnectAllEvents disconnects node from all window events -- typically
// disconnect when not visible -- pri is priority -- pass AllPris for all priorities.
// This goes down the entire tree from this node on down, as typically everything under
// will not get an explicit disconnect call because no further updating will happen
func (nb *Node3DBase) DisconnectAllEvents(win *gi.Window, pri gi.EventPris) {
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
		_, ni := KiToNode3D(k)
		if ni == nil {
			return ki.Break // going into a different type of thing, bail
		}
		win.EventMgr.DisconnectAllEvents(ni.This(), pri)
		return ki.Continue
	})
}

// TrackCamera moves this node to pose of camera
func (nb *Node3DBase) TrackCamera(sc *Scene) {
	nb.PoseMu.Lock()
	sc.Camera.CamMu.RLock()
	nb.Pose.CopyFrom(&sc.Camera.Pose)
	sc.Camera.CamMu.RUnlock()
	nb.PoseMu.Unlock()
}

// TrackLight moves node to position of light of given name.
// For SpotLight, copies entire Pose. Does not work for Ambient light
// which has no position information.
func (nb *Node3DBase) TrackLight(sc *Scene, lightName string) error {
	nb.PoseMu.Lock()
	defer nb.PoseMu.Unlock()
	lt, ok := sc.Lights[lightName]
	if !ok {
		return fmt.Errorf("gi3d Node: %v TrackLight named: %v not found", nb.Path(), lightName)
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
