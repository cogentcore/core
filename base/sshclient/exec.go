// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"os"
	"path/filepath"
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
func (cl *Client) Exec(sio *exec.StdIOState, start, output bool, cmd string, args ...string) (string, error) {
	ses, err := cl.NewSession()
	if err != nil {
		return "", err
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
	return cl.run(ses, sio, start, output, cmd, args...)
}

func (cl *Client) run(ses *ssh.Session, sio *exec.StdIOState, start, output bool, cmd string, args ...string) (string, error) {
	// todo: env is established on connection!
	// for k, v := range cl.Env {
	// 	ses.Env = append(ses.Env, k+"="+v)
	// }
	var err error
	out := ""
	ses.Stderr = sio.Err // note: no need to save previous b/c not retained
	ses.Stdout = sio.Out
	if cl.Echo != nil {
		cl.PrintCmd(cmd+" "+strings.Join(args, " "), err)
	}
	if exec.IsPipe(sio.In) {
		ses.Stdin = sio.In
	}

	cdto := ""
	cmds := cmd + " " + strings.Join(args, " ")
	if cl.Dir != "" {
		if cmd == "cd" {
			if len(args) > 0 {
				cdto = args[0]
			} else {
				cdto = cl.HomeDir
			}
		}
		cmds = `cd '` + cl.Dir + `'; ` + cmds
	}

	if !cl.PrintOnly {
		switch {
		case start:
			err = ses.Start(cmds)
			go func() {
				ses.Wait()
				sio.PopToStart()
			}()
		case output:
			buf := &bytes.Buffer{}
			ses.Stdout = buf
			err = ses.Run(cmds)
			if sio.Out != nil {
				sio.Out.Write(buf.Bytes())
			}
			out = strings.TrimSuffix(buf.String(), "\n")
		default:
			err = ses.Run(cmds)
		}

		// we must call InitColor after calling a system command
		// TODO(kai): maybe figure out a better solution to this
		// or expand this list
		if cmd == "cp" || cmd == "ls" || cmd == "mv" {
			logx.InitColor()
		}

		if cdto != "" {
			if filepath.IsAbs(cdto) {
				cl.Dir = filepath.Clean(cdto)
			} else {
				cl.Dir = filepath.Clean(filepath.Join(cl.Dir, cdto))
			}
		}
	}
	return out, err
}
