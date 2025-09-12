// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzcore

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
)

// SelectionModes are selection modes for Scene
type SelectionModes int32 //enums:enum

const (
	// NotSelectable means that selection events are ignored entirely
	NotSelectable SelectionModes = iota

	// Selectable means that nodes can be selected but no visible consequence occurs
	Selectable

	// SelectionBox means that a selection bounding box is drawn around selected nodes
	SelectionBox

	// Manipulable means that a manipulation box will be created for selected nodes,
	// which can update the Pose parameters dynamically.
	Manipulable
)

const (
	// SelectedBoxName is the reserved top-level Group name for holding
	// a bounding box or manipulator for currently selected object.
	// also used for meshes representing the box.
	SelectedBoxName = "__SelectedBox"

	// ManipBoxName is the reserved top-level name for meshes
	// representing the manipulation box.
	ManipBoxName = "__ManipBox"
)

// SelectionParams are parameters for selection / manipulation box
type SelectionParams struct {
	// color for selection box (default yellow)
	Color color.RGBA

	// width of the box lines, scaled by view distance
	Width float32 `default:"0.001"`

	// radius of the manipulation control point spheres, scaled by view distance
	Radius float32 `default:"0.005"`
}

func (sp *SelectionParams) Defaults() {
	sp.Color = colors.Yellow
	sp.Width = .001
	sp.Radius = .005
}

/////////////////////////////////////////////////////////////////////////////////////
// 		Scene interface

// SetSelected -- if Selectable is true, then given object is selected
// if node is nil then selection is reset.
func (sw *Scene) SetSelected(nd xyz.Node) {
	if sw.SelectionMode == NotSelectable {
		return
	}
	if sw.CurrentSelected == nd {
		return
	}
	xy := sw.XYZ
	if nd == nil {
		// if sv.CurSel != nil {
		// 	sv.CurSel.AsNode().SetSelected(false)
		// }
		sw.CurrentManipPoint = nil
		sw.CurrentSelected = nil
		xy.DeleteChildByName(SelectedBoxName)
		xy.DeleteChildByName(ManipBoxName)
		return
	}
	manip, ok := nd.(*ManipPoint)
	if ok {
		sw.CurrentManipPoint = manip
		return
	}
	sw.CurrentSelected = nd
	// nd.AsNode().SetSelected()
	switch sw.SelectionMode {
	case Selectable:
		return
	case SelectionBox:
		sw.SelectBox()
	case Manipulable:
		sw.ManipBox()
	}
}

// SelectBox draws a selection box around selected node
func (sw *Scene) SelectBox() {
	if sw.CurrentSelected == nil {
		return
	}
	sc := sw.XYZ

	nb := sw.CurrentSelected.AsNodeBase()
	sc.DeleteChildByName(SelectedBoxName) // get rid of existing
	clr := sw.SelectionParams.Color
	g := xyz.NewGroup(sc)
	g.SetName(SelectedBoxName)
	xyz.InitLineBox(sc, SelectedBoxName, nb.WorldBBox.BBox, sw.SelectionParams.Width, clr, xyz.Inactive)(g)
	sw.NeedsRender()
}

// ManipBox draws a manipulation box around selected node
func (sw *Scene) ManipBox() {
	sw.CurrentManipPoint = nil
	if sw.CurrentSelected == nil {
		return
	}
	sc := sw.XYZ

	nm := ManipBoxName

	nb := sw.CurrentSelected.AsNodeBase()
	// todo: definitely need to use plan based updating here!
	sc.DeleteChildByName(nm) // get rid of existing
	clr := sw.SelectionParams.Color

	cdist := math32.Max(sc.Camera.DistanceTo(sc.Camera.Target), 1.0)

	bbox := nb.WorldBBox.BBox
	mb := xyz.NewGroup(sc)
	mb.SetName(nm)
	xyz.InitLineBox(sc, nm, bbox, sw.SelectionParams.Width*cdist, clr, xyz.Inactive)(mb)

	mbspm := xyz.NewSphere(sc, nm+"-pt", sw.SelectionParams.Radius*cdist, 16)

	bbox.Min.SetSub(mb.Pose.Pos)
	bbox.Max.SetSub(mb.Pose.Pos)
	NewManipPoint(mb, nm+"-lll", mbspm.Name, clr, bbox.Min)
	NewManipPoint(mb, nm+"-llu", mbspm.Name, clr, math32.Vec3(bbox.Min.X, bbox.Min.Y, bbox.Max.Z))
	NewManipPoint(mb, nm+"-lul", mbspm.Name, clr, math32.Vec3(bbox.Min.X, bbox.Max.Y, bbox.Min.Z))
	NewManipPoint(mb, nm+"-ull", mbspm.Name, clr, math32.Vec3(bbox.Max.X, bbox.Min.Y, bbox.Min.Z))
	NewManipPoint(mb, nm+"-luu", mbspm.Name, clr, math32.Vec3(bbox.Min.X, bbox.Max.Y, bbox.Max.Z))
	NewManipPoint(mb, nm+"-ulu", mbspm.Name, clr, math32.Vec3(bbox.Max.X, bbox.Min.Y, bbox.Max.Z))
	NewManipPoint(mb, nm+"-uul", mbspm.Name, clr, math32.Vec3(bbox.Max.X, bbox.Max.Y, bbox.Min.Z))
	NewManipPoint(mb, nm+"-uuu", mbspm.Name, clr, bbox.Max)

	sw.NeedsRender()
}

