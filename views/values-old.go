// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

/* TODO(config): remove

import (
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// This file contains the standard [Value]s built into views.

// TreeValue represents a [tree.Node] value with a button.
type TreeValue struct {
	ValueBase[*core.Button]
}

func (v *TreeValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *TreeValue) Update() {
	path := "None"
	k := v.TreeValue()
	if k != nil && k.This() != nil {
		path = k.AsTreeNode().String()
	}
	v.Widget.SetText(path).Update()
}

func (v *TreeValue) ConfigDialog(d *core.Body) (bool, func()) {
	k := v.TreeValue()
	if k == nil {
		return false, nil
	}
	InspectorView(d, k)
	return true, nil
}

// TreeValue returns the actual underlying [tree.Node] value, or nil.
func (vv *TreeValue) TreeValue() tree.Node {
	if !vv.Value.IsValid() || vv.Value.IsNil() {
		return nil
	}
	npv := reflectx.NonPointerValue(vv.Value)
	if npv.Kind() == reflect.Interface {
		return npv.Interface().(tree.Node)
	}
	opv := reflectx.OnePointerValue(vv.Value)
	if opv.IsNil() {
		return nil
	}
	return opv.Interface().(tree.Node)
}

// TypeValue represents a [types.Type] value with a chooser.
type TypeValue struct {
	ValueBase[*core.Chooser]
}

func (v *TypeValue) Config() {
	typEmbeds := core.WidgetBaseType
	if tetag, ok := v.Tag("type-embeds"); ok {
		typ := types.TypeByName(tetag)
		if typ != nil {
			typEmbeds = typ
		}
	}

	tl := types.AllEmbeddersOf(typEmbeds)
	v.Widget.SetTypes(tl...)
	v.Widget.OnChange(func(e events.Event) {
		tval := v.Widget.CurrentItem.Value.(*types.Type)
		v.SetValue(tval)
	})
}

func (v *TypeValue) Update() {
	opv := reflectx.OnePointerValue(v.Value)
	typ := opv.Interface().(*types.Type)
	v.Widget.SetCurrentValue(typ)
}

*/
