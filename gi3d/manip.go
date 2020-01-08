// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SelMode are selection modes for Scene
type SelMode int

const (
	// NotSelectable means that selection events are ignored entirely
	NotSelectable SelMode = iota

	// Selectable means that nodes can be selected but no visible consequence occurs
	Selectable

	// SelectionBox means that a selection bounding box is drawn around selected nodes
	SelectionBox

	// Manipulable means that a manipulation box will be created for selected nodes,
	// which can update the Pose parameters dynamically.
	Manipulable

	SelModeN
)

//go:generate stringer -type=SelMode

var KiT_SelMode = kit.Enums.AddEnum(SelModeN, kit.NotBitFlag, nil)

/////////////////////////////////////////////////////////////////////////////////////
// 		Scene interface

// SetSel -- if Selectable is true, then given object is selected
// if node is nil then selection is reset.
func (sc *Scene) SetSel(nd Node3D) {
	if sc.SelMode == NotSelectable {
		return
	}
	if sc.CurSel == nd {
		return
	}
	if nd == nil {
		if sc.CurSel != nil {
			sc.CurSel.AsNode3D().ClearSelected()
		}
		sc.CurManipPt = nil
		sc.CurSel = nil
		updt := sc.UpdateStart()
		sc.DeleteChildByName(SelBoxName, true)
		sc.DeleteChildByName(ManipBoxName, true)
		sc.UpdateEnd(updt)
		return
	}
	sc.CurSel = nd
	nd.AsNode3D().SetSelected()
	switch sc.SelMode {
	case Selectable:
		return
	case SelectionBox:
		sc.SelectBox()
	case Manipulable:
		sc.ManipBox()
	}
}

// SelectBox draws a selection box around selected node
func (sc *Scene) SelectBox() {
	if sc.CurSel == nil {
		return
	}
	updt := sc.UpdateStart()
	defer sc.UpdateEnd(updt)

	nb := sc.CurSel.AsNode3D()
	sc.DeleteChildByName(SelBoxName, true) // get rid of existing
	clr, _ := gi.ColorFromName(string(sc.SelBoxColorName))
	AddNewLineBox(sc, sc, SelBoxName, SelBoxName, nb.WorldBBox.BBox, .01, clr, Inactive)
	sc.InitMesh(SelBoxName + "-front")
	sc.InitMesh(SelBoxName + "-side")
}

// ManipBox draws a manipulation box around selected node
func (sc *Scene) ManipBox() {
	sc.CurManipPt = nil
	if sc.CurSel == nil {
		return
	}
	updt := sc.UpdateStart()
	defer sc.UpdateEnd(updt)
	nm := ManipBoxName

	nb := sc.CurSel.AsNode3D()
	sc.DeleteChildByName(nm, true) // get rid of existing
	clr, _ := gi.ColorFromName(string(sc.SelBoxColorName))

	bbox := nb.WorldBBox.BBox
	mb := AddNewLineBox(sc, sc, nm, nm, bbox, .01, clr, Inactive)

	mbspm := AddNewSphere(sc, nm+"-pt", 0.05, 16)

	bbox.Min.SetSub(mb.Pose.Pos)
	bbox.Max.SetSub(mb.Pose.Pos)
	AddNewManipPt(sc, mb, nm+"-lll", mbspm.Name(), clr, bbox.Min)
	AddNewManipPt(sc, mb, nm+"-llu", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Min.Y, bbox.Max.Z})
	AddNewManipPt(sc, mb, nm+"-lul", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Max.Y, bbox.Min.Z})
	AddNewManipPt(sc, mb, nm+"-ull", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Min.Y, bbox.Min.Z})
	AddNewManipPt(sc, mb, nm+"-luu", mbspm.Name(), clr, mat32.Vec3{bbox.Min.X, bbox.Max.Y, bbox.Max.Z})
	AddNewManipPt(sc, mb, nm+"-ulu", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Min.Y, bbox.Max.Z})
	AddNewManipPt(sc, mb, nm+"-uul", mbspm.Name(), clr, mat32.Vec3{bbox.Max.X, bbox.Max.Y, bbox.Min.Z})
	AddNewManipPt(sc, mb, nm+"-uuu", mbspm.Name(), clr, bbox.Max)

	sc.InitMesh(nm + "-front")
	sc.InitMesh(nm + "-side")
	sc.InitMesh(nm + "-pt")
}

