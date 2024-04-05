// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"image"
	"io"
	"io/fs"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
	"golang.org/x/image/draw"
)

// SVG is a Widget that renders an [svg.SVG] object.
// If it is not [states.ReadOnly], the user can pan and zoom the display.
// By default, it is [states.ReadOnly]. See [giv.ConfigSVGToolbar] for a
// toolbar with panning, selecting, and I/O buttons.
type SVG struct {
	Box

	// SVG is the SVG object associated with the element.
	SVG *svg.SVG `set:"-"`
}

func (sv *SVG) OnInit() {
	sv.SVG = svg.NewSVG(10, 10)
	sv.WidgetBase.OnInit()
	sv.SetStyles()
	sv.HandleEvents()
}

func (sv *SVG) SetStyles() {
	sv.SetReadOnly(true)
	sv.Style(func(s *styles.Style) {
		ro := sv.IsReadOnly()
		s.SetAbilities(!ro, abilities.Slideable, abilities.Clickable, abilities.Scrollable)
		s.Min.Set(units.Dp(256))
	})
	sv.StyleFinal(func(s *styles.Style) {
		sv.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
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
		sv.NeedsRender()
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
		sv.NeedsRender()
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

// Open opens an XML-formatted SVG file
func (sv *SVG) Open(filename Filename) error { //gti:add
	return sv.SVG.OpenXML(string(filename))
}

// OpenSVG opens an XML-formatted SVG file from the given fs.
func (sv *SVG) OpenFS(fsys fs.FS, filename string) error {
	return sv.SVG.OpenFS(fsys, filename)
}

// Read reads an XML-formatted SVG file from the given reader
func (sv *SVG) Read(r io.Reader) error {
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

func (sv *SVG) Render() {
	sv.Box.Render()

	if sv.SVG == nil {
		return
	}
	// need to make the image again to prevent it from
	// rendering over itself
	sv.SVG.Pixels = image.NewRGBA(sv.SVG.Pixels.Rect)
	sv.SVG.RenderState.Init(sv.SVG.Pixels.Rect.Dx(), sv.SVG.Pixels.Rect.Dy(), sv.SVG.Pixels)

	sv.SVG.Render()

	r := sv.Geom.ContentBBox
	sp := sv.Geom.ScrollOffset()
	draw.Draw(sv.Scene.Pixels, r, sv.SVG.Pixels, sp, draw.Over)
}
