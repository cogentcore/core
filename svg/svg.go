// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

//go:generate goki generate

import (
	"image"
	"image/color"
	"strings"
	"sync"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// SVG is an SVG object.
type SVG struct {
	// Name is the name of the SVG -- e.g., the filename if loaded
	Name string

	// the title of the svg
	Title string `xml:"title"`

	// the description of the svg
	Desc string `xml:"desc"`

	// fill the viewport with Background
	Fill bool

	// image/color to fill background with if Fill is on
	Background image.Image

	// Color can be set to provide a default Fill and Stroke Color value
	Color image.Image

	// Size is size of image, Pos is offset within any parent viewport.
	// Node bounding boxes are based on 0 Pos offset within Pixels image
	Geom mat32.Geom2DInt

	// physical width of the drawing, e.g., when printed -- does not affect rendering -- metadata
	PhysWidth units.Value

	// physical height of the drawing, e.g., when printed -- does not affect rendering -- metadata
	PhysHeight units.Value

	// Norm installs a transform that renormalizes so that the specified
	// ViewBox exactly fits within the allocated SVG size.
	Norm bool

	// InvertY, when doing Norm transform, also flip the Y axis so that
	// the smallest Y value is at the bottom of the SVG box,
	// instead of being at the top as it is by default.
	InvertY bool

	// Translate specifies a translation to apply beyond what is specified in the SVG,
	// and in addition to the effects of Norm if active.
	Translate mat32.Vec2

	// Scale specifies a zoom scale factor to apply beyond what is specified in the SVG,
	// and in addition to the effects of Norm if active.
	Scale float32

	// render state for rendering
	RenderState paint.State `copy:"-" json:"-" xml:"-" edit:"-"`

	// live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" edit:"-"`

	// all defs defined elements go here (gradients, symbols, etc)
	Defs Group

	// root of the svg tree -- top-level viewbox and paint style here
	Root SVGNode

	// map of def names to index -- uses starting index to find element -- always updated after each search
	DefIdxs map[string]int `view:"-" json:"-" xml:"-"`

	// map of unique numeric ids for all elements -- used for allocating new unique id numbers, appended to end of elements -- see NewUniqueId, GatherIds
	UniqueIds map[int]struct{} `view:"-" json:"-" xml:"-"`

	// mutex for protecting rendering
	RenderMu sync.Mutex `view:"-" json:"-" xml:"-"`
}

// NewSVG creates a SVG with Pixels Image of the specified width and height
func NewSVG(width, height int) *SVG {
	sv := &SVG{}
	sv.Config(width, height)
	return sv
}

// Config configures the SVG, setting image to given size
// and initializing all relevant fields.
func (sv *SVG) Config(width, height int) {
	sz := image.Point{width, height}
	sv.Geom.Size = sz
	sv.Scale = 1
	sv.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	sv.RenderState.Init(width, height, sv.Pixels)
	sv.Root.InitName(&sv.Root, "svg")
	sv.Defs.InitName(&sv.Defs, "defs")
}

// Resize resizes the viewport, creating a new image -- updates Geom Size
func (sv *SVG) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if sv.Root.Ths == nil {
		sv.Config(nwsz.X, nwsz.Y)
		return
	}
	if sv.Pixels != nil {
		ib := sv.Pixels.Bounds().Size()
		if ib == nwsz {
			sv.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sv.Pixels != nil {
		sv.Pixels = nil
	}
	sv.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	sv.RenderState.Init(nwsz.X, nwsz.Y, sv.Pixels)
	sv.Geom.Size = nwsz // make sure
}

func (sv *SVG) CopyFrom(fr *SVG) {
	sv.Title = fr.Title
	sv.Desc = fr.Desc
	sv.Fill = fr.Fill
	sv.Background = fr.Background
	sv.Geom = fr.Geom
	sv.Norm = fr.Norm
	sv.InvertY = fr.InvertY
	sv.Defs.CopyFrom(&fr.Defs)
	sv.Root.CopyFrom(&fr.Root)
	sv.UniqueIds = nil
}

// DeleteAll deletes any existing elements in this svg
func (sv *SVG) DeleteAll() {
	if sv.Root.This() == nil {
		return
	}
	updt := sv.Root.UpdateStart() // don't really need update logic here
	sv.Root.Paint.Defaults()
	sv.Root.DeleteChildren(ki.DestroyKids)
	sv.Defs.DeleteChildren(ki.DestroyKids)
	sv.Root.UpdateEnd(updt)
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
	// set the Defs flags
	sv.Defs.WalkPre(func(k ki.Ki) bool {
		ni := k.(Node)
		if ni == nil || ni.Is(ki.Deleted) || ni.Is(ki.Destroyed) {
			return ki.Break
		}
		ni.SetFlag(true, IsDef)
		return ki.Continue
	})

	sv.Root.Paint.Defaults()
	if sv.Color != nil {
		// TODO(kai/imageColor): handle non-uniform colors here
		c := colors.ToUniform(sv.Color)
		sv.Root.SetColorProps("stroke", colors.AsHex(c))
		sv.Root.SetColorProps("fill", colors.AsHex(c))
	}
	sv.SetUnitContext(&sv.Root.Paint, mat32.Vec2{}, mat32.Vec2{})

	sv.Root.WalkPre(func(k ki.Ki) bool {
		ni := k.(Node)
		if ni == nil || ni.Is(ki.Deleted) || ni.Is(ki.Destroyed) {
			return ki.Break
		}
		ni.Style(sv)
		return ki.Continue
	})
}

func (sv *SVG) Render() {
	sv.RenderMu.Lock()
	defer sv.RenderMu.Unlock()

	sv.Style()

	sv.SetRootTransform()

	rs := &sv.RenderState
	rs.PushBounds(sv.Pixels.Bounds())
	if sv.Fill {
		sv.FillViewport()
	}
	sv.Root.Render(sv)
	rs.PopBounds()
}

func (sv *SVG) FillViewport() {
	pc := &paint.Context{&sv.RenderState, &sv.Root.Paint}
	pc.Lock()
	pc.FillBox(mat32.Vec2{}, mat32.V2FromPoint(sv.Geom.Size), sv.Background)
	pc.Unlock()
}

// SetRootTransform sets the Root node transform based on Norm, Translate, Scale
// parameters set on the SVG object.
func (sv *SVG) SetRootTransform() {
	sc := mat32.V2(1, 1)
	tr := mat32.Vec2{}
	if sv.Norm {
		vb := &sv.Root.ViewBox
		if vb.Size != (mat32.Vec2{}) {
			sc.X = float32(sv.Geom.Size.X) / vb.Size.X
			sc.Y = float32(sv.Geom.Size.Y) / vb.Size.Y
			if sv.InvertY {
				sc.Y *= -1
			}
			tr = vb.Min.MulScalar(-1)
		} else {
			sz := sv.Root.LocalBBox().Size()
			if sz != (mat32.Vec2{}) {
				sc.X = float32(sv.Geom.Size.X) / sz.X
				sc.Y = float32(sv.Geom.Size.Y) / sz.Y
			}
		}
	}
	tr.SetAdd(sv.Translate)
	sc.SetMulScalar(sv.Scale)
	pc := &sv.Root.Paint
	pc.Transform = pc.Transform.Scale(sc.X, sc.Y).Translate(tr.X, tr.Y)
	if sv.InvertY {
		pc.Transform.Y0 = -pc.Transform.Y0
	}
}

// SetDPITransform sets a scaling transform to compensate for
// a given LogicalDPI factor.
// svg rendering is done within a 96 DPI context.
func (sv *SVG) SetDPITransform(logicalDPI float32) {
	pc := &sv.Root.Paint
	dpisc := logicalDPI / 96.0
	pc.Transform = mat32.Scale2D(dpisc, dpisc)
}

// SavePNG saves the Pixels to a PNG file
func (sv *SVG) SavePNG(fname string) error {
	return images.Save(sv.Pixels, fname)
}

//////////////////////////////////////////////////////////////
// 	SVGNode

// SVGNode represents the root of an SVG tree
type SVGNode struct {
	Group

	// viewbox defines the coordinate system for the drawing -- these units are mapped into the screen space allocated for the SVG during rendering
	ViewBox ViewBox
}

func (g *SVGNode) CopyFieldsFrom(frm any) {
	fr := frm.(*SVGNode)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.ViewBox = fr.ViewBox
}

func (g *SVGNode) SVGName() string { return "svg" }

func (g *SVGNode) EnforceSVGName() bool { return false }

func (g *SVGNode) NodeBBox(sv *SVG) image.Rectangle {
	// todo: return viewbox
	return sv.Geom.SizeRect()
}

// SetUnitContext sets the unit context based on size of viewport, element,
// and parent element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering -- call at start of render
func (sv *SVG) SetUnitContext(pc *styles.Paint, el, par mat32.Vec2) {
	pc.UnContext.Defaults()
	pc.UnContext.DPI = 96 // paint (SVG) context is always 96 = 1to1
	if sv.RenderState.Image != nil {
		sz := sv.RenderState.Image.Bounds().Size()
		pc.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
	} else {
		pc.UnContext.SetSizes(0, 0, el.X, el.Y, par.X, par.Y)
	}
	pc.FontStyle.SetUnitContext(&pc.UnContext)
	pc.ToDots()
}