// ManipPoint is a manipulation control point.
type ManipPoint struct { //core:no-new
	xyz.Solid
}

// NewManipPoint adds a new manipulation point
func NewManipPoint(parent tree.Node, name string, meshName string, clr color.RGBA, pos math32.Vector3) *ManipPoint { // TODO: more standard New functions?
	mpt := tree.New[ManipPoint](parent)
	mpt.SetName(name)
	mpt.SetMeshName(meshName)
	mpt.Defaults()
	mpt.Pose.Pos = pos
	mpt.Material.Color = clr
	return mpt
}

func (sw *Scene) handleSelectEvents() {
	sw.On(events.MouseDown, sw.handleSelectEventsImpl)
	sw.On(events.DoubleClick, sw.handleSelectEventsImpl)
}

func (sw *Scene) handleSelectEventsImpl(e events.Event) {
	xy := sw.XYZ
	pos := sw.Geom.ContentBBox.Min
	e.SetLocalOff(e.LocalOff().Add(pos))
	ns := xyz.NodesUnderPoint(xy, e.Pos())
	nsel := len(ns)
	switch {
	case nsel == 0:
		sw.SetSelected(nil)
	case nsel == 1:
		sw.SetSelected(ns[0])
	default:
		for _, n := range ns {
			if _, ok := n.(*ManipPoint); ok {
				sw.SetSelected(n)
				return
			}
		}
		if sw.CurrentSelected == nil {
			sw.SetSelected(ns[0])
		} else {
			got := false
			for i, n := range ns {
				if sw.CurrentSelected == n {
					if i < nsel-1 {
						sw.SetSelected(ns[i+1])
					} else {
						sw.SetSelected(ns[0])
					}
					got = true
					break
				}
			}
			if !got {
				sw.SetSelected(ns[0])
			}
		}
	}
}

func (sw *Scene) handleSlideEvents() {
	sw.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		pos := sw.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		xy := sw.XYZ
		if sw.CurrentManipPoint == nil || sw.CurrentSelected == nil {
			xy.SlideMoveEvent(e)
			sw.NeedsRender()
			return
		}
		sn := sw.CurrentSelected.AsNodeBase()
		mpt := sw.CurrentManipPoint
		mb := mpt.Parent.(*xyz.Group)
		del := e.PrevDelta()
		dx := float32(del.X)
		dy := float32(del.Y)
		mpos := mpt.Name[len(ManipBoxName)+1:] // has ull etc for where positioned
		camd, sgn := xy.Camera.ViewMainAxis()
		var dm math32.Vector3 // delta multiplier
		if mpos[math32.X] == 'u' {
			dm.X = 1
		} else {
			dm.X = -1
		}
		if mpos[math32.Y] == 'u' {
			dm.Y = 1
		} else {
			dm.Y = -1
		}
		if mpos[math32.Z] == 'u' {
			dm.Z = 1
		} else {
			dm.Z = -1
		}
		var dd math32.Vector3
		switch camd {
		case math32.X:
			dd.Z = -sgn * dx
			dd.Y = -dy
		case math32.Y:
			dd.X = dx
			dd.Z = sgn * dy
		case math32.Z:
			dd.X = sgn * dx
			dd.Y = -dy
		}
		// fmt.Printf("mpos: %v  camd: %v  sgn: %v  dm: %v\n", mpos, camd, sgn, dm)
		cdist := xy.Camera.DistanceTo(xy.Camera.Target)
		scDel := float32(.0005) * cdist
		panDel := float32(.0005) * cdist
		// todo: use SVG ApplyDeltaXForm logic
		switch {
		case e.HasAllModifiers(key.Control): // scale
			dsc := dd.Mul(dm).MulScalar(scDel)
			mb.Pose.Scale.SetAdd(dsc)
			msc := dsc.MulMatrix4AsVector4(&sn.Pose.ParMatrix, 0) // this is not quite right but close enough
			sn.Pose.Scale.SetAdd(msc)
		case e.HasAllModifiers(key.Alt): // rotation
			dang := -sgn * dm.Y * (dx + dy)
			if camd == math32.Y {
				dang = -sgn * dm.X * (dy + dx)
			}
			dang *= 0.01 * cdist
			var rvec math32.Vector3
			rvec.SetDim(camd, 1)
			mb.Pose.RotateOnAxis(rvec.X, rvec.Y, rvec.Z, dang)
			inv, _ := sn.Pose.WorldMatrix.Inverse() // undo full transform
			mvec := rvec.MulMatrix4AsVector4(inv, 0)
			sn.Pose.RotateOnAxis(mvec.X, mvec.Y, mvec.Z, dang)
		// case key.HasAllModifierBits(e.Modifiers, key.Shift):
		default: // position
			dpos := dd.MulScalar(panDel)
			inv, _ := sn.Pose.ParMatrix.Inverse() // undo parent's transform
			mpos := dpos.MulMatrix4AsVector4(inv, 0)
			sn.Pose.Pos.SetAdd(mpos)
			mb.Pose.Pos.SetAdd(dpos)
		}
		sw.NeedsRender()
	})
}
