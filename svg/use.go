// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log/slog"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// Symbol is an SVG symbol element, which is only rendered by the use
// elements that refer to it, scaled from its ViewBox to their size.
type Symbol struct {
	Group

	// ViewBox is the view box of the symbol, if any.
	ViewBox ViewBox
}

func (g *Symbol) SVGName() string { return "symbol" }

func (g *Symbol) EnforceSVGName() bool { return false }

func (g *Symbol) Render(sv *SVG) {} // only through use elements

// pendingUse is a use element, as a group, whose element is resolved
// after the parsing.
type pendingUse struct {
	group               *Group
	link                string
	x, y, width, height float32
}

// resolveUses adds a copy of the element that each of the given use
// elements refers to, to its group. A symbol is added as a group, with
// the transform from its view box to the size of the use (or its view
// box size, if the use has none).
func (sv *SVG) resolveUses(uses []pendingUse) {
	for _, pu := range uses {
		itm := sv.findByName(pu.link)
		if itm == nil {
			slog.Debug("svg: use refers to an element that does not exist", "href", pu.link)
			continue
		}
		if isAncestor(itm, pu.group) { // a use of its own ancestor
			continue
		}
		cln := itm.AsTree().Clone().(Node)
		if sym, ok := cln.(*Symbol); ok {
			g := NewGroup(pu.group)
			g.Properties = sym.Properties
			delete(g.Properties, "transform")
			if vb := sym.ViewBox; vb.Size.X > 0 && vb.Size.Y > 0 {
				size := math32.Vec2(pu.width, pu.height)
				if size.X <= 0 {
					size.X = vb.Size.X
				}
				if size.Y <= 0 {
					size.Y = vb.Size.Y
				}
				_, trans, scale := vb.Transform(size)
				trans.SetSub(vb.Min)
				xf := math32.Scale2D(scale.X, scale.Y).Translate(trans.X, trans.Y)
				g.SetProperty("transform", xf.String())
			}
			for _, k := range sym.Children {
				g.AddChild(k)
			}
			sym.Children = nil
			cln = g
		} else {
			pu.group.AddChild(cln)
		}
		cln.AsTree().WalkDown(func(n tree.Node) bool {
			if p, ok := n.(*Path); ok {
				p.SetData(p.DataStr) // defs don't apply paths
			}
			return tree.Continue
		})
	}
}

// isAncestor returns whether the given node is the given other node or
// one of its ancestors.
func isAncestor(n Node, of tree.Node) bool {
	for p := of; p != nil; p = p.AsTree().Parent {
		if p == n {
			return true
		}
	}
	return false
}
