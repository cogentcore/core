// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/phong"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
)

// Text2D presents 2D rendered text on a vertically oriented plane, using a texture.
// Call SetText() which calls RenderText to update for text changes (re-renders texture).
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
	TextPos math32.Vector2 `set:"-" xml:"-" json:"-"`

	// richText is the conversion of the HTML text source.
	richText rich.Text

	// render data for text label
	textRender *shaped.Lines `set:"-" xml:"-" json:"-"`

	// automatically set to true if the font render color is the default
	// colors.Scheme.OnSurface.  If so, it is automatically updated if the default
	// changes, e.g., in light mode vs dark mode switching.
	usesDefaultColor bool
}

func (txt *Text2D) Init() {
	txt.Defaults()
}

func (txt *Text2D) Defaults() {
	txt.Solid.Defaults()
	txt.Pose.Scale.SetScalar(.005)
	txt.Styles.Defaults()
	txt.Styles.Font.Size.Pt(36)
	txt.Styles.Margin.Set(units.Dp(2))
	txt.Material.Bright = 4 // this is key for making e.g., a white background show up as white..
}

// TextSize returns the size of the text plane, applying all *local* scaling factors
// if nothing rendered yet, returns false
func (txt *Text2D) TextSize() (math32.Vector2, bool) {
	txt.Pose.Defaults() // only if nil
	sz := math32.Vector2{}
	tx := txt.Material.Texture
	if tx == nil {
		return sz, false
	}
	tsz := tx.Image().Bounds().Size()
	fsz := txt.Styles.Font.FontHeight()
	if fsz == 0 {
		fsz = 36
	}
	sz.Set(txt.Pose.Scale.X*float32(tsz.X)/fsz, txt.Pose.Scale.Y*float32(tsz.Y)/fsz)
	return sz, true
}

func (txt *Text2D) Config() {
	tm := txt.Scene.PlaneMesh2D()
	txt.SetMesh(tm)
	txt.Solid.Config()
	txt.RenderText()
	txt.Validate()
}

func (txt *Text2D) RenderText() {
	if txt.Scene == nil || txt.Scene.TextShaper == nil {
		return
	}
	// TODO(kai): do we need to set unit context sizes? (units.Context.SetSizes)
	st := &txt.Styles

	st.ToDots()
	sty, tsty := st.NewRichText()
	if sty.FillColor() == nil {
		txt.usesDefaultColor = true
	}
	if txt.usesDefaultColor {
		tsty.Color = colors.ToUniform(colors.Scheme.OnSurface)
	}
	sz := math32.Vec2(10000, 1000) // just a big size
	txt.richText, _ = htmltext.HTMLToRich([]byte(txt.Text), sty, nil)
	txt.textRender = txt.Scene.TextShaper.WrapLines(txt.richText, sty, tsty, sz)
	sz = txt.textRender.Bounds.Size().Ceil()
	if sz.X == 0 {
		sz.X = 10
	}
	if sz.Y == 0 {
		sz.Y = 10
	}
	marg := txt.Styles.TotalMargin()
	sz.SetAdd(marg.Size())
	txt.TextPos = marg.Pos().Round()

	psty := styles.NewPaint()
	psty.FromStyle(&txt.Styles)
	pc := paint.NewPainter(sz)
	pc.Paint = psty
	if txt.Styles.Background != nil {
		pc.Fill.Color = txt.Styles.Background
		pc.Clear()
	}
	pc.DrawText(txt.textRender, txt.TextPos)
	img := imagex.AsRGBA(paint.RenderToImage(pc))
	var tx Texture
	var err error
	if txt.Material.Texture == nil {
		txname := "__Text2D_" + txt.Name
		tx, err = txt.Scene.TextureByName(txname)
		if err != nil {
			tx = &TextureBase{Name: txname}
			tx.AsTextureBase().RGBA = img
			txt.Scene.SetTexture(tx)
			txt.Material.SetTexture(tx)
		} else {
			if gpu.Debug {
				fmt.Printf("xyz.Text2D: error: texture name conflict: %s\n", txname)
			}
			txt.Material.SetTexture(tx)
		}
	} else {
		tx = txt.Material.Texture
		tx.AsTextureBase().RGBA = img
		txt.Scene.Phong.SetTexture(tx.AsTextureBase().Name, phong.NewTexture(img))
	}
}

// Validate checks that text has valid mesh and texture settings, etc
func (txt *Text2D) Validate() error {
	// todo: validate more stuff here
	return txt.Solid.Validate()
}

func (txt *Text2D) UpdateWorldMatrix(parWorld *math32.Matrix4) {
	sz, ok := txt.TextSize()
	if ok {
		sc := math32.Vec3(sz.X, sz.Y, txt.Pose.Scale.Z)
		ax := txt.Styles.Text.Align.Factor()
		ay := float32(0.0)
		al := txt.Styles.Text.AlignV
		switch al {
		case text.Start:
			ay = -0.5
		case text.Center:
			ay = 0
		case text.End:
			ay = 0.5
		}
		ps := txt.Pose.Pos
		ps.X += (0.5 - ax) * sz.X
		ps.Y += ay * sz.Y
		quat := txt.Pose.Quat
		quat.SetMul(math32.NewQuatAxisAngle(math32.Vec3(0, 1, 0), math32.DegToRad(180)))
		txt.Pose.Matrix.SetTransform(ps, quat, sc)
	} else {
		txt.Pose.UpdateMatrix()
	}
	txt.Pose.UpdateWorldMatrix(parWorld)
}

func (txt *Text2D) IsTransparent() bool {
	return colors.ToUniform(txt.Styles.Background).A < 255
}

func (txt *Text2D) RenderClass() RenderClasses {
	if txt.IsTransparent() {
		return RClassTransTexture
	}
	return RClassOpaqueTexture
}