// SetManipPt sets the CurManipPt
func (sc *Scene) SetManipPt(pt *ManipPt) {
	sc.CurManipPt = pt
}

///////////////////////////////////////////////////////////////////////////
//  ManipPt is a manipulation point

// ManipPt is a manipulation control point
type ManipPt struct {
	Solid
}

var KiT_ManipPt = kit.Types.AddType(&ManipPt{}, ManipPtProps)

// AddNewManipPt adds a new manipulation point
func AddNewManipPt(sc *Scene, parent ki.Ki, name string, meshName string, clr gi.Color, pos mat32.Vec3) *ManipPt {
	mpt := parent.AddNewChild(KiT_ManipPt, name).(*ManipPt)
	mpt.SetMeshName(sc, meshName)
	mpt.Defaults()
	mpt.Pose.Pos = pos
	mpt.Mat.Color = clr
	return mpt
}

// Default ManipPt can be selected and manipulated
func (mpt *ManipPt) ConnectEvents3D(sc *Scene) {
	mpt.ConnectEvent(sc.Win, oswin.MouseEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		if me.Action != mouse.Press || !sc.IsVisible() {
			return
		}
		sci, err := recv.ParentByTypeTry(KiT_Scene, false)
		if err != nil {
			return
		}
		scc := sci.Embed(KiT_Scene).(*Scene)
		scc.SetManipPt(mpt)
		me.SetProcessed()
	})
	mpt.ConnectEvent(sc.Win, oswin.MouseDragEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		mpt := recv.Embed(KiT_ManipPt).(*ManipPt)
		mb := mpt.Par.(*Group)
		sci, err := mpt.ParentByTypeTry(KiT_Scene, false)
		if err != nil {
			return
		}
		ssc := sci.Embed(KiT_Scene).(*Scene)
		if ssc.CurSel == nil {
			return
		}
		sn := ssc.CurSel.AsNode3D()
		updt := ssc.UpdateStart()
		if mpt.IsDragging() {
			scDel := float32(.01)
			panDel := float32(.01)
			if !ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Push(cursor.HandOpen)
				ssc.SetDragCursor = true
			}
			del := me.Where.Sub(me.From)
			dx := float32(del.X)
			dy := float32(del.Y)
			switch {
			case key.HasAllModifierBits(me.Modifiers, key.Shift):
				ssc.Camera.Pan(dx*panDel, -dy*panDel)
			case key.HasAllModifierBits(me.Modifiers, key.Control):
				sn.Pose.Scale.X += dx * scDel
				sn.Pose.Scale.Y += -dy * scDel
				mb.Pose.Scale.X += dx * scDel
				mb.Pose.Scale.Y += -dy * scDel
			case key.HasAllModifierBits(me.Modifiers, key.Alt):
				ssc.Camera.PanTarget(dx*panDel, -dy*panDel, 0)
			default:
				// if mat32.Abs(dx) > mat32.Abs(dy) {
				// 	dy = 0
				// } else {
				// 	dx = 0
				// }
				// ssc.Camera.Orbit(-dx*orbDel, -dy*orbDel)
				sn.Pose.Pos.X += dx * panDel
				sn.Pose.Pos.Y += -dy * panDel
				mb.Pose.Pos.X += dx * panDel
				mb.Pose.Pos.Y += -dy * panDel
			}
		} else {
			if ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.Viewport.Win.OSWin).Pop()
				ssc.SetDragCursor = false
			}
		}
		ssc.UpdateEnd(updt)
	})
}

var ManipPtProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
