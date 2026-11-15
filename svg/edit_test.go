// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg_test

import (
	"image"
	"image/color"
	"path/filepath"
	"testing"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	. "cogentcore.org/core/svg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNameIDs tests naming elements with unique numeric ids.
func TestNameIDs(t *testing.T) {
	nm, id := SplitNameIDDig("rect12")
	assert.Equal(t, "rect", nm)
	assert.Equal(t, 12, id)
	nm, id = SplitNameIDDig("rect")
	assert.Equal(t, "rect", nm)
	assert.Equal(t, 0, id)

	ok, id := SplitNameID("rect", "rect34")
	assert.True(t, ok)
	assert.Equal(t, 34, id)
	ok, _ = SplitNameID("rect", "circle34")
	assert.False(t, ok)

	assert.Equal(t, "rect5", NameID("rect", 5))
	assert.Equal(t, "rect", NameID("rect", 0))
	assert.Equal(t, "url(#g1)", NameToURL("g1"))
	assert.Equal(t, "g1", NameFromURL("url(#g1)"))
	assert.Equal(t, "", NameFromURL("#g1"))

	sv := NewSVG(math32.Vec2(100, 100))
	named := func(n Node, name string) {
		n.AsTree().SetName(name)
	}
	r1, r2, c, o, g := NewRect(sv.Root), NewRect(sv.Root), NewCircle(sv.Root), NewRect(sv.Root), NewGroup(sv.Root)
	named(r1, "rect3")
	named(r2, "rect3") // a duplicate
	named(c, "circle")
	named(o, "box7") // renamed to rect7
	named(g, "layer9")
	sv.GatherIDs()
	assert.Equal(t, "rect3", r1.Name)
	assert.NotEqual(t, "rect3", r2.Name, "the duplicate gets a new id")
	ok, id = SplitNameID("rect", r2.Name)
	assert.True(t, ok)
	assert.Greater(t, id, 0)
	ok, id = SplitNameID("circle", c.Name)
	assert.True(t, ok && id > 0, c.Name)
	assert.Equal(t, "rect7", o.Name)
	assert.Equal(t, "layer9", g.Name, "groups keep their names")
	for _, id := range []int{3, 7, 9} {
		_, has := sv.UniqueIDs[id]
		assert.True(t, has, id)
	}

	nid := sv.NewUniqueID()
	_, has := sv.UniqueIDs[nid]
	assert.True(t, has)

	assert.Equal(t, Node(c), sv.FindNamedElement("#"+c.Name))
	assert.Nil(t, sv.FindNamedElement("nothing"))
	assert.Nil(t, sv.NodeFindURL(c, "none"))
}

// TestGradientsForNodes tests making and updating the gradients
// specific to nodes, as a drawing app does.
func TestGradientsForNodes(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	stops, surl := sv.GradientNew(false)
	stops.Grad.AsBase().AddStop(color.RGBA{255, 0, 0, 255}, 0).AddStop(color.RGBA{0, 0, 255, 255}, 1)
	assert.Equal(t, "linearGradient", stops.GradientTypeName())
	assert.False(t, stops.IsRadial())
	assert.Equal(t, NameToURL(stops.Name), surl)

	r := NewRect(sv.Root)
	r.Pos = math32.Vec2(10, 20)
	r.Size = math32.Vec2(30, 40)
	gr, url := sv.GradientUpdateNodeProp(r, "fill", true, stops.Name)
	require.NotNil(t, gr)
	assert.True(t, gr.IsRadial())
	assert.Equal(t, "radialGradient", gr.GradientTypeName())
	assert.Equal(t, url, r.Property("fill"))
	assert.Equal(t, gradient.UserSpaceOnUse, gr.Grad.AsBase().Units)
	assert.Len(t, gr.Grad.AsBase().Stops, 2, "copied from the stops gradient")
	rg := gr.Grad.(*gradient.Radial)
	assert.InDelta(t, 25, rg.Center.X, 1, "centered on the rect")
	assert.InDelta(t, 40, rg.Center.Y, 1)
	assert.Equal(t, gr, sv.GradientByName(r, url))
	assert.Nil(t, sv.GradientByName(r, NameToURL(r.Name)), "not a gradient")

	// the same kind updates the existing gradient
	gr2, url2 := sv.GradientUpdateNodeProp(r, "fill", true, stops.Name)
	assert.Equal(t, gr, gr2)
	assert.Equal(t, url, url2)

	// a different kind replaces it
	gr3, url3 := sv.GradientUpdateNodeProp(r, "fill", false, stops.Name)
	assert.False(t, gr3.IsRadial())
	assert.NotEqual(t, url, url3)
	assert.Nil(t, sv.FindDefByName(NameFromURL(url)), "the old one is deleted")
	lg := gr3.Grad.(*gradient.Linear)
	assert.Equal(t, math32.Vec2(10, 20), lg.Start, "from the bbox")

	// transforms of the node geometry go to its gradients
	r.ApplyTransform(sv, math32.Translate2D(5, 0))
	assert.Equal(t, math32.Vec2(15, 20), r.Pos)
	assert.Equal(t, math32.Translate2D(5, 0), gr3.Grad.AsBase().Transform)
	r.GradientFill = math32.Identity2()
	r.GradientFromGradients(sv)
	assert.Equal(t, math32.Translate2D(5, 0), r.GradientFill)
	sv.GradientFromGradients()

	stops.Grad.AsBase().AddStop(color.RGBA{0, 255, 0, 255}, 0.5)
	sv.GradientUpdateAllStops()
	assert.Len(t, gr3.Grad.AsBase().Stops, 3)

	// a duplicated node gets its own gradient
	r2 := NewRect(sv.Root)
	r2.SetProperty("fill", url3)
	sv.GradientDuplicateNode(r2)
	assert.NotEqual(t, url3, r2.Property("fill"))
	assert.NotNil(t, sv.GradientByName(r2, r2.Property("fill").(string)))

	assert.True(t, sv.GradientDeleteForNode(r, url3))
	assert.Nil(t, sv.FindDefByName(NameFromURL(url3)))
	assert.False(t, sv.GradientDeleteForNode(r, surl), "the stops gradient is not node-specific")

	// orphaned defs are removed, keeping the stops that are referred to
	_, orphan := sv.GradientNew(true)
	assert.True(t, sv.RemoveOrphanedDefs())
	assert.Nil(t, sv.FindDefByName(NameFromURL(orphan)))
	assert.NotNil(t, sv.FindDefByName(stops.Name), "referred to by the stops of another")
	assert.False(t, sv.RemoveOrphanedDefs())
}

// TestGradientDuplicateStroke tests duplicating the gradient of the
// stroke of a node.
func TestGradientDuplicateStroke(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	stops, _ := sv.GradientNew(false)
	r := NewRect(sv.Root)
	_, surl := sv.GradientUpdateNodeProp(r, "stroke", false, stops.Name)
	r.SetProperty("fill", "red")
	sv.GradientDuplicateNode(r)
	assert.Equal(t, "red", r.Property("fill"))
	assert.NotEqual(t, surl, r.Property("stroke"))
}

// TestGradientDeleteBareName tests deleting the gradient of a node by
// its bare name, which the documentation allows.
func TestGradientDeleteBareName(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	stops, _ := sv.GradientNew(false)
	r := NewRect(sv.Root)
	_, url := sv.GradientUpdateNodeProp(r, "fill", false, stops.Name)
	name := NameFromURL(url)
	assert.True(t, sv.GradientDeleteForNode(r, name))
	assert.Nil(t, sv.FindDefByName(name))
}

// TestZoom tests zooming and fitting to the contents.
func TestZoom(t *testing.T) {
	sv := NewSVG(math32.Vec2(200, 100))
	sv.Root.ViewBox.Size = math32.Vec2(200, 100)
	r := NewRect(sv.Root)
	r.Pos = math32.Vec2(20, 10)
	r.Size = math32.Vec2(40, 20)
	r.Paint.Stroke.Color = nil
	sv.RenderImage()

	cb := sv.ContentBounds()
	assert.InDelta(t, 20, cb.Min.X, 1)
	assert.InDelta(t, 60, cb.Max.X, 1)

	// the point under the mouse stays put when zooming
	pt := image.Pt(40, 20)
	sv.ZoomAt(pt, 0.5)
	assert.InDelta(t, 1.5, sv.Scale, 1e-5)
	sv.UpdateBBoxes()
	sv.ZoomAt(pt, 2)
	assert.InDelta(t, 4.5, sv.Scale, 1e-4)
	sv.ZoomAt(pt, -0.5)
	assert.InDelta(t, 2.25, sv.Scale, 1e-4)
	sv.ZoomAtScroll(10, pt)
	assert.Greater(t, sv.Scale, float32(2.25), "scrolling down zooms in")
	sv.ScaleAt(pt, 1000)
	assert.Equal(t, float32(1000), sv.Scale)
	sv.ZoomAt(pt, 0.5)
	assert.Equal(t, float32(100), sv.Scale, "clamped")

	sv.ZoomReset()
	assert.Equal(t, float32(1), sv.Scale)
	assert.Equal(t, math32.Vector2{}, sv.Translate)

	sv.ZoomToContents(math32.Vec2(200, 100))
	assert.InDelta(t, 5, sv.Scale, 0.1, "the 40x20 contents fit 200x100")
	assert.InDelta(t, -20, sv.Translate.X, 1)

	sv.ResizeToContents(1)
	assert.InDelta(t, 20, sv.Root.ViewBox.Min.X, 1)
	assert.InDelta(t, 40, sv.Root.ViewBox.Size.X, 2)
	sv.ResizeToContents(16)
	assert.Equal(t, float32(16), sv.Root.ViewBox.Min.X, "on the grid")
	assert.Equal(t, float32(0), math32.Mod(sv.Root.ViewBox.Size.X, 16))

	sv.SetDPITransform(192)
	assert.Equal(t, math32.Scale2D(2, 2), sv.Root.Paint.Transform)
	sv.SetSize(math32.Vec2(300, 150))
	assert.Equal(t, image.Pt(300, 150), sv.Geom.Size)
}

// TestApplyTransform tests applying transforms to the geometry of
// each kind of node, and setting their positions and sizes.
func TestApplyTransform(t *testing.T) {
	sv := NewSVG(math32.Vec2(100, 100))
	scale := math32.Scale2D(2, 3).Translate(1, 1)
	rot := math32.Rotate2D(math32.Pi / 2)

	r := NewRect(sv.Root)
	r.Pos, r.Size = math32.Vec2(1, 2), math32.Vec2(3, 4)
	r.ApplyTransform(sv, scale)
	assert.Equal(t, math32.Vec2(4, 9), r.Pos)
	assert.Equal(t, math32.Vec2(6, 12), r.Size)
	r.ApplyTransform(sv, rot)
	assert.Equal(t, math32.Vec2(4, 9), r.Pos, "rotations go to the transform")
	assert.NotEmpty(t, r.Property("transform"))
	r.SetNodePos(math32.Vec2(5, 6))
	r.SetNodeSize(math32.Vec2(7, 8))
	assert.Equal(t, math32.Vec2(5, 6), r.Pos)
	assert.Equal(t, math32.Vec2(7, 8), r.Size)

	c := NewCircle(sv.Root)
	c.Pos, c.Radius = math32.Vec2(1, 1), 2
	c.ApplyTransform(sv, math32.Scale2D(2, 4))
	assert.Equal(t, math32.Vec2(2, 4), c.Pos)
	assert.Equal(t, float32(6), c.Radius, "the average scale")
	c.ApplyTransform(sv, rot)
	assert.NotEmpty(t, c.Property("transform"))
	c.SetNodeSize(math32.Vec2(2, 6))
	assert.Equal(t, float32(2), c.Radius)

	e := NewEllipse(sv.Root)
	e.Pos, e.Radii = math32.Vec2(1, 1), math32.Vec2(2, 3)
	e.ApplyTransform(sv, math32.Scale2D(2, 2))
	assert.Equal(t, math32.Vec2(2, 2), e.Pos)
	assert.Equal(t, math32.Vec2(4, 6), e.Radii)
	e.ApplyTransform(sv, rot)
	assert.NotEmpty(t, e.Property("transform"))
	e.SetNodeSize(math32.Vec2(10, 20))
	assert.Equal(t, math32.Vec2(5, 10), e.Radii)

	l := NewLine(sv.Root)
	l.Start, l.End = math32.Vec2(1, 1), math32.Vec2(2, 3)
	l.ApplyTransform(sv, math32.Translate2D(1, 2))
	assert.Equal(t, math32.Vec2(2, 3), l.Start)
	assert.Equal(t, math32.Vec2(3, 5), l.End)
	l.ApplyTransform(sv, rot)
	assert.NotEmpty(t, l.Property("transform"))
	l.SetPos(math32.Vec2(0, 0))
	l.SetSize(math32.Vec2(4, 5))
	assert.Equal(t, math32.Vec2(4, 5), l.End)

	pl := NewPolyline(sv.Root)
	pl.Points = []math32.Vector2{{X: 0, Y: 0}, {X: 1, Y: 2}}
	pl.ApplyTransform(sv, math32.Scale2D(2, 2))
	assert.Equal(t, []math32.Vector2{{X: 0, Y: 0}, {X: 2, Y: 4}}, pl.Points)
	pl.ApplyTransform(sv, rot)
	assert.NotEmpty(t, pl.Property("transform"))
	pl.SetPos(math32.Vec2(1, 1))
	pl.SetSize(math32.Vec2(1, 1))
	assert.Equal(t, []math32.Vector2{{X: 0, Y: 0}, {X: 2, Y: 4}}, pl.Points, "no-ops")

	p := NewPath(sv.Root)
	require.NoError(t, p.SetData("M0 0L2 4"))
	p.ApplyTransform(sv, math32.Translate2D(1, 1))
	assert.Equal(t, "M1 1L3 5", p.Data.String())
	p.SetSize(math32.Vec2(4, 8))
	assert.Equal(t, "M2 2L6 10", p.Data.String(), "scaled to the size")
	p.UpdatePathString()
	assert.Equal(t, "M2 2L6 10", p.DataStr)
	assert.Error(t, p.SetData("M0 0 X"))

	g := NewGroup(sv.Root)
	gr := NewRect(g)
	gr.Pos, gr.Size = math32.Vec2(1, 1), math32.Vec2(1, 1)
	g.ApplyTransform(sv, math32.Translate2D(2, 2))
	assert.Equal(t, math32.Vec2(1, 1), gr.Pos, "a group transforms itself")
	assert.Equal(t, math32.Translate2D(2, 2), gr.ParentTransform(false))
	assert.Equal(t, Node(gr), FirstNonGroupNode(g))

	dx := gr.DeltaTransform(math32.Vec2(1, 0), math32.Vec2(1, 1), 0, math32.Vec2(0, 0))
	assert.Equal(t, math32.Vec2(1, 0), dx.MulPoint(math32.Vector2{}))
}

// TestColorProperties tests setting color properties from strings,
// with alpha as opacity.
func TestColorProperties(t *testing.T) {
	r := NewRect()
	r.SetColorProperties("fill", "#00ff00")
	assert.Equal(t, "#00FF00", r.Property("fill"))
	assert.Equal(t, "1", r.Property("fill-opacity"))
	r.SetColorProperties("stroke", "blue")
	r.SetProperty("stroke-width", "2")
	r.SetColorProperties("stroke", "none")
	assert.Equal(t, "none", r.Property("stroke"))
	assert.Nil(t, r.Property("stroke-width"))
	r.SetColorProperties("fill", "url(#g1)")
	assert.Equal(t, "#00FF00", r.Property("fill"), "urls are not set")

	r.SetColorProperties("fill", "#ff000080")
	assert.Equal(t, "0.5019608", r.Property("fill-opacity"))
	assert.Equal(t, "#FF0000", r.Property("fill"))
}

// TestImageNode tests an image element: setting, scaling, rendering,
// saving, and opening its image.
func TestImageNode(t *testing.T) {
	sv := NewSVG(math32.Vec2(40, 40))
	sv.Root.ViewBox.Size = math32.Vec2(40, 40)
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	im := NewImage(sv.Root)
	im.SetImage(nil, 0, 0)
	assert.Nil(t, im.Pixels)
	im.SetImage(src, 0, 0)
	assert.Equal(t, math32.Vec2(4, 2), im.Size, "the size of the image")
	im.SetImage(src, 8, 0)
	assert.Equal(t, image.Pt(8, 2), im.Pixels.Image.Bounds().Size())
	im.SetImage(src, 0, 6)
	assert.Equal(t, image.Pt(4, 6), im.Pixels.Image.Bounds().Size())
	im.SetNodePos(math32.Vec2(10, 10))
	im.SetNodeSize(math32.Vec2(20, 20))
	assert.Equal(t, math32.B2(10, 10, 30, 30), im.LocalBBox(sv))

	img := sv.RenderImage()
	assert.Equal(t, uint32(0xffff), func() uint32 { _, _, _, a := img.At(20, 20).RGBA(); return a }())

	im.ApplyTransform(sv, math32.Scale2D(0.5, 0.5))
	assert.Equal(t, math32.Vec2(5, 5), im.Pos)
	assert.Equal(t, math32.Vec2(10, 10), im.Size)
	im.ApplyTransform(sv, math32.Rotate2D(1))
	assert.NotEmpty(t, im.Property("transform"))

	fn := filepath.Join(t.TempDir(), "im.png")
	require.NoError(t, im.SaveImage(fn))
	im2 := NewImage(sv.Root)
	assert.Error(t, im2.SaveImage(fn), "no pixels")
	require.NoError(t, im2.OpenImage(fn, 0, 0))
	assert.Equal(t, fn, im2.Filename)
	assert.Equal(t, im.Pixels.Image.Bounds(), im2.Pixels.Image.Bounds())
	assert.Error(t, im2.OpenImage(filepath.Join(t.TempDir(), "none.png"), 0, 0))
}

// TestViewBoxAspect tests the view box defaults, strings, and setting
// its aspect ratio from a style.
func TestViewBoxAspect(t *testing.T) {
	var vb ViewBox
	vb.Defaults()
	assert.Equal(t, math32.Vec2(100, 100), vb.Max())
	assert.Contains(t, vb.String(), `preserveAspectRatio="`)
	assert.Contains(t, vb.BoxString(), "0 0 100 100")

	s := styles.NewStyle()
	s.Justify.Content = styles.Start
	s.Align.Content = styles.End
	s.ObjectFit = styles.FitCover
	pa := &vb.PreserveAspectRatio
	pa.SetFromStyle(s)
	assert.Equal(t, AlignMin, pa.Align.X)
	assert.Equal(t, AlignMax, pa.Align.Y)
	assert.Equal(t, Slice, pa.MeetOrSlice)
	s.ObjectFit = styles.FitContain
	pa.SetFromStyle(s)
	assert.Equal(t, Meet, pa.MeetOrSlice)
	s.ObjectFit = styles.FitFill
	pa.SetFromStyle(s)
	assert.Equal(t, AlignNone, pa.Align.X)

	// slice covers the box, and none stretches
	vb.Size = math32.Vec2(100, 50)
	vb.PreserveAspectRatio.Align.Set(AlignMid)
	vb.PreserveAspectRatio.MeetOrSlice = Slice
	size, _, _ := vb.Transform(math32.Vec2(100, 100))
	assert.Equal(t, math32.Vec2(200, 100), size)
	vb.PreserveAspectRatio.Align.Set(AlignNone)
	size, _, scale := vb.Transform(math32.Vec2(100, 100))
	assert.Equal(t, math32.Vec2(100, 100), size)
	assert.Equal(t, math32.Vec2(1, 2), scale)
}

// TestNodeInit tests that new nodes have the default paint and
// gradient transforms.
func TestNodeInit(t *testing.T) {
	r := NewRect()
	assert.Equal(t, math32.Identity2(), r.GradientFill)
	assert.Equal(t, float32(1), r.Paint.Stroke.Width.Value)
	c := NewCircle()
	assert.Equal(t, math32.Identity2(), c.GradientFill)
	assert.Equal(t, r.Paint.Stroke.Width, c.Paint.Stroke.Width)
}
