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
