// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"cogentcore.org/core/vgpu"
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

	// the text string to display
	Text string

	// styling settings for the text
	Styles styles.Style `set:"-" json:"-" xml:"-"`

	// position offset of start of text rendering relative to upper-left corner
	TxtPos mat32.Vec2 `set:"-" xml:"-" json:"-"`

	// render data for text label
	TxtRender paint.Text `set:"-" xml:"-" json:"-"`

	// render state for rendering text
	RenderState paint.State `set:"-" copier:"-" json:"-" xml:"-" view:"-"`
}

func (txt *Text2D) OnInit() {
	txt.Defaults()
}

func (txt *Text2D) Defaults() {
	txt.Solid.Defaults()
	txt.Pose.Scale.SetScalar(.005)
	txt.Styles.Defaults()
	txt.Styles.Font.Size.Pt(36)
	txt.Styles.Margin.Set(units.Px(2))
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

func (txt *Text2D) Config() {
	tm := txt.Sc.PlaneMesh2D()
	txt.SetMesh(tm)
	txt.Solid.Config()
	txt.RenderText()
	txt.Validate()
}

func (txt *Text2D) RenderText() {
	// TODO(kai): do we need to set unit context sizes? (units.Context.SetSizes)
	st := &txt.Styles
	fr := st.FontRender()
	if st.Font.Face == nil {
		st.Font = paint.OpenFont(fr, &st.UnContext)
	}
	st.ToDots()

	txt.TxtRender.SetHTML(txt.Text, fr, &txt.Styles.Text, &txt.Styles.UnContext, nil)
	sz := txt.TxtRender.Size
	txt.TxtRender.LayoutStdLR(&txt.Styles.Text, fr, &txt.Styles.UnContext, sz)
	if txt.TxtRender.Size != sz {
		sz = txt.TxtRender.Size
		txt.TxtRender.LayoutStdLR(&txt.Styles.Text, fr, &txt.Styles.UnContext, sz)
		if txt.TxtRender.Size != sz {
			sz = txt.TxtRender.Size
		}
	}
	marg := txt.Styles.TotalMargin()
	sz.SetAdd(marg.Size())
	txt.TxtPos = marg.Pos().Round()
	szpt := sz.ToPointRound()
	if szpt == (image.Point{}) {
		szpt = image.Point{X: 10, Y: 10}
	}
	bounds := image.Rectangle{Max: szpt}
	var img *image.RGBA
	var tx Texture
	var err error
	if txt.Mat.TexPtr == nil {
		txname := "__Text2D: " + txt.Nm
		tx, err = txt.Sc.TextureByNameTry(txname)
		if err != nil {
			tx = &TextureBase{Nm: txname}
			txt.Sc.AddTexture(tx)
			img = image.NewRGBA(bounds)
			tx.SetImage(img)
			txt.Mat.SetTexture(tx)
		} else {
			if vgpu.Debug {
				fmt.Printf("xyz.Text2D: error: texture name conflict: %s\n", txname)
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
		txt.Sc.Phong.UpdateTextureName(tx.Name())
	}
	rs := &txt.RenderState
	if rs.Image != img || rs.Image.Bounds() != img.Bounds() {
		rs.Init(szpt.X, szpt.Y, img)
	}
	rs.PushBounds(bounds)
	pt := styles.Paint{}
	pt.Defaults()
	pt.FromStyle(st)
	ctx := &paint.Context{State: rs, Paint: &pt}
	if st.Background != nil {
		draw.Draw(img, bounds, st.Background, image.Point{}, draw.Src)
	}
	txt.TxtRender.Render(ctx, txt.TxtPos)
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
		sc := mat32.V3(sz.X, sz.Y, txt.Pose.Scale.Z)
		ax, ay := txt.Styles.Text.AlignFactors()
		al := txt.Styles.Text.AlignV
		switch al {
		case styles.Start:
			ay = -0.5
		case styles.Center:
			ay = 0
		case styles.End:
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
	// TODO(kai/imageColor)
	return colors.ToUniform(txt.Styles.Background).A < 255
}

func (txt *Text2D) RenderClass() RenderClasses {
	if txt.IsTransparent() {
		return RClassTransTexture
	}
	return RClassOpaqueTexture
}
