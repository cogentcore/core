// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import "reflect"

// Symbols variable stores the map of stdlib symbols per package.
var Symbols = map[string]map[string]reflect.Value{}

// MapTypes variable contains a map of functions which have an interface{} as parameter but
// do something special if the parameter implements a given interface.
var MapTypes = map[reflect.Value][]reflect.Type{}

func init() {
	Symbols["cogentcore.org/core/shell/interpreter/interpreter"] = map[string]reflect.Value{
		"Symbols": reflect.ValueOf(Symbols),
	}
	Symbols["."] = map[string]reflect.Value{
		"MapTypes": reflect.ValueOf(MapTypes),
	}
}
