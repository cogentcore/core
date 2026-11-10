// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"golang.org/x/net/html"
)

// This file has the containers of CSS (container-type: size or
// inline-size), for the container query units (cqw, cqh, cqi, cqb, cqmin,
// cqmax), which are percentages of the size of the content box of the
// nearest container that an element is in. They are resolved when the
// element is styled, as the other lengths of CSS (in [lengthEnv]), from the
// size of its container in its last layout: after a layout in which the
// size of a container changed, the elements in it are styled again and
// laid out again (as for the screenshots of the platform features of
// github.com). Before the first layout of a container, they are of the
// viewport, as in CSS for an element that is in no container, or of the
// size that it had when the page was last read (see
// [Context.KeepContainerSizes]).

// containerFrame is the frame of an element that is a container.
type containerFrame struct {
	core.Frame

	// inline is whether only the width of the container is queried
	// (container-type: inline-size), so that its height is not used.
	inline bool

	// size is the size of the content box of the container (in dots) in
	// its last layout, which the elements in it are styled for, and prev
	// is the size before it, for stopping sizes that alternate (set with
	// the containerMu of ctx held, for [Context.KeepContainerSizes]).
	size, prev math32.Vector2

	// ctx is the context that read the container.
	ctx *Context
}

// containerSizes are the sizes of a container (see [containerFrame]).
type containerSizes struct {
	size, prev math32.Vector2
}

// KeepContainerSizes sets the containers of the page that the context
// reads, which is the document that the given context read, read again
// (as after its scripts change it), to start with the sizes that they had
// in its last layout, so that the elements in them are styled for those
// first, not for the viewport until they are laid out (as the panels of
// repositories of github.com, of font-size: 1cqw, which were shown at
// the size of the viewport each time that the page was read again).
// It can be called as the given context's page is laid out.
func (c *Context) KeepContainerSizes(prev *Context) {
	if prev == nil {
		return
	}
	prev.containerMu.Lock()
	defer prev.containerMu.Unlock()
	c.keptContainers = make(map[*html.Node]containerSizes, len(prev.containers))
	for n, cf := range prev.containers {
		if cf.size.X > 0 {
			c.keptContainers[n] = containerSizes{cf.size, cf.prev}
		}
	}
}

// isContainer returns whether the given element is a container, and
// whether only its width is queried (container-type: inline-size).
func (c *Context) isContainer(n *html.Node) (container, inline bool) {
	typ := strings.ToLower(c.declaration(n, "container-type"))
	if v := c.declaration(n, "container"); v != "" { // name / type
		if _, t, ok := strings.Cut(v, "/"); ok {
			typ = strings.ToLower(t)
		}
	}
	switch strings.TrimSpace(typ) {
	case "size":
		return true, false
	case "inline-size":
		return true, true
	}
	return false, false
}

// SizeFinal sizes the container, and when the size of its content box
// changed (by more than a dot) from the one that the elements in it were
// styled for, it styles them again for it, and lays them out again, after
// this render (not during the layout).
func (cf *containerFrame) SizeFinal() {
	cf.Frame.SizeFinal()
	sz := cf.Geom.Size.Actual.Content
	if cf.inline {
		sz.Y = 0
	}
	changed := func(a, b math32.Vector2) bool {
		return math32.Abs(a.X-b.X) > 1 || math32.Abs(a.Y-b.Y) > 1
	}
	if !changed(sz, cf.size) || !changed(sz, cf.prev) && cf.prev != (math32.Vector2{}) {
		return
	}
	cf.ctx.containerMu.Lock()
	cf.prev, cf.size = cf.size, sz
	cf.ctx.containerMu.Unlock()
	cf.Defer(func() {
		cf.Restyle()
		cf.NeedsLayout()
	})
}

