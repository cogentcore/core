// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"golang.org/x/crypto/ssh"
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
func (cl *Client) exec(cmd string, args ...string) (ran bool, err error) {
	ses, err := cl.NewSession()
	if err != nil {
		return
	}
	defer ses.Close()

	expand := func(s string) string {
		s2, ok := cl.Env[s]
		if ok {
			return s2
		}
		return os.Getenv(s)
	}
	// todo: what does this do?
	cmd = os.Expand(cmd, expand)
	for i := range args {
		args[i] = os.Expand(args[i], expand)
	}
	ran, code, err := cl.run(ses, cmd, args...)
	_ = code
	if err == nil {
		return true, nil
	}
	return ran, fmt.Errorf(`failed to run "%s %s: %v"`, cmd, strings.Join(args, " "), err)
}

func (cl *Client) run(ses *ssh.Session, cmd string, args ...string) (ran bool, code int, err error) {
	// todo: env is established on connection!
	// for k, v := range cl.Env {
	// 	ses.Env = append(ses.Env, k+"="+v)
	// }
	// need to store in buffer so we can color and print commands and stdout correctly
	// (need to declare regardless even if we aren't using so that it is accessible)
	ebuf := &bytes.Buffer{}
	obuf := &bytes.Buffer{}
	if cl.Buffer {
		ses.Stderr = ebuf
		ses.Stdout = obuf
	} else {
		ses.Stderr = cl.Stderr
		ses.Stdout = cl.Stdout
	}
	// need to do now because we aren't buffering, or we are guaranteed to print them
	// regardless of whether there is an error anyway, so we should print it now so
	// people can see it earlier (especially important if it runs for a long time).
	if !cl.Buffer || cl.Echo != nil {
		cl.PrintCmd(cmd+" "+strings.Join(args, " "), err)
	}

	// ses.Stdin = cl.Stdin
	// todo: add cd command first.
	// ses.Dir = cl.Dir

	cmds := cmd + " " + strings.Join(args, " ")

	if !cl.PrintOnly {
		err = ses.Run(cmds)

		// we must call InitColor after calling a system command
		// TODO(kai): maybe figure out a better solution to this
		// or expand this list
		if cmd == "cp" || cmd == "ls" || cmd == "mv" {
			logx.InitColor()
		}
	}

	if cl.Buffer {
		// if we have an error, we print the commands and stdout regardless of the config info
		// (however, we don't print the command if we are guaranteed to print it regardless, as
		// we already printed it above in that case)
		if cl.Echo == nil {
			cl.PrintCmd(cmds, err)
		}
		sout := cl.GetWriter(cl.Stdout, err)
		if sout != nil {
			sout.Write(obuf.Bytes())
		}
		estr := ebuf.String()
		if estr != "" && cl.Stderr != nil {
			cl.Stderr.Write([]byte(logx.ErrorColor(estr)))
		}
	}
	return exec.CmdRan(err), exec.ExitStatus(err), err
}
