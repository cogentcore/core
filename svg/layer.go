// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image/color"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Mask is an SVG mask element: the luminance (times the alpha) of the
// rendering of its children masks the elements that refer to it with
// the mask property. It is not rendered itself.
type Mask struct {
	NodeBase
}

func (g *Mask) SVGName() string { return "mask" }

func (g *Mask) EnforceSVGName() bool { return false }

func (g *Mask) Render(sv *SVG) {} // only for the elements that refer to it

func (g *ClipPath) Render(sv *SVG) {} // only for the elements that refer to it

func (g *ClipPath) EnforceSVGName() bool { return false }

// urlName returns the name (id) of the element that the given url()
// value refers to (url(#name), with or without quotes), or "" if it is
// not one (as none).
func urlName(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	s = strings.TrimSpace(s)
	rest, ok := strings.CutPrefix(s, "url(")
	if !ok {
		return ""
	}
	rest = strings.TrimSpace(strings.TrimSuffix(rest, ")"))
	rest = strings.Trim(rest, `"'`)
	_, name, ok := strings.Cut(rest, "#")
	if !ok {
		return ""
	}
	return name
}

// findByName returns the element with the given name (id), in the defs
// or the tree of the SVG, or nil if there is none.
func (sv *SVG) findByName(name string) Node {
	if name == "" {
		return nil
	}
	var found Node
	find := func(n tree.Node) bool {
		if found != nil {
			return tree.Break
		}
		if n.AsTree().Name == name {
			found = n.(Node)
			return tree.Break
		}
		return tree.Continue
	}
	sv.Defs.WalkDown(find)
	if found == nil {
		sv.Root.WalkDown(find)
	}
	return found
}

// setLayerRefs sets the clip-path and mask references of the node from
// the given properties (as of its attributes or the rules of its style
// sheet that apply to it), if they have them.
func (g *NodeBase) setLayerRefs(properties map[string]any) {
	if v, ok := properties["clip-path"]; ok {
		g.clipRef = urlName(v)
	}
	if v, ok := properties["mask"]; ok {
		g.maskRef = urlName(v)
	}
}

// isGroupOpacity returns whether the opacity of the node applies to its
// rendering as a whole (as for the opacity of groups), rather than to
// its fill and stroke.
func (g *NodeBase) isGroupOpacity() bool {
	_, ok := g.This.(*Group)
	return ok
}

// pushLayer pushes a context for the clip-path and mask of a shape (which
// renders with its transform without pushing a context), if it has them,
// returning whether it did, in which case the shape pops it after it
// renders. The context has no transform of its own, but its clip mask and
// mask are in the user space of the shape (with its transform).
func (g *NodeBase) pushLayer(sv *SVG) bool {
	g.preparePatterns(sv, g.Paint.Transform)
	if g.clipRef == "" && g.maskRef == "" {
		return false
	}
	sv.pushTransform(math32.Identity2())
	g.setLayer(sv, g.Paint.Transform)
	return true
}

// setLayer sets the clip mask, mask, and opacity of the context of the
// node, which it has just pushed, if it has a clip-path or mask that
// refers to an element, or the opacity of a group (see
// [paint.State.SetLayer]). The clip mask and mask are rendered with the
// given transform, from the user space of the context.
func (g *NodeBase) setLayer(sv *SVG, xf math32.Matrix2) {
	var opacity float32
	if g.isGroupOpacity() && g.Paint.Opacity < 1 {
		opacity = g.Paint.Opacity
	}
	var cp *ClipPath
	var mk *Mask
	if g.clipRef != "" {
		cp, _ = sv.findByName(g.clipRef).(*ClipPath)
	}
	if g.maskRef != "" {
		mk, _ = sv.findByName(g.maskRef).(*Mask)
	}
	if cp == nil && mk == nil && opacity == 0 {
		return
	}
	pc := g.Painter(sv)
	var clip, mask render.Render
	if cp != nil {
		clip = pc.RenderSeparately(func() {
			sv.pushTransform(xf)
			cp.renderClip(sv, g)
			sv.painter.PopContext()
		})
		if len(clip) == 0 { // nothing is in the clip path: nothing is rendered
			clip = render.Render{&render.ContextPush{}, &render.ContextPop{}}
		}
	}
	if mk != nil {
		mask = pc.RenderSeparately(func() {
			sv.pushTransform(xf)
			mk.renderMask(sv, g)
			sv.painter.PopContext()
		})
		if len(mask) == 0 {
			mask = render.Render{&render.ContextPush{}, &render.ContextPop{}}
		}
	}
	pc.SetLayer(clip, mask, opacity)
}

// userBBox returns the bounding box of the given node in its user space,
// for the objectBoundingBox units of clip paths, masks, and patterns.
func (sv *SVG) userBBox(g *NodeBase) math32.Box2 {
	if bb := g.This.(Node).LocalBBox(sv); bb.Size() != (math32.Vector2{}) {
		return bb
	}
	// groups: their box (in the coordinates of the SVG) in their user space
	inv := sv.painter.Context().Cumulative.Inverse()
	return g.BBox.MulMatrix2(inv)
}

