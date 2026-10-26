// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package basesymbols

import (
	"reflect"
	"strings"

	"github.com/cogentcore/mvm/goparser"
	"github.com/cogentcore/mvm/interp"
	"github.com/cogentcore/mvm/stdlib"
	"github.com/cogentcore/mvm/symbol"
)

// Import imports the symbols of the given packages, as [Values], into the
// given interpreter, and makes all of its packages usable without an import,
// by their names. Packages outside of the standard library take the names
// that they share with standard library packages, as cogentcore.org/core/base/errors
// does that of errors, whose functions it also has. It also gives the
// interpreter the source of the standard library packages that it
// interprets, if mvm does not have it embedded (see [StdSource]).
func Import(in *interp.Interp, values ...map[string]map[string]reflect.Value) {
	if stdlib.EmbeddedStd() == nil {
		in.SetStdlibFS(StdSource())
	}
	for _, v := range values {
		in.ImportPackageValues(v)
	}
	in.AutoImportPackages()
	for path := range in.Packages {
		if first, _, _ := strings.Cut(path, "/"); !strings.Contains(first, ".") {
			continue
		}
		name := goparser.PackageName(path)
		in.SymSet(name, &symbol.Symbol{Kind: symbol.Pkg, PkgPath: path, Index: symbol.UnsetAddr, Name: name, AutoImport: true})
	}
}
