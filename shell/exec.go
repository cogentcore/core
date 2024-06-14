// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/sshclient"
	"github.com/mitchellh/go-homedir"
)

// Exec handles command execution for all cases, parameterized by the args.
// It executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the shell, and triggers CancelExecution.
//   - errOk = don't call AddError so execution will not stop on error
//   - start = calls Start on the command, which then runs asynchronously, with
//     a goroutine forked to Wait for it and close its IO
//   - output = return the output of the command as a string (otherwise return is "")
func (sh *Shell) Exec(errOk, start, output bool, cmd any, args ...any) string {
	out := ""
	if !errOk && len(sh.Errors) > 0 {
		return out
	}
	cmdIO := exec.NewCmdIO(&sh.Config)
	cmdIO.StackStart()
	if start {
		cmdIO.PushIn(nil) // no stdin for bg
	}
	cl, scmd, sargs := sh.ExecArgs(cmdIO, errOk, cmd, args...)
	if scmd == "" {
		return out
	}
	var err error
	if cl != nil {
		switch {
		case start:
			err = cl.Start(&cmdIO.StdIOState, scmd, sargs...)
		case output:
			cmdIO.PushOut(nil)
			out, err = cl.Output(&cmdIO.StdIOState, scmd, sargs...)
		default:
			err = cl.Run(&cmdIO.StdIOState, scmd, sargs...)
		}
		if !errOk {
			sh.AddError(err)
		}
	} else {
		ran := false
		ran, out = sh.RunBuiltinOrCommand(cmdIO, errOk, output, scmd, sargs...)
		if !ran {
			sh.isCommand.Push(false)
			switch {
			case start:
				err = sh.Config.StartIO(cmdIO, scmd, sargs...)
				sh.Jobs.Push(cmdIO)
				go func() {
					if !cmdIO.OutIsPipe() {
						fmt.Printf("[%d]  %s\n", len(sh.Jobs), cmdIO.String())
					}
					cmdIO.Cmd.Wait()
					cmdIO.PopToStart()
					sh.DeleteJob(cmdIO)
				}()
			case output:
				cmdIO.PushOut(nil)
				out, err = sh.Config.OutputIO(cmdIO, scmd, sargs...)
			default:
				err = sh.Config.RunIO(cmdIO, scmd, sargs...)
			}
			if !errOk {
				sh.AddError(err)
			}
			sh.isCommand.Pop()
		}
	}
	if !start {
		cmdIO.PopToStart()
	}
	return out
}

// RunBuiltinOrCommand runs a builtin or a command, returning true if it ran,
// and the output string if running in output mode.
func (sh *Shell) RunBuiltinOrCommand(cmdIO *exec.CmdIO, errOk, output bool, cmd string, args ...string) (bool, string) {
	out := ""
	cmdFun, hasCmd := sh.Commands[cmd]
	bltFun, hasBlt := sh.Builtins[cmd]

	if !hasCmd && !hasBlt {
		return false, out
	}

	if hasCmd {
		sh.commandArgs.Push(args)
		sh.isCommand.Push(true)
	}

	// note: we need to set both os. and wrapper versions, so it works the same
	// in compiled vs. interpreted mode
	oldsh := sh.Config.StdIO.Set(&cmdIO.StdIO)
	oldwrap := sh.StdIOWrappers.SetWrappers(&cmdIO.StdIO)
	oldstd := cmdIO.SetToOS()
	if output {
		obuf := &bytes.Buffer{}
		// os.Stdout = obuf // needs a file
		sh.Config.StdIO.Out = obuf
		sh.StdIOWrappers.SetWrappedOut(obuf)
		cmdIO.PushOut(obuf)
		if hasCmd {
			cmdFun(args...)
		} else {
			sh.AddError(bltFun(cmdIO, args...))
		}
		out = strings.TrimSuffix(obuf.String(), "\n")
	} else {
		if hasCmd {
			cmdFun(args...)
		} else {
			sh.AddError(bltFun(cmdIO, args...))
		}
	}

	if hasCmd {
		sh.isCommand.Pop()
		sh.commandArgs.Pop()
	}
	oldstd.SetToOS()
	sh.StdIOWrappers.SetWrappers(oldwrap)
	sh.Config.StdIO = *oldsh

	return true, out
}

func (sh *Shell) HandleArgErr(errok bool, err error) error {
	if err == nil {
		return err
	}
	if errok {
		sh.Config.StdIO.ErrPrintln(err.Error())
	} else {
		sh.AddError(err)
	}
	return err
}

