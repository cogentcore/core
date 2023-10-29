// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/vgpu/v2/vgpu"
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
type Text2D struct { //goki:no-new
	Solid

	// the text string to display
	Text string

	// styling settings for the text
	Styles styles.Style `set:"-" json:"-" xml:"-"`

	// position offset of start of text rendering relative to upper-left corner
	TxtPos mat32.Vec2 `set:"-" xml:"-" json:"-"`

	// render data for text label
	TxtRender paint.Text `set:"-" view:"-" xml:"-" json:"-"`

	// render state for rendering text
	RenderState paint.State `set:"-" copy:"-" json:"-" xml:"-" view:"-"`
}

// NewText2D adds a new text of given name and text string to given parent
func NewText2D(parent ki.Ki, name string) *Text2D {
	txt := parent.NewChild(Text2DType, name).(*Text2D)
	txt.Defaults()
	return txt
}

func (txt *Text2D) Defaults() {
	txt.Solid.Defaults()
	txt.Pose.Scale.SetScalar(.005)
	txt.Styles.Defaults()
	txt.Styles.Font.Size.Pt(36)
	txt.Styles.Margin.Set(units.Px(2))
	txt.Styles.Color = colors.Scheme.OnSurface
	txt.Styles.BackgroundColor.SetSolid(colors.Transparent)
	txt.Mat.Bright = 4 // this is key for making e.g., a white background show up as white..
}

// TextSize returns the size of the text plane, applying all *local* scaling factors
// if nothing rendered yet, returns false
func (txt *Text2D) TextSize() (mat32.Vec2, bool) {
	txt.Pose.Defaults() // only if nil
	sz := mat32.Vec2{}
	tx := txt.Mat.TexPtr
	if tx == nil {
		return sz, false
	}
	tsz := tx.Image().Bounds().Size()
	fsz := float32(txt.Styles.Font.Size.Dots)
	if fsz == 0 {
		fsz = 36
	}
	sz.Set(txt.Pose.Scale.X*float32(tsz.X)/fsz, txt.Pose.Scale.Y*float32(tsz.Y)/fsz)
	return sz, true
}

func (txt *Text2D) Config(sc *Scene) {
	txt.Solid.Config(sc)
	tm := sc.PlaneMesh2D()
	txt.SetMesh(tm)
	txt.RenderText(sc)
	txt.Validate()
}

func (txt *Text2D) RenderText(sc *Scene) {
	gi.SetUnitContext(&txt.Styles, nil, mat32.Vec2{}, mat32.Vec2{})
	txt.TxtRender.SetHTML(txt.Text, txt.Styles.FontRender(), &txt.Styles.Text, &txt.Styles.UnContext, nil)
	sz := txt.TxtRender.Size
	txt.TxtRender.LayoutStdLR(&txt.Styles.Text, txt.Styles.FontRender(), &txt.Styles.UnContext, sz)
	if txt.TxtRender.Size != sz {
		sz = txt.TxtRender.Size
		txt.TxtRender.LayoutStdLR(&txt.Styles.Text, txt.Styles.FontRender(), &txt.Styles.UnContext, sz)
		if txt.TxtRender.Size != sz {
			sz = txt.TxtRender.Size
		}
	}
	marg := txt.Styles.TotalMargin()
	sz.SetAdd(marg.Size())
	txt.TxtPos = marg.Pos()
	szpt := sz.ToPoint()
	if szpt == (image.Point{}) {
		szpt = image.Point{10, 10}
	}
	bounds := image.Rectangle{Max: szpt}
	var img *image.RGBA
	var tx Texture
	var err error
	if txt.Mat.TexPtr == nil {
		txname := "__Text2D: " + txt.Nm
		tx, err = sc.TextureByNameTry(txname)
		if err != nil {
			tx = &TextureBase{Nm: txname}
			sc.AddTexture(tx)
			img = image.NewRGBA(bounds)
			tx.SetImage(img)
			txt.Mat.SetTexture(tx)
		} else {
			if vgpu.Debug {
				fmt.Printf("gi3d.Text2D: error: texture name conflict: %s\n", txname)
			}
			txt.Mat.SetTexture(tx)
			img = tx.Image()
		}
	} else {
		tx = txt.Mat.TexPtr
		img = tx.Image()
		if img.Bounds() != bounds {
			img = image.NewRGBA(bounds)
		}
		tx.SetImage(img)
		sc.Phong.UpdateTextureName(tx.Name())
	}
	rs := &txt.RenderState
	if rs.Image != img || rs.Image.Bounds() != img.Bounds() {
		rs.Init(szpt.X, szpt.Y, img)
	}
	rs.PushBounds(bounds)
	// draw.Draw(img, bounds, &image.Uniform{txt.Styles.BackgroundColor.Color}, image.Point{}, draw.Src)
	txt.TxtRender.Render(rs, txt.TxtPos)
	rs.PopBounds()
}

// Validate checks that text has valid mesh and texture settings, etc
func (txt *Text2D) Validate() error {
	// todo: validate more stuff here
	return txt.Solid.Validate()
}

func (txt *Text2D) UpdateWorldMatrix(parWorld *mat32.Mat4) {
	txt.PoseMu.Lock()
	defer txt.PoseMu.Unlock()
	sz, ok := txt.TextSize()
	if ok {
		sc := mat32.Vec3{sz.X, sz.Y, txt.Pose.Scale.Z}
		ax, ay := txt.Styles.Text.AlignFactors()
		al := txt.Styles.Text.AlignV
		switch {
		case styles.IsAlignStart(al):
			ay = -0.5
		case styles.IsAlignMiddle(al):
			ay = 0
		case styles.IsAlignEnd(al):
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
	txt.SetFlag(true, WorldMatrixUpdated)
}

func (txt *Text2D) IsTransparent() bool {
	if txt.Styles.BackgroundColor.Solid.A < 255 {
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
