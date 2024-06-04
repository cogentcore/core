// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

//go:generate core generate

import (
	"image/color"
	"reflect"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// This file handles converting values to [Value]s.

func init() {
	core.SettingsWindow = SettingsWindow
	core.InspectorWindow = InspectorWindow

	core.AddValueType[icons.Icon, *IconButton]()
	core.AddValueType[time.Time, *TimeText]()
	// AddValue(time.Duration(0), func() Value { return &DurationValue{} })
	core.AddValueType[types.Type, *TypeChooser]()
	core.AddValueType[core.Filename, *FileButton]()
	core.AddValueType[core.FontName, *FontButton]()
	core.AddValueType[keymap.MapName, *KeyMapButton]()
	core.AddValueType[key.Chord, *KeyChordButton]()

	core.AddValueConverter(func(value any, tags reflect.StructTag) core.Value {
		if _, ok := value.(color.Color); ok {
			return NewColorButton()
		}
		if _, ok := value.(tree.Node); ok {
			return NewTreeButton()
		}

		forceInline := tags.Get("view") == "inline"
		forceNoInline := tags.Get("view") == "no-inline"
		uv := reflectx.Underlying(reflect.ValueOf(value))
		if !uv.IsValid() {
			return nil
		}
		typ := uv.Type()
		kind := typ.Kind()
		switch kind {
		case reflect.Struct:
			num := reflectx.NumAllFields(uv)
			if !forceNoInline && (forceInline || num <= core.SystemSettings.StructInlineLength) {
				return NewStructView().SetInline(true)
			} else {
				return NewStructButton()
			}
		case reflect.Map:
			return NewMapButton() // TODO(config): inline map value
		case reflect.Array, reflect.Slice:
			sz := uv.Len()
			elemType := reflectx.SliceElementType(value)
			if _, ok := value.([]byte); ok {
				return core.NewTextField()
			}
			if _, ok := value.([]rune); ok {
				return core.NewTextField()
			}
			isStruct := (reflectx.NonPointerType(elemType).Kind() == reflect.Struct)
			if !forceNoInline && (forceInline || (!isStruct && sz <= core.SystemSettings.SliceInlineLength && !tree.IsNode(elemType))) {
				return NewSliceViewInline()
			} else {
				return NewSliceButton()
			}
		case reflect.Func:
			return tree.New[*FuncButton]() // TODO(config): update to NewFuncButton after changing its signature
		}
		return nil
	})
}
