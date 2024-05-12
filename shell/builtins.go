// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/sshclient"
	"github.com/mitchellh/go-homedir"
)

// InstallBuiltins adds the builtin shell commands to [Shell.Builtins].
func (sh *Shell) InstallBuiltins() {
	sh.Builtins = make(map[string]func(cmdIO *exec.CmdIO, args ...string) error)
	sh.Builtins["cd"] = sh.Cd
	sh.Builtins["exit"] = sh.Exit
	sh.Builtins["jobs"] = sh.JobsCmd
	sh.Builtins["kill"] = sh.Kill
	sh.Builtins["set"] = sh.Set
	sh.Builtins["add-path"] = sh.AddPath
	sh.Builtins["cossh"] = sh.CoSSH
}

// Cd changes the current directory.
func (sh *Shell) Cd(cmdIO *exec.CmdIO, args ...string) error {
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
func (sh *Shell) Exit(cmdIO *exec.CmdIO, args ...string) error {
	os.Exit(0)
	return nil
}

// Set sets the given environment variable to the given value.
func (sh *Shell) Set(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected two arguments, got %d", len(args))
	}
	return os.Setenv(args[0], args[1])
}

// JobsCmd is the builtin jobs command
func (sh *Shell) JobsCmd(cmdIO *exec.CmdIO, args ...string) error {
	for i, jb := range sh.Jobs {
		fmt.Fprintf(cmdIO.Out, "[%d]  %s\n", i+1, jb.String())
	}
	return nil
}

// Kill kills a job by job number or PID
func (sh *Shell) Kill(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("cosh kill expected at least one argument")
	}
	for _, id := range args {
		if id[0] == '%' {
			idx, err := strconv.Atoi(id[1:])
			if err == nil {
				if idx > 0 && idx <= len(sh.Jobs) {
					jb := sh.Jobs[idx-1]
					if jb.Cmd != nil && jb.Cmd.Process != nil {
						jb.Cmd.Process.Kill()
					}
				} else {
					sh.AddError(fmt.Errorf("cosh kill: job number out of range: %d", idx))
				}
			}
		}
	}
	return nil
}

// AddPath adds the given path(s) to $PATH.
func (sh *Shell) AddPath(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("cosh add-path expected at least one argument")
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

// CoSSH manages SSH connections, which are referenced by the @name
// identifier.  It handles the following cases:
//   - @name -- switches to using given host for all subsequent commands
//   - host [name] -- connects to a server specified in first arg and switches
//     to using it, with optional name instead of default sequential number.
//   - close -- closes all open connections, or the specified one
func (sh *Shell) CoSSH(cmdIO *exec.CmdIO, args ...string) error {
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
