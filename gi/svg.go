// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
	"golang.org/x/image/draw"
)

// SVG is a Widget that renders an [svg.SVG] object.
// If it is not [states.ReadOnly], the user can pan and zoom the display.
// SVGs do not render a background or border independent of their SVG object.
type SVG struct {
	WidgetBase

	// SVG is the SVG object associated with the element.
	SVG *svg.SVG `set:"-"`
}

func (sv *SVG) CopyFieldsFrom(frm any) {
	fr := frm.(*SVG)
	sv.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sv.SVG = fr.SVG
}

func (sv *SVG) OnInit() {
	sv.SVG = svg.NewSVG(10, 10)
	sv.SVG.Norm = true
	sv.WidgetBase.OnInit()
	sv.SetStyles()
	sv.HandleEvents()
}

func (sv *SVG) SetStyles() {
	sv.Style(func(s *styles.Style) {
		ro := sv.IsReadOnly()
		s.SetAbilities(!ro, abilities.Slideable, abilities.Pressable, abilities.LongHoverable, abilities.Scrollable)
		s.Grow.Set(1, 1)
		s.Min.Set(units.Dp(sv.SVG.Root.ViewBox.Size.X), units.Dp(sv.SVG.Root.ViewBox.Size.Y))
	})
}

func (sv *SVG) ConfigToolbar(tb *Toolbar) {
	NewButton(tb).SetIcon(icons.PanTool).
		SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
		sv.SetReadOnly(!sv.IsReadOnly())
		sv.ApplyStyleUpdate()
	})
	NewButton(tb).SetIcon(icons.ArrowForward).
		SetTooltip("turn on select mode for selecting SVG elements").
		OnClick(func(e events.Event) {
			fmt.Println("this will select select mode")
		})
	NewSeparator(tb)
	NewButton(tb).SetText("Open SVG").SetIcon(icons.Open).
		SetTooltip("Open from SVG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.OpenSVG)
	})
	NewButton(tb).SetText("Save SVG").SetIcon(icons.Save).
		SetTooltip("Save to SVG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.SaveSVG)
	})
	NewButton(tb).SetText("Save PNG").SetIcon(icons.Save).
		SetTooltip("Save to PNG file").OnClick(func(e events.Event) {
		CallFunc(sv, sv.SavePNG)
	})
}

func (sv *SVG) HandleEvents() {
	sv.On(events.SlideMove, func(e events.Event) {
		if sv.IsReadOnly() {
			return
		}
		e.SetHandled()
		del := e.PrevDelta()
		sv.SVG.Translate.X += float32(del.X)
		sv.SVG.Translate.Y += float32(del.Y)
		sv.SetNeedsRender(true)
	})
	sv.On(events.Scroll, func(e events.Event) {
		if sv.IsReadOnly() {
			return
		}
		e.SetHandled()
		se := e.(*events.MouseScroll)
		sv.SVG.Scale += float32(se.Delta.Y) / 100
		if sv.SVG.Scale <= 0.0000001 {
			sv.SVG.Scale = 0.01
		}
		sv.SetNeedsRender(true)
	})
	sv.On(events.LongHoverStart, func(e events.Event) {
		pos := e.Pos()
		objs := svg.NodesContainingPoint(&sv.SVG.Root, pos, true)
		sv.Tooltip = "no objects under mouse"
		if len(objs) > 0 {
			els := ""
			for _, o := range objs {
				els += o.KiType().Name + ": " + o.Name() + "\n"
			}
			sv.Tooltip = els
		}
	})
}

// OpenSVG opens an XML-formatted SVG file
func (sv *SVG) OpenSVG(filename Filename) error { //gti:add
	return sv.SVG.OpenXML(string(filename))
}

// OpenSVG opens an XML-formatted SVG file from the given fs
func (sv *SVG) OpenFS(fsys fs.FS, filename Filename) error {
	return sv.SVG.OpenFS(fsys, string(filename))
}

// ReadSVG reads an XML-formatted SVG file from the given reader
func (sv *SVG) ReadSVG(r io.Reader) error {
	return sv.SVG.ReadXML(r)
}

// ReadBytes reads an XML-formatted SVG file from the given bytes
func (sv *SVG) ReadBytes(b []byte) error {
	return sv.SVG.ReadXML(bytes.NewReader(b))
}

// SaveSVG saves the current SVG to an XML-encoded standard SVG file
func (sv *SVG) SaveSVG(filename Filename) error { //gti:add
	return sv.SVG.SaveXML(string(filename))
}

// SavePNG saves the current rendered SVG image to an PNG image file
func (sv *SVG) SavePNG(filename Filename) error { //gti:add
	return sv.SVG.SavePNG(string(filename))
}

func (sv *SVG) SizeFinal() {
	sv.WidgetBase.SizeFinal()
	sv.SVG.Resize(sv.Geom.Size.Actual.Content.ToPoint())
}

func (sv *SVG) DrawIntoScene() {
	if sv.SVG == nil {
		return
	}
	sv.SVG.Render()
	r := sv.Geom.ContentBBox
	sp := sv.Geom.ScrollOffset()
	draw.Draw(sv.Sc.Pixels, r, sv.SVG.Pixels, sp, draw.Over)
}

func (sv *SVG) Render() {
	if sv.PushBounds() {
		sv.DrawIntoScene()
		sv.RenderChildren()
		sv.PopBounds()
	}
}
