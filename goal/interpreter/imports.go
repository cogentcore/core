// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

//go:generate ./make

import (
	"reflect"

	"github.com/cogentcore/yaegi/interp"
)

var Symbols = map[string]map[string]reflect.Value{}

// ImportShell imports special symbols from the shell package.
func (in *Interpreter) ImportShell() {
	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/shell/shell": map[string]reflect.Value{
			"Run":         reflect.ValueOf(in.Shell.Run),
			"RunErrOK":    reflect.ValueOf(in.Shell.RunErrOK),
			"Output":      reflect.ValueOf(in.Shell.Output),
			"OutputErrOK": reflect.ValueOf(in.Shell.OutputErrOK),
			"Start":       reflect.ValueOf(in.Shell.Start),
			"AddCommand":  reflect.ValueOf(in.Shell.AddCommand),
			"RunCommands": reflect.ValueOf(in.Shell.RunCommands),
		},
	})
}
