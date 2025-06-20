// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"image"
	"io"
	"io/fs"
	"strings"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
)

// todo: rewrite svg.SVG to accept an external painter to render to,
// and use that for this, so it renders directly instead of via image.

// SVG is a Widget that renders an [svg.SVG] object.
// If it is not [states.ReadOnly], the user can pan and zoom the display.
// By default, it is [states.ReadOnly].
type SVG struct {
	WidgetBase

	// SVG is the SVG drawing to display.
	SVG *svg.SVG `set:"-"`

	// image renderer
	renderer render.Renderer

	// cached rendered image
	image image.Image

	// prevSize is the cached allocated size for the last rendered image.
	prevSize image.Point `xml:"-" json:"-" set:"-"`
}

func (sv *SVG) Init() {
	sv.WidgetBase.Init()
	sz := math32.Vec2(10, 10)
	sv.SVG = svg.NewSVG(sz)
	sv.renderer = paint.NewImageRenderer(sz)
	sv.SetReadOnly(true)
	sv.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))

		ro := sv.IsReadOnly()
		s.SetAbilities(!ro, abilities.Slideable, abilities.Activatable, abilities.Scrollable)
		if !ro {
			if s.Is(states.Active) {
				s.Cursor = cursors.Grabbing
				s.StateLayer = 0
			} else {
				s.Cursor = cursors.Grab
			}
		}
	})
	sv.FinalStyler(func(s *styles.Style) {
		sv.SVG.Root.ViewBox.PreserveAspectRatio.SetFromStyle(s)
	})

	sv.On(events.SlideMove, func(e events.Event) {
		if sv.IsReadOnly() {
			return
		}
		e.SetHandled()
		sv.SVG.Translate.SetAdd(math32.FromPoint(e.PrevDelta()))
		sv.NeedsRender()
	})
	sv.On(events.Scroll, func(e events.Event) {
		if sv.IsReadOnly() {
			return
		}
		e.SetHandled()
		se := e.(*events.MouseScroll)
		sv.SVG.ZoomAtScroll(se.Delta.Y, se.Pos())
		sv.NeedsRender()
	})
}

// Open opens an XML-formatted SVG file
func (sv *SVG) Open(filename Filename) error { //types:add
	return sv.SVG.OpenXML(string(filename))
}

// OpenSVG opens an XML-formatted SVG file from the given fs.
func (sv *SVG) OpenFS(fsys fs.FS, filename string) error {
	return sv.SVG.OpenFS(fsys, filename)
}

// Read reads an XML-formatted SVG file from the given reader.
func (sv *SVG) Read(r io.Reader) error {
	return sv.SVG.ReadXML(r)
}

// ReadBytes reads an XML-formatted SVG file from the given bytes.
func (sv *SVG) ReadBytes(b []byte) error {
	return sv.SVG.ReadXML(bytes.NewReader(b))
}

// ReadString reads an XML-formatted SVG file from the given string.
func (sv *SVG) ReadString(s string) error {
	return sv.SVG.ReadXML(strings.NewReader(s))
}

// SaveSVG saves the current SVG to an XML-encoded standard SVG file.
func (sv *SVG) SaveSVG(filename Filename) error { //types:add
	return sv.SVG.SaveXML(string(filename))
}

// SaveImage saves the current rendered SVG image to an image file,
// using the filename extension to determine the file type.
func (sv *SVG) SaveImage(filename Filename) error { //types:add
	return sv.SVG.SaveImage(string(filename))
}

func (sv *SVG) SizeFinal() {
	sv.WidgetBase.SizeFinal()
	sz := sv.Geom.Size.Actual.Content
	sv.SVG.SetSize(sz)
	sv.renderer.SetSize(units.UnitDot, sz)
}

// renderSVG renders the SVG
func (sv *SVG) renderSVG() {
	if sv.SVG == nil {
		return
	}
	sv.SVG.TextShaper = sv.Scene.TextShaper()
	sv.renderer.Render(sv.SVG.Render(nil).RenderDone())
	sv.image = imagex.WrapJS(sv.renderer.Image())
	sv.prevSize = sv.image.Bounds().Size()
}

func (sv *SVG) Render() {
	sv.WidgetBase.Render()
	if sv.SVG == nil {
		return
	}
	needsRender := !sv.IsReadOnly()
	if !needsRender {
		if sv.image == nil {
			needsRender = true
		} else {
			sz := sv.image.Bounds().Size()
			if sz != sv.prevSize || sz == (image.Point{}) {
				needsRender = true
			}
		}
	}
	if needsRender {
		sv.renderSVG()
	}
	r := sv.Geom.ContentBBox
	sp := sv.Geom.ScrollOffset()
	sv.Scene.Painter.DrawImage(sv.image, r, sp, draw.Over)
}

func (sv *SVG) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *Button) {
		w.SetText("Pan").SetIcon(icons.PanTool)
		w.SetTooltip("Toggle the ability to zoom and pan")
		w.OnClick(func(e events.Event) {
			sv.SetReadOnly(!sv.IsReadOnly())
			sv.Restyle()
		})
	})
	tree.Add(p, func(w *Separator) {})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(sv.Open).SetIcon(icons.Open)
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(sv.SaveSVG).SetIcon(icons.Save)
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(sv.SaveImage).SetIcon(icons.Save)
	})
}
