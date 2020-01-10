// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
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
	Viewport *gi.Viewport2D `desc:"the viewport to display"`
	Tex      *TextureBase   `view:"-" xml:"-" json:"-" desc:"texture object -- this is used directly instead of pointing to the Scene Texture resources"`
}

var KiT_Embed2D = kit.Types.AddType(&Embed2D{}, Embed2DProps)

// AddNewEmbed2D adds a new embedded 2D viewport of given name and size
func AddNewEmbed2D(sc *Scene, parent ki.Ki, name string, width, height int) *Embed2D {
	em := parent.AddNewChild(KiT_Embed2D, name).(*Embed2D)
	em.Defaults(sc)
	em.Viewport = gi.NewViewport2D(width, height)
	em.Viewport.InitName(em.Viewport, name)
	return em
}

func (em *Embed2D) Defaults(sc *Scene) {
	tm := sc.PlaneMesh2D()
	em.SetMesh(sc, tm)
	em.Solid.Defaults()
	em.Pose.Scale.SetScalar(.005)
	em.Mat.Bright = 1.5 // this is key for making e.g., a white background show up as white..
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
	em.RenderView(sc)
	err := em.Validate(sc)
	if err != nil {
		em.SetInvisible()
	}
	em.Node3DBase.Init3D(sc)
}

func (em *Embed2D) RenderView(sc *Scene) {
	em.Viewport.FullRender2DTree()
	img := em.Viewport.Pixels
	bounds := em.Viewport.Pixels.Bounds()
	setImg := false
	if em.Tex == nil {
		em.Tex = &TextureBase{Nm: em.Nm}
		tx := em.Tex.NewTex()
		tx.SetImage(img) // safe here
	} else {
		im := em.Tex.Tex.Image()
		if im == nil {
			setImg = true // needs to be set on main
		} else {
			tim := im.(*image.RGBA)
			if tim == nil {
				setImg = true
			} else {
				if tim.Bounds() != bounds {
					setImg = true
				}
			}
		}
	}
	if sc.Win != nil {
		oswin.TheApp.RunOnMain(func() {
			sc.Win.OSWin.Activate()
			if setImg {
				em.Tex.Tex.SetImage(img) // does transfer if active
			} else {
				em.Tex.Tex.Transfer(0) // update
			}
		})
	}
	em.Mat.SetTexture(sc, em.Tex)
	// gi.SavePNG("text-test.png", img)
}

// Validate checks that text has valid mesh and texture settings, etc
func (em *Embed2D) Validate(sc *Scene) error {
	// todo: validate more stuff here
	return em.Solid.Validate(sc)
}

func (em *Embed2D) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	if em.Viewport != nil {
		sz := em.Viewport.Geom.Size
		sc := mat32.Vec3{.01 * float32(sz.X), .01 * float32(sz.Y), em.Pose.Scale.Z}
		em.Pose.Matrix.SetTransform(em.Pose.Pos, em.Pose.Quat, sc)
	} else {
		em.Pose.UpdateMatrix()
	}
	em.Pose.UpdateWorldMatrix(parWorld)
	em.SetFlag(int(WorldMatrixUpdated))
}

func (em *Embed2D) UpdateNode3D(sc *Scene) {
	// em.RenderView(sc)
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
	fmt.Printf("ispt: %v   ppt: %v\n", ispt, ppt)
	if ppt.X < 0 || ppt.Y < 0 {
		return ppt, false
	}
	return ppt, true
}

func (em *Embed2D) ConnectEvents3D(sc *Scene) {
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
		_ = ppt
	})
	em.ConnectEvent(sc.Win, oswin.MouseHoverEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		if !sc.IsVisible() {
			return
		}
		me := d.(*mouse.HoverEvent)
		emm := recv.Embed(KiT_Embed2D).(*Embed2D)
		ppt, ok := emm.Project2D(sc, me.Where)
		if !ok {
			return
		}
		_ = ppt
	})
}
