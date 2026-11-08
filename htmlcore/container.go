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
// viewport, as in CSS for an element that is in no container.

// containerFrame is the frame of an element that is a container.
type containerFrame struct {
	core.Frame

	// inline is whether only the width of the container is queried
	// (container-type: inline-size), so that its height is not used.
	inline bool

	// size is the size of the content box of the container (in dots) in
	// its last layout, which the elements in it are styled for, and prev
	// is the size before it, for stopping sizes that alternate.
	size, prev math32.Vector2
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
	cf.prev, cf.size = cf.size, sz
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
