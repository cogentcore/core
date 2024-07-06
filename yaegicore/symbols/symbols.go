// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package symbols contains yaegi symbols for core packages.
package symbols

//go:generate ./make

import "reflect"

var Symbols = map[string]map[string]reflect.Value{}
