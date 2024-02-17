// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted in part from: https://github.com/magefile/mage
// Copyright presumably by Nate Finch, primary contributor
// Apache License, Version 2.0, January 2004

package xe

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"cogentcore.org/core/grog"
)

// Exec executes the command, piping its stdout and stderr to the config
// writers. If the command fails, it will return an error with the command output.
// The given cmd and args may include references
// to environment variables in $FOO format, in which case these will be
// expanded before the command is run.
//
// Ran reports if the command ran (rather than was not found or not executable).
// Code reports the exit code the command returned if it ran. If err == nil, ran
// is always true and code is always 0.
func (c *Config) Exec(cmd string, args ...string) (ran bool, err error) {
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
	ran, code, err := c.run(cmd, args...)
	_ = code
	if err == nil {
		return true, nil
	}
	return ran, fmt.Errorf(`failed to run "%s %s: %v"`, cmd, strings.Join(args, " "), err)
}

// Exec calls [Config.Exec] on [Major]
func Exec(cmd string, args ...string) (ran bool, err error) {
	return Major().Exec(cmd, args...)
}

func (c *Config) run(cmd string, args ...string) (ran bool, code int, err error) {
	cm := exec.Command(cmd, args...)
	cm.Env = os.Environ()
	for k, v := range c.Env {
		cm.Env = append(cm.Env, k+"="+v)
	}
	// need to store in buffer so we can color and print commands and stdout correctly
	// (need to declare regardless even if we aren't using so that it is accessible)
	ebuf := &bytes.Buffer{}
	obuf := &bytes.Buffer{}
	if c.Buffer {
		cm.Stderr = ebuf
		cm.Stdout = obuf
	} else {
		cm.Stderr = c.Stderr
		cm.Stdout = c.Stdout
	}
	// need to do now because we aren't buffering, or we are guaranteed to print them
	// regardless of whether there is an error anyway, so we should print it now so
	// people can see it earlier (especially important if it runs for a long time).
	if !c.Buffer || c.Commands != nil {
		c.PrintCmd(cmd+" "+strings.Join(args, " "), err)
	}

	cm.Stdin = c.Stdin
	cm.Dir = c.Dir

	if !c.PrintOnly {
		err = cm.Run()

		// we must call InitColor after calling a system command
		// TODO(kai): maybe figure out a better solution to this
		// or expand this list
		if cmd == "cp" || cmd == "ls" || cmd == "mv" {
			grog.InitColor()
		}
	}

	if c.Buffer {
		// if we have an error, we print the commands and stdout regardless of the config info
		// (however, we don't print the command if we are guaranteed to print it regardless, as
		// we already printed it above in that case)
		if c.Commands == nil {
			c.PrintCmd(cmd+" "+strings.Join(args, " "), err)
		}
		sout := c.GetWriter(c.Stdout, err)
		if sout != nil {
			sout.Write(obuf.Bytes())
		}
		estr := ebuf.String()
		if estr != "" && c.Stderr != nil {
			c.Stderr.Write([]byte(grog.ErrorColor(estr)))
		}
	}
	return CmdRan(err), ExitStatus(err), err
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
	var ee *exec.ExitError
	ok := errors.As(err, &ee)
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
	var e *exec.ExitError
	if errors.As(err, &e) {
		if ex, ok := e.Sys().(exitStatus); ok {
			return ex.ExitStatus()
		}
	}
	return 1
}
