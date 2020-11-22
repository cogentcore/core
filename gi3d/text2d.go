// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image"
	"image/draw"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Text2D presents 2D rendered text on a vertically-oriented plane, using a texture.
// Call SetText() which calls RenderText to update fortext changes (re-renders texture).
// The native scale is such that a unit height value is the height of the default font
// set by the font-size property, and the X axis is scaled proportionally based on the
// rendered text size to maintain the aspect ratio.  Further scaling can be applied on
// top of that by setting the Pose.Scale values as usual.
// Standard styling properties can be set on the node to set font size, family,
// and text alignment relative to the Pose.Pos position (e.g., Left, Top puts the
// upper-left corner of text at Pos).
// Note that higher quality is achieved by using a larger font size (36 default).
// The margin property creates blank margin of the background color around the text
// (2 px default) and the background-color defaults to transparent
// but can be set to any color.
type Text2D struct {
	Solid
	Text        string       `desc:"the text string to display"`
	Sty         gist.Style   `json:"-" xml:"-" desc:"styling settings for the text"`
	TxtPos      mat32.Vec2   `xml:"-" json:"-" desc:"position offset of start of text rendering relative to upper-left corner"`
	TxtRender   girl.Text    `view:"-" xml:"-" json:"-" desc:"render data for text label"`
	TxtTex      *TextureBase `view:"-" xml:"-" json:"-" desc:"texture object for the text -- this is used directly instead of pointing to the Scene Texture resources"`
	RenderState girl.State   `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering text"`
}

var KiT_Text2D = kit.Types.AddType(&Text2D{}, Text2DProps)

// AddNewText2D adds a new text of given name and text string to given parent
func AddNewText2D(sc *Scene, parent ki.Ki, name string, text string) *Text2D {
	txt := parent.AddNewChild(KiT_Text2D, name).(*Text2D)
	txt.Defaults(sc)
	txt.Text = text
	return txt
}

func (txt *Text2D) Defaults(sc *Scene) {
	tm := sc.PlaneMesh2D()
	txt.SetMesh(sc, tm)
	txt.Solid.Defaults()
	txt.Pose.Scale.SetScalar(.005)
	txt.SetProp("font-size", units.NewPt(36))
	txt.SetProp("margin", units.NewPx(2))
	txt.SetProp("color", &gi.Prefs.Colors.Font)
	txt.SetProp("background-color", gist.Color{0, 0, 0, 0})
	txt.Mat.Bright = 5 // this is key for making e.g., a white background show up as white..
}

func (txt *Text2D) Disconnect() {
	if txt.TxtTex != nil && txt.TxtTex.Tex.IsActive() {
		scc, err := txt.ParentByTypeTry(KiT_Scene, ki.Embeds)
		if err == nil {
			sc := scc.Embed(KiT_Scene).(*Scene)
			if sc.Win != nil && sc.Win.IsVisible() {
				oswin.TheApp.RunOnMain(func() {
					sc.Win.OSWin.Activate()
					txt.TxtTex.Tex.Delete()
				})
			}
		}
	}
	txt.Solid.Disconnect()
}

// SetText sets the text and renders it to the texture image
func (txt *Text2D) SetText(sc *Scene, str string) {
	txt.Text = str
	txt.RenderText(sc)
}

// TextSize returns the size of the text plane, applying all *local* scaling factors
// if nothing rendered yet, returns false
func (txt *Text2D) TextSize() (mat32.Vec2, bool) {
	txt.Pose.Defaults() // only if nil
	sz := mat32.Vec2{}
	if txt.TxtTex == nil {
		return sz, false
	}
	tsz := txt.TxtTex.Tex.Size()
	fsz := float32(txt.Sty.Font.Size.Dots)
	if fsz == 0 {
		fsz = 36
	}
	sz.Set(txt.Pose.Scale.X*float32(tsz.X)/fsz, txt.Pose.Scale.Y*float32(tsz.Y)/fsz)
	return sz, true
}

func (txt *Text2D) Init3D(sc *Scene) {
	txt.RenderText(sc)
	err := txt.Validate(sc)
	if err != nil {
		txt.SetInvisible()
	}
	txt.Node3DBase.Init3D(sc)
}

// StyleText does basic 2D styling
func (txt *Text2D) StyleText(sc *Scene) {
	txt.Sty.Defaults()
	txt.Sty.SetStyleProps(nil, *txt.Properties(), sc.Viewport)
	pagg := txt.ParentCSSAgg()
	if pagg != nil {
		gi.AggCSS(&txt.CSSAgg, *pagg)
	} else {
		txt.CSSAgg = nil // restart
	}
	// css stuff only works for node2d
	// gi.AggCSS(&txt.CSSAgg, txt.CSS)
	// txt.Sty.StyleCSS(txt.This().(gi.Node2D), txt.CSSAgg, "", sc.Viewport)
	gi.SetUnitContext(&txt.Sty, sc.Viewport, mat32.Vec2Zero)
}

