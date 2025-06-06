// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package styleprops provides infrastructure for property-list-based setting
// of style values, where a property list is a map[string]any collection of
// key, value pairs.
package styleprops

import (
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/base/num"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/units"
)

// Func is the signature for styleprops functions
type Func func(obj any, key string, val any, parent any, cc colors.Context)

// InhInit detects the style values of "inherit" and "initial",
// setting the corresponding bool return values
func InhInit(val, parent any) (inh, init bool) {
	if str, ok := val.(string); ok {
		switch str {
		case "inherit":
			return !reflectx.IsNil(reflect.ValueOf(parent)), false
		case "initial":
			return false, true
		default:
			return false, false
		}
	}
	return false, false
}

// FuncInt returns a style function for any numerical value
func Int[T any, F num.Integer](initVal F, getField func(obj *T) *F) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, err := reflectx.ToInt(val)
		if err == nil {
			*fp = F(fv)
		}
	}
}

// Float returns a style function for any numerical value.
// Automatically removes a trailing % -- see FloatProportion.
func Float[T any, F num.Float](initVal F, getField func(obj *T) *F) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		if vstr, ok := val.(string); ok {
			val = strings.TrimSuffix(vstr, "%")
		}
		fv, err := reflectx.ToFloat(val)
		if err == nil {
			*fp = F(fv)
		}
	}
}

// FloatProportion returns a style function for a proportion that can be
// represented as a percentage (divides value by 100).
func FloatProportion[T any, F num.Float](initVal F, getField func(obj *T) *F) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		isPct := false
		if vstr, ok := val.(string); ok {
			val = strings.TrimSuffix(vstr, "%")
			isPct = true
		}
		fv, err := reflectx.ToFloat(val)
		if isPct {
			fv /= 100
		}
		if err == nil {
			*fp = F(fv)
		}
	}
}

// Bool returns a style function for a bool value
func Bool[T any](initVal bool, getField func(obj *T) *bool) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, err := reflectx.ToBool(val)
		if err == nil {
			*fp = fv
		}
	}
}

// Units returns a style function for units.Value
func Units[T any](initVal units.Value, getField func(obj *T) *units.Value) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				*fp = *getField(parent.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fp.SetAny(val, key)
	}
}

// Enum returns a style function for any enum value
func Enum[T any](initVal enums.Enum, getField func(obj *T) enums.EnumSetter) Func {
	return func(obj any, key string, val any, parent any, cc colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := InhInit(val, parent); inh || init {
			if inh {
				fp.SetInt64(getField(parent.(*T)).Int64())
			} else if init {
				fp.SetInt64(initVal.Int64())
			}
			return
		}
		if st, ok := val.(string); ok {
			fp.SetString(st)
			return
		}
		if en, ok := val.(enums.Enum); ok {
			fp.SetInt64(en.Int64())
			return
		}
		iv, err := reflectx.ToInt(val)
		if err == nil {
			fp.SetInt64(int64(iv))
		}
	}
}

// SetError reports that cannot set property of given key with given value due to given error
func SetError(key string, val any, err error) {
	slog.Error("styleprops: error setting value", "key", key, "value", val, "err", err)
}
