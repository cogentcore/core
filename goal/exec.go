// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/sshclient"
	"github.com/mitchellh/go-homedir"
)

// Exec handles command execution for all cases, parameterized by the args.
// It executes the given command string, waiting for the command to finish,
// handling the given arguments appropriately.
// If there is any error, it adds it to the goal, and triggers CancelExecution.
//   - errOk = don't call AddError so execution will not stop on error
//   - start = calls Start on the command, which then runs asynchronously, with
//     a goroutine forked to Wait for it and close its IO
//   - output = return the output of the command as a string (otherwise return is "")
func (gl *Goal) Exec(errOk, start, output bool, cmd any, args ...any) string {
	out := ""
	if !errOk && len(gl.Errors) > 0 {
		return out
	}
	cmdIO := exec.NewCmdIO(&gl.Config)
	cmdIO.StackStart()
	if start {
		cmdIO.PushIn(nil) // no stdin for bg
	}
	cl, scmd, sargs := gl.ExecArgs(cmdIO, errOk, cmd, args...)
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
			gl.AddError(err)
		}
	} else {
		ran := false
		ran, out = gl.RunBuiltinOrCommand(cmdIO, errOk, start, output, scmd, sargs...)
		if !ran {
			gl.isCommand.Push(false)
			switch {
			case start:
				// fmt.Fprintf(gl.debugTrace, "start exe %s in: %#v  out: %#v  %v\n  ", scmd, cmdIO.In, cmdIO.Out, cmdIO.OutIsPipe())
				err = gl.Config.StartIO(cmdIO, scmd, sargs...)
				job := &Job{CmdIO: cmdIO}
				gl.Jobs.Push(job)
				go func() {
					if !cmdIO.OutIsPipe() {
						fmt.Printf("[%d]  %s\n", len(gl.Jobs), cmdIO.String())
					}
					cmdIO.Cmd.Wait()
					cmdIO.PopToStart()
					gl.DeleteJob(job)
				}()
			case output:
				cmdIO.PushOut(nil)
				out, err = gl.Config.OutputIO(cmdIO, scmd, sargs...)
			default:
				// fmt.Fprintf(gl.debugTrace, "run exe %s in: %#v  out: %#v  %v\n  ", scmd, cmdIO.In, cmdIO.Out, cmdIO.OutIsPipe())
				err = gl.Config.RunIO(cmdIO, scmd, sargs...)
			}
			if !errOk {
				gl.AddError(err)
			}
			gl.isCommand.Pop()
		}
	}
	if !start {
		cmdIO.PopToStart()
	}
	return out
}

// RunBuiltinOrCommand runs a builtin or a command, returning true if it ran,
// and the output string if running in output mode.
func (gl *Goal) RunBuiltinOrCommand(cmdIO *exec.CmdIO, errOk, start, output bool, cmd string, args ...string) (bool, string) {
	out := ""
	cmdFun, hasCmd := gl.Commands[cmd]
	bltFun, hasBlt := gl.Builtins[cmd]

	if !hasCmd && !hasBlt {
		return false, out
	}

	if hasCmd {
		gl.commandArgs.Push(args)
		gl.isCommand.Push(true)
	}

	// note: we need to set both os. and wrapper versions, so it works the same
	// in compiled vs. interpreted mode
	var oldsh, oldwrap, oldstd *exec.StdIO
	save := func() {
		oldsh = gl.Config.StdIO.Set(&cmdIO.StdIO)
		oldwrap = gl.StdIOWrappers.SetWrappers(&cmdIO.StdIO)
		oldstd = cmdIO.SetToOS()
	}

	done := func() {
		if hasCmd {
			gl.isCommand.Pop()
			gl.commandArgs.Pop()
		}
		// fmt.Fprintf(gl.debugTrace, "%s restore %#v\n", cmd, oldstd.In)
		oldstd.SetToOS()
		gl.StdIOWrappers.SetWrappers(oldwrap)
		gl.Config.StdIO = *oldsh
	}

	switch {
	case start:
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			if !cmdIO.OutIsPipe() {
				fmt.Printf("[%d]  %s\n", len(gl.Jobs), cmd)
			}
			if hasCmd {
				oldwrap = gl.StdIOWrappers.SetWrappers(&cmdIO.StdIO)
				// oldstd = cmdIO.SetToOS()
				// fmt.Fprintf(gl.debugTrace, "%s oldstd in: %#v  out: %#v\n", cmd, oldstd.In, oldstd.Out)
				cmdFun(args...)
				// oldstd.SetToOS()
				gl.StdIOWrappers.SetWrappers(oldwrap)
				gl.isCommand.Pop()
				gl.commandArgs.Pop()
			} else {
				gl.AddError(bltFun(cmdIO, args...))
			}
			time.Sleep(time.Millisecond)
			wg.Done()
		}()
		// fmt.Fprintf(gl.debugTrace, "%s push: %#v  out: %#v  %v\n", cmd, cmdIO.In, cmdIO.Out, cmdIO.OutIsPipe())
		job := &Job{CmdIO: cmdIO}
		gl.Jobs.Push(job)
		go func() {
			wg.Wait()
			cmdIO.PopToStart()
			gl.DeleteJob(job)
		}()
	case output:
		save()
		obuf := &bytes.Buffer{}
		// os.Stdout = obuf // needs a file
		gl.Config.StdIO.Out = obuf
		gl.StdIOWrappers.SetWrappedOut(obuf)
		cmdIO.PushOut(obuf)
		if hasCmd {
			cmdFun(args...)
		} else {
			gl.AddError(bltFun(cmdIO, args...))
		}
		out = strings.TrimSuffix(obuf.String(), "\n")
		done()
	default:
		save()
		if hasCmd {
			cmdFun(args...)
		} else {
			gl.AddError(bltFun(cmdIO, args...))
		}
		done()
	}
	return true, out
}

