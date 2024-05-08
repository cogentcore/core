// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

// InstallBuiltins adds the builtin shell commands to [Shell.Builtins].
func (sh *Shell) InstallBuiltins() {
	sh.Builtins = make(map[string]func(args ...string) error)
	sh.Builtins["cd"] = sh.Cd
	sh.Builtins["exit"] = sh.Exit
	sh.Builtins["set"] = sh.Set
	sh.Builtins["add-path"] = sh.AddPath
	sh.Builtins["cossh"] = sh.CoSSH
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
	if len(args) != 2 {
		return fmt.Errorf("expected two arguments, got %d", len(args))
	}
	return os.Setenv(args[0], args[1])
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

// CoSSH does connects to a server specified in first arg, which is then
// used for executing any shell commands until called with `stop` or `close`.
// Should call 'close' when no longer needed.
// * stop:  goes back to running locally.
// * start: If a previous connection has been established, resumes running remotely.
// * close: closes the connection; start after this will re-open.
func (sh *Shell) CoSSH(args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("cossh: requires one argument")
	}
	cmd := args[0]
	host := ""
	switch cmd {
	case "stop":
		sh.SSHActive = false
		return nil
	case "close":
		sh.SSHActive = false
		sh.SSH.Close()
		return nil
	case "start":
		if sh.SSH.Client != nil { // already running
			sh.SSHActive = true
			return nil
		}
		if sh.SSH.Host != "" {
			host = sh.SSH.Host
		} else {
			return fmt.Errorf("cossh: start can only be called if a host name was previously specified")
		}
	default:
		host = args[0]
	}
	err := sh.SSH.Connect(host)
	if err != nil {
		return err
	}
	// sh.SSH.Stdin = sh.Config.Stdin // this causes it to hang!  do not set.
	sh.SSH.Stdout = sh.Config.Stdout
	sh.SSH.Stderr = sh.Config.Stderr
	sh.SSHActive = true
	return nil
}

// example alias:
// shell list(args ...string) {
// 	ls -la {args...}
// }
