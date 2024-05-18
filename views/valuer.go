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
		typ := reflectx.NonPointerType(reflect.TypeOf(value))
		kind := typ.Kind()
		switch kind {
		case reflect.Array, reflect.Slice:
			v := reflect.ValueOf(value)
			sz := v.Len()
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
			nfld := reflectx.NumAllFields(typ)
			if nfld > 0 && !forceNoInline && (forceInline || nfld <= core.SystemSettings.StructInlineLength) {
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
