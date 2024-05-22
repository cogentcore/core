// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"image/color"
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// This file handles converting values to [Value]s.

func init() {
	core.SettingsWindow = SettingsWindow
	core.InspectorWindow = InspectorWindow

	core.AddValueConverter(func(value any, tags reflect.StructTag) core.Value {
		if _, ok := value.(color.Color); ok {
			return NewColorButton()
		}

		forceInline := tags.Get("view") == "inline"
		forceNoInline := tags.Get("view") == "no-inline"
		rval := reflectx.Underlying(reflect.ValueOf(value))
		if rval == (reflect.Value{}) {
			return nil
		}
		typ := rval.Type()
		kind := typ.Kind()
		switch kind {
		case reflect.Array, reflect.Slice:
			sz := rval.Len()
			eltyp := reflectx.SliceElementType(value)
			if _, ok := value.([]byte); ok {
				return core.NewTextField()
			}
			if _, ok := value.([]rune); ok {
				return core.NewTextField()
			}
			isstru := (reflectx.NonPointerType(eltyp).Kind() == reflect.Struct)
			if !forceNoInline && (forceInline || (!isstru && sz <= core.SystemSettings.SliceInlineLength && !tree.IsNode(eltyp))) {
				return NewSliceViewInline()
			} else {
				return NewSliceButton()
			}
		case reflect.Struct:
			num := reflectx.NumAllFields(rval)
			if !forceNoInline && (forceInline || num <= core.SystemSettings.StructInlineLength) {
				return NewStructView().SetInline(true)
			} else {
				return NewStructButton()
			}
		case reflect.Map:
			return NewMapButton()
		}
		return nil
	})
}
