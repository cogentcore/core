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
func Run(app any, defaultFile ...string) error {
	leftovers, err := Config(app, defaultFile...)
	if err != nil {
		return fmt.Errorf("error configuring app: %w", err)
	}
	if len(leftovers) == 0 {
		GUI(app)
		return nil
	}
	err = RunCommand(app, leftovers[0])
	if err != nil {
		return fmt.Errorf("error running command %q: %w", leftovers[0], err)
	}
	return nil
}

// RunCommand runs the command with the given
// name on the given app. It looks for the
// method with the name of the command converted
// to camel case suffixed with "Cmd"; for example,
// for a command named "build", it will look for a
// method named "BuildCmd".
func RunCommand(app any, cmd string) error {
	name := strcase.ToCamel(cmd) + "Cmd"
	val := reflect.ValueOf(app)
	meth := val.MethodByName(name)
	if !meth.IsValid() {
		return fmt.Errorf("command %q not found", cmd)
	}

	res := meth.Call(nil)

	if len(res) != 1 {
		return fmt.Errorf("programmer error: expected 1 return value (of type error) from %q but got %d instead", name, len(res))
	}
	r := res[0]
	if !r.IsValid() || !r.CanInterface() {
		return fmt.Errorf("programmer error: expected valid return value (of type error) from %q but got %v instead", name, r)
	}
	i := r.Interface()
	err, ok := i.(error)
	if !ok && i != nil { // if i is nil, then it won't be an error, even if it returns one
		return fmt.Errorf("programmer error: expected return value of type error from %q but got value '%v' of type %T instead", name, i, i)
	}
	if err != nil {
		return err
	}
	return nil
}
