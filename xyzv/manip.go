// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

import (
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/xyz"
)

// SelModes are selection modes for Scene
type SelModes int32 //enums:enum

const (
	// NotSelectable means that selection events are ignored entirely
	NotSelectable SelModes = iota

	// Selectable means that nodes can be selected but no visible consequence occurs
	Selectable

	// SelectionBox means that a selection bounding box is drawn around selected nodes
	SelectionBox

	// Manipulable means that a manipulation box will be created for selected nodes,
	// which can update the Pose parameters dynamically.
	Manipulable
)

const (
	// SelBoxName is the reserved top-level Group name for holding
	// a bounding box or manipulator for currently selected object.
	// also used for meshes representing the box.
	SelBoxName = "__SelectedBox"

	// ManipBoxName is the reserved top-level name for meshes
	// representing the manipulation box.
	ManipBoxName = "__ManipBox"
)

// SelParams are parameters for selection / manipulation box
type SelParams struct {
	// color for selection box (default yellow)
	Color color.RGBA

	// width of the box lines, scaled by view distance
	Width float32 `default:"0.001"`

	// radius of the manipulation control point spheres, scaled by view distance
	Radius float32 `default:"0.005"`
}

func (sp *SelParams) Defaults() {
	sp.Color = colors.Yellow
	sp.Width = .001
	sp.Radius = .005
}

/////////////////////////////////////////////////////////////////////////////////////
// 		Scene interface

