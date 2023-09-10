// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
	"reflect"

	"github.com/iancoleman/strcase"
)

var (
	// AppName is the internal name of the Grease app
	// (typically in kebab-case) (see also [AppTitle])
	AppName string = "grease"

	// AppTitle is the user-visible name of the Grease app
	// (typically in Title Case) (see also [AppName])
	AppTitle string = "Grease"

	// AppAbout is the description of the Grease app
	AppAbout string = "Grease allows you to edit configuration information and run commands through a CLI and a GUI interface."
)

// Run runs the given app with the given default
// configuration file paths. It does not run the
// GUI; see [goki.dev/greasi.Run] for that. The app should be
// a pointer, and configuration options should
// be defined as fields on the app type. Also,
// commands should be defined as methods on the
// app type with the suffix "Cmd"; for example,
// for a command named "build", there should be
// the method:
//
//	func (a *App) BuildCmd() error
//
// If no command is provided, Run calls the
// method "RootCmd" if it exists. If it does
// not exist, or the command provided is "help"
// and "HelpCmd" does not exist, Run prints
// the result of [Usage].
// Run uses [os.Args] for its arguments.
func Run(app, cfg any) error {
	leftovers, err := Config(cfg)
	if err != nil {
		return fmt.Errorf("error configuring app: %w", err)
	}
	// root command if no other command is specified
	cmd := ""
	if len(leftovers) > 0 {
		cmd = leftovers[0]
	}
	err = RunCmd(app, cfg, cmd)
	if err != nil {
		return fmt.Errorf("error running command %q: %w", cmd, err)
	}
	return nil
}

// RunCmd runs the command with the given
// name on the given app. It looks for the
// method with the name of the command converted
// to camel case suffixed with "Cmd"; for example,
// for a command named "build", it will look for a
// method named "BuildCmd".
func RunCmd(app, cfg any, cmd string) error {
	name := strcase.ToCamel(cmd) + "Cmd"
	val := reflect.ValueOf(app)
	meth := val.MethodByName(name)
	if !meth.IsValid() {
		if cmd == "" || cmd == "help" { // handle root and help here so that people can still override them if they want to
			fmt.Println(Usage(app))
			os.Exit(0)
		}
		return fmt.Errorf("command %q not found", cmd)
	}

	res := meth.Call([]reflect.Value{reflect.ValueOf(cfg)})

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
