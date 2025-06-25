// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

//go:generate core generate

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"sync"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/tree"
)

var (
	// svgShaper is a shared text shaper.
	svgShaper shaped.Shaper

	// mutex for initializing the svgShaper.
	shaperMu sync.Mutex
)

// SVGToImage generates an image from given svg source,
// with given width and height size.
func SVGToImage(svg []byte, size math32.Vector2) (image.Image, error) {
	sv := NewSVG(size)
	err := sv.ReadXML(bytes.NewBuffer(svg))
	return sv.RenderImage(), err
}

// SVG represents a structured SVG vector graphics drawing,
// with nodes allocated for each element.
// It renders to a [paint.Painter] via the Render method.
// Any supported representation can then be rendered from that.
type SVG struct {
	// Name is the name of the SVG -- e.g., the filename if loaded
	Name string

	// the title of the svg
	Title string `xml:"title"`

	// the description of the svg
	Desc string `xml:"desc"`

	// Background is the image/color to fill the background with,
	// if any.
	Background image.Image

	// DefaultFill can be set to provide a default Fill color for the root node.
	DefaultFill image.Image

	// Size is size of image, Pos is offset within any parent viewport.
	// The bounding boxes within the scene _include_ the Pos offset already.
	Geom math32.Geom2DInt

	// physical width of the drawing, e.g., when printed.
	// Does not affect rendering, metadata.
	PhysicalWidth units.Value

	// physical height of the drawing, e.g., when printed.
	// Does not affect rendering, metadata.
	PhysicalHeight units.Value

	// InvertY, when applying the ViewBox transform, also flip the Y axis so that
	// the smallest Y value is at the bottom of the SVG box,
	// instead of being at the top as it is by default.
	InvertY bool

	// Translate specifies a translation to apply beyond what is specified in the SVG,
	// and its ViewBox transform, in top-level rendering units (dots, pixels).
	Translate math32.Vector2

	// Scale specifies a zoom scale factor to apply beyond what is specified in the SVG,
	// and its ViewBox transform. See [SVG.ZoomAt] for convenient zooming method.
	Scale float32

	// painter is the current painter being used, which is only valid during rendering.
	painter *paint.Painter

	// TextShaper for shaping text. Can set to a shared external one,
	// or else the shared svgShaper is used.
	TextShaper shaped.Shaper

	// all defs defined elements go here (gradients, symbols, etc)
	Defs *Group

	// Root is the root of the svg tree, which has the top-level viewbox and styles.
	Root *Root

	// GroupFilter is used to filter group names, skipping any that don't contain
	// this string, if non-empty. This is needed e.g., for reading SVG font files
	// which pack many elements into the same file.
	GroupFilter string

	// groupFilterSkip is whether to skip the current group based on GroupFilter.
	groupFilterSkip bool

	// groupFilterSkipName is name of group currently skipping.
	groupFilterSkipName string

	// map of def names to index. uses starting index to find element.
	// always updated after each search.
	DefIndexes map[string]int `display:"-" json:"-" xml:"-"`

	// map of unique numeric ids for all elements.
	// Used for allocating new unique id numbers, appended to end of elements.
	// See NewUniqueID, GatherIDs
	UniqueIDs map[int]struct{} `display:"-" json:"-" xml:"-"`

	// mutex for protecting rendering
	sync.Mutex
}

// NewSVG creates a SVG with the given viewport size,
// which is typically in pixel dots.
func NewSVG(size math32.Vector2) *SVG {
	sv := &SVG{}
	sv.Init(size)
	return sv
}

// Init initializes the SVG with given viewport size,
// which is typically in pixel dots.
func (sv *SVG) Init(size math32.Vector2) {
	sv.Geom.Size = size.ToPointCeil()
	sv.Scale = 1
	sv.Root = NewRoot()
	sv.Root.SetName("svg")
	sv.Defs = NewGroup()
	sv.Defs.SetName("defs")
	sv.SetUnitContext(&sv.Root.Paint)
}

// SetSize updates the viewport size.
func (sv *SVG) SetSize(size math32.Vector2) {
	sv.Geom.Size = size.ToPointCeil()
	sv.SetUnitContext(&sv.Root.Paint)
}

// DeleteAll deletes any existing elements in this svg
func (sv *SVG) DeleteAll() {
	if sv.Root == nil || sv.Root.This == nil {
		return
	}
	sv.Root.Paint.Defaults()
	sv.Root.DeleteChildren()
	sv.Defs.DeleteChildren()
}

// Base returns the current Color activated in the context.
// Color has support for special color names that are relative to
// this current color.
func (sv *SVG) Base() color.RGBA {
	return colors.AsRGBA(colors.ToUniform(sv.Background))
}

// ImageByURL finds a Node by an element name (URL-like path), and
// attempts to convert it to an [image.Image].
// Used for color styling based on url() value.
func (sv *SVG) ImageByURL(url string) image.Image {
	// TODO(kai): support taking snapshot of element as image in SVG.ImageByURL
	if sv == nil {
		return nil
	}
	val := url[4:]
	val = strings.TrimPrefix(strings.TrimSuffix(val, ")"), "#")
	def := sv.FindDefByName(val)
	if def != nil {
		if grad, ok := def.(*Gradient); ok {
			return grad.Grad
		}
	}
	ne := sv.FindNamedElement(val)
	if grad, ok := ne.(*Gradient); ok {
		return grad.Grad
	}
	return nil
}

