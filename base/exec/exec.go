// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted in part from: https://github.com/magefile/mage
// Copyright presumably by Nate Finch, primary contributor
// Apache License, Version 2.0, January 2004

package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"cogentcore.org/core/base/logx"
)

// exec executes the command, piping its stdout and stderr to the config
// writers. If start, uses cmd.Start, else.Run.
// If the command fails, it will return an error with the command output.
// The given cmd and args may include references
// to environment variables in $FOO format, in which case these will be
// expanded before the command is run.
//
// Ran reports if the command ran (rather than was not found or not executable).
// Code reports the exit code the command returned if it ran. If err == nil, ran
// is always true and code is always 0.
//
// If ctx is not nil, a command that is run (not started) is interrupted
// when it is done (see [Config.RunIOContext]).
func (c *Config) exec(ctx context.Context, sio *StdIO, start bool, cmd string, args ...string) (excmd *exec.Cmd, ran bool, err error) {
	expand := func(s string) string {
		s2, ok := c.Env[s]
		if ok {
			return s2
		}
		return os.Getenv(s)
	}
	cmd = os.Expand(cmd, expand)
	for i := range args {
		args[i] = os.Expand(args[i], expand)
	}
	excmd, ran, code, err := c.run(ctx, sio, start, cmd, args...)
	_ = code
	if err == nil {
		return excmd, true, nil
	}
	return excmd, ran, runError(cmd, args, err)
}

// RunError is an error of running a command, which wraps the error of
// running it, such as [exec.ErrNotFound] or an [exec.ExitError].
type RunError struct {
	// Msg is the message of the error.
	Msg string

	// Err is the error of running the command.
	Err error
}

func (e *RunError) Error() string { return e.Msg }

func (e *RunError) Unwrap() error { return e.Err }

// runError returns a [RunError] for the given error of running the given
// command with the given args: one that it was not found, as in a shell,
// or that it failed.
func runError(cmd string, args []string, err error) error {
	if errors.Is(err, exec.ErrNotFound) {
		return &RunError{Msg: cmd + ": command not found", Err: err}
	}
	return &RunError{Msg: fmt.Sprintf("%q: %v", strings.Join(append([]string{cmd}, args...), " "), err), Err: err}
}

// InterruptWait is how long a command that is interrupted, when the
// context it is run with is done, has to end, before it is killed.
var InterruptWait = time.Second

func (c *Config) run(ctx context.Context, sio *StdIO, start bool, cmd string, args ...string) (excmd *exec.Cmd, ran bool, code int, err error) {
	var cm *exec.Cmd
	if ctx != nil && !start && interruptible() {
		cm = exec.CommandContext(ctx, cmd, args...)
		cm.Cancel = func() error { return interrupt(cm.Process) }
		cm.WaitDelay = InterruptWait
	} else {
		cm = exec.Command(cmd, args...)
	}
	cm.Env = os.Environ()
	for k, v := range c.Env {
		cm.Env = append(cm.Env, k+"="+v)
	}
	// need to store in buffer so we can color and print commands and stdout correctly
	// (need to declare regardless even if we aren't using so that it is accessible)
	ebuf := &bytes.Buffer{}
	obuf := &bytes.Buffer{}
	if !start && c.Buffer {
		cm.Stderr = ebuf
		cm.Stdout = obuf
	} else {
		cm.Stderr = sio.Err
		cm.Stdout = sio.Out
	}
	// need to do now because we aren't buffering, or we are guaranteed to print them
	// regardless of whether there is an error anyway, so we should print it now so
	// people can see it earlier (especially important if it runs for a long time).
	if !start || !c.Buffer || c.Echo != nil {
		c.PrintCmd(cmd+" "+strings.Join(args, " "), err)
	}

	cm.Stdin = sio.In
	cm.Dir = c.Dir

	if !c.PrintOnly {
		if start {
			err = cm.Start()
			excmd = cm
		} else {
			restore := saveConsole(sio.In, sio.Out)
			err = cm.Run()
			restore()
		}

		// we must call InitColor after calling a system command
		// TODO(kai): maybe figure out a better solution to this
		// or expand this list
		if cmd == "cp" || cmd == "ls" || cmd == "mv" {
			logx.InitColor()
		}
	}

	if !start && c.Buffer {
		// if we have an error, we print the commands and stdout regardless of the config info
		// (however, we don't print the command if we are guaranteed to print it regardless, as
		// we already printed it above in that case)
		if c.Echo == nil {
			c.PrintCmd(cmd+" "+strings.Join(args, " "), err)
		}
		sout := c.GetWriter(sio.Out, err)
		if sout != nil {
			sout.Write(obuf.Bytes())
		}
		estr := ebuf.String()
		if estr != "" && sio.Err != nil {
			sio.Err.Write([]byte(logx.ErrorColor(estr)))
		}
	}
	return excmd, CmdRan(err), ExitStatus(err), err
}

// CmdRan examines the error to determine if it was generated as a result of a
// command running via os/exec.Command.  If the error is nil, or the command ran
// (even if it exited with a non-zero exit code), CmdRan reports true.  If the
// error is an unrecognized type, or it is an error from exec.Command that says
// the command failed to run (usually due to the command not existing or not
// being executable), it reports false.
func CmdRan(err error) bool {
	if err == nil {
		return true
	}
	ee, ok := err.(*exec.ExitError)
	if ok {
		return ee.Exited()
	}
	return false
}

type exitStatus interface {
	ExitStatus() int
}

// ExitStatus returns the exit status of the error if it is an exec.ExitError
// or if it implements ExitStatus() int.
// 0 if it is nil or 1 if it is a different error.
func ExitStatus(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(exitStatus); ok {
		return e.ExitStatus()
	}
	if e, ok := err.(*exec.ExitError); ok {
		if ex, ok := e.Sys().(exitStatus); ok {
			return ex.ExitStatus()
		}
	}
	return 1
}