// containerUnits returns the size of 1% of the width and the height of the
// nearest containers that the given widget is in (cqw and cqh), in dp, or
// the given ones (of the viewport) if it is in none, or none has been laid
// out yet.
func containerUnits(w core.Widget, vw, vh float64) (cqw, cqh float64) {
	cqw, cqh = vw, vh
	gotW, gotH := false, false
	for p := w.AsTree().Parent; p != nil && !(gotW && gotH); p = p.AsTree().Parent {
		cf, ok := p.(*containerFrame)
		if !ok {
			continue
		}
		dp := cf.Styles.UnitContext.Dots(units.UnitDp)
		if dp <= 0 || cf.size.X <= 0 {
			continue
		}
		if !gotW {
			cqw, gotW = float64(cf.size.X/dp/100), true
		}
		if !gotH && !cf.inline {
			cqh, gotH = float64(cf.size.Y/dp/100), true
		}
	}
	return
}

// frameOf returns the given widget as the frame of an element (a plain
// frame or that of a container), or nil if it is not one.
func frameOf(w any) *core.Frame {
	switch f := w.(type) {
	case *core.Frame:
		return f
	case *containerFrame:
		return &f.Frame
	}
	return nil
}

// containerCond is the condition of an @container rule: the name of the
// container that it queries ("" for the nearest one), and its query (as
// (width <= 799px)), and that of an @container rule that it is in.
type containerCond struct {
	name, query string
	outer       *containerCond
}

// parseContainerCond returns the condition of an @container rule of the
// given prelude (as "spotlight (width <= 799px)"), in the given one.
func parseContainerCond(prelude string, outer *containerCond) *containerCond {
	prelude = strings.TrimSpace(prelude)
	cc := &containerCond{query: prelude, outer: outer}
	if i := strings.IndexAny(prelude, "( "); i > 0 && !strings.HasPrefix(strings.ToLower(prelude), "not") {
		name := strings.TrimSpace(prelude[:i])
		if name != "" && !strings.ContainsAny(name, "()") {
			cc.name = name
			cc.query = strings.TrimSpace(prelude[i:])
		}
	}
	return cc
}

// containerDecls returns the given declarations of the given element
// without those of @container rules whose conditions it does not match.
func (c *Context) containerDecls(n *html.Node, ds []*cssDecl) []*cssDecl {
	out := make([]*cssDecl, 0, len(ds))
	for _, d := range ds {
		if d.container == nil || c.containerMatches(n, d.container) {
			out = append(out, d)
		}
	}
	return out
}

// containerMatches returns whether the given element matches the given
// @container condition: of the width of its nearest container (of that
// name) in its last layout, as the container query units (see
// [containerUnits]), or of the viewport before it is laid out, as the
// spotlight cards of edition.cnn.com, whose aspect ratios were those of
// the last of their @container rules, which all applied.
func (c *Context) containerMatches(n *html.Node, cc *containerCond) bool {
	for ; cc != nil; cc = cc.outer {
		width := c.mediaWidth()
		for p := n.Parent; p != nil && p.Type == html.ElementNode; p = p.Parent {
			if ok, _ := c.isContainer(p); !ok {
				continue
			}
			if cc.name != "" && !c.containerNamed(p, cc.name) {
				continue
			}
			if cf := c.containers[p]; cf != nil && cf.size.X > 0 {
				if dp := cf.Styles.UnitContext.Dots(units.UnitDp); dp > 0 {
					width = cf.size.X / dp
				}
			}
			break
		}
		saved, checking := c.MediaWidth, c.checkingMedia
		c.MediaWidth, c.checkingMedia = width, true
		m := c.mediaListMatches(strings.ToLower(cc.query))
		c.MediaWidth, c.checkingMedia = saved, checking
		if !m {
			return false
		}
	}
	return true
}

// containerNamed returns whether the given container has the given name
// (of container-name, or the container shorthand).
func (c *Context) containerNamed(n *html.Node, name string) bool {
	names := c.declaration(n, "container-name")
	if v := c.declaration(n, "container"); v != "" {
		names, _, _ = strings.Cut(v, "/")
	}
	for _, f := range strings.Fields(names) {
		if f == name {
			return true
		}
	}
	return false
}
