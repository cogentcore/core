// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

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
	"cogentcore.org/core/core"
	"cogentcore.org/core/system"
	"github.com/mitchellh/go-homedir"
)

// InstallBuiltins adds the builtin goal commands to [Goal.Builtins].
func (gl *Goal) InstallBuiltins() {
	gl.Builtins = make(map[string]func(cmdIO *exec.CmdIO, args ...string) error)
	gl.Builtins["cd"] = gl.Cd
	gl.Builtins["exit"] = gl.Exit
	gl.Builtins["jobs"] = gl.JobsCmd
	gl.Builtins["kill"] = gl.Kill
	gl.Builtins["set"] = gl.Set
	gl.Builtins["unset"] = gl.Unset
	gl.Builtins["add-path"] = gl.AddPath
	gl.Builtins["which"] = gl.Which
	gl.Builtins["source"] = gl.Source
	gl.Builtins["cossh"] = gl.CoSSH
	gl.Builtins["scp"] = gl.Scp
	gl.Builtins["debug"] = gl.Debug
	gl.Builtins["history"] = gl.History
}

// Cd changes the current directory.
func (gl *Goal) Cd(cmdIO *exec.CmdIO, args ...string) error {
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
	gl.Config.Dir = dir
	return nil
}

// Exit exits the shell.
func (gl *Goal) Exit(cmdIO *exec.CmdIO, args ...string) error {
	os.Exit(0)
	return nil
}

// Set sets the given environment variable to the given value.
func (gl *Goal) Set(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected two arguments, got %d", len(args))
	}
	err := os.Setenv(args[0], args[1])
	if core.TheApp.Platform() == system.MacOS {
		gl.Config.RunIO(cmdIO, "/bin/launchctl", "setenv", args[0], args[1])
	}
	return err
}

// Unset un-sets the given environment variable.
func (gl *Goal) Unset(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected one argument, got %d", len(args))
	}
	err := os.Unsetenv(args[0])
	if core.TheApp.Platform() == system.MacOS {
		gl.Config.RunIO(cmdIO, "/bin/launchctl", "unsetenv", args[0])
	}
	return err
}

// JobsCmd is the builtin jobs command
func (gl *Goal) JobsCmd(cmdIO *exec.CmdIO, args ...string) error {
	for i, jb := range gl.Jobs {
		cmdIO.Printf("[%d]  %s\n", i+1, jb.String())
	}
	return nil
}

// Kill kills a job by job number or PID.
// Just expands the job id expressions %n into PIDs and calls system kill.
func (gl *Goal) Kill(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("goal kill: expected at least one argument")
	}
	gl.JobIDExpand(args)
	gl.Config.RunIO(cmdIO, "kill", args...)
	return nil
}

// Fg foregrounds a job by job number
func (gl *Goal) Fg(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("goal fg: requires exactly one job id argument")
	}
	jid := args[0]
	exp := gl.JobIDExpand(args)
	if exp != 1 {
		return fmt.Errorf("goal fg: argument was not a job id in the form %%n")
	}
	jno, _ := strconv.Atoi(jid[1:]) // guaranteed good
	job := gl.Jobs[jno]
	cmdIO.Printf("foregrounding job [%d]\n", jno)
	_ = job
	// todo: the problem here is we need to change the stdio for running job
	// job.Cmd.Wait() // wait
	// * probably need to have wrapper StdIO for every exec so we can flexibly redirect for fg, bg commands.
	// * likewise, need to run everything effectively as a bg job with our own explicit Wait, which we can then communicate with to move from fg to bg.

	return nil
}

// AddPath adds the given path(s) to $PATH.
func (gl *Goal) AddPath(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("goal add-path expected at least one argument")
	}
	path := os.Getenv("PATH")
	for _, arg := range args {
		arg, err := homedir.Expand(arg)
		if err != nil {
			return err
		}
		path = path + ":" + arg
	}
	err := os.Setenv("PATH", path)
	if core.TheApp.Platform() == system.MacOS {
		gl.Config.RunIO(cmdIO, "/bin/launchctl", "setenv", "PATH", path)
	}
	return err
}

// Which reports the executable associated with the given command.
// Processes builtins and commands, and if not found, then passes on
// to exec which.
func (gl *Goal) Which(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) != 1 {
		return fmt.Errorf("goal which: requires one argument")
	}
	cmd := args[0]
	if _, hasCmd := gl.Commands[cmd]; hasCmd {
		cmdIO.Println(cmd, "is a user-defined command")
		return nil
	}
	if _, hasBlt := gl.Builtins[cmd]; hasBlt {
		cmdIO.Println(cmd, "is a goal builtin command")
		return nil
	}
	gl.Config.RunIO(cmdIO, "which", args...)
	return nil
}

// Source loads and evaluates the given file(s)
func (gl *Goal) Source(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("goal source: requires at least one argument")
	}
	for _, fn := range args {
		gl.TranspileCodeFromFile(fn)
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
func (gl *Goal) CoSSH(cmdIO *exec.CmdIO, args ...string) error {
	if len(args) < 1 {
		return fmt.Errorf("cossh: requires at least one argument")
	}
	cmd := args[0]
	var err error
	host := ""
	name := fmt.Sprintf("%d", 1+len(gl.SSHClients))
	con := false
	switch {
	case cmd == "close":
		gl.CloseSSH()
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
		cl := sshclient.NewClient(gl.SSH)
		err = cl.Connect(host)
		if err != nil {
			return err
		}
		gl.SSHClients[name] = cl
		gl.SSHActive = name
	} else {
		if name == "0" {
			gl.SSHActive = ""
		} else {
			gl.SSHActive = name
			cl := gl.ActiveSSH()
			if cl == nil {
				err = fmt.Errorf("goal: ssh connection named: %q not found", name)
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
func (gl *Goal) Scp(cmdIO *exec.CmdIO, args ...string) error {
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

	cl, err := gl.SSHByHost(host)
	if err != nil {
		return err
	}

	ctx := gl.Ctx
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
func (gl *Goal) Debug(cmdIO *exec.CmdIO, args ...string) error {
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
func (gl *Goal) History(cmdIO *exec.CmdIO, args ...string) error {
	n := len(gl.Hist)
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
		cmdIO.Printf("%d:\t%s\n", i, gl.Hist[i])
	}
	return nil
}
