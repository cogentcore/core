// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
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
	sh.Builtins["which"] = sh.Which
	sh.Builtins["source"] = sh.Source
	sh.Builtins["cossh"] = sh.CoSSH
	sh.Builtins["scp"] = sh.Scp
	sh.Builtins["debug"] = sh.Debug
	sh.Builtins["history"] = sh.History
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
		cmdIO.Printf("[%d]  %s\n", i+1, jb.String())
	}
	return nil
}

// Kill kills a job by job number or PID.
// Just expands the job id expressions %n into PIDs and calls system kill.
func (sh *Shell) Kill(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("cosh kill: expected at least one argument")
	}
	sh.JobIDExpand(args)
	sh.Config.RunIO(cmdIO, "kill", args...)
	return nil
}

// Fg foregrounds a job by job number
func (sh *Shell) Fg(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("cosh fg: requires exactly one job id argument")
	}
	jid := args[0]
	exp := sh.JobIDExpand(args)
	if exp != 1 {
		return fmt.Errorf("cosh fg: argument was not a job id in the form %%n")
	}
	jno, _ := strconv.Atoi(jid[1:]) // guaranteed good
	job := sh.Jobs[jno]
	cmdIO.Printf("foregrounding job [%d]\n", jno)
	_ = job
	// todo: the problem here is we need to change the stdio for running job
	// job.Cmd.Wait() // wait
	// * probably need to have wrapper StdIO for every exec so we can flexibly redirect for fg, bg commands.
	// * likewise, need to run everything effectively as a bg job with our own explicit Wait, which we can then communicate with to move from fg to bg.

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

// Which reports the executable associated with the given command.
// Processes builtins and commands, and if not found, then passes on
// to exec which.
func (sh *Shell) Which(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("cosh which: requires one argument")
	}
	cmd := args[0]
	if _, hasCmd := sh.Commands[cmd]; hasCmd {
		cmdIO.Println(cmd, "is a user-defined command")
		return nil
	}
	if _, hasBlt := sh.Builtins[cmd]; hasBlt {
		cmdIO.Println(cmd, "is a cosh builtin command")
		return nil
	}
	sh.Config.RunIO(cmdIO, "which", args...)
	return nil
}

// Source loads and evaluates the given file(s)
func (sh *Shell) Source(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("cosh source: requires at least one argument")
	}
	for _, fn := range args {
		sh.TranspileCodeFromFile(fn)
	}
	// note that we do not execute the file -- just loads it in
	return nil
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

// Scp performs file copy over SSH connection, with the remote filename
// prefixed with the @name: and the local filename un-prefixed.
// The order is from -> to, as in standard cp.
// The remote filename is automatically relative to the current working
// directory on the remote host.
func (sh *Shell) Scp(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 2 {
		return fmt.Errorf("scp: requires exactly two arguments")
	}
	var lfn, hfn string
	toHost := false
	if args[0][0] == '@' {
		hfn = args[0]
		lfn = args[1]
	} else if args[1][0] == '@' {
		hfn = args[1]
		lfn = args[0]
		toHost = true
	} else {
		return fmt.Errorf("scp: one of the files must a remote host filename, specified by @name:")
	}

	ci := strings.Index(hfn, ":")
	if ci < 0 {
		return fmt.Errorf("scp: remote host filename does not contain a : after the host name")
	}
	host := hfn[1:ci]
	hfn = hfn[ci+1:]

	cl, err := sh.SSHByHost(host)
	if err != nil {
		return err
	}

	ctx := sh.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if toHost {
		err = cl.CopyLocalFileToHost(ctx, lfn, hfn)
	} else {
		err = cl.CopyHostToLocalFile(ctx, hfn, lfn)
	}
	return err
}

// Debug changes log level
func (sh *Shell) Debug(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		if logx.UserLevel == slog.LevelDebug {
			logx.UserLevel = slog.LevelInfo
		} else {
			logx.UserLevel = slog.LevelDebug
		}
	}
	if len(args) == 1 {
		lev := args[0]
		if lev == "on" || lev == "true" || lev == "1" {
			logx.UserLevel = slog.LevelDebug
		} else {
			logx.UserLevel = slog.LevelInfo
		}
	}
	return nil
}

// History shows history
func (sh *Shell) History(cmdIO *exec.CmdIO, args ...string) error {
	n := len(sh.Hist)
	nh := n
	if len(args) == 1 {
		an, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("history: error parsing number of history items: %q, error: %s", args[0], err.Error())
		}
		nh = min(n, an)
	} else if len(args) > 1 {
		return fmt.Errorf("history: uses at most one argument")
	}
	for i := n - nh; i < n; i++ {
		cmdIO.Printf("%d:\t%s\n", i, sh.Hist[i])
	}
	return nil
}
