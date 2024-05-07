// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

// InstallBuiltins adds the builtin shell commands to [Shell.Builtins].
func (sh *Shell) InstallBuiltins() {
	sh.Builtins = make(map[string]func(args ...string) error)
	sh.Builtins["cd"] = sh.Cd
	sh.Builtins["exit"] = sh.Exit
	sh.Builtins["set"] = sh.Set
	sh.Builtins["add-path"] = sh.AddPath
}

// Cd changes the current directory.
func (sh *Shell) Cd(args ...string) error {
	if len(args) > 1 {
		return fmt.Errorf("no more than one argument can be passed to cd")
	}
	dir := ""
	if len(args) == 1 {
		dir = args[0]
	}
	dir, err := homedir.Expand(dir)
	if err != nil {
		return err
	}
	if dir == "" {
		dir, err = homedir.Dir()
		if err != nil {
			return err
		}
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	err = os.Chdir(dir)
	if err != nil {
		return err
	}
	sh.Config.Dir = dir
	return nil
}

// Exit exits the shell.
func (sh *Shell) Exit(args ...string) error {
	os.Exit(0)
	return nil
}

// Set sets the given environment variable to the given value.
func (sh *Shell) Set(args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected one argument")
	}
	key, value, has := strings.Cut(args[0], "=")
	if !has {
		return fmt.Errorf("expected equal sign (=)")
	}
	return os.Setenv(key, value)
}

// AddPath adds the given path(s) to $PATH.
func (sh *Shell) AddPath(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("expected at least one argument")
	}
	path := os.Getenv("PATH")
	for _, arg := range args {
		arg, err := homedir.Expand(arg)
		if err != nil {
			return err
		}
		path = path + ":" + arg
	}
	return os.Setenv("PATH", path)
}

// example alias:
// shell list(args ...string) {
// 	ls -la {args...}
// }
