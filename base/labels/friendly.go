// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package labels

import (
	"fmt"
	"reflect"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
)

// FriendlyTypeName returns a user-friendly version of the name of the given type.
// It transforms it into sentence case, excludes the package, and converts various
// builtin types into more friendly forms (eg: "int" to "Number").
func FriendlyTypeName(typ reflect.Type) string {
	nptyp := reflectx.NonPointerType(typ)
	if nptyp == nil {
		return "None"
	}
	nm := nptyp.Name()

	// if it is named, we use that
	if nm != "" {
		switch nm {
		case "string":
			return "Text"
		case "float32", "float64", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
			return "Number"
		}
		return strcase.ToSentence(nm)
	}

	// otherwise, we fall back on Kind
	switch nptyp.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		bnm := FriendlyTypeName(nptyp.Elem())
		if strings.HasSuffix(bnm, "s") {
			return "List of " + bnm
		} else if strings.Contains(bnm, "Function of") {
			return strings.ReplaceAll(bnm, "Function of", "Functions of") + "s"
		}
		return bnm + "s"
	case reflect.Func:
		str := "Function"
		ni := nptyp.NumIn()
		if ni > 0 {
			str += " of"
		}
		for i := 0; i < ni; i++ {
			str += " " + FriendlyTypeName(nptyp.In(i))
			if ni == 2 && i == 0 {
				str += " and"
			} else if i == ni-2 {
				str += ", and"
			} else if i < ni-1 {
				str += ","
			}
		}
		return str
	}
	if nptyp.String() == "interface {}" {
		return "Value"
	}
	return nptyp.String()
}

// FriendlyStructLabel returns a user-friendly label for the given struct value.
func FriendlyStructLabel(v reflect.Value) string {
	npv := reflectx.NonPointerValue(v)
	if !v.IsValid() || v.IsZero() {
		return "None"
	}
	opv := reflectx.UnderlyingPointer(v)
	if lbler, ok := opv.Interface().(Labeler); ok {
		return lbler.Label()
	}
	return FriendlyTypeName(npv.Type())
}

// FriendlySliceLabel returns a user-friendly label for the given slice value.
func FriendlySliceLabel(v reflect.Value) string {
	uv := reflectx.Underlying(v)
	label := ""
	if !uv.IsValid() {
		label = "None"
	} else {
		if uv.Kind() == reflect.Array || !uv.IsNil() {
			bnm := FriendlyTypeName(reflectx.SliceElementType(v.Interface()))
			if strings.HasSuffix(bnm, "s") {
				label = strcase.ToSentence(fmt.Sprintf("%d lists of %s", uv.Len(), bnm))
			} else {
				label = strcase.ToSentence(fmt.Sprintf("%d %ss", uv.Len(), bnm))
			}
		} else {
			label = "None"
		}
	}
	return label
}

// FriendlyMapLabel returns a user-friendly label for the given map value.
func FriendlyMapLabel(v reflect.Value) string {
	uv := reflectx.Underlying(v)
	mpi := v.Interface()
	label := ""
	if !uv.IsValid() || uv.IsNil() {
		label = "None"
	} else {
		bnm := FriendlyTypeName(reflectx.MapValueType(mpi))
		if strings.HasSuffix(bnm, "s") {
			label = strcase.ToSentence(fmt.Sprintf("%d lists of %s", uv.Len(), bnm))
		} else {
			label = strcase.ToSentence(fmt.Sprintf("%d %ss", uv.Len(), bnm))
		}
	}
	return label
}
