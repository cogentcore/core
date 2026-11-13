// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

//go:generate core generate

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"cogentcore.org/core/base/fileinfo"
)

// Script is a script written by the user, for things that are more than
// a command line: its code starts with a #! line of the program that runs
// it, such as #!/bin/bash, #!/usr/bin/env python3, or #!/usr/bin/env goal
// (the shell, /bin/sh, if there is none). Its arguments are prompted for
// when it runs, and passed to it as $1, $2, and so on. The scripts of a
// command, such as git, are in its menu after its macros (see
// [Cmd.Scripts]), and others are entries of the commands (see
// [Entry.Script]).
type Script struct {

	// Name is the name of the script, which is shown for it.
	Name string

	// Doc is the documentation of the script, which is its tooltip.
	Doc string `json:",omitempty"`

	// Confirm is whether the script is confirmed before it runs, if it has
	// no arguments to prompt for, as for scripts that cannot be undone.
	Confirm bool `json:",omitempty"`

	// Code is the code of the script, starting with its #! line.
	Code string `display:"-"`

	// Args are the arguments of the script, which are prompted for when it
	// runs, in their order: $1, $2, and so on.
	Args []ScriptArg `json:",omitempty" display:"-"`
}

// ScriptArg is an argument of a [Script], which is prompted for when it runs.
type ScriptArg struct {

	// Name is the name of the argument, the label of its prompt.
	Name string

	// Kind is the kind of value of the argument.
	Kind ArgKinds

	// Doc is the documentation of the argument, the tooltip of its prompt.
	Doc string `json:",omitempty" width:"40"`

	// Default is the value of the argument when it is first prompted for
	// (after that, the last value used): for a switch, "true" if it is on.
	Default string `json:",omitempty"`

	// On is the value passed for a switch that is on, such as --force, or
	// "true" if it is empty; an empty string is passed if it is off.
	On string `json:",omitempty"`
}

// ArgKinds are the kinds of values of the arguments of a [Script].
type ArgKinds int32 //enums:enum -trim-prefix Arg

const (
	// ArgString is text, such as a name or a message.
	ArgString ArgKinds = iota

	// ArgSwitch is a switch that is on or off (see [ScriptArg.On]).
	ArgSwitch

	// ArgFile is the path of a file or directory, which can be chosen.
	ArgFile
)

// clone returns a copy of the script, with its own arguments.
func (s *Script) clone() *Script {
	c := *s
	c.Args = slices.Clone(s.Args)
	return &c
}

// interpreter returns the program of the #! line of the code of the
// script, with its arguments, such as ["/usr/bin/env", "python3"], or nil
// if there is none.
func (s *Script) interpreter() []string {
	line, _, _ := strings.Cut(s.Code, "\n")
	if !strings.HasPrefix(line, "#!") {
		return nil
	}
	return strings.Fields(line[2:])
}

// Language returns the language of the script, from its #! line.
func (s *Script) Language() fileinfo.Known {
	in := s.interpreter()
	if len(in) == 0 {
		return fileinfo.Bash
	}
	prog := filepath.Base(in[0])
	if prog == "env" && len(in) > 1 {
		prog = in[len(in)-1]
	}
	prog = strings.TrimRightFunc(prog, func(r rune) bool { return unicode.IsDigit(r) || r == '.' })
	switch prog {
	case "sh", "bash", "zsh", "dash", "ksh":
		return fileinfo.Bash
	case "goal":
		return fileinfo.Goal
	case "python":
		return fileinfo.Python
	case "perl":
		return fileinfo.Perl
	case "ruby":
		return fileinfo.Ruby
	}
	return fileinfo.Unknown
}

// argValues returns the values passed to the script for the given values
// of its arguments, by name (see [Commands.ScriptValues]): the value of a
// switch that is on is its [ScriptArg.On] value, and empty if it is off.
func (s *Script) argValues(values map[string]string) []string {
	var vals []string
	for _, a := range s.Args {
		v := values[a.Name]
		if a.Kind == ArgSwitch {
			v = ""
			if values[a.Name] == "true" {
				v = cmpOr(a.On, "true")
			}
		}
		vals = append(vals, v)
	}
	return vals
}

// cmpOr returns a if it is not empty, and otherwise b.
func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// CommandLine returns the command line that the script stands for with
// the given values of its arguments, by name, as shown to the user: its
// name and the values of its arguments, quoted.
func (s *Script) CommandLine(values map[string]string) string {
	words := []string{shellQuote(s.Name)}
	for _, v := range s.argValues(values) {
		words = append(words, shellQuote(v))
	}
	return strings.Join(words, " ")
}

// fileName returns the given name as the name of a file: its letters,
// digits, dots, and dashes, with dashes in place of the others.
func fileName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, strings.TrimSpace(name))
	return cmpOr(strings.Trim(name, ".-"), "script")
}

// scriptsDir returns the directory of the files of the scripts, next to
// the file of the commands.
func (cs *Commands) scriptsDir() string {
	dir := os.TempDir()
	if cs.file.Path != "" {
		dir = filepath.Dir(cs.file.Path)
	}
	return filepath.Join(dir, "scripts")
}

// ScriptCommandLine writes the file of the given script of the given
// command ("" for one of an entry), and returns the command line that runs
// it with the given values of its arguments, by name. The arguments are in
// a launcher, a shell script that runs the script with them, so that the
// command line is the same in any shell, such as Goal, which cannot pass
// empty arguments: /bin/sh and the path of the launcher. On Windows, it is
// the program of the #! line and the path of the script, with the
// arguments.
func (cs *Commands) ScriptCommandLine(root string, s *Script, values map[string]string) (string, error) {
	dir := filepath.Join(cs.scriptsDir(), fileName(cmpOr(root, "commands")))
	name := fileName(s.Name)
	file := filepath.Join(dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	code := s.Code
	if !strings.HasSuffix(code, "\n") {
		code += "\n"
	}
	if err := os.WriteFile(file, []byte(code), 0o755); err != nil {
		return "", err
	}
	if err := os.Chmod(file, 0o755); err != nil { // (if it existed)
		return "", err
	}
	in := s.interpreter()
	args := s.argValues(values)
	if runtime.GOOS == "windows" {
		prog := "sh"
		if len(in) > 0 {
			prog = filepath.Base(in[0])
			if prog == "env" && len(in) > 1 {
				prog = in[1]
			}
		}
		words := []string{prog, shellQuote(file)}
		for _, a := range args {
			words = append(words, shellQuote(a))
		}
		return strings.Join(words, " "), nil
	}
	words := []string{"exec", shellQuote(file)}
	if len(in) == 0 {
		words = []string{"exec", "/bin/sh", shellQuote(file)}
	}
	for _, a := range args {
		words = append(words, shellQuote(a))
	}
	rdir := filepath.Join(cs.scriptsDir(), "run", fileName(cmpOr(root, "commands")))
	if err := os.MkdirAll(rdir, 0o755); err != nil {
		return "", err
	}
	launcher := filepath.Join(rdir, name)
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\n"+strings.Join(words, " ")+"\n"), 0o755); err != nil {
		return "", err
	}
	return "/bin/sh " + shellQuote(launcher), nil
}
