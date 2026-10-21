// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript && !noyaegi

package main

import (
	"reflect"

	"cogentcore.org/core/yaegicore"
	"cogentcore.org/core/yaegicore/coresymbols"
)

// The code blocks of the docs are compiled into the app (contentcode_gen.go).
// Yaegi runs code that users edit, natively and on wasm; it is not in js
// (GoScript) builds, or with the noyaegi tag.

var _ = yaegicore.BindTextEditor

func init() {
	coresymbols.Symbols["."]["econtent"] = reflect.ValueOf(econtent)
	coresymbols.Symbols["."]["myImage"] = reflect.ValueOf(myImage)
	coresymbols.Symbols["."]["mySVG"] = reflect.ValueOf(mySVG)
	coresymbols.Symbols["."]["myFile"] = reflect.ValueOf(myFile)
}
