// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Embed2D embeds a 2D Viewport on a vertically-oriented plane, using a texture.
// The native scale is such that a unit height value is the height of the default font
// set by the font-size property, and the X axis is scaled proportionally based on the
// rendered text size to maintain the aspect ratio.  Further scaling can be applied on
// top of that by setting the Pose.Scale values as usual.
type Embed2D struct {
	Solid
	Viewport *EmbedViewport `desc:"the embedded viewport to display"`
	Zoom     float32        `desc:"overall scaling factor relative to an arbitrary but sensible default scale based on size of viewport -- increase to increase size of view"`
	Tex      *TextureBase   `view:"-" xml:"-" json:"-" desc:"texture object -- this is used directly instead of pointing to the Scene Texture resources"`
}

var KiT_Embed2D = kit.Types.AddType(&Embed2D{}, Embed2DProps)

// AddNewEmbed2D adds a new embedded 2D viewport of given name and size
func AddNewEmbed2D(sc *Scene, parent ki.Ki, name string, width, height int) *Embed2D {
	em := parent.AddNewChild(KiT_Embed2D, name).(*Embed2D)
	em.Defaults(sc)
	em.Viewport = NewEmbedViewport(sc, em, name, width, height)
	em.Viewport.Fill = true
	return em
}

func (em *Embed2D) Defaults(sc *Scene) {
	tm := sc.PlaneMesh2D()
	em.SetMesh(sc, tm)
	em.Solid.Defaults()
	em.Zoom = 1
	em.Mat.Bright = 1.4 // this is key for making e.g., a white background show up as white..
}

func (em *Embed2D) Disconnect() {
	if em.Tex != nil && em.Tex.Tex.IsActive() {
		scc, err := em.ParentByTypeTry(KiT_Scene, true)
		if err == nil {
			sc := scc.(*Scene)
			if sc.Win != nil && sc.Win.IsVisible() {
				oswin.TheApp.RunOnMain(func() {
					sc.Win.OSWin.Activate()
					em.Tex.Tex.Delete()
				})
			}
		}
	}
	em.Solid.Disconnect()
}

func (em *Embed2D) Init3D(sc *Scene) {
	em.Viewport.SetWin(sc.Win) // make sure
	em.Viewport.FullRender2DTree()
	em.UploadViewTex(sc)
	em.Mat.SetTexture(sc, em.Tex)
	err := em.Validate(sc)
	if err != nil {
		em.SetInvisible()
	}
	em.Node3DBase.Init3D(sc)
}

// UploadViewTex uploads the viewport image to the texture
func (em *Embed2D) UploadViewTex(sc *Scene) {
	img := em.Viewport.Pixels
	if em.Tex == nil {
		em.Tex = &TextureBase{Nm: em.Nm}
		tx := em.Tex.NewTex()
		tx.SetImage(img) // safe here
	}
	if sc.Win != nil {
		oswin.TheApp.RunOnMain(func() {
			sc.Win.OSWin.Activate()
			em.Tex.Tex.SetImage(img) // does transfer if active
		})
	}
	// gi.SavePNG("emb-test.png", img)
}

// Validate checks that text has valid mesh and texture settings, etc
func (em *Embed2D) Validate(sc *Scene) error {
	// todo: validate more stuff here
	return em.Solid.Validate(sc)
}

func (em *Embed2D) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	if em.Viewport != nil {
		sz := em.Viewport.Geom.Size
		sc := mat32.Vec3{.006 * em.Zoom * float32(sz.X), .006 * em.Zoom * float32(sz.Y), em.Pose.Scale.Z}
		em.Pose.Matrix.SetTransform(em.Pose.Pos, em.Pose.Quat, sc)
	} else {
		em.Pose.UpdateMatrix()
	}
	em.Pose.UpdateWorldMatrix(parWorld)
	em.SetFlag(int(WorldMatrixUpdated))
}

func (em *Embed2D) UpdateBBox2D(size mat32.Vec2, sc *Scene) {
	em.Solid.UpdateBBox2D(size, sc)
	em.Viewport.Geom.Pos = em.WinBBox.Min
	em.Viewport.WinBBox.Min = em.WinBBox.Min
	em.Viewport.WinBBox.Max = em.WinBBox.Min.Add(em.Viewport.Geom.Size)
	em.Viewport.FuncDownMeFirst(0, em.Viewport.This(), func(k ki.Ki, level int, d interface{}) bool {
		if k == em.Viewport.This() {
			return true
		}
		_, ni := gi.KiToNode2D(k)
		if ni == nil {
			return false // going into a different type of thing, bail
		}
		ni.SetWinBBox()
		return true
	})
}

