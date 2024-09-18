// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

//go:generate ./make

import (
	"reflect"

	_ "cogentcore.org/core/yaegicore/symbols"
	"github.com/cogentcore/yaegi/interp"
)

var Symbols = map[string]map[string]reflect.Value{}

// ImportGoal imports special symbols from the goal package.
func (in *Interpreter) ImportGoal() {
	in.Interp.Use(interp.Exports{
		"cogentcore.org/core/goal/goal": map[string]reflect.Value{
			"Run":         reflect.ValueOf(in.Goal.Run),
			"RunErrOK":    reflect.ValueOf(in.Goal.RunErrOK),
			"Output":      reflect.ValueOf(in.Goal.Output),
			"OutputErrOK": reflect.ValueOf(in.Goal.OutputErrOK),
			"Start":       reflect.ValueOf(in.Goal.Start),
			"AddCommand":  reflect.ValueOf(in.Goal.AddCommand),
			"RunCommands": reflect.ValueOf(in.Goal.RunCommands),
		},
	})
}
