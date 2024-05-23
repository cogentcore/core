// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"reflect"

	"cogentcore.org/core/shell"
	"github.com/traefik/yaegi/interp"
)

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

// ImportShell imports special symbols from shell package
func (in *Interpreter) ImportShell() {
	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/shell/shell": map[string]reflect.Value{
			"Run":           reflect.ValueOf(in.Shell.Run),
			"RunErrOK":      reflect.ValueOf(in.Shell.RunErrOK),
			"Output":        reflect.ValueOf(in.Shell.Output),
			"OutputErrOK":   reflect.ValueOf(in.Shell.OutputErrOK),
			"Start":         reflect.ValueOf(in.Shell.Start),
			"AddCommand":    reflect.ValueOf(in.Shell.AddCommand),
			"SplitLines":    reflect.ValueOf(shell.SplitLines),
			"FileExists":    reflect.ValueOf(shell.FileExists),
			"WriteFile":     reflect.ValueOf(shell.WriteFile),
			"ReadFile":      reflect.ValueOf(shell.ReadFile),
			"ReplaceInFile": reflect.ValueOf(shell.ReplaceInFile),
		},
	})
}
