// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SelModes are selection modes for Scene
type SelModes int

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

	SelModesN
)

//go:generate stringer -type=SelModes

var KiT_SelModes = kit.Enums.AddEnum(SelModesN, kit.NotBitFlag, nil)

// SelParams are parameters for selection / manipulation box
type SelParams struct {
	Color  gi.ColorName `desc:"name of color to use for selection box (default yellow)"`
	Width  float32      `desc:"width of the box lines (.01 default)"`
	Radius float32      `desc:"radius of the manipulation control point spheres"`
}

func (sp *SelParams) Defaults() {
	sp.Color = gi.ColorName("yellow")
	sp.Width = .01
	sp.Radius = .05
}

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
		sc.DeleteChildByName(SelBoxName, ki.DestroyKids)
		sc.DeleteChildByName(ManipBoxName, ki.DestroyKids)
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
	sc.DeleteChildByName(SelBoxName, ki.DestroyKids) // get rid of existing
	clr, _ := gist.ColorFromName(string(sc.SelParams.Color))
	AddNewLineBox(sc, sc, SelBoxName, SelBoxName, nb.WorldBBox.BBox, sc.SelParams.Width, clr, Inactive)
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
	sc.DeleteChildByName(nm, ki.DestroyKids) // get rid of existing
	clr, _ := gist.ColorFromName(string(sc.SelParams.Color))

	bbox := nb.WorldBBox.BBox
	mb := AddNewLineBox(sc, sc, nm, nm, bbox, sc.SelParams.Width, clr, Inactive)

	mbspm := AddNewSphere(sc, nm+"-pt", sc.SelParams.Radius, 16)

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
func AddNewManipPt(sc *Scene, parent ki.Ki, name string, meshName string, clr gist.Color, pos mat32.Vec3) *ManipPt {
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
		sci, err := recv.ParentByTypeTry(KiT_Scene, ki.Embeds)
		if err != nil {
			return
		}
		ssc := sci.Embed(KiT_Scene).(*Scene)
		ssc.SetManipPt(mpt)
		me.SetProcessed()
	})
	mpt.ConnectEvent(sc.Win, oswin.MouseDragEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.DragEvent)
		me.SetProcessed()
		mpt := recv.Embed(KiT_ManipPt).(*ManipPt)
		mb := mpt.Par.(*Group)
		sci, err := mpt.ParentByTypeTry(KiT_Scene, ki.Embeds)
		if err != nil {
			return
		}
		ssc := sci.Embed(KiT_Scene).(*Scene)
		if ssc.CurSel == nil {
			return
		}
		sn := ssc.CurSel.AsNode3D()
		if !mpt.IsDragging() {
			if ssc.SetDragCursor {
				oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Pop()
				ssc.SetDragCursor = false
			}
			return
		}
		if !ssc.SetDragCursor {
			oswin.TheApp.Cursor(ssc.ParentWindow().OSWin).Push(cursor.HandOpen)
			ssc.SetDragCursor = true
		}
		del := me.Where.Sub(me.From)
		dx := float32(del.X)
		dy := float32(del.Y)
		mpos := mpt.Nm[len(ManipBoxName)+1:] // has ull etc for where positioned
		camd, sgn := ssc.Camera.ViewMainAxis()
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
		updt := ssc.UpdateStart()
		scDel := float32(.01)
		panDel := float32(.01)
		// todo: use SVG ApplyDeltaXForm logic
		switch {
		case key.HasAllModifierBits(me.Modifiers, key.Control): // scale
			dsc := dd.Mul(dm).MulScalar(scDel)
			mb.Pose.Scale.SetAdd(dsc)
			msc := dsc.MulMat4AsVec4(&sn.Pose.ParMatrix, 0) // this is not quite right but close enough
			sn.Pose.Scale.SetAdd(msc)
		case key.HasAllModifierBits(me.Modifiers, key.Alt): // rotation
			dang := -sgn * dm.Y * (dx + dy)
			if camd == mat32.Y {
				dang = -sgn * dm.X * (dy + dx)
			}
			var rvec mat32.Vec3
			rvec.SetDim(camd, 1)
			mb.Pose.RotateOnAxis(rvec.X, rvec.Y, rvec.Z, dang)
			inv, _ := sn.Pose.WorldMatrix.Inverse() // undo full transform
			mvec := rvec.MulMat4AsVec4(inv, 0)
			sn.Pose.RotateOnAxis(mvec.X, mvec.Y, mvec.Z, dang)
		// case key.HasAllModifierBits(me.Modifiers, key.Shift):
		default: // position
			dpos := dd.MulScalar(panDel)
			inv, _ := sn.Pose.ParMatrix.Inverse() // undo parent's transform
			mpos := dpos.MulMat4AsVec4(inv, 0)
			sn.Pose.Pos.SetAdd(mpos)
			mb.Pose.Pos.SetAdd(dpos)
		}
		ssc.UpdateEnd(updt)
	})
}

var ManipPtProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
