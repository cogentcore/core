// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/sshclient"
	"github.com/mitchellh/go-homedir"
)

func (sh *Shell) HandleArgErr(errok bool, err error) error {
	if err == nil {
		return err
	}
	if errok {
		fmt.Fprintln(sh.Config.StdIO.Err, err.Error())
	} else {
		sh.AddError(err)
	}
	return err
}

// ExecArgs processes the args to given exec command,
// handling all of the input / output redirection and
// file globbing, homedir expansion, etc.
func (sh *Shell) ExecArgs(errok bool, cmd any, args ...any) (*sshclient.Client, string, []string) {
	scmd := reflectx.ToString(cmd)
	cl := sh.ActiveSSH()
	if len(args) == 0 {
		return cl, scmd, nil
	}
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
			sh.HandleArgErr(errok, err)
			// note: handling globbing in a later pass, to not clutter..
		} else {
			if s[0] == '~' {
				s = "$HOME/" + s[1:]
			}
		}
		sargs = append(sargs, s)
	}
	if scmd == "@" {
		newHost := ""
		if sargs[0] == "0" { // local
			cl = nil
		} else {
			if scl, ok := sh.SSHClients[sargs[0]]; ok {
				newHost = sargs[0]
				cl = scl
			} else {
				sh.HandleArgErr(errok, fmt.Errorf("cosh: ssh connection named: %q not found", sargs[0]))
			}
		}
		if len(sargs) > 1 {
			scmd = sargs[1]
			sargs = sargs[2:]
		} else { // just a ssh switch
			sh.SSHActive = newHost
			return nil, "", nil
		}
	}
	for i := 0; i < len(sargs); i++ { // we modify so no range
		s := sargs[i]
		switch {
		case s[0] == '>':
			sargs = sh.OutToFile(errok, sargs, i)
		case isCmd && strings.HasPrefix(s, "args"):
			sargs = sh.CmdArgs(errok, sargs, i)
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
func (sh *Shell) OutToFile(errok bool, sargs []string, i int) []string {
	n := len(sargs)
	s := sargs[i]
	sn := len(s)
	fn := ""
	narg := 1
	if i < n-1 {
		fn = sargs[1+1]
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
	fmt.Println(s, appn, errf, narg, fn)
	sargs = slices.Delete(sargs, i, i+narg)
	if fn == "" {
		sh.HandleArgErr(errok, fmt.Errorf("cosh: no output file specified"))
		return sargs
	}
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
		sh.Config.StdIO.PushOut(f)
		if errf {
			sh.Config.StdIO.PushErr(f)
		}
	} else {
		sh.HandleArgErr(errok, err)
	}
	return sargs
}

// CmdArgs processes expressions involving "args" for commands
func (sh *Shell) CmdArgs(errok bool, sargs []string, i int) []string {
	n := len(sargs)
	// s := sargs[i]
	// sn := len(s)
	args := sh.commandArgs.Peek()

	fmt.Println("command args:", args)

	switch {
	case i < n-1 && sargs[i+1] == "...":
		sargs = slices.Delete(sargs, i, i+2)
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
