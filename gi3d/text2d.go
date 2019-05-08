// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"image"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Text2D presents 2D rendered text on a vertically-oriented plane
type Text2D struct {
	Object
	Text      string        `desc:"the text string to display"`
	Sty       gi.Style      `json:"-" xml:"-" desc:"styling settings for the text"`
	TxtPos    gi.Vec2D      `xml:"-" json:"-" desc:"position offset of start of text rendering relative to upper-left corner"`
	TxtRender gi.TextRender `view:"-" xml:"-" json:"-" desc:"render data for text label"`
	TxtTex    gpu.Texture2D `view:"-" desc:"gpu texture object for the text -- this is used directly instead of pointing to the Scene Texture resources"`
}

var KiT_Text2D = kit.Types.AddType(&Text2D{}, nil)

// AddNewText2D adds a new object of given name and text string to given parent
func AddNewText2D(sc *Scene, parent ki.Ki, name string, text string) *Text2D {
	txt := parent.AddNewChild(KiT_Text2D, name).(*Text2D)
	tm := sc.Text2DPlaneMesh()
	txt.SetMesh(sc, tm.Name())
	txt.Defaults()
	txt.Text = text
	return txt
}

func (txt *Text2D) Defaults() {
	txt.Object.Defaults()
}

func (txt *Text2D) Init3D(sc *Scene) {
	err := txt.Validate(sc)
	if err != nil {
		txt.SetInvisible()
	}
	txt.Node3DBase.Init3D(sc)
}

// StyleText does basic 2D styling
func (txt *Text2D) StyleText(sc *Scene) {
	txt.Sty.SetStyleProps(nil, *txt.Properties(), sc.Viewport)
	pagg := txt.ParentCSSAgg()
	if pagg != nil {
		gi.AggCSS(&txt.CSSAgg, *pagg)
	} else {
		txt.CSSAgg = nil // restart
	}
	gi.AggCSS(&txt.CSSAgg, txt.CSS)
	txt.Sty.StyleCSS(txt.This().(gi.Node2D), txt.CSSAgg, "", sc.Viewport)
	txt.Sty.SetUnitContext(sc.Viewport, gi.Vec2DZero)
}

func (txt *Text2D) RenderText(sc *Scene) {
	txt.StyleText(sc)
	txt.TxtRender.SetHTML(txt.Text, &txt.Sty.Font, &txt.Sty.Text, &txt.Sty.UnContext, txt.CSSAgg)
	sz := txt.TxtRender.Size
	fmt.Printf("sz1: %v\n", sz)
	txt.TxtRender.LayoutStdLR(&txt.Sty.Text, &txt.Sty.Font, &txt.Sty.UnContext, sz)
	sz = txt.TxtRender.Size
	fmt.Printf("sz2: %v\n", sz)
	txt.TxtRender.LayoutStdLR(&txt.Sty.Text, &txt.Sty.Font, &txt.Sty.UnContext, sz)
	sz = txt.TxtRender.Size
	fmt.Printf("sz3: %v\n", sz)
	szpt := sz.ToPoint()
	var img *image.RGBA
	if txt.TxtTex == nil {
		txt.TxtTex = gpu.TheGPU.NewTexture2D(txt.Nm)
		img = image.NewRGBA(image.Rectangle{Max: szpt})
	} else {
		img = txt.TxtTex.Image().(*image.RGBA)
	}
	txt.TxtTex.SetSize(szpt)
	sc.RenderState.Init(szpt.X, szpt.Y, img)
	txt.TxtRender.Render(&sc.RenderState, txt.TxtPos)
	// todo: need to change world matrix update to include scaling..
}

// Validate checks that object has valid mesh and texture settings, etc
func (txt *Text2D) Validate(sc *Scene) error {
	if txt.Mesh == "" {
		err := fmt.Errorf("gi3d.Object: %s Mesh name is empty", txt.PathUnique())
		log.Println(err)
		return err
	}
	if txt.MeshPtr == nil || txt.MeshPtr.Name() != string(txt.Mesh) {
		err := txt.SetMesh(sc, string(txt.Mesh))
		if err != nil {
			return err
		}
	}
	return txt.Mat.Validate(sc)
}