func (em *Embed2D) RenderClass() RenderClasses {
	return RClassOpaqueTexture
}

var Embed2DProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}

func (em *Embed2D) Project2D(sc *Scene, pt image.Point) (image.Point, bool) {
	var ppt image.Point
	if em.Viewport == nil || em.Viewport.Pixels == nil {
		return ppt, false
	}
	sz := em.Viewport.Geom.Size
	relpos := pt.Sub(sc.ObjBBox.Min)
	ray := em.RayPick(relpos, sc)
	// is in XY plane with norm pointing up in Z axis
	plane := mat32.Plane{Norm: mat32.Vec3{0, 0, 1}, Off: 0}
	ispt, ok := ray.IntersectPlane(plane)
	if !ok || ispt.Z > 0 { // Z > 0 means clicked "in front" of plane
		return ppt, false
	}
	ppt.X = int((ispt.X + 0.5) * float32(sz.X))
	ppt.Y = int((ispt.Y + 0.5) * float32(sz.Y))
	if ppt.X < 0 || ppt.Y < 0 {
		return ppt, false
	}
	ppt.Y = sz.Y - ppt.Y // top-zero
	// fmt.Printf("ppt: %v\n", ppt)
	ppt = ppt.Add(em.Viewport.Geom.Pos)
	return ppt, true
}

func (em *Embed2D) ConnectEvents3D(sc *Scene) {
	em.SetCanFocus()
	em.ConnectEvent(sc.Win, oswin.MouseEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if !sc.IsVisible() {
			return
		}
		me := d.(*mouse.Event)
		emm := recv.Embed(KiT_Embed2D).(*Embed2D)
		ppt, ok := emm.Project2D(sc, me.Where)
		if !ok {
			return
		}
		md := &mouse.Event{}
		*md = *me
		evToPopup := !sc.Win.CurPopupIsTooltip() // don't send events to tooltips!
		if !evToPopup {
			md.Where = ppt
			sc.Win.SetFocus(em)
		}
		em.Viewport.EventMgr.MouseEvents(md)
		em.Viewport.EventMgr.SendEventSignal(md, evToPopup)
		me.SetProcessed() // must always
	})
	em.ConnectEvent(sc.Win, oswin.MouseMoveEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if !sc.IsVisible() {
			return
		}
		me := d.(*mouse.MoveEvent)
		emm := recv.Embed(KiT_Embed2D).(*Embed2D)
		ppt, ok := emm.Project2D(sc, me.Where)
		if !ok {
			return
		}
		md := &mouse.MoveEvent{}
		*md = *me
		evToPopup := !sc.Win.CurPopupIsTooltip() // don't send events to tooltips!
		if !evToPopup {
			del := ppt.Sub(me.Where)
			md.Where = ppt
			md.From.Add(del)
		}
		em.Viewport.EventMgr.MouseEvents(md)
		em.Viewport.EventMgr.SendEventSignal(md, evToPopup)
		em.Viewport.EventMgr.GenMouseFocusEvents(md, evToPopup)
		me.SetProcessed() // must always
	})
	em.ConnectEvent(sc.Win, oswin.MouseDragEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if !sc.IsVisible() {
			return
		}
		me := d.(*mouse.DragEvent)
		emm := recv.Embed(KiT_Embed2D).(*Embed2D)
		ppt, ok := emm.Project2D(sc, me.Where)
		if !ok {
			return
		}
		md := &mouse.DragEvent{}
		*md = *me
		evToPopup := !sc.Win.CurPopupIsTooltip() // don't send events to tooltips!
		if !evToPopup {
			del := ppt.Sub(me.Where)
			md.Where = ppt
			md.From.Add(del)
		}
		em.Viewport.EventMgr.MouseEvents(md)
		em.Viewport.EventMgr.SendEventSignal(md, evToPopup)
		me.SetProcessed() // must always
	})
	em.ConnectEvent(sc.Win, oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if !sc.IsVisible() {
			return
		}
		kt := d.(*key.ChordEvent)
		evToPopup := !sc.Win.CurPopupIsTooltip() // don't send events to tooltips!
		em.Viewport.EventMgr.SendEventSignal(kt, evToPopup)
		kt.SetProcessed() // must always
	})
	// em.ConnectEvent(sc.Win, oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
	// 	if !sc.IsVisible() {
	// 		return
	// 	}
	// 	me := d.(*mouse.HoverEvent)
	// 	emm := recv.Embed(KiT_Embed2D).(*Embed2D)
	// 	ppt, ok := emm.Project2D(sc, me.Where)
	// 	if !ok {
	// 		return
	// 	}
	// 	_ = ppt
	// })
}