func (sv *SVG) Style() {
	// set isDef
	sv.Defs.WalkDown(func(n tree.Node) bool {
		sn := n.(Node)
		sn.AsNodeBase().isDef = true
		sn.AsNodeBase().Style(sv)
		return tree.Continue
	})

	sv.Root.Paint.Defaults()
	if sv.DefaultFill != nil {
		// TODO(kai): consider handling non-uniform colors here
		sv.Root.SetColorProperties("fill", colors.AsHex(colors.ToUniform(sv.DefaultFill)))
	}
	sv.SetUnitContext(&sv.Root.Paint)

	sv.Root.WalkDown(func(k tree.Node) bool {
		sn := k.(Node)
		sn.AsNodeBase().Style(sv)
		return tree.Continue
	})
}

// Render renders the SVG to given Painter, which can be nil
// to have a new one created. Returns the painter used.
// Set the TextShaper prior to calling to use an existing one,
// otherwise it will use shared svgShaper.
func (sv *SVG) Render(pc *paint.Painter) *paint.Painter {
	sv.Lock()
	defer sv.Unlock()

	if pc != nil {
		sv.painter = pc
	} else {
		sv.UpdateSize()
		sv.painter = paint.NewPainter(math32.FromPoint(sv.Geom.Size))
		pc = sv.painter
	}
	if sv.TextShaper == nil {
		shaperMu.Lock()
		if svgShaper == nil {
			svgShaper = shaped.NewShaper()
		}
		sv.TextShaper = svgShaper
		shaperMu.Unlock()
		defer func() {
			sv.TextShaper = nil
		}()
	}

	sv.Style()
	sv.UpdateBBoxes()

	if sv.Background != nil {
		sv.FillViewport()
	}
	sv.Root.Render(sv)

	sv.painter = nil
	return pc
}

// RenderImage renders the SVG to an image and returns it.
func (sv *SVG) RenderImage() image.Image {
	return paint.RenderToImage(sv.Render(nil))
}

// SaveImage renders the SVG to an image and saves it to given filename,
// using the filename extension to determine the file type.
func (sv *SVG) SaveImage(fname string) error {
	pos := sv.Geom.Pos
	sv.Geom.Pos = image.Point{}
	err := imagex.Save(sv.RenderImage(), fname)
	sv.Geom.Pos = pos
	return err
}

// SaveImageSize renders the SVG to an image and saves it to given filename,
// using the filename extension to determine the file type.
// Specify either width or height of resulting image, or nothing for
// current physical size as set.
func (sv *SVG) SaveImageSize(fname string, width, height float32) error {
	sz := sv.Geom
	sv.Geom.Pos = image.Point{}
	sv.Geom.Size.X = int(width)
	sv.Geom.Size.Y = int(height)
	err := imagex.Save(sv.RenderImage(), fname)
	sv.Geom = sz
	return err
}

func (sv *SVG) FillViewport() {
	sty := styles.NewPaint() // has no transform
	pc := &paint.Painter{sv.painter.State, sty}
	pc.FillBox(math32.Vector2{}, math32.FromPoint(sv.Geom.Size), sv.Background)
}

// UpdateBBoxes updates the bounding boxes for all nodes
// using current transform settings.
func (sv *SVG) UpdateBBoxes() {
	sv.setRootTransform()
	sv.Root.BBoxes(sv, math32.Identity2())
}

//////// Root

// Root represents the root of an SVG tree.
type Root struct {
	Group

	// ViewBox defines the coordinate system for the drawing.
	// These units are mapped into the screen space allocated
	// for the SVG during rendering.
	ViewBox ViewBox
}

func (g *Root) SVGName() string { return "svg" }

func (g *Root) EnforceSVGName() bool { return false }

// SetUnitContext sets the unit context based on size of viewport, element,
// and parent element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering -- call at start of render
func (sv *SVG) SetUnitContext(pc *styles.Paint) {
	pc.UnitContext.Defaults()
	pc.UnitContext.DPI = 96 // paint (SVG) context is always 96 = 1to1
	wd := float32(sv.Geom.Size.X)
	ht := float32(sv.Geom.Size.Y)
	pc.UnitContext.SetSizes(wd, ht, wd, ht, wd, ht) // self, element, parent -- all same
	pc.ToDots()
	sv.ToDots(&pc.UnitContext)
}

func (sv *SVG) ToDots(uc *units.Context) {
	sv.PhysicalWidth.ToDots(uc)
	sv.PhysicalHeight.ToDots(uc)
}

func (g *Root) Render(sv *SVG) {
	pc := g.Painter(sv)
	pc.PushContext(&g.Paint, render.NewBoundsRect(sv.Geom.Bounds(), sides.NewFloats()))
	g.RenderChildren(sv)
	pc.PopContext()
}