// SetSel -- if Selectable is true, then given object is selected
// if node is nil then selection is reset.
func (sw *Scene) SetSel(nd xyz.Node) {
	if sw.SelMode == NotSelectable {
		return
	}
	if sw.CurSel == nd {
		return
	}
	xy := sw.XYZ
	if nd == nil {
		// if sv.CurSel != nil {
		// 	sv.CurSel.AsNode().SetSelected(false)
		// }
		sw.CurManipPt = nil
		sw.CurSel = nil
		xy.DeleteChildByName(SelBoxName)
		xy.DeleteChildByName(ManipBoxName)
		xy.NeedsRender()
		return
	}
	manip, ok := nd.(*ManipPt)
	if ok {
		sw.CurManipPt = manip
		return
	}
	sw.CurSel = nd
	// nd.AsNode().SetSelected()
	switch sw.SelMode {
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
	if sw.CurSel == nil {
		return
	}
	xy := sw.XYZ

	nb := sw.CurSel.AsNode()
	xy.DeleteChildByName(SelBoxName) // get rid of existing
	clr := sw.SelParams.Color
	xyz.NewLineBox(xy, xy, SelBoxName, SelBoxName, nb.WorldBBox.BBox, sw.SelParams.Width, clr, xyz.Inactive)

	xy.NeedsUpdate()
	sw.NeedsRender()
}

// ManipBox draws a manipulation box around selected node
func (sw *Scene) ManipBox() {
	sw.CurManipPt = nil
	if sw.CurSel == nil {
		return
	}
	xy := sw.XYZ

	nm := ManipBoxName

	nb := sw.CurSel.AsNode()
	xy.DeleteChildByName(nm) // get rid of existing
	clr := sw.SelParams.Color

	cdist := mat32.Max(xy.Camera.DistTo(xy.Camera.Target), 1.0)

	bbox := nb.WorldBBox.BBox
	mb := xyz.NewLineBox(xy, xy, nm, nm, bbox, sw.SelParams.Width*cdist, clr, xyz.Inactive)

	mbspm := xyz.NewSphere(xy, nm+"-pt", sw.SelParams.Radius*cdist, 16)

	bbox.Min.SetSub(mb.Pose.Pos)
	bbox.Max.SetSub(mb.Pose.Pos)
	NewManipPt(mb, nm+"-lll", mbspm.Name(), clr, bbox.Min)
	NewManipPt(mb, nm+"-llu", mbspm.Name(), clr, mat32.V3(bbox.Min.X, bbox.Min.Y, bbox.Max.Z))
	NewManipPt(mb, nm+"-lul", mbspm.Name(), clr, mat32.V3(bbox.Min.X, bbox.Max.Y, bbox.Min.Z))
	NewManipPt(mb, nm+"-ull", mbspm.Name(), clr, mat32.V3(bbox.Max.X, bbox.Min.Y, bbox.Min.Z))
	NewManipPt(mb, nm+"-luu", mbspm.Name(), clr, mat32.V3(bbox.Min.X, bbox.Max.Y, bbox.Max.Z))
	NewManipPt(mb, nm+"-ulu", mbspm.Name(), clr, mat32.V3(bbox.Max.X, bbox.Min.Y, bbox.Max.Z))
	NewManipPt(mb, nm+"-uul", mbspm.Name(), clr, mat32.V3(bbox.Max.X, bbox.Max.Y, bbox.Min.Z))
	NewManipPt(mb, nm+"-uuu", mbspm.Name(), clr, bbox.Max)

	xy.NeedsConfig()
	sw.NeedsRender()
}

// SetManipPt sets the CurManipPt
func (sw *Scene) SetManipPt(pt *ManipPt) {
	sw.CurManipPt = pt
}

///////////////////////////////////////////////////////////////////////////
//  ManipPt is a manipulation point

// ManipPt is a manipulation control point
//
//core:no-new
type ManipPt struct {
	xyz.Solid
}

// NewManipPt adds a new manipulation point
func NewManipPt(par ki.Ki, name string, meshName string, clr color.RGBA, pos mat32.Vec3) *ManipPt {
	mpt := par.NewChild(ManipPtType, name).(*ManipPt)
	mpt.SetMeshName(meshName)
	mpt.Defaults()
	mpt.Pose.Pos = pos
	mpt.Mat.Color = clr
	return mpt
}

func (sw *Scene) HandleSelectEvents() {
	sw.On(events.MouseDown, func(e events.Event) {
		sw.HandleSelectEventsImpl(e)
	})
	sw.On(events.DoubleClick, func(e events.Event) {
		sw.HandleSelectEventsImpl(e)
	})
}

func (sw *Scene) HandleSelectEventsImpl(e events.Event) {
	xy := sw.XYZ
	pos := sw.Geom.ContentBBox.Min
	e.SetLocalOff(e.LocalOff().Add(pos))
	ns := xyz.NodesUnderPoint(xy, e.Pos())
	nsel := len(ns)
	switch {
	case nsel == 0:
		sw.SetSel(nil)
	case nsel == 1:
		sw.SetSel(ns[0])
	default:
		for _, n := range ns {
			if _, ok := n.(*ManipPt); ok {
				sw.SetSel(n)
				return
			}
		}
		if sw.CurSel == nil {
			sw.SetSel(ns[0])
		} else {
			got := false
			for i, n := range ns {
				if sw.CurSel == n {
					if i < nsel-1 {
						sw.SetSel(ns[i+1])
					} else {
						sw.SetSel(ns[0])
					}
					got = true
					break
				}
			}
			if !got {
				sw.SetSel(ns[0])
			}
		}
	}
}

func (sw *Scene) HandleSlideEvents() {
	sw.On(events.SlideMove, func(e events.Event) {
		pos := sw.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		xy := sw.XYZ
		if sw.CurManipPt == nil || sw.CurSel == nil {
			xy.SlideMoveEvent(e)
			sw.NeedsRender()
			return
		}
		sn := sw.CurSel.AsNode()
		mpt := sw.CurManipPt
		mb := mpt.Par.(*xyz.Group)
		del := e.PrevDelta()
		dx := float32(del.X)
		dy := float32(del.Y)
		mpos := mpt.Nm[len(ManipBoxName)+1:] // has ull etc for where positioned
		camd, sgn := xy.Camera.ViewMainAxis()
		var dm mat32.Vec3 // delta multiplier
		if mpos[mat32.X] == 'u' {
			dm.X = 1
		} else {
			dm.X = -1
		}
		if mpos[mat32.Y] == 'u' {
			dm.Y = 1
		} else {
			dm.Y = -1
		}
		if mpos[mat32.Z] == 'u' {
			dm.Z = 1
		} else {
			dm.Z = -1
		}
		var dd mat32.Vec3
		switch camd {
		case mat32.X:
			dd.Z = -sgn * dx
			dd.Y = -dy
		case mat32.Y:
			dd.X = dx
			dd.Z = sgn * dy
		case mat32.Z:
			dd.X = sgn * dx
			dd.Y = -dy
		}
		// fmt.Printf("mpos: %v  camd: %v  sgn: %v  dm: %v\n", mpos, camd, sgn, dm)
		cdist := xy.Camera.DistTo(xy.Camera.Target)
		scDel := float32(.0005) * cdist
		panDel := float32(.0005) * cdist
		// todo: use SVG ApplyDeltaXForm logic
		switch {
		case e.HasAllModifiers(key.Control): // scale
			dsc := dd.Mul(dm).MulScalar(scDel)
			mb.Pose.Scale.SetAdd(dsc)
			msc := dsc.MulMat4AsVec4(&sn.Pose.ParMatrix, 0) // this is not quite right but close enough
			sn.Pose.Scale.SetAdd(msc)
		case e.HasAllModifiers(key.Alt): // rotation
			dang := -sgn * dm.Y * (dx + dy)
			if camd == mat32.Y {
				dang = -sgn * dm.X * (dy + dx)
			}
			dang *= 0.01 * cdist
			var rvec mat32.Vec3
			rvec.SetDim(camd, 1)
			mb.Pose.RotateOnAxis(rvec.X, rvec.Y, rvec.Z, dang)
			inv, _ := sn.Pose.WorldMatrix.Inverse() // undo full transform
			mvec := rvec.MulMat4AsVec4(inv, 0)
			sn.Pose.RotateOnAxis(mvec.X, mvec.Y, mvec.Z, dang)
		// case key.HasAllModifierBits(e.Modifiers, key.Shift):
		default: // position
			dpos := dd.MulScalar(panDel)
			inv, _ := sn.Pose.ParMatrix.Inverse() // undo parent's transform
			mpos := dpos.MulMat4AsVec4(inv, 0)
			sn.Pose.Pos.SetAdd(mpos)
			mb.Pose.Pos.SetAdd(dpos)
		}
		xy.NeedsUpdate()
		sw.NeedsRender()
	})
}