// unitsTransform returns the transform from the objectBoundingBox units
// of the given property of the given node (as clipPathUnits) to the
// user space of the given target node, or the identity for
// userSpaceOnUse units (or the given default).
func (sv *SVG) unitsTransform(n *NodeBase, prop string, def bool, target *NodeBase) math32.Matrix2 {
	obb := def
	if v, ok := n.Properties[prop]; ok {
		obb = fmt.Sprint(v) == "objectBoundingBox"
	}
	if !obb {
		return math32.Identity2()
	}
	bb := sv.userBBox(target)
	sz := bb.Size()
	return math32.Translate2D(bb.Min.X, bb.Min.Y).Scale(sz.X, sz.Y)
}

// pushTransform pushes a context with the given transform, from the user
// space of the current one.
func (sv *SVG) pushTransform(xf math32.Matrix2) {
	sty := styles.NewPaint()
	sty.UnitContext = sv.painter.Context().Style.UnitContext
	sty.Transform = xf
	sv.painter.PushContext(sty, nil)
}

// renderClip renders the shapes of the clip path, for the clip mask of
// the given target node (in its user space, as its context is pushed):
// as opaque fills (with their clip-rule), whatever their fill and stroke.
func (g *ClipPath) renderClip(sv *SVG, target *NodeBase) {
	xf := sv.unitsTransform(&g.NodeBase, "clipPathUnits", false, target).Mul(g.Paint.Transform)
	sv.pushTransform(xf)
	defer sv.painter.PopContext()
	type saved struct {
		nb    *NodeBase
		paint styles.Paint
		isDef bool
	}
	var restore []saved
	white := colors.Uniform(color.RGBA{255, 255, 255, 255})
	rule := clipRule(g.Properties, ppath.NonZero)
	var setRules func(n tree.Node, rule ppath.FillRules)
	setRules = func(n tree.Node, rule ppath.FillRules) {
		nb := n.(Node).AsNodeBase()
		rule = clipRule(nb.Properties, rule)
		restore = append(restore, saved{nb, nb.Paint, nb.isDef})
		pc := &nb.Paint
		pc.Fill.Color, pc.Fill.Opacity, pc.Fill.Rule = white, 1, rule
		pc.Stroke.Color = nil
		pc.Opacity = 1
		pc.Off = false
		nb.isDef = true // rendered whatever their visible box
		for _, k := range nb.Children {
			setRules(k, rule)
		}
	}
	for _, k := range g.Children {
		setRules(k, rule)
	}
	defer func() {
		for _, s := range restore {
			s.nb.Paint, s.nb.isDef = s.paint, s.isDef
		}
	}()
	for _, k := range g.Children {
		k.(Node).Render(sv)
	}
}

// clipRule returns the fill rule of the clip-rule in the given
// properties, or the given one (inherited) if they have none.
func clipRule(properties map[string]any, rule ppath.FillRules) ppath.FillRules {
	switch fmt.Sprint(properties["clip-rule"]) {
	case "evenodd":
		return ppath.EvenOdd
	case "nonzero":
		return ppath.NonZero
	}
	return rule
}

// renderMask renders the children of the mask, for the mask of the given
// target node (in its user space, as its context is pushed), clipped to
// the region of the mask (by default 10% larger than the bounding box of
// the target on each side).
func (g *Mask) renderMask(sv *SVG, target *NodeBase) {
	// the region of the mask, in maskUnits (objectBoundingBox by default)
	region := math32.B2(-0.1, -0.1, 1.1, 1.1)
	pos := func(prop string, def float32) float32 {
		v, ok := g.Properties[prop]
		if !ok {
			return def
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if pct, ok := strings.CutSuffix(s, "%"); ok {
			f, _ := math32.ParseFloat32(pct)
			return f / 100
		}
		f, _ := math32.ParseFloat32(s)
		return f
	}
	x, y := pos("x", region.Min.X), pos("y", region.Min.Y)
	w, h := pos("width", region.Size().X), pos("height", region.Size().Y)
	unitsXf := sv.unitsTransform(&g.NodeBase, "maskUnits", true, target)
	regionPath := ppath.New().Rectangle(x, y, w, h)

	// the children, clipped to the region
	pc := g.Painter(sv)
	sv.pushTransform(math32.Identity2())
	regionClip := pc.RenderSeparately(func() {
		sv.pushTransform(unitsXf)
		sty := styles.NewPaint()
		sty.Fill.Color = colors.Uniform(color.RGBA{255, 255, 255, 255})
		sty.Stroke.Color = nil
		rp := &render.Path{Path: *regionPath, Context: *render.NewContext(sty, nil, sv.painter.Context())}
		sv.painter.Render.Add(rp)
		sv.painter.PopContext()
	})
	pc.SetLayer(regionClip, nil, 0)
	sv.pushTransform(sv.unitsTransform(&g.NodeBase, "maskContentUnits", false, target))
	for _, k := range g.Children {
		nb := k.(Node).AsNodeBase()
		def := nb.isDef
		nb.isDef = true // rendered whatever their visible box
		k.(Node).Render(sv)
		nb.isDef = def
	}
	sv.painter.PopContext()
	sv.painter.PopContext()
}
