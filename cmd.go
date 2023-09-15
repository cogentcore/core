// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/gti"
)

// Cmd represents a runnable command with configuration options
// that can be passed to [grease.Run] and/or [grease.RunCmd].
// The type constraint is the type of the configuration
// information passed to the command.
type Cmd[T any] struct {
	// Func is the actual function that runs the command.
	// It takes configuration information and returns an error.
	Func func(T) error
	// Name is the name of the command.
	Name string
	// Doc is the documentation for the command.
	Doc string
	// Root is whether the command is the root command
	// (what is called when no subcommands are passed)
	Root bool
	// Icon is the icon of the command in the tool bar
	// when running in the GUI via greasi
	Icon string
	// SepBefore is whether to add a separator before the
	// command in the tool bar when running in the GUI via greasi
	SepBefore bool
	// SepAfter is whether to add a separator after the
	// command in the tool bar when running in the GUI via greasi
	SepAfter bool
}

// CmdOrFunc is a generic type constraint that represents either
// a [Cmd] with the given config type or a command function that
// takes the given config type and returns an error.
type CmdOrFunc[T any] interface {
	*Cmd[T] | func(T) error
}

// CmdFromFunc returns a new [Cmd] object from the given function
// and any information specified on it using comment directives,
// which requires the use of gti.
func CmdFromFunc[T any](fun func(T) error) (*Cmd[T], error) {
	cmd := &Cmd[T]{
		Func: fun,
		Name: gti.FuncName(fun),
	}
	if f := gti.FuncByName(cmd.Name); f != nil {
		cmd.Doc = f.Doc
		for _, dir := range f.Directives {
			if dir.Tool != "grease" {
				continue
			}
			if dir.Directive != "cmd" {
				return cmd, fmt.Errorf("unrecognized comment directive %q (from comment %q)", dir.Directive, dir.String())
			}
			leftovers, err := SetFromArgs(cmd, dir.Args)
			if err != nil {
				return cmd, fmt.Errorf("error setting command from directive arguments (from comment %q): %w", dir.String(), err)
			}
			if len(leftovers) != 0 {
				return cmd, fmt.Errorf("expected no leftover arguments, but got %d", len(leftovers))
			}
		}
	}
	if strings.Contains(cmd.Name, ".") {
		strs := strings.Split(cmd.Name, ".")
		cmd.Name = strs[len(strs)-1]
	}
	cmd.Name = strcase.ToKebab(cmd.Name)
	return cmd, nil
}

// CmdFromCmdOrFunc returns a new [Cmd] object from the given
// [CmdOrFunc] object, using [CmdFromFunc] if it is a function.
func CmdFromCmdOrFunc[T any, C CmdOrFunc[T]](cmd C) (*Cmd[T], error) {
	switch c := any(cmd).(type) {
	case *Cmd[T]:
		return c, nil
	case func(T) error:
		return CmdFromFunc(c)
	default:
		panic(fmt.Errorf("internal/programmer error: grease.CmdFromCmdOrFunc: impossible type %T for command %v", cmd, cmd))
	}
}

// CmdsFromFuncs is a helper function that returns a slice
// of command objects from the given slice of command functions,
// using [CmdFromFunc].
func CmdsFromFuncs[T any](funcs []func(T) error) ([]*Cmd[T], error) {
	res := make([]*Cmd[T], len(funcs))
	for i, fun := range funcs {
		cmd, err := CmdFromFunc(fun)
		if err != nil {
			return nil, err
		}
		res[i] = cmd
	}
	return res, nil
}

// CmdsFromFuncs is a helper function that returns a slice
// of command objects from the given slice of [CmdOrFunc] objects,
// using [CmdFromFuncOrFunc].
func CmdsFromCmdOrFuncs[T any, C CmdOrFunc[T]](cmds []C) ([]*Cmd[T], error) {
	res := make([]*Cmd[T], len(cmds))
	for i, cmd := range cmds {
		cmd, err := CmdFromCmdOrFunc[T, C](cmd)
		if err != nil {
			return nil, err
		}
		res[i] = cmd
	}
	return res, nil
}