func (txt *Text2D) RenderText(sc *Scene) {
	txt.StyleText(sc)
	txt.TxtRender.SetHTML(txt.Text, &txt.Sty.Font, &txt.Sty.Text, &txt.Sty.UnContext, txt.CSSAgg)
	sz := txt.TxtRender.Size
	txt.TxtRender.LayoutStdLR(&txt.Sty.Text, &txt.Sty.Font, &txt.Sty.UnContext, sz)
	if txt.TxtRender.Size != sz {
		sz = txt.TxtRender.Size
		txt.TxtRender.LayoutStdLR(&txt.Sty.Text, &txt.Sty.Font, &txt.Sty.UnContext, sz)
		if txt.TxtRender.Size != sz {
			sz = txt.TxtRender.Size
		}
	}
	marg := txt.Sty.Layout.Margin.Dots
	sz.SetAddScalar(2 * marg)
	txt.TxtPos.SetScalar(marg)
	szpt := sz.ToPoint()
	if szpt == image.ZP {
		szpt = image.Point{10, 10}
	}
	bounds := image.Rectangle{Max: szpt}
	var img *image.RGBA
	setImg := false
	if txt.TxtTex == nil {
		txt.TxtTex = &TextureBase{Nm: txt.Nm}
		tx := txt.TxtTex.NewTex()
		img = image.NewRGBA(bounds)
		tx.SetImage(img) // safe here
	} else {
		im := txt.TxtTex.Tex.Image()
		if im == nil {
			img = image.NewRGBA(bounds)
			setImg = true // needs to be set on main
		} else {
			img = im.(*image.RGBA)
			if img == nil {
				img = image.NewRGBA(bounds)
				setImg = true
			} else {
				if img.Bounds() != bounds {
					img = image.NewRGBA(bounds)
					setImg = true
				}
			}
		}
	}
	rs := &txt.RenderState
	if rs.Image != img || rs.Image.Bounds() != img.Bounds() {
		rs.Init(szpt.X, szpt.Y, img)
	}
	rs.PushBounds(bounds)
	draw.Draw(img, bounds, &image.Uniform{txt.Sty.Font.BgColor.Color}, image.ZP, draw.Src)
	txt.TxtRender.Render(rs, txt.TxtPos)
	rs.PopBounds()
	if sc.Win != nil {
		oswin.TheApp.RunOnMain(func() {
			sc.Win.OSWin.Activate()
			if setImg {
				txt.TxtTex.Tex.SetImage(img) // does transfer if active
			} else {
				txt.TxtTex.Tex.Transfer(0) // update
			}
		})
	}
	txt.Mat.SetTexture(sc, txt.TxtTex)
	// gi.SavePNG("text-test.png", img)
}

// Validate checks that text has valid mesh and texture settings, etc
func (txt *Text2D) Validate(sc *Scene) error {
	// todo: validate more stuff here
	return txt.Solid.Validate(sc)
}

func (txt *Text2D) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	txt.PoseMu.Lock()
	defer txt.PoseMu.Unlock()
	sz, ok := txt.TextSize()
	if ok {
		sc := mat32.Vec3{sz.X, sz.Y, txt.Pose.Scale.Z}
		ax, ay := txt.Sty.Text.AlignFactors()
		al := txt.Sty.Text.AlignV
		switch {
		case gist.IsAlignStart(al):
			ay = -0.5
		case gist.IsAlignMiddle(al):
			ay = 0
		case gist.IsAlignEnd(al):
			ay = 0.5
		}
		ps := txt.Pose.Pos
		ps.X += (0.5 - ax) * sz.X
		ps.Y += ay * sz.Y
		txt.Pose.Matrix.SetTransform(ps, txt.Pose.Quat, sc)
	} else {
		txt.Pose.UpdateMatrix()
	}
	txt.Pose.UpdateWorldMatrix(parWorld)
	txt.SetFlag(int(WorldMatrixUpdated))
}

func (txt *Text2D) IsTransparent() bool {
	if txt.Sty.Font.BgColor.Color.A < 255 {
		return true
	}
	return false
}

func (txt *Text2D) RenderClass() RenderClasses {
	if txt.IsTransparent() {
		return RClassTransTexture
	}
	return RClassOpaqueTexture
}

var Text2DProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}
