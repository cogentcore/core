// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzv

import (
	"image/color"

	"goki.dev/colors"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/xyz"
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
	Color  color.RGBA `desc:"name of color to use for selection box (default yellow)"`
	Width  float32    `desc:"width of the box lines (.01 default)"`
	Radius float32    `desc:"radius of the manipulation control point spheres"`
}

func (sp *SelParams) Defaults() {
	sp.Color = colors.Yellow
	sp.Width = .01
	sp.Radius = .05
}

/////////////////////////////////////////////////////////////////////////////////////
// 		Scene interface

// SetSel -- if Selectable is true, then given object is selected
// if node is nil then selection is reset.
func (se *Scene3D) SetSel(nd xyz.Node) {
	if se.SelMode == NotSelectable {
		return
	}
	if se.CurSel == nd {
		return
	}
	sc := se.Scene
	if nd == nil {
		// if sv.CurSel != nil {
		// 	sv.CurSel.AsNode().SetSelected(false)
		// }
		se.CurManipPt = nil
		se.CurSel = nil
		updt := sc.UpdateStart()
		sc.DeleteChildByName(SelBoxName, ki.DestroyKids)
		sc.DeleteChildByName(ManipBoxName, ki.DestroyKids)
		sc.UpdateEndRender(updt)
		return
	}
	manip, ok := nd.(*ManipPt)
	if ok {
		se.CurManipPt = manip
		return
	}
	se.CurSel = nd
	// nd.AsNode().SetSelected()
	switch se.SelMode {
	case Selectable:
		return
	case SelectionBox:
		se.SelectBox()
	case Manipulable:
		se.ManipBox()
	}
}

// SelectBox draws a selection box around selected node
func (se *Scene3D) SelectBox() {
	if se.CurSel == nil {
		return
	}
	sc := se.Scene

	updt := sc.UpdateStart()
	defer sc.UpdateEndUpdate(updt)

	nb := se.CurSel.AsNode()
	sc.DeleteChildByName(SelBoxName, ki.DestroyKids) // get rid of existing
	clr := se.SelParams.Color
	xyz.NewLineBox(sc, sc, SelBoxName, SelBoxName, nb.WorldBBox.BBox, se.SelParams.Width, clr, xyz.Inactive)

	se.SetNeedsRender(true)
}

// ManipBox draws a manipulation box around selected node
func (se *Scene3D) ManipBox() {
	se.CurManipPt = nil
	if se.CurSel == nil {
		return
	}
	sc := se.Scene

	updt := sc.UpdateStart()
	defer sc.UpdateEndConfig(updt)

	nm := ManipBoxName

	nb := se.CurSel.AsNode()
	sc.DeleteChildByName(nm, ki.DestroyKids) // get rid of existing
	clr := se.SelParams.Color

	bbox := nb.WorldBBox.BBox
	mb := xyz.NewLineBox(sc, sc, nm, nm, bbox, se.SelParams.Width, clr, xyz.Inactive)

	mbspm := xyz.NewSphere(sc, nm+"-pt", se.SelParams.Radius, 16)

	bbox.Min.SetSub(mb.Pose.Pos)
	bbox.Max.SetSub(mb.Pose.Pos)
	NewManipPt(mb, nm+"-lll", mbspm.Name(), clr, bbox.Min)
	NewManipPt(mb, nm+"-llu", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Min.Y, bbox.Max.Z})
	NewManipPt(mb, nm+"-lul", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Max.Y, bbox.Min.Z})
	NewManipPt(mb, nm+"-ull", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Min.Y, bbox.Min.Z})
	NewManipPt(mb, nm+"-luu", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Max.Y, bbox.Max.Z})
	NewManipPt(mb, nm+"-ulu", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Min.Y, bbox.Max.Z})
	NewManipPt(mb, nm+"-uul", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Max.Y, bbox.Min.Z})
	NewManipPt(mb, nm+"-uuu", mbspm.Name(), clr, bbox.Max)

	se.SetNeedsRender(true)
}

// SetManipPt sets the CurManipPt
func (se *Scene3D) SetManipPt(pt *ManipPt) {
	se.CurManipPt = pt
}

///////////////////////////////////////////////////////////////////////////
//  ManipPt is a manipulation point

// ManipPt is a manipulation control point
//
//goki:no-new
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

func (se *Scene3D) HandleSelectEvents() {
	se.On(events.MouseDown, func(e events.Event) {
		sc := se.Scene
		pos := se.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		ns := xyz.NodesUnderPoint(sc, e.LocalPos())
		nsel := len(ns)
		switch {
		case nsel == 0:
			se.SetSel(nil)
		case nsel == 1:
			se.SetSel(ns[0])
		default:
			for _, n := range ns {
				if _, ok := n.(*ManipPt); ok {
					se.SetSel(n)
					return
				}
			}
			if se.CurSel == nil {
				se.SetSel(ns[0])
			} else {
				got := false
				for i, n := range ns {
					if se.CurSel == n {
						if i < nsel-1 {
							se.SetSel(ns[i+1])
						} else {
							se.SetSel(ns[0])
						}
						got = true
						break
					}
				}
				if !got {
					se.SetSel(ns[0])
				}
			}
		}
	})
}

func (se *Scene3D) HandleSlideEvents() {
	se.On(events.SlideMove, func(e events.Event) {
		pos := se.Geom.ContentBBox.Min
		e.SetLocalOff(e.LocalOff().Add(pos))
		sc := se.Scene
		if se.CurManipPt == nil || se.CurSel == nil {
			sc.SlideMoveEvent(e)
			se.SetNeedsRender(true)
			return
		}
		sn := se.CurSel.AsNode()
		mpt := se.CurManipPt
		mb := mpt.Par.(*xyz.Group)
		del := e.StartDelta()
		dx := float32(del.X)
		dy := float32(del.Y)
		mpos := mpt.Nm[len(ManipBoxName)+1:] // has ull etc for where positioned
		camd, sgn := sc.Camera.ViewMainAxis()
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
		updt := sc.UpdateStart()
		cdist := sc.Camera.DistTo(sc.Camera.Target)
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
		sc.UpdateEnd(updt)
		se.SetNeedsRender(updt)
	})
}
