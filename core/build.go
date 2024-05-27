// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/tree"
)

// Builder adds a new function to [WidgetBase.Builders], which are called in sequential
// descending (reverse) order in [Widget.Build].
func (wb *WidgetBase) Builder(builder func()) {
	wb.Builders = append(wb.Builders, builder)
}

// Maker adds a new function to [WidgetBase.Makers], which are called in sequential
// ascending order in [Widget.Make].
func (wb *WidgetBase) Maker(maker func(p *Plan)) {
	wb.Makers = append(wb.Makers, maker)
}

// Build is the base implementation of [Widget.Build] that
// runs [WidgetBase.Builders] after [WidgetBase.ValueBuild].
func (wb *WidgetBase) Build() {
	if wb.ValueBuild != nil {
		wb.ValueBuild()
	}
	for i := len(wb.Builders) - 1; i >= 0; i-- {
		wb.Builders[i]()
	}
}

// baseBuild is the base builder added to [WidgetBase.Builders] in OnInit.
func (wb *WidgetBase) baseBuild() {
	p := Plan{}
	wb.This().(Widget).Make(&p)
	p.Build(wb)
}

// Make is the base implementation of [Widget.Make] that runs
// [WidgetBase.Makers].
func (wb *WidgetBase) Make(p *Plan) {
	for _, maker := range wb.Makers {
		maker(p)
	}
}

// ConfigTree calls [Widget.Build] on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree() {
	if wb.This() == nil {
		return
	}
	// pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wi.Build()
		return tree.Continue
	})
	// pr.End()
}

// Update does a general purpose update of the widget and everything
// below it by reconfiguring it, applying its styles, and indicating
// that it needs a new layout pass. It is the main way that end users
// should update widgets, and it should be called after making any
// changes to the core properties of a widget (for example, the text
// of [Text], the icon of a [Button], or the slice of a table view).
//
// If you are calling this in a separate goroutine outside of the main
// configuration, rendering, and event handling structure, you need to
// call [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock] before and
// after this, respectively.
func (wb *WidgetBase) Update() { //types:add
	if wb == nil || wb.This() == nil {
		return
	}
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace Update:", wb)
	}
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wi.Build()
		wi.ApplyStyle()
		return tree.Continue
	})
	wb.NeedsLayout()
}
