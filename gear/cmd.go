// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gear provides the generation of GUIs and interactive CLIs for any existing command line tools.
package gear

import (
	"cogentcore.org/core/glop/sentence"
	"github.com/iancoleman/strcase"
)

// Cmd contains all of the data for a parsed command line command.
type Cmd struct {
	// Cmd is the actual name of the command (eg: "git", "go build")
	Cmd string
	// Name is the formatted name of the command (eg: "Git", "Go build")
	Name string
	// Doc is the documentation for the command (eg: "compile packages and dependencies")
	Doc string
	// Flags contains the flags for the command
	Flags []*Flag
	// Cmds contains the subcommands of the command
	Cmds []*Cmd
}

// NewCmd makes a new [App] object from the given command name.
// It does not parse it; see [App.Parse].
func NewCmd(cmd string) *Cmd {
	return &Cmd{
		Cmd:  cmd,
		Name: sentence.Case(strcase.ToCamel(cmd)),
	}
}

// Flag contains the information for a parsed command line flag.
type Flag struct {
	// Name is the canonical (longest) name of the flag.
	// It includes the leading dashes of the flag.
	Name string
	// Names are the different names the flag can go by.
	// They include the leading dashes of the flag.
	Names []string
	// Type is the type or value hint for the flag.
	Type string
	// Doc is the documentation for the flag.
	Doc string
}
