// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SliceView

// SliceView represents a slice, creating an interactive viewer / editor of the
// elements as rows in a table.  Widgets to show the index / value pairs, within an
// overall frame.
// Set to Inactive for select-only mode, which emits WidgetSig WidgetSelected
// signals when selection is updated.
type SliceView struct {
	SliceViewBase

	// [view: -] optional styling function
	StyleFunc SliceViewStyleFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional styling function"`
}

var TypeSliceView = kit.Types.AddType(&SliceView{}, SliceViewProps)

// AddNewSliceView adds a new sliceview to given parent node, with given name.
func AddNewSliceView(parent ki.Ki, name string) *SliceView {
	return parent.AddNewChild(TypeSliceView, name).(*SliceView)
}

// check for interface impl
var _ SliceViewer = (*SliceView)(nil)

// SliceViewStyleFunc is a styling function for custom styling /
// configuration of elements in the view.  If style properties are set
// then you must call widg.AsNode2dD().SetFullReRender() to trigger
// re-styling during re-render
type SliceViewStyleFunc func(sv *SliceView, slice any, widg gi.Node2D, row int, vv ValueView)

var SliceViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

func (sv *SliceView) StyleRow(svnp reflect.Value, widg gi.Node2D, idx, fidx int, vv ValueView) {
	if sv.StyleFunc != nil {
		sv.StyleFunc(sv, svnp.Interface(), widg, idx, vv)
	}
}
