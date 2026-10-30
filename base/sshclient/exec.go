// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	return cl.ExecContext(nil, sio, start, output, cmd, args...)
}

// ExecContext is [Client.Exec] with a context: if it is not nil, a command
// that is run (not started) is interrupted when the context is done, as a
// shell does for Control+C: it gets an interrupt signal, and its session
// is closed if it has not ended after [exec.InterruptWait], which ends it
// with a hangup. A command that is run with its standard input from a
// terminal gets the input of that terminal while it runs, such that
// interactive programs such as more and editors work (not on Windows).
func (cl *Client) ExecContext(ctx context.Context, sio *exec.StdIOState, start, output bool, cmd string, args ...string) (string, error) {
	var tty *os.File
	if !start && !output {
		tty = terminalInput(sio.In)
	}
	ses, err := cl.newSession(tty)
	if err != nil {
		return "", err
	}
	defer ses.Close()

	expand := func(s string) string {
		s2, ok := cl.Env[s]
		if ok {
			return s2
		}
		switch s {
		case "!", "?":
			return "$" + s
		}
		return os.Getenv(s)
	}
	cmd = os.Expand(cmd, expand)
	for i := range args {
		args[i] = os.Expand(args[i], expand)
	}
	if ctx != nil && !start {
		done := make(chan struct{})
		defer close(done)
		go interruptOnDone(ctx, done, ses)
	}
	return cl.run(ses, sio, tty, start, output, cmd, args...)
}

// interruptOnDone interrupts the command of the given session when the
// given context is done, unless done is closed first (see
// [Client.ExecContext]).
func interruptOnDone(ctx context.Context, done chan struct{}, ses *ssh.Session) {
	select {
	case <-done:
		return
	case <-ctx.Done():
	}
	ses.Signal(ssh.SIGINT) // not all servers support this
	select {
	case <-done:
	case <-time.After(exec.InterruptWait):
		ses.Close()
	}
}

func (cl *Client) run(ses *ssh.Session, sio *exec.StdIOState, tty *os.File, start, output bool, cmd string, args ...string) (string, error) {
	for k, v := range cl.Env {
		ses.Setenv(k, v)
	}
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
	if tty != nil && !cl.PrintOnly {
		stop, err := forwardInput(ses, tty)
		if err != nil {
			return "", err
		}
		defer stop()
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
	cmds = cl.exports + cmds

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

// Shell runs an interactive login shell on the host, in the current
// directory of the client, with the terminal of the given standard input,
// whose input it gets while it runs (see [Client.ExecContext]), until it
// ends, as with exit or Control+D. Its exit status is not an error, as
// it is that of the last command in it. It needs a terminal (not on
// Windows).
func (cl *Client) Shell(sio *exec.StdIOState) error {
	tty := terminalInput(sio.In)
	if tty == nil {
		return errors.New("ssh: an interactive shell needs the input of a terminal")
	}
	ses, err := cl.newSession(tty)
	if err != nil {
		return err
	}
	defer ses.Close()
	for k, v := range cl.Env {
		ses.Setenv(k, v)
	}
	ses.Stdout, ses.Stderr = sio.Out, sio.Err
	stop, err := forwardInput(ses, tty)
	if err != nil {
		return err
	}
	defer stop()
	if cl.Dir != "" && cl.Dir != cl.HomeDir {
		err = ses.Start(`cd '` + cl.Dir + `' && exec "$SHELL" -l`)
	} else {
		err = ses.Shell()
	}
	if err != nil {
		return err
	}
	err = ses.Wait()
	var ee *ssh.ExitError
	if errors.As(err, &ee) {
		return nil
	}
	return err
}
