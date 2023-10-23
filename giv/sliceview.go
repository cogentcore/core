// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"goki.dev/gi/v2/gi"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating an interactive viewer / editor of the
// elements as rows in a table.  Widgets to show the index / value pairs, within an
// overall frame.
// Set to ReadOnly for select-only mode, which emits WidgetSig WidgetSelected
// signals when selection is updated.
type SliceView struct {
	SliceViewBase

	// optional styling function
	StyleFunc SliceViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-"`
}

// check for interface impl
var _ SliceViewer = (*SliceView)(nil)

// SliceViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view.  If style properties are set
// then you must call widg.AsNode2dD().SetFullReRender() to trigger
// re-styling during re-render
type SliceViewStyleFunc func(sv *SliceView, slice any, widg gi.Widget, row int, vv Value)

func (sv *SliceView) StyleRow(svnp reflect.Value, widg gi.Widget, idx, fidx int, vv Value) {
	if sv.StyleFunc != nil {
		sv.StyleFunc(sv, svnp.Interface(), widg, idx, vv)
	}
}
