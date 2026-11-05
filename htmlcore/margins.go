// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/sides"
)

// collapseMargins collapses the given top and bottom margins (in dots) of
// the given block widget with the adjoining margins of its previous
// sibling and its parent, as browsers do. The margins of the widgets stay
// within their parents, but the space between them is that of the
// collapsed margins.
func (c *Context) collapseMargins(wb *core.WidgetBase, s, ps *styles.Style, bs, pbs *blockState, pad, bw sides.Floats, top, bottom *float32) {
	flow := s.Display == styles.Flex && s.Direction == styles.Column
	bs.openTop = flow && pad.Top == 0 && bw.Top == 0
	bs.openBottom = flow && pad.Bottom == 0 && bw.Bottom == 0 && s.Min.Y.Value == 0
	bs.topBase, bs.bottomBase = *top, *bottom
	if pbs == nil || !bs.block || !(pbs.display == "block" || pbs.display == "list-item" || pbs.display == "table-cell" || pbs.display == "inline-block") {
		return
	}
	pw := wb.Parent.(core.Widget)
	prev, isLast := c.blockSiblings(pw, wb.This.(core.Widget))

	adj := float32(0)
	switch {
	case prev != nil:
		adj = c.chainBelow(prev)
	case pbs.openTop:
		adj = pbs.topBase
	}
	if *top >= 0 && adj > 0 {
		*top = max(0, *top-adj)
	}
	bs.topBase = adj + *top

	if isLast && pbs.openBottom {
		base := pbs.bottomBase
		if *bottom >= 0 && base > 0 {
			*bottom = max(0, *bottom-base)
		}
		bs.bottomBase = base + *bottom
	}
}

// blockSiblings returns the previous block sibling of the given widget,
// and whether it is the last child of the parent.
func (c *Context) blockSiblings(pw, w core.Widget) (prev core.Widget, isLast bool) {
	kids := pw.AsTree().Children
	idx := -1
	for i, k := range kids {
		if k == w {
			idx = i
			break
		}
	}
	for i := idx - 1; i >= 0; i-- {
		kw, ok := kids[i].(core.Widget)
		if !ok {
			continue
		}
		if kbs := c.blocks[kw]; kbs != nil && kbs.block && kw.AsWidget().Styles.Display != styles.DisplayNone {
			prev = kw
			break
		}
	}
	return prev, idx == len(kids)-1
}

// chainBelow returns the total of the collapsed margins below the given
// block widget, through its last descendants.
func (c *Context) chainBelow(w core.Widget) float32 {
	bs := c.blocks[w]
	total := bs.bottomBase
	for bs != nil && bs.openBottom {
		kids := w.AsTree().Children
		if len(kids) == 0 {
			break
		}
		kw, ok := kids[len(kids)-1].(core.Widget)
		if !ok {
			break
		}
		kbs := c.blocks[kw]
		if kbs == nil || !kbs.block {
			break
		}
		w, bs = kw, kbs
		total = bs.bottomBase
	}
	return total
}