// ExecArgs processes the args to given exec command,
// handling all of the input / output redirection and
// file globbing, homedir expansion, etc.
func (sh *Shell) ExecArgs(cmdIO *exec.CmdIO, errOk bool, cmd any, args ...any) (*sshclient.Client, string, []string) {
	if len(sh.Jobs) > 0 {
		jb := sh.Jobs.Peek()
		if jb.OutIsPipe() {
			cmdIO.PushIn(jb.PipeIn.Peek())
		}
	}
	scmd := reflectx.ToString(cmd)
	cl := sh.ActiveSSH()
	isCmd := sh.isCommand.Peek()
	sargs := make([]string, 0, len(args))
	var err error
	for _, a := range args {
		s := reflectx.ToString(a)
		if s == "" {
			continue
		}
		if cl == nil {
			s, err = homedir.Expand(s)
			sh.HandleArgErr(errOk, err)
			// note: handling globbing in a later pass, to not clutter..
		} else {
			if s[0] == '~' {
				s = "$HOME/" + s[1:]
			}
		}
		sargs = append(sargs, s)
	}
	if scmd[0] == '@' {
		newHost := ""
		if scmd == "@0" { // local
			cl = nil
		} else {
			hnm := scmd[1:]
			if scl, ok := sh.SSHClients[hnm]; ok {
				newHost = hnm
				cl = scl
			} else {
				sh.HandleArgErr(errOk, fmt.Errorf("cosh: ssh connection named: %q not found", hnm))
			}
		}
		if len(sargs) > 0 {
			scmd = sargs[0]
			sargs = sargs[1:]
		} else { // just a ssh switch
			sh.SSHActive = newHost
			return nil, "", nil
		}
	}
	for i := 0; i < len(sargs); i++ { // we modify so no range
		s := sargs[i]
		switch {
		case s[0] == '>':
			sargs = sh.OutToFile(cl, cmdIO, errOk, sargs, i)
		case s[0] == '|':
			sargs = sh.OutToPipe(cl, cmdIO, errOk, sargs, i)
		case cl == nil && isCmd && strings.HasPrefix(s, "args"):
			sargs = sh.CmdArgs(errOk, sargs, i)
			i-- // back up because we consume this one
		}
	}
	// do globbing late here so we don't have to wade through everything.
	// only for local.
	if cl == nil {
		gargs := make([]string, 0, len(sargs))
		for _, s := range sargs {
			g, err := filepath.Glob(s)
			if err != nil || len(g) == 0 { // not valid
				gargs = append(gargs, s)
			} else {
				gargs = append(gargs, g...)
			}
		}
		sargs = gargs
	}
	return cl, scmd, sargs
}

// OutToFile processes the > arg that sends output to a file
func (sh *Shell) OutToFile(cl *sshclient.Client, cmdIO *exec.CmdIO, errOk bool, sargs []string, i int) []string {
	n := len(sargs)
	s := sargs[i]
	sn := len(s)
	fn := ""
	narg := 1
	if i < n-1 {
		fn = sargs[i+1]
		narg = 2
	}
	appn := false
	errf := false
	switch {
	case sn > 1 && s[1] == '>':
		appn = true
		if sn > 2 && s[2] == '&' {
			errf = true
		}
	case sn > 1 && s[1] == '&':
		errf = true
	case sn > 1:
		fn = s[1:]
		narg = 1
	}
	if fn == "" {
		sh.HandleArgErr(errOk, fmt.Errorf("cosh: no output file specified"))
		return sargs
	}
	if cl != nil {
		if !strings.HasPrefix(fn, "@0:") {
			return sargs
		}
		fn = fn[3:]
	}
	sargs = slices.Delete(sargs, i, i+narg)
	// todo: process @n: expressions here -- if @0 then it is the same
	// if @1, then need to launch an ssh "cat >[>] file" with pipe from command as stdin
	var f *os.File
	var err error
	if appn {
		f, err = os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	} else {
		f, err = os.Create(fn)
	}
	if err == nil {
		cmdIO.PushOut(f)
		if errf {
			cmdIO.PushErr(f)
		}
	} else {
		sh.HandleArgErr(errOk, err)
	}
	return sargs
}

// OutToPipe processes the | arg that sends output to a pipe
func (sh *Shell) OutToPipe(cl *sshclient.Client, cmdIO *exec.CmdIO, errOk bool, sargs []string, i int) []string {
	s := sargs[i]
	sn := len(s)
	errf := false
	if sn > 1 && s[1] == '&' {
		errf = true
	}
	// todo: what to do here?
	sargs = slices.Delete(sargs, i, i+1)
	cmdIO.PushOutPipe()
	if errf {
		cmdIO.PushErr(cmdIO.Out)
	}
	// sh.HandleArgErr(errok, err)
	return sargs
}

// CmdArgs processes expressions involving "args" for commands
func (sh *Shell) CmdArgs(errOk bool, sargs []string, i int) []string {
	// n := len(sargs)
	// s := sargs[i]
	// sn := len(s)
	args := sh.commandArgs.Peek()

	// fmt.Println("command args:", args)

	switch {
	case sargs[i] == "args...":
		sargs = slices.Delete(sargs, i, i+1)
		sargs = slices.Insert(sargs, i, args...)
	}

	return sargs
}

// CancelExecution calls the Cancel() function if set.
func (sh *Shell) CancelExecution() {
	if sh.Cancel != nil {
		sh.Cancel()
	}
}