///////////////////////////////////////////////////////////////////
//  EmbedViewport

// EmbedViewport is an embedded viewport with its own event manager to handle
// events instead of using the Window.
type EmbedViewport struct {
	gi.Viewport2D
	EventMgr gi.EventMgr `json:"-" xml:"-" desc:"event manager that handles dispersing events to nodes"`
	Scene    *Scene      `json:"-" xml:"-" desc:"parent scene -- trigger updates"`
	EmbedPar *Embed2D    `json:"-" xml:"-" desc:"parent Embed2D -- render updates"`
}

var KiT_EmbedViewport = kit.Types.AddType(&EmbedViewport{}, EmbedViewportProps)

var EmbedViewportProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Background,
}

// NewEmbedViewport creates a new Pixels Image with the specified width and height,
// and initializes the renderer etc
func NewEmbedViewport(sc *Scene, em *Embed2D, name string, width, height int) *EmbedViewport {
	sz := image.Point{width, height}
	vp := &EmbedViewport{}
	vp.Geom = gi.Geom2DInt{Size: sz}
	vp.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	vp.Render.Init(width, height, vp.Pixels)
	vp.InitName(vp, name)
	vp.Scene = sc
	vp.EmbedPar = em
	vp.Win = vp.Scene.Win
	vp.EventMgr.Master = vp.Win
	return vp
}

func (vp *EmbedViewport) SetWin(win *gi.Window) {
	vp.Win = win
	vp.EventMgr.Master = win
}

func (vp *EmbedViewport) VpTop() gi.Viewport {
	return vp.This().(gi.Viewport)
}

func (vp *EmbedViewport) VpTopNode() gi.Node {
	return vp.This().(gi.Node)
}

func (vp *EmbedViewport) VpEventMgr() *gi.EventMgr {
	return &vp.EventMgr
}

func (vp *EmbedViewport) VpIsVisible() bool {
	if vp.Scene == nil || vp.EmbedPar == nil {
		return false
	}
	return vp.Scene.IsVisible()
}

func (vp *EmbedViewport) VpUploadAll() {
	if !vp.This().(gi.Viewport).VpIsVisible() {
		return
	}
	vp.EmbedPar.UploadViewTex(vp.Scene)
	vp.Scene.UpdateSig() // todo: maybe go up to its viewport?
}

// VpUploadVp uploads our viewport image into the parent window -- e.g., called
// by popups when updating separately
func (vp *EmbedViewport) VpUploadVp() {
	if !vp.This().(gi.Viewport).VpIsVisible() {
		return
	}
	vp.EmbedPar.UploadViewTex(vp.Scene)
	vp.Scene.UpdateSig()
}

// VpUploadRegion uploads node region of our viewport image
func (vp *EmbedViewport) VpUploadRegion(vpBBox, winBBox image.Rectangle) {
	if !vp.This().(gi.Viewport).VpIsVisible() {
		return
	}
	vp.EmbedPar.UploadViewTex(vp.Scene)
	vp.Scene.UpdateSig()
}

///////////////////////////////////////
//  EventMaster API

func (vp *EmbedViewport) EventTopNode() ki.Ki {
	return vp
}

// IsInScope returns whether given node is in scope for receiving events
func (vp *EmbedViewport) IsInScope(node *gi.Node2DBase, popup bool) bool {
	return true // no popups as yet
}

// CurPopupIsTooltip returns true if current popup is a tooltip
func (vp *EmbedViewport) CurPopupIsTooltip() bool {
	return false
}

// DeleteTooltip deletes any tooltip popup (called when hover ends)
func (vp *EmbedViewport) DeleteTooltip() {

}

// IsFocusActive returns true if focus is active in this master
func (vp *EmbedViewport) IsFocusActive() bool {
	return vp.HasFocus()
}

// SetFocusActiveState sets focus active state
func (vp *EmbedViewport) SetFocusActiveState(active bool) {
	vp.SetFocusState(active)
}
