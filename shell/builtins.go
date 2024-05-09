// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/sshclient"
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
// * close: closes the connection; start after this will re-open.
func (sh *Shell) CoSSH(args ...string) error {
	if len(args) < 1 {
		return fmt.Errorf("cossh: requires at least one argument")
	}
	cmd := args[0]
	var err error
	host := ""
	name := fmt.Sprintf("%d", 1+len(sh.SSHClients))
	con := false
	switch {
	case cmd == "close":
		sh.CloseSSH()
		return nil
	case cmd == "@" && len(args) == 2:
		name = args[1]
	case len(args) == 2:
		con = true
		host = args[0]
		name = args[1]
	default:
		con = true
		host = args[0]
	}
	if con {
		cl := sshclient.NewClient(sh.SSH)
		err = cl.Connect(host)
		if err != nil {
			return err
		}
		sh.SSHClients[name] = cl
		sh.SSHActive = name
	} else {
		if name == "0" {
			sh.SSHActive = ""
		} else {
			sh.SSHActive = name
			cl := sh.ActiveSSH()
			if cl == nil {
				err = fmt.Errorf("cosh: ssh connection named: %q not found", name)
			}
		}
	}
	return err
}

// example alias:
// shell list(args ...string) {
// 	ls -la {args...}
// }
