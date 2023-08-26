// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"fmt"
	"reflect"

	"github.com/iancoleman/strcase"
)

// Run runs the given app with the given default
// configuration file paths. The app should be
// a pointer, and configuration options should
// be defined as fields on the app type. Also,
// commands should be defined as methods on the
// app type with the suffix "Cmd"; for example,
// for a command named "build", there should be
// the method:
//
//	func (a *App) BuildCmd() error
//
// Run uses [os.Args] for its arguments.
func Run(app any, defaultFile ...string) {
	leftovers, err := Config(app, defaultFile...)
	_ = err
	if len(leftovers) == 0 {
		GUI(app)
		return
	}
	cmd := strcase.ToCamel(leftovers[0]) + "Cmd"
	fmt.Printf("running command: %s\n", cmd)
	val := reflect.ValueOf(app)
	meth := val.MethodByName(cmd)
	// todo: check for bad
	meth.Call(nil) // no args!!
}