func (gl *Goal) HandleArgErr(errok bool, err error) error {
	if err == nil {
		return err
	}
	if errok {
		gl.Config.StdIO.ErrPrintln(err.Error())
	} else {
		gl.AddError(err)
	}
	return err
}

// ExecArgs processes the args to given exec command,
// handling all of the input / output redirection and
// file globbing, homedir expansion, etc.
func (gl *Goal) ExecArgs(cmdIO *exec.CmdIO, errOk bool, cmd any, args ...any) (*sshclient.Client, string, []string) {
	if len(gl.Jobs) > 0 {
		jb := gl.Jobs.Peek()
		if jb.OutIsPipe() && !jb.GotPipe {
			jb.GotPipe = true
			cmdIO.PushIn(jb.PipeIn.Peek())
		}
	}
	scmd := reflectx.ToString(cmd)
	cl := gl.ActiveSSH()
	// isCmd := gl.isCommand.Peek()
	sargs := make([]string, 0, len(args))
	var err error
	for _, a := range args {
		s := reflectx.ToString(a)
		if s == "" {
			continue
		}
		if cl == nil {
			s, err = homedir.Expand(s)
			gl.HandleArgErr(errOk, err)
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
			if scl, ok := gl.SSHClients[hnm]; ok {
				newHost = hnm
				cl = scl
			} else {
				gl.HandleArgErr(errOk, fmt.Errorf("goal: ssh connection named: %q not found", hnm))
			}
		}
		if len(sargs) > 0 {
			scmd = sargs[0]
			sargs = sargs[1:]
		} else { // just a ssh switch
			gl.SSHActive = newHost
			return nil, "", nil
		}
	}
	for i := 0; i < len(sargs); i++ { // we modify so no range
		s := sargs[i]
		switch {
		case s[0] == '>':
			sargs = gl.OutToFile(cl, cmdIO, errOk, sargs, i)
		case s[0] == '|':
			sargs = gl.OutToPipe(cl, cmdIO, errOk, sargs, i)
		case cl == nil && strings.HasPrefix(s, "args"):
			sargs = gl.CmdArgs(errOk, sargs, i)
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
func (gl *Goal) OutToFile(cl *sshclient.Client, cmdIO *exec.CmdIO, errOk bool, sargs []string, i int) []string {
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
		gl.HandleArgErr(errOk, fmt.Errorf("goal: no output file specified"))
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
		gl.HandleArgErr(errOk, err)
	}
	return sargs
}

// OutToPipe processes the | arg that sends output to a pipe
func (gl *Goal) OutToPipe(cl *sshclient.Client, cmdIO *exec.CmdIO, errOk bool, sargs []string, i int) []string {
	s := sargs[i]
	sn := len(s)
	errf := false
	if sn > 1 && s[1] == '&' {
		errf = true
	}
	sargs = slices.Delete(sargs, i, i+1)
	cmdIO.PushOutPipe()
	if errf {
		cmdIO.PushErr(cmdIO.Out)
	}
	return sargs
}

// CmdArgs processes expressions involving "args" for commands
func (gl *Goal) CmdArgs(errOk bool, sargs []string, i int) []string {
	// n := len(sargs)
	// s := sargs[i]
	// sn := len(s)
	args := gl.commandArgs.Peek()

	// fmt.Println("command args:", args)

	switch {
	case sargs[i] == "args...":
		sargs = slices.Delete(sargs, i, i+1)
		sargs = slices.Insert(sargs, i, args...)
	}

	return sargs
}

// CancelExecution calls the Cancel() function if set.
func (gl *Goal) CancelExecution() {
	if gl.Cancel != nil {
		gl.Cancel()
	}
}
